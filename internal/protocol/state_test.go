package protocol

import "testing"

func TestEstadosTemRotulosEstaveis(t *testing.T) {
	casos := map[PeerState]string{
		StateDiscovered:           "discovered",
		StateConnecting:           "connecting",
		StateAwaitingVerification: "awaiting_verification",
		StateTrusted:              "trusted",
		StateIdentityChanged:      "identity_changed",
		StateDisconnected:         "disconnected",
	}
	for estado, esperado := range casos {
		if got := estado.String(); got != esperado {
			t.Errorf("PeerState(%d).String() = %q, esperado %q", int(estado), got, esperado)
		}
	}
}

func TestEstadoDesconhecidoTemRotuloPadrao(t *testing.T) {
	if got := PeerState(99).String(); got != "unknown" {
		t.Fatalf("estado desconhecido = %q, esperado \"unknown\"", got)
	}
}
