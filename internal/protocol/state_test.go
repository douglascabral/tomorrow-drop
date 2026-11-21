package protocol

import "testing"

func TestStatesHaveStableLabels(t *testing.T) {
	cases := map[PeerState]string{
		StateDiscovered:           "discovered",
		StateConnecting:           "connecting",
		StateAwaitingVerification: "awaiting_verification",
		StateTrusted:              "trusted",
		StateIdentityChanged:      "identity_changed",
		StateDisconnected:         "disconnected",
	}
	for state, want := range cases {
		if got := state.String(); got != want {
			t.Errorf("PeerState(%d).String() = %q, want %q", int(state), got, want)
		}
	}
}

func TestUnknownStateHasDefaultLabel(t *testing.T) {
	if got := PeerState(99).String(); got != "unknown" {
		t.Fatalf("unknown state = %q, want \"unknown\"", got)
	}
}
