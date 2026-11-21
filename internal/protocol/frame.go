// Package protocol defines the framing, version, states and limits of the
// transfer protocol. See docs/PROTOCOLO.md.
package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

const frameHeaderSize = 4

// WriteFrame writes payload preceded by its size (uint32, big-endian).
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("empty frame: %w", apperr.ErrMalformed)
	}
	if len(payload) > limits.MaxControlFrameBytes {
		return fmt.Errorf("frame of %d bytes: %w", len(payload), apperr.ErrLimitExceeded)
	}

	var header [frameHeaderSize]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload))) //nolint:gosec // G115: len(payload) is already bounded by MaxControlFrameBytes above.
	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// ReadFrame reads one frame. Invalid sizes are rejected before any body
// allocation. io.EOF is returned only if the connection closes before any
// header byte is received.
func ReadFrame(r io.Reader) ([]byte, error) {
	var header [frameHeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, readError("incomplete header", err)
	}

	size := binary.BigEndian.Uint32(header[:])
	if size == 0 {
		return nil, fmt.Errorf("empty frame: %w", apperr.ErrMalformed)
	}
	if size > limits.MaxControlFrameBytes {
		return nil, fmt.Errorf("frame of %d bytes: %w", size, apperr.ErrLimitExceeded)
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, readError("truncated body", err)
	}
	return payload, nil
}

// readError classifies a failed read. A network deadline is a timeout, not a
// malformed frame, so callers can tell a silent peer from a corrupt one.
func readError(what string, err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%s: %w: %w", what, apperr.ErrTimeout, err)
	}
	return fmt.Errorf("%s: %w", what, apperr.ErrMalformed)
}
