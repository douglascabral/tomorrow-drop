package protocol

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func TestProvisionalPolicyAllowsOnlyPairing(t *testing.T) {
	if !AllowedIn(StateAwaitingVerification, MsgPairConfirm) {
		t.Fatalf("pairing should be allowed in a provisional state")
	}
	if AllowedIn(StateAwaitingVerification, MsgTransferOffer) {
		t.Fatalf("transfer must not be allowed in a provisional state")
	}
}

func TestTrustedPolicyAllowsTransferWithoutPairing(t *testing.T) {
	if !AllowedIn(StateTrusted, MsgTransferOffer) {
		t.Fatalf("transfer should be allowed in a trusted state")
	}
	if AllowedIn(StateTrusted, MsgPairConfirm) {
		t.Fatalf("pairing must not happen again in a trusted state")
	}
}

func TestPolicyBlocksEverythingWithoutEstablishedSession(t *testing.T) {
	for _, state := range []PeerState{StateDiscovered, StateDisconnected, StateIdentityChanged} {
		for _, kind := range []MessageType{MsgHello, MsgPairConfirm, MsgTransferOffer} {
			if AllowedIn(state, kind) {
				t.Errorf("%s allowed %s", state, kind)
			}
		}
	}
}

func TestConnectingPolicyAllowsOnlyHello(t *testing.T) {
	if !AllowedIn(StateConnecting, MsgHello) {
		t.Fatalf("HELLO should be allowed while connecting")
	}
	if AllowedIn(StateConnecting, MsgPairConfirm) {
		t.Fatalf("pairing must not happen before the connection is established")
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	data, err := EncodeMessage(MsgHello, Hello{Versions: []Version{1}})
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	msg, err := DecodeMessage(data)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	if msg.Type != MsgHello {
		t.Fatalf("type = %s, want HELLO", msg.Type)
	}
	var h Hello
	if err := msg.DecodeBody(&h); err != nil || len(h.Versions) != 1 || h.Versions[0] != 1 {
		t.Fatalf("body = %+v (%v), want versions [1]", h, err)
	}
}

func TestDecodeRejectsUnknownType(t *testing.T) {
	_, err := DecodeMessage([]byte(`{"type":"RUN_COMMAND"}`))
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("unknown type = %v, want ErrMalformed", err)
	}
}

func TestDecodeRejectsInvalidJSON(t *testing.T) {
	_, err := DecodeMessage([]byte(`{not json`))
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("invalid JSON = %v, want ErrMalformed", err)
	}
}

func TestEncodeRejectsUnknownType(t *testing.T) {
	if _, err := EncodeMessage(MessageType("MADE_UP"), nil); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("EncodeMessage made-up type = %v, want ErrMalformed", err)
	}
}
