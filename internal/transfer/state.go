// Package transfer sends and receives files, texts and links over an
// authenticated trusted session (PRD §11–16). Each transfer is all-or-nothing:
// content is staged in private temporary files and published only after every
// item has been received and verified.
package transfer

import (
	"fmt"

	"tomorrow-drop/internal/apperr"
)

// State is the lifecycle state of one transfer (tasks 025 and 033).
type State string

const (
	StateIdle         State = "idle"
	StateOffered      State = "offered"
	StateDeclined     State = "declined"
	StateAccepted     State = "accepted"
	StateTransferring State = "transferring"
	StateFinishing    State = "finishing"
	StateDone         State = "done"
	StateFailed       State = "failed"
	StateCanceled     State = "canceled"
)

var transitions = map[State][]State{
	StateIdle:         {StateOffered},
	StateOffered:      {StateAccepted, StateDeclined},
	StateAccepted:     {StateTransferring},
	StateTransferring: {StateFinishing},
	StateFinishing:    {StateDone},
}

// Machine enforces the transfer lifecycle. Terminal states cannot be left, and
// any active state may move to failed or canceled.
type Machine struct {
	state State
}

// NewMachine starts a transfer in the idle state.
func NewMachine() *Machine {
	return &Machine{state: StateIdle}
}

// State returns the current state.
func (m *Machine) State() State {
	return m.state
}

// Move changes state if the transition is allowed.
func (m *Machine) Move(next State) error {
	if next == StateFailed || next == StateCanceled {
		if isActive(m.state) {
			m.state = next
			return nil
		}
	} else {
		for _, allowed := range transitions[m.state] {
			if allowed == next {
				m.state = next
				return nil
			}
		}
	}
	return fmt.Errorf("transfer cannot move from %s to %s: %w", m.state, next, apperr.ErrUnauthorized)
}

func isActive(s State) bool {
	switch s {
	case StateIdle, StateOffered, StateAccepted, StateTransferring, StateFinishing:
		return true
	}
	return false
}
