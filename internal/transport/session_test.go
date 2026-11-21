package transport

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/trust"
)

const testTimeout = 10 * time.Second

func testContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func newID(t *testing.T) *identity.Identity {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return id
}

func newTrustStore(t *testing.T) *trust.Store {
	t.Helper()
	return trust.Open(filepath.Join(t.TempDir(), "trust.json"))
}

// dial connects a client and a server over an in-memory TCP pair.
func dial(t *testing.T, ctx context.Context, client, server *identity.Identity, pin ed25519.PublicKey, clientTrust, serverTrust TrustLookup) (cs, ss *Session, errCli, errSrv error) {
	t.Helper()
	c1, c2 := pipeTCP(t)
	type result struct {
		s   *Session
		err error
	}
	ch := make(chan result, 1)
	go func() {
		s, err := Server(ctx, c2, server, serverTrust)
		ch <- result{s, err}
	}()
	cs, errCli = Client(ctx, c1, client, pin, clientTrust)
	r := <-ch
	return cs, r.s, errCli, r.err
}

// pairDevices runs the full pairing between a and b and makes each trust the other.
func pairDevices(t *testing.T, ctx context.Context, a, b *identity.Identity, storeA, storeB *trust.Store) {
	t.Helper()
	ca, cb, errA, errB := dial(t, ctx, a, b, nil, storeA, storeB)
	if errA != nil || errB != nil {
		t.Fatalf("provisional connection: %v / %v", errA, errB)
	}
	pa, err := ca.Pairing()
	if err != nil {
		t.Fatalf("Pairing a: %v", err)
	}
	pb, err := cb.Pairing()
	if err != nil {
		t.Fatalf("Pairing b: %v", err)
	}
	pa.ConfirmLocal()
	pb.ConfirmLocal()

	ch := make(chan error, 2)
	go func() { ch <- ExchangePairing(ctx, ca, pa) }()
	go func() { ch <- ExchangePairing(ctx, cb, pb) }()
	for i := 0; i < 2; i++ {
		if err := <-ch; err != nil {
			t.Fatalf("confirmation exchange: %v", err)
		}
	}
	if err := trust.CommitPairing(storeA, pa, "b"); err != nil {
		t.Fatalf("commit a: %v", err)
	}
	if err := trust.CommitPairing(storeB, pb, "a"); err != nil {
		t.Fatalf("commit b: %v", err)
	}
	_ = ca.Close()
	_ = cb.Close()
}

func TestTLSRequiresVersion13(t *testing.T) {
	cert, err := certificateFor(newID(t))
	if err != nil {
		t.Fatal(err)
	}
	if cfg := tlsConfig(cert, nil); cfg.MinVersion != tls.VersionTLS13 {
		t.Fatalf("MinVersion = %x, want TLS 1.3", cfg.MinVersion)
	}
}

func TestMutualAuthenticationBindsIdentities(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)

	cs, ss, errCli, errSrv := dial(t, ctx, alice, bob, nil, newTrustStore(t), newTrustStore(t))
	if errCli != nil || errSrv != nil {
		t.Fatalf("handshake: client=%v server=%v", errCli, errSrv)
	}
	if !cs.PeerKey().Equal(bob.PublicKey()) {
		t.Fatalf("client saw the wrong server key")
	}
	if !ss.PeerKey().Equal(alice.PublicKey()) {
		t.Fatalf("server saw the wrong client key")
	}
	if cs.State() != protocol.StateAwaitingVerification {
		t.Fatalf("state = %s, want awaiting_verification", cs.State())
	}
	if cs.conn.ConnectionState().Version != tls.VersionTLS13 {
		t.Fatalf("negotiated TLS version is not 1.3")
	}
}

func TestServerRequiresClientCertificate(t *testing.T) {
	ctx := testContext(t)
	server := newID(t)
	store := newTrustStore(t)
	c1, c2 := pipeTCP(t)
	serverErr := make(chan error, 1)
	go func() {
		_, err := Server(ctx, c2, server, store)
		serverErr <- err
	}()

	client := tls.Client(c1, &tls.Config{MinVersion: tls.VersionTLS13, InsecureSkipVerify: true}) //nolint:gosec // test: client without certificate
	_ = client.HandshakeContext(ctx)
	if err := <-serverErr; err == nil {
		t.Fatalf("server accepted a client without a certificate")
	}
}

