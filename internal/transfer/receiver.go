package transfer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"os"
	"time"
	"unicode/utf8"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
)

// Decision asks the person whether to accept an offer. It runs for every
// transfer, including those from trusted devices. It must honor ctx, which
// ends at the decision timeout.
type Decision func(ctx context.Context, offer Offer) bool

// ReceiverConfig configures a Receiver.
type ReceiverConfig struct {
	Decide Decision
	// StagingDir holds temporary files while data arrives. It is created with
	// 0700 permissions when missing.
	StagingDir string
	// DestDir receives completed files when Destination is nil.
	DestDir     string
	Destination Destination
	// MaxConcurrent limits simultaneous transfers. Zero means the default.
	MaxConcurrent int
	// OnProgress, when set, is called as data arrives. Optional.
	OnProgress func(ReceiveProgress)
}

// ReceiveProgress reports how much of the incoming offer has arrived.
type ReceiveProgress struct {
	OfferID  string
	Item     int
	Name     string
	Received int64
	Total    int64
}

// Receiver accepts transfers on trusted sessions.
type Receiver struct {
	cfg             ReceiverConfig
	limiter         *Limiter
	decisionTimeout time.Duration
	idleTimeout     time.Duration
}

// NewReceiver validates the configuration, prepares the staging directory and
// removes staging files abandoned by an earlier shutdown.
func NewReceiver(cfg ReceiverConfig) (*Receiver, error) {
	if cfg.Decide == nil {
		return nil, fmt.Errorf("receiver needs a decision function: %w", apperr.ErrMalformed)
	}
	if cfg.StagingDir == "" {
		return nil, fmt.Errorf("receiver needs a staging directory: %w", apperr.ErrMalformed)
	}
	if err := os.MkdirAll(cfg.StagingDir, 0o700); err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	if _, err := CleanupAbandoned(cfg.StagingDir, time.Hour, time.Now()); err != nil {
		return nil, err
	}
	if cfg.Destination == nil {
		cfg.Destination = DesktopDestination{Dir: cfg.DestDir}
	}
	n := cfg.MaxConcurrent
	if n == 0 {
		n = limits.MaxConcurrentTransfers
	}
	return &Receiver{
		cfg:             cfg,
		limiter:         NewLimiter(n),
		decisionTimeout: protocol.DecisionTimeout,
		idleTimeout:     protocol.IdleTimeout,
	}, nil
}

func (r *Receiver) active() int {
	return r.limiter.InUse()
}

// staged is one item being received into a temporary file.
type staged struct {
	name    string
	size    int64
	path    string
	file    *os.File
	hash    hash.Hash
	written int64
}

// Serve handles one offer on the session and returns when the transfer ends.
// It refuses sessions that are not trusted.
func (r *Receiver) Serve(ctx context.Context, s *transport.Session) (Result, error) {
	if s.State() != protocol.StateTrusted {
		return Result{}, fmt.Errorf("transfer on a %s session: %w", s.State(), apperr.ErrUnauthorized)
	}
	stop := context.AfterFunc(ctx, func() { _ = s.Close() })
	defer stop()

	if !r.limiter.TryAcquire() {
		_ = s.Send(protocol.MsgTransferDecline, declineBody{ID: "", Reason: reasonBusy})
		return Result{}, apperr.ErrBusy
	}
	defer r.limiter.Release()

	m := NewMachine()
	msg, err := s.ReceiveWithin(r.idleTimeout)
	if err != nil {
		if ctx.Err() != nil {
			return Result{}, contextOr(ctx, err)
		}
		if errors.Is(err, apperr.ErrPeerWantsToPair) {
			return Result{}, err
		}
		// Nothing was offered, so no transfer failed (see ErrNoOffer).
		return Result{}, fmt.Errorf("connection ended before an offer: %w", apperr.ErrNoOffer)
	}
	if msg.Type != protocol.MsgTransferOffer {
		return Result{}, fmt.Errorf("received %s instead of an offer: %w", msg.Type, apperr.ErrMalformed)
	}
	var body offerBody
	if err := msg.DecodeBody(&body); err != nil {
		return Result{}, r.reject(s, body.ID, reasonInvalid, err)
	}
	offer := Offer{ID: body.ID, Kind: body.Kind, Items: toItems(body.Items), Peer: s.PeerKey()}
	if err := ValidateOffer(offer); err != nil {
		return Result{}, r.reject(s, body.ID, reasonInvalid, err)
	}
	_ = m.Move(StateOffered)

	dctx, cancel := context.WithTimeout(ctx, r.decisionTimeout)
	accepted := r.cfg.Decide(dctx, offer)
	cancel()
	if !accepted {
		_ = m.Move(StateDeclined)
		_ = s.Send(protocol.MsgTransferDecline, declineBody{ID: offer.ID, Reason: reasonDeclined})
		return Result{}, apperr.ErrDeclined
	}
	if err := s.Send(protocol.MsgTransferAccept, idBody{ID: offer.ID}); err != nil {
		return Result{}, contextOr(ctx, err)
	}
	_ = m.Move(StateAccepted)
	_ = m.Move(StateTransferring)

	items, err := r.receiveItems(ctx, s, offer)
	if err != nil {
		_ = m.Move(StateFailed)
		discard(items)
		return Result{}, r.fail(s, offer.ID, err)
	}
	_ = m.Move(StateFinishing)

	res, err := r.complete(offer, items)
	if err != nil {
		_ = m.Move(StateFailed)
		return Result{}, r.fail(s, offer.ID, err)
	}
	if err := s.Send(protocol.MsgTransferResult, resultBody{ID: offer.ID, OK: true}); err != nil {
		return Result{}, contextOr(ctx, err)
	}
	_ = m.Move(StateDone)
	return res, nil
}

