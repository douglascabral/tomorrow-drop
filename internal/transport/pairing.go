package transport

import (
	"context"
	"fmt"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/trust"
)

type confirmationBody struct {
	Tag []byte `json:"tag"`
}

// ExchangePairing sends the local confirmation and verifies the peer's. It is
// called only after ConfirmLocal. If the peer does not confirm, the exchange
// fails through the context.
func ExchangePairing(ctx context.Context, s *Session, p *trust.Pairing) error {
	tag, err := p.LocalTag()
	if err != nil {
		return err
	}
	reply, err := s.exchange(ctx, protocol.MsgPairConfirm, confirmationBody{Tag: tag})
	if err != nil {
		return err
	}
	if reply.Type != protocol.MsgPairConfirm {
		return fmt.Errorf("received %s instead of PAIR_CONFIRM: %w", reply.Type, apperr.ErrUnauthorized)
	}
	var body confirmationBody
	if err := reply.DecodeBody(&body); err != nil {
		return err
	}
	return p.VerifyRemote(body.Tag)
}
