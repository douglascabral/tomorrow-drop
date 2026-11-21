package protocol

import (
	"testing"
	"time"
)

func TestTimeoutsArePositive(t *testing.T) {
	for name, value := range map[string]time.Duration{
		"HandshakeTimeout": HandshakeTimeout,
		"DecisionTimeout":  DecisionTimeout,
		"IdleTimeout":      IdleTimeout,
	} {
		if value <= 0 {
			t.Errorf("%s = %v, want a positive duration", name, value)
		}
	}
}

func TestDiscoveryIntervalMatchesPRD(t *testing.T) {
	// PRD v1.9, section 5.2: approved interval of 30 seconds.
	if DiscoveryInterval != 30*time.Second {
		t.Fatalf("DiscoveryInterval = %v, want 30s per the PRD", DiscoveryInterval)
	}
}