// receiveItems reads DATA and END messages until DONE, staging every item.
func (r *Receiver) receiveItems(ctx context.Context, s *transport.Session, offer Offer) ([]*staged, error) {
	items, err := createStaging(r.cfg.StagingDir, offer)
	if err != nil {
		return items, err
	}
	rc := &receiveState{offerID: offer.ID, items: items}
	if r.cfg.OnProgress != nil {
		rc.onProgress = func(id string, item int, name string, written, size, total int64) {
			r.cfg.OnProgress(ReceiveProgress{OfferID: id, Item: item, Name: name, Received: rc.received(item, written), Total: total})
		}
	}
	for {
		msg, err := s.ReceiveWithin(r.idleTimeout)
		if err != nil {
			return items, contextOr(ctx, err)
		}
		done, err := rc.handle(msg)
		if err != nil || done {
			return items, err
		}
	}
}

func createStaging(dir string, offer Offer) ([]*staged, error) {
	items := make([]*staged, len(offer.Items))
	for i, it := range offer.Items {
		f, err := os.CreateTemp(dir, stagingPattern)
		if err != nil {
			return items, fmt.Errorf("create staging file: %w", err)
		}
		items[i] = &staged{name: it.Name, size: it.Size, path: f.Name(), file: f, hash: sha256.New()}
	}
	return items, nil
}

// receiveState tracks which item the sender is on.
type receiveState struct {
	offerID    string
	items      []*staged
	current    int
	onProgress func(offerID string, item int, name string, written, size, total int64)
}

// total is the size of the whole offer.
func (rc *receiveState) total() int64 {
	var sum int64
	for _, it := range rc.items {
		sum += it.size
	}
	return sum
}

// received is the bytes of the whole offer received so far.
func (rc *receiveState) received(item int, written int64) int64 {
	var sum int64
	for i, it := range rc.items {
		if i < item {
			sum += it.size
		}
	}
	return sum + written
}

// handle applies one incoming message. It reports true when the transfer is over.
func (rc *receiveState) handle(msg protocol.Message) (bool, error) {
	switch msg.Type {
	case protocol.MsgTransferData:
		return false, rc.data(msg)
	case protocol.MsgTransferEnd:
		return false, rc.end(msg)
	case protocol.MsgTransferDone:
		return true, rc.done(msg)
	case protocol.MsgTransferCancel:
		return true, apperr.ErrCanceled
	default:
		return true, fmt.Errorf("unexpected %s during transfer: %w", msg.Type, apperr.ErrMalformed)
	}
}

