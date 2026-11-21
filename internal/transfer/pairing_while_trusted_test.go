package transfer

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
)

// Regression: when one device unpairs and the other still has it paired, the
// pairing that follows reached the trusted side as a pairing message in a
// transfer, and failed as an unexpected error. That case is named, so the
// application can ask whether to unpair on this side too.
func TestPairingMessageOnATrustedConnectionIsNamed(t *testing.T) {
	ctx := testCtx(t)
	p := newPairedParties(t, ctx)
	c1, c2 := tcpPair(t)

	// The sender has unpaired the receiver: its connection is not trusted.
	type result struct {
		s   *transport.Session
		err error
	}
	server := make(chan result, 1)
	go func() {
		s, err := transport.Server(ctx, c2, p.b, p.sb)
		server <- result{s, err}
	}()
	sender, err := transport.Client(ctx, c1, p.a, nil, newStore(t))
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	r := <-server
	if r.err != nil {
		t.Fatalf("server: %v", r.err)
	}
	if sender.State() == protocol.StateTrusted {
		t.Fatalf("the sender must not be trusted for this case")
	}
	if err := sender.Send(protocol.MsgPairConfirm, nil); err != nil {
		t.Fatalf("send: %v", err)
	}
	_, err = receiverFor(t, t.TempDir(), acceptAll).Serve(ctx, r.s)
	if !errors.Is(err, apperr.ErrPeerWantsToPair) {
		t.Fatalf("Serve = %v, want ErrPeerWantsToPair", err)
	}
}
