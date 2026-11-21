package transfer

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func TestMachineFollowsHappyPath(t *testing.T) {
	m := NewMachine()
	for _, next := range []State{StateOffered, StateAccepted, StateTransferring, StateFinishing, StateDone} {
		if err := m.Move(next); err != nil {
			t.Fatalf("Move(%s): %v", next, err)
		}
	}
	if m.State() != StateDone {
		t.Fatalf("state = %s, want done", m.State())
	}
}

func TestMachineRejectsSkippedStates(t *testing.T) {
	m := NewMachine()
	if err := m.Move(StateTransferring); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Move skipping offer = %v, want ErrUnauthorized", err)
	}
}

func TestMachineTerminalStatesAreFinal(t *testing.T) {
	m := NewMachine()
	_ = m.Move(StateOffered)
	_ = m.Move(StateDeclined)
	if err := m.Move(StateAccepted); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("move out of declined = %v, want ErrUnauthorized", err)
	}
}

func TestMachineAnyActiveStateCanFail(t *testing.T) {
	for _, from := range []State{StateOffered, StateAccepted, StateTransferring, StateFinishing} {
		m := NewMachine()
		_ = m.Move(StateOffered)
		if from != StateOffered {
			_ = m.Move(StateAccepted)
		}
		if from == StateTransferring || from == StateFinishing {
			_ = m.Move(StateTransferring)
		}
		if from == StateFinishing {
			_ = m.Move(StateFinishing)
		}
		if err := m.Move(StateFailed); err != nil {
			t.Fatalf("Failed from %s: %v", m.State(), err)
		}
	}
}
