package protocol

import (
	"encoding/json"
	"fmt"

	"tomorrow-drop/internal/apperr"
)

// MessageType identifies the type of a control message.
type MessageType string

const (
	MsgHello         MessageType = "HELLO"
	MsgPairConfirm   MessageType = "PAIR_CONFIRM"
	MsgPairCancel    MessageType = "PAIR_CANCEL"
	MsgTransferOffer MessageType = "TRANSFER_OFFER"
)

var knownTypes = map[MessageType]bool{
	MsgHello:         true,
	MsgPairConfirm:   true,
	MsgPairCancel:    true,
	MsgTransferOffer: true,
}

// Hello is the body of the HELLO message, listing the versions the peer supports.
type Hello struct {
	Versions []Version `json:"versions"`
}

// Message is the JSON envelope of a control message.
type Message struct {
	Type MessageType     `json:"type"`
	Body json.RawMessage `json:"body,omitempty"`
}

// DecodeBody decodes the message body into v.
func (m Message) DecodeBody(v any) error {
	if len(m.Body) == 0 {
		return fmt.Errorf("missing body in %s: %w", m.Type, apperr.ErrMalformed)
	}
	if err := json.Unmarshal(m.Body, v); err != nil {
		return fmt.Errorf("body of %s: %w: %w", m.Type, apperr.ErrMalformed, err)
	}
	return nil
}

// AllowedIn defines which message types are accepted in each security state
// (PRD §8.1 and §14). A provisional connection allows only pairing; transfers
// require the trusted state.
func AllowedIn(state PeerState, kind MessageType) bool {
	switch state {
	case StateConnecting:
		return kind == MsgHello
	case StateAwaitingVerification:
		return kind == MsgHello || kind == MsgPairConfirm || kind == MsgPairCancel
	case StateTrusted:
		return kind == MsgHello || kind == MsgTransferOffer
	default:
		return false
	}
}

// EncodeMessage serializes a message with the given body (may be nil).
func EncodeMessage(kind MessageType, body any) ([]byte, error) {
	if !knownTypes[kind] {
		return nil, fmt.Errorf("message type %q: %w", kind, apperr.ErrMalformed)
	}
	var raw json.RawMessage
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode body of %s: %w: %w", kind, apperr.ErrMalformed, err)
		}
		raw = data
	}
	return json.Marshal(Message{Type: kind, Body: raw})
}

// DecodeMessage validates and decodes the envelope of a received message.
func DecodeMessage(data []byte) (Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return Message{}, fmt.Errorf("%w: %w", apperr.ErrMalformed, err)
	}
	if !knownTypes[msg.Type] {
		return Message{}, fmt.Errorf("message type %q: %w", msg.Type, apperr.ErrMalformed)
	}
	return msg, nil
}