func TestPinRejectsDifferentKey(t *testing.T) {
	ctx := testContext(t)
	bob := newID(t)
	impostor := newID(t)

	_, _, errCli, _ := dial(t, ctx, newID(t), impostor, bob.PublicKey(), newTrustStore(t), newTrustStore(t))
	if !errors.Is(errCli, apperr.ErrIdentityChanged) {
		t.Fatalf("substituted key = %v, want ErrIdentityChanged", errCli)
	}
}

func TestPinAcceptsExpectedKey(t *testing.T) {
	ctx := testContext(t)
	bob := newID(t)

	_, _, errCli, errSrv := dial(t, ctx, newID(t), bob, bob.PublicKey(), newTrustStore(t), newTrustStore(t))
	if errCli != nil || errSrv != nil {
		t.Fatalf("correct key rejected: %v / %v", errCli, errSrv)
	}
}

func TestRejectsCertificateWithoutPrivateKeyPossession(t *testing.T) {
	ctx := testContext(t)
	victim := newID(t)
	victimCert, err := identity.NewCertificate(victim, time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	_, attackerKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	// The victim's certificate, paired with the attacker's private key.
	forged := tls.Certificate{Certificate: victimCert.Certificate, PrivateKey: attackerKey}
	store := newTrustStore(t)

	c1, c2 := pipeTCP(t)
	go func() {
		_, _ = establish(ctx, c2, trust.Responder, forged, victim.PublicKey(), nil, store)
	}()
	if _, err := Client(ctx, c1, newID(t), victim.PublicKey(), newTrustStore(t)); err == nil {
		t.Fatalf("certificate without possession of the private key was accepted")
	}
}

func TestRejectsIncompatibleVersionInHello(t *testing.T) {
	ctx := testContext(t)
	cert, err := certificateFor(newID(t))
	if err != nil {
		t.Fatal(err)
	}
	c1, c2 := pipeTCP(t)
	go func() {
		tc := tls.Server(c2, tlsConfig(cert, nil))
		if err := tc.HandshakeContext(ctx); err != nil {
			return
		}
		data, _ := protocol.EncodeMessage(protocol.MsgHello, protocol.Hello{Versions: []protocol.Version{99}})
		_ = protocol.WriteFrame(tc, data)
		_, _ = protocol.ReadFrame(tc)
	}()

	if _, err := Client(ctx, c1, newID(t), nil, newTrustStore(t)); !errors.Is(err, apperr.ErrVersionMismatch) {
		t.Fatalf("incompatible version = %v, want ErrVersionMismatch", err)
	}
}

func TestProvisionalBlocksTransfer(t *testing.T) {
	ctx := testContext(t)
	cs, _, errCli, errSrv := dial(t, ctx, newID(t), newID(t), nil, newTrustStore(t), newTrustStore(t))
	if errCli != nil || errSrv != nil {
		t.Fatalf("handshake: %v / %v", errCli, errSrv)
	}
	if err := cs.Send(protocol.MsgTransferOffer, map[string]string{"name": "x"}); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("transfer in a provisional connection = %v, want ErrUnauthorized", err)
	}
}

func TestProvisionalRejectsForgedIncomingTransfer(t *testing.T) {
	ctx := testContext(t)
	cs, ss, _, _ := dial(t, ctx, newID(t), newID(t), nil, newTrustStore(t), newTrustStore(t))

	// A malicious peer ignores the local policy and sends a transfer offer.
	data, _ := protocol.EncodeMessage(protocol.MsgTransferOffer, map[string]string{"name": "x"})
	go func() { _ = protocol.WriteFrame(ss.conn, data) }()

	if _, err := cs.Receive(); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Receive of a disallowed type = %v, want ErrUnauthorized", err)
	}
}

