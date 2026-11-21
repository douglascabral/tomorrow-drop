// Package discovery finds devices on the local network with UDP DISCOVER and
// ANNOUNCE messages (PRD §5). Announcements are untrusted: they never
// authorize a transfer. Trust comes only from the authenticated TLS identity.
package discovery

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/trust"
)

const (
	// ProtocolVersion is the version of the discovery datagram format.
	ProtocolVersion = 1
	// DefaultPort is the UDP port used for discovery.
	DefaultPort = 47520

	// TypeDiscover is a query sent to find devices.
	TypeDiscover = "DISCOVER"
	// TypeAnnounce is the answer sent by an available device.
	TypeAnnounce = "ANNOUNCE"

	nonceBytes = 8
)

// Message is one discovery datagram. ANNOUNCE echoes the nonce of the DISCOVER
// it answers, so answers to other queries are discarded. It carries only the
// protocol version, the message type, the nonce, the friendly name and the TCP
// port; no addresses of other devices, keys or user data.
type Message struct {
	Version int    `json:"v"`
	Type    string `json:"type"`
	Nonce   string `json:"nonce"`
	Name    string `json:"name,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// NewDiscover creates a query with a random nonce.
func NewDiscover() (Message, error) {
	nonce := make([]byte, nonceBytes)
	if _, err := rand.Read(nonce); err != nil {
		return Message{}, fmt.Errorf("generate nonce: %w", err)
	}
	return Message{Version: ProtocolVersion, Type: TypeDiscover, Nonce: hex.EncodeToString(nonce)}, nil
}

// NewAnnounce creates the answer to query.
func NewAnnounce(query Message, name string, port int) (Message, error) {
	m := Message{Version: ProtocolVersion, Type: TypeAnnounce, Nonce: query.Nonce, Name: name, Port: port}
	if err := m.validate(); err != nil {
		return Message{}, err
	}
	return m, nil
}

// Encode serializes a valid message within the datagram size limit.
func Encode(m Message) ([]byte, error) {
	if err := m.validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encode discovery message: %w", apperr.ErrMalformed)
	}
	if len(data) > limits.MaxDiscoveryDatagram {
		return nil, fmt.Errorf("datagram of %d bytes: %w", len(data), apperr.ErrLimitExceeded)
	}
	return data, nil
}

// Decode validates a received datagram. Unknown fields, trailing data, wrong
// versions and out-of-range values are rejected.
func Decode(data []byte) (Message, error) {
	if len(data) > limits.MaxDiscoveryDatagram {
		return Message{}, fmt.Errorf("datagram of %d bytes: %w", len(data), apperr.ErrLimitExceeded)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Message
	if err := dec.Decode(&m); err != nil {
		return Message{}, fmt.Errorf("discovery message: %w: %w", apperr.ErrMalformed, err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return Message{}, fmt.Errorf("trailing data after message: %w", apperr.ErrMalformed)
	}
	if err := m.validate(); err != nil {
		return Message{}, err
	}
	return m, nil
}

func (m Message) validate() error {
	if m.Version != ProtocolVersion {
		return fmt.Errorf("protocol version %d: %w", m.Version, apperr.ErrMalformed)
	}
	if len(m.Nonce) != 2*nonceBytes {
		return fmt.Errorf("nonce of %d characters: %w", len(m.Nonce), apperr.ErrMalformed)
	}
	if _, err := hex.DecodeString(m.Nonce); err != nil {
		return fmt.Errorf("nonce is not hexadecimal: %w", apperr.ErrMalformed)
	}
	switch m.Type {
	case TypeDiscover:
		if m.Name != "" || m.Port != 0 {
			return fmt.Errorf("DISCOVER carries announcement fields: %w", apperr.ErrMalformed)
		}
	case TypeAnnounce:
		if err := trust.ValidateName(m.Name); err != nil {
			return err
		}
		if m.Port < 1 || m.Port > 65535 {
			return fmt.Errorf("port %d out of range: %w", m.Port, apperr.ErrMalformed)
		}
	default:
		return fmt.Errorf("message type %q: %w", m.Type, apperr.ErrMalformed)
	}
	return nil
}
