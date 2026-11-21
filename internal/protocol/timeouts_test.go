package protocol

import (
	"testing"
	"time"
)

func TestTimeoutsSaoPositivos(t *testing.T) {
	for nome, valor := range map[string]time.Duration{
		"HandshakeTimeout": HandshakeTimeout,
		"DecisionTimeout":  DecisionTimeout,
		"IdleTimeout":      IdleTimeout,
	} {
		if valor <= 0 {
			t.Errorf("%s = %v, esperado duração positiva", nome, valor)
		}
	}
}

func TestIntervaloDeDescobertaEhDoPRD(t *testing.T) {
	// PRD v1.9, seção 5.2: intervalo aprovado de 30 segundos.
	if DiscoveryInterval != 30*time.Second {
		t.Fatalf("DiscoveryInterval = %v, esperado 30s conforme PRD", DiscoveryInterval)
	}
}