func TestPromoteRequiresTrust(t *testing.T) {
	ctx := testContext(t)
	cs, _, _, _ := dial(t, ctx, newID(t), newID(t), nil, newTrustStore(t), newTrustStore(t))
	if err := cs.Promote(newTrustStore(t)); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Promote without trust = %v, want ErrUnauthorized", err)
	}
}

func TestCompletedPairingMakesConnectionTrustedAndReconnectionAllowsTransfer(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)
	storeA, storeB := newTrustStore(t), newTrustStore(t)

	pairDevices(t, ctx, alice, bob, storeA, storeB)

	cs, ss, errCli, errSrv := dial(t, ctx, alice, bob, nil, storeA, storeB)
	if errCli != nil || errSrv != nil {
		t.Fatalf("reconnection: %v / %v", errCli, errSrv)
	}
	if cs.State() != protocol.StateTrusted || ss.State() != protocol.StateTrusted {
		t.Fatalf("states after reconnection = %s / %s, want trusted", cs.State(), ss.State())
	}

	sent := make(chan error, 1)
	go func() {
		sent <- cs.Send(protocol.MsgTransferOffer, map[string]string{"name": "file.txt"})
	}()
	msg, err := ss.Receive()
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}
	if msg.Type != protocol.MsgTransferOffer {
		t.Fatalf("received type = %s, want TRANSFER_OFFER", msg.Type)
	}
	if err := <-sent; err != nil {
		t.Fatalf("Send on a trusted connection: %v", err)
	}
}

func TestTrustedConnectionBlocksPairing(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)
	storeA, storeB := newTrustStore(t), newTrustStore(t)
	pairDevices(t, ctx, alice, bob, storeA, storeB)

	cs, _, _, _ := dial(t, ctx, alice, bob, nil, storeA, storeB)
	if _, err := cs.Pairing(); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Pairing on a trusted connection = %v, want ErrUnauthorized", err)
	}
	if err := cs.Send(protocol.MsgPairConfirm, map[string]string{}); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("PAIR_CONFIRM on a trusted connection = %v, want ErrUnauthorized", err)
	}
}

func TestRevocationRequiresNewPairing(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)
	storeA, storeB := newTrustStore(t), newTrustStore(t)
	pairDevices(t, ctx, alice, bob, storeA, storeB)

	if err := storeA.Remove(bob.Fingerprint()); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	cs, _, errCli, errSrv := dial(t, ctx, alice, bob, nil, storeA, storeB)
	if errCli != nil || errSrv != nil {
		t.Fatalf("handshake after revocation: %v / %v", errCli, errSrv)
	}
	if cs.State() != protocol.StateAwaitingVerification {
		t.Fatalf("state after revocation = %s, want awaiting_verification", cs.State())
	}
	if err := cs.Send(protocol.MsgTransferOffer, map[string]string{}); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("transfer after revocation = %v, want ErrUnauthorized", err)
	}
}

func TestConfirmationFromPreviousSessionIsNotReused(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)

	ca1, cb1, _, _ := dial(t, ctx, alice, bob, nil, newTrustStore(t), newTrustStore(t))
	pb1, err := cb1.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	pb1.ConfirmLocal()
	tagSession1, err := pb1.LocalTag()
	if err != nil {
		t.Fatal(err)
	}
	_ = ca1.Close()
	_ = cb1.Close()

	// New connection between the same devices: a different TLS exporter, a different binding.
	ca2, _, _, _ := dial(t, ctx, alice, bob, nil, newTrustStore(t), newTrustStore(t))
	pa2, err := ca2.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	if err := pa2.VerifyRemote(tagSession1); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("confirmation from a previous session was accepted: %v", err)
	}
}

