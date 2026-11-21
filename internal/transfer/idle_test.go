package transfer

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

// Regression: opening a trusted device in the application connects to it only
// to check trust, and closes the connection. The receiving device used to report
// that as a failed transfer. A connection that ends before any offer is not a
// transfer, so it must report ErrNoOffer, which the interface does not show.
func TestConnectionWithoutOfferIsNotATransfer(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	_ = sender.Close()

	_, err := receiverFor(t, t.TempDir(), acceptAll).Serve(ctx, receiver)
	if !errors.Is(err, apperr.ErrNoOffer) {
		t.Fatalf("Serve after a connection without offer = %v, want ErrNoOffer", err)
	}
}
