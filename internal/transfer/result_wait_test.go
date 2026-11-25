package transfer

import (
	"testing"
	"time"

	"tomorrow-drop/internal/protocol"
)

// Regression: on phones the receiver asks where to save every file, inside
// Publish, after the sender sent DONE. The sender waited only IdleTimeout
// (30 s) for the result, so a transfer of several files showed an error on the
// sender although the receiver saved every file. The wait for the result must
// cover one save prompt per file.
func TestResultWaitCoversSavePrompts(t *testing.T) {
	if finishTimeout <= protocol.IdleTimeout {
		t.Fatalf("finishTimeout = %s, must exceed the idle timeout %s", finishTimeout, protocol.IdleTimeout)
	}
	if finishTimeout < 5*time.Minute {
		t.Fatalf("finishTimeout = %s, want at least 5m for several save prompts", finishTimeout)
	}
}