func (rc *receiveState) data(msg protocol.Message) error {
	var d dataBody
	if err := msg.DecodeBody(&d); err != nil || d.ID != rc.offerID || d.Index != rc.current || rc.current >= len(rc.items) {
		return fmt.Errorf("unexpected data message: %w", apperr.ErrMalformed)
	}
	it := rc.items[rc.current]
	if it.written+int64(len(d.Data)) > it.size {
		return fmt.Errorf("%s exceeds its declared size: %w", it.name, apperr.ErrMalformed)
	}
	if _, err := it.file.Write(d.Data); err != nil {
		return fmt.Errorf("write staging file: %w", err)
	}
	_, _ = it.hash.Write(d.Data)
	it.written += int64(len(d.Data))
	if rc.onProgress != nil {
		rc.onProgress(rc.offerID, rc.current, it.name, it.written, it.size, rc.total())
	}
	return nil
}

func (rc *receiveState) end(msg protocol.Message) error {
	var e endBody
	if err := msg.DecodeBody(&e); err != nil || e.ID != rc.offerID || e.Index != rc.current || rc.current >= len(rc.items) {
		return fmt.Errorf("unexpected end message: %w", apperr.ErrMalformed)
	}
	it := rc.items[rc.current]
	if e.Size != it.size || it.written != it.size {
		return fmt.Errorf("%s has %d bytes, declared %d: %w", it.name, it.written, it.size, apperr.ErrMalformed)
	}
	if hex.EncodeToString(it.hash.Sum(nil)) != e.SHA256 {
		return fmt.Errorf("checksum of %s does not match: %w", it.name, apperr.ErrIntegrity)
	}
	if err := it.file.Close(); err != nil {
		return fmt.Errorf("close staging file: %w", err)
	}
	it.file = nil
	rc.current++
	return nil
}

func (rc *receiveState) done(msg protocol.Message) error {
	var d idBody
	if err := msg.DecodeBody(&d); err != nil || d.ID != rc.offerID || rc.current != len(rc.items) {
		return fmt.Errorf("transfer ended before every item arrived: %w", apperr.ErrMalformed)
	}
	return nil
}

// complete verifies and publishes the staged items. Files are published only
// when every item is intact; if one publish fails, the ones already published
// are removed, so the transfer is all or nothing.
func (r *Receiver) complete(offer Offer, items []*staged) (Result, error) {
	res := Result{OfferID: offer.ID}
	if offer.Kind == KindFile {
		for _, it := range items {
			final, err := r.cfg.Destination.Publish(it.path, it.name)
			if err != nil {
				for _, done := range res.Files {
					_ = os.Remove(done)
				}
				return Result{}, err
			}
			res.Files = append(res.Files, final)
		}
		return res, nil
	}
	data, err := os.ReadFile(items[0].path) //nolint:gosec // G304: staging file created by this receiver
	_ = os.Remove(items[0].path)
	if err != nil {
		return Result{}, fmt.Errorf("read staged text: %w", err)
	}
	if len(data) > limits.MaxTextBytes || !utf8.Valid(data) {
		return Result{}, fmt.Errorf("received %s is not valid text: %w", offer.Kind, apperr.ErrMalformed)
	}
	if offer.Kind == KindLink {
		if err := ValidateLink(string(bytes.TrimSpace(data))); err != nil {
			return Result{}, err
		}
	}
	res.Text = string(data)
	return res, nil
}

func (r *Receiver) reject(s *transport.Session, id, reason string, cause error) error {
	_ = s.Send(protocol.MsgTransferDecline, declineBody{ID: id, Reason: reason})
	return fmt.Errorf("offer rejected: %w", errors.Join(cause, apperr.ErrMalformed))
}

// fail removes staged data, tells the sender why, and returns the error.
func (r *Receiver) fail(s *transport.Session, id string, err error) error {
	reason := reasonCanceled
	switch {
	case errors.Is(err, apperr.ErrIntegrity):
		reason = reasonIntegrity
	case errors.Is(err, apperr.ErrMalformed), errors.Is(err, apperr.ErrLimitExceeded):
		reason = reasonSize
	}
	_ = s.Send(protocol.MsgTransferResult, resultBody{ID: id, OK: false, Reason: reason})
	return err
}

func discard(items []*staged) {
	for _, it := range items {
		if it == nil {
			continue
		}
		if it.file != nil {
			_ = it.file.Close()
		}
		_ = os.Remove(it.path)
	}
}
