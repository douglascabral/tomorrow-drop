package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/trust"
)

// Session is a TLS connection with a security state and a message policy.
type Session struct {
	conn    *tls.Conn
	local   ed25519.PublicKey
	peer    ed25519.PublicKey
	role    trust.Role
	version protocol.Version
	state   protocol.PeerState
}

// Client opens the session as the initiator. If pin is not nil, the peer must
// present exactly that key, otherwise the handshake fails with ErrIdentityChanged.
func Client(ctx context.Context, conn net.Conn, id *identity.Identity, pin ed25519.PublicKey, trustList TrustLookup) (*Session, error) {
	return connect(ctx, conn, trust.Initiator, id, pin, trustList)
}

// Server accepts the session as the responder. It requires the client's certificate.
func Server(ctx context.Context, conn net.Conn, id *identity.Identity, trustList TrustLookup) (*Session, error) {
	return connect(ctx, conn, trust.Responder, id, nil, trustList)
}

func connect(ctx context.Context, conn net.Conn, role trust.Role, id *identity.Identity, pin ed25519.PublicKey, trustList TrustLookup) (*Session, error) {
	cert, err := certificateFor(id)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return establish(ctx, conn, role, cert, id.PublicKey(), pin, trustList)
}

func establish(ctx context.Context, conn net.Conn, role trust.Role, cert tls.Certificate, local, pin ed25519.PublicKey, trustList TrustLookup) (*Session, error) {
	cfg := tlsConfig(cert, pin)
	var tc *tls.Conn
	if role == trust.Initiator {
		tc = tls.Client(conn, cfg)
	} else {
		tc = tls.Server(conn, cfg)
	}

	handshakeCtx, cancel := context.WithTimeout(ctx, protocol.HandshakeTimeout)
	defer cancel()
	if err := tc.HandshakeContext(handshakeCtx); err != nil {
		_ = conn.Close()
		return nil, contextOr(handshakeCtx, fmt.Errorf("TLS handshake: %w", err))
	}

	peerKey, err := peerKey(tc)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	s := &Session{conn: tc, local: local, peer: peerKey, role: role, state: protocol.StateAwaitingVerification}
	if trustList != nil && trustList.IsTrusted(peerKey) {
		s.state = protocol.StateTrusted
	}
	if err := s.negotiate(handshakeCtx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func peerKey(tc *tls.Conn) (ed25519.PublicKey, error) {
	state := tc.ConnectionState()
	if len(state.PeerCertificates) != 1 {
		return nil, fmt.Errorf("peer certificate missing: %w", apperr.ErrMalformed)
	}
	key, ok := state.PeerCertificates[0].PublicKey.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("peer key is not Ed25519: %w", apperr.ErrMalformed)
	}
	return key, nil
}

// negotiate exchanges HELLO and chooses the highest common version. The
// exchange happens inside TLS, so an attacker cannot alter the version list.
func (s *Session) negotiate(ctx context.Context) error {
	reply, err := s.exchange(ctx, protocol.MsgHello, protocol.Hello{Versions: protocol.SupportedVersions})
	if err != nil {
		return err
	}
	if reply.Type != protocol.MsgHello {
		return fmt.Errorf("received %s instead of HELLO: %w", reply.Type, apperr.ErrMalformed)
	}
	var hello protocol.Hello
	if err := reply.DecodeBody(&hello); err != nil {
		return err
	}
	version, err := protocol.Negotiate(protocol.SupportedVersions, hello.Versions)
	if err != nil {
		return err
	}
	s.version = version
	return nil
}

// exchange sends a message and waits for the peer's reply. Sending runs in
// parallel with reading, so the exchange does not stall on synchronous connections.
func (s *Session) exchange(ctx context.Context, kind protocol.MessageType, body any) (protocol.Message, error) {
	stop := context.AfterFunc(ctx, func() { _ = s.conn.Close() })
	defer stop()

	var sendErr error
	sent := make(chan struct{})
	go func() {
		sendErr = s.Send(kind, body)
		close(sent)
	}()

	msg, err := s.Receive()
	if err != nil {
		_ = s.conn.Close()
		<-sent
		return protocol.Message{}, contextOr(ctx, err)
	}
	<-sent
	if sendErr != nil {
		_ = s.conn.Close()
		return protocol.Message{}, contextOr(ctx, sendErr)
	}
	return msg, nil
}

// Send sends a message if the current state allows its type.
func (s *Session) Send(kind protocol.MessageType, body any) error {
	if !protocol.AllowedIn(s.state, kind) {
		return fmt.Errorf("%s not allowed in %s: %w", kind, s.state, apperr.ErrUnauthorized)
	}
	data, err := protocol.EncodeMessage(kind, body)
	if err != nil {
		return err
	}
	return protocol.WriteFrame(s.conn, data)
}

// Receive reads a message. Types not allowed in the current state close the
// connection with ErrUnauthorized.
func (s *Session) Receive() (protocol.Message, error) {
	data, err := protocol.ReadFrame(s.conn)
	if err != nil {
		return protocol.Message{}, err
	}
	msg, err := protocol.DecodeMessage(data)
	if err != nil {
		return protocol.Message{}, err
	}
	if !protocol.AllowedIn(s.state, msg.Type) {
		_ = s.conn.Close()
		return protocol.Message{}, fmt.Errorf("%s received in %s: %w", msg.Type, s.state, apperr.ErrUnauthorized)
	}
	return msg, nil
}

// ReceiveWithin is Receive with a deadline. It fails with apperr.ErrTimeout if
// no message arrives within d. After a timeout the connection must be closed.
func (s *Session) ReceiveWithin(d time.Duration) (protocol.Message, error) {
	if err := s.conn.SetReadDeadline(time.Now().Add(d)); err != nil {
		return protocol.Message{}, fmt.Errorf("set read deadline: %w", err)
	}
	defer func() { _ = s.conn.SetReadDeadline(time.Time{}) }()
	msg, err := s.Receive()
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return protocol.Message{}, fmt.Errorf("no message within %s: %w", d, apperr.ErrTimeout)
	}
	return msg, err
}

// Promote marks the session as trusted if the peer's key is in the trust list.
func (s *Session) Promote(trustList TrustLookup) error {
	if trustList == nil || !trustList.IsTrusted(s.peer) {
		return fmt.Errorf("untrusted peer: %w", apperr.ErrUnauthorized)
	}
	s.state = protocol.StateTrusted
	return nil
}

// Pairing starts pairing. It is allowed only in a provisional connection.
func (s *Session) Pairing() (*trust.Pairing, error) {
	if s.state != protocol.StateAwaitingVerification {
		return nil, fmt.Errorf("pairing in %s: %w", s.state, apperr.ErrUnauthorized)
	}
	tlsState := s.conn.ConnectionState()
	binding, err := tlsState.ExportKeyingMaterial(bindingLabel, nil, bindingSize)
	if err != nil {
		return nil, fmt.Errorf("session binding: %w", err)
	}
	return trust.NewPairing(s.role, s.local, s.peer, binding, s.version)
}

// PeerKey returns the peer's authenticated public key.
func (s *Session) PeerKey() ed25519.PublicKey {
	return s.peer
}

// State returns the current security state.
func (s *Session) State() protocol.PeerState {
	return s.state
}

// Close closes the connection.
func (s *Session) Close() error {
	s.state = protocol.StateDisconnected
	return s.conn.Close()
}

// contextOr replaces connection errors caused by cancellation with the context error.
func contextOr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return apperr.FromContext(ctx.Err())
	}
	return err
}