func TestUnilateralConfirmationDoesNotPersist(t *testing.T) {
	ctx := testContext(t)
	alice, bob := newID(t), newID(t)
	storeA := newTrustStore(t)
	ca, cb, _, _ := dial(t, ctx, alice, bob, nil, storeA, newTrustStore(t))

	pa, _ := ca.Pairing()
	pb, _ := cb.Pairing()
	pa.ConfirmLocal() // bob never confirms.

	shortCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	errA := ExchangePairing(shortCtx, ca, pa)
	errB := ExchangePairing(ctx, cb, pb)
	if errA == nil || errB == nil {
		t.Fatalf("unilateral exchange succeeded: alice=%v bob=%v", errA, errB)
	}
	if err := trust.CommitPairing(storeA, pa, "bob"); err == nil {
		t.Fatalf("commit after a unilateral confirmation was accepted")
	}
	if peers, _ := storeA.List(); len(peers) != 0 {
		t.Fatalf("trust persisted after a unilateral confirmation")
	}
}

func TestMITMRetransmissionOfConfirmationIsRejected(t *testing.T) {
	ctx := testContext(t)
	alice, bob, mallory := newID(t), newID(t), newID(t)
	storeAlice, storeBob := newTrustStore(t), newTrustStore(t)
	storeMalServer, storeMalClient := newTrustStore(t), newTrustStore(t)

	// Alice <-> Mallory <-> Bob: Mallory terminates two distinct TLS sessions.
	cA, mA := pipeTCP(t)
	mB, bB := pipeTCP(t)

	malServerCh := make(chan *Session, 1)
	malClientCh := make(chan *Session, 1)
	bobCh := make(chan *Session, 1)
	go func() {
		s, _ := Server(ctx, mA, mallory, storeMalServer)
		malServerCh <- s
	}()
	go func() {
		s, _ := Client(ctx, mB, mallory, nil, storeMalClient)
		malClientCh <- s
	}()
	go func() {
		s, _ := Server(ctx, bB, bob, storeBob)
		bobCh <- s
	}()
	alicePair, err := Client(ctx, cA, alice, nil, storeAlice)
	if err != nil {
		t.Fatalf("alice: %v", err)
	}
	malServer, malClient, bobPair := <-malServerCh, <-malClientCh, <-bobCh
	if malServer == nil || malClient == nil || bobPair == nil {
		t.Fatalf("handshakes failed")
	}

	// Mallory relays the pairing messages without altering them.
	go relay(malServer, malClient)
	go relay(malClient, malServer)

	pa, err := alicePair.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	pb, err := bobPair.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	pa.ConfirmLocal()
	pb.ConfirmLocal()

	ch := make(chan error, 2)
	go func() { ch <- ExchangePairing(ctx, alicePair, pa) }()
	go func() { ch <- ExchangePairing(ctx, bobPair, pb) }()
	for i := 0; i < 2; i++ {
		if err := <-ch; err == nil {
			t.Fatalf("pairing with Mallory's retransmission was accepted")
		}
	}
	if err := trust.CommitPairing(storeAlice, pa, "mallory"); err == nil {
		t.Fatalf("alice trusted after Mallory's retransmission")
	}
}

// relay forwards one pairing message from origin to destination, unaltered.
func relay(origin, destination *Session) {
	msg, err := origin.Receive()
	if err != nil {
		return
	}
	_ = destination.Send(msg.Type, msg.Body)
}

// Regression: with TLS session resumption enabled, Go does not call
// VerifyPeerCertificate on resumed connections, which would bypass the pinned key check.
func TestConfigDisablesSessionResumption(t *testing.T) {
	cert, err := certificateFor(newID(t))
	if err != nil {
		t.Fatal(err)
	}
	if cfg := tlsConfig(cert, nil); !cfg.SessionTicketsDisabled {
		t.Fatalf("TLS session resumption enabled: resumed connections would not verify the peer")
	}
}

// pipeTCP returns a pair of TCP connections over loopback. net.Pipe is
// synchronous and can stall TLS when one side sends an alert while the other
// is writing; the real transport is TCP, so the tests use TCP.
func pipeTCP(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = listener.Close() }()

	type accepted struct {
		conn net.Conn
		err  error
	}
	ch := make(chan accepted, 1)
	go func() {
		c, err := listener.Accept()
		ch <- accepted{c, err}
	}()
	client, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	r := <-ch
	if r.err != nil {
		t.Fatalf("accept: %v", r.err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = r.conn.Close()
	})
	return client, r.conn
}
