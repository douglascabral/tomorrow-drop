package protocol

import (
	"bytes"
	"testing"

	"tomorrow-drop/internal/limits"
)

// FuzzReadFrame checks that no input can make the frame reader panic, read
// more than the control limit, or return a payload for a truncated frame.
func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{0, 0, 0, 2, '{', '}'})
	f.Add([]byte{0, 0, 0, 0})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{0, 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		payload, err := ReadFrame(bytes.NewReader(data))
		if err != nil {
			return
		}
		if len(payload) == 0 || len(payload) > limits.MaxControlFrameBytes {
			t.Fatalf("accepted a payload of %d bytes", len(payload))
		}
		if len(data) < 4+len(payload) {
			t.Fatalf("returned %d bytes from %d input bytes", len(payload), len(data))
		}
	})
}

// FuzzDecodeMessage checks that every accepted message has a known type and
// that decoding never panics.
func FuzzDecodeMessage(f *testing.F) {
	f.Add([]byte(`{"type":"HELLO","body":{"versions":[1]}}`))
	f.Add([]byte(`{"type":"TRANSFER_DATA"}`))
	f.Add([]byte(`{"type":"NOPE"}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, data []byte) {
		msg, err := DecodeMessage(data)
		if err != nil {
			return
		}
		if !knownTypes[msg.Type] {
			t.Fatalf("accepted unknown type %q", msg.Type)
		}
	})
}
