package transfer

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// memoryWriter collects what the destination writes and records whether it was closed.
type memoryWriter struct {
	bytes.Buffer
	closed bool
}

func (w *memoryWriter) Close() error {
	w.closed = true
	return nil
}

func stagedFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "staged")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWriterDestinationCopiesStagedFileAndCloses(t *testing.T) {
	var got *memoryWriter
	dest := WriterDestination{Open: func(name string) (io.WriteCloser, error) {
		if name != "report.pdf" {
			t.Fatalf("Open name = %q, want report.pdf", name)
		}
		got = &memoryWriter{}
		return got, nil
	}}

	final, err := dest.Publish(stagedFile(t, "payload"), "report.pdf")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if final != "report.pdf" {
		t.Fatalf("Publish returned %q, want report.pdf", final)
	}
	if got.String() != "payload" {
		t.Fatalf("written = %q, want payload", got.String())
	}
	if !got.closed {
		t.Fatal("writer was not closed, so the platform may not have saved the file")
	}
}

func TestWriterDestinationReturnsOpenError(t *testing.T) {
	boom := errors.New("user cancelled")
	dest := WriterDestination{Open: func(string) (io.WriteCloser, error) { return nil, boom }}

	if _, err := dest.Publish(stagedFile(t, "x"), "a.txt"); !errors.Is(err, boom) {
		t.Fatalf("Publish error = %v, want %v", err, boom)
	}
}

func TestWriterDestinationMissingStagedFileWritesNothing(t *testing.T) {
	opened := false
	dest := WriterDestination{Open: func(string) (io.WriteCloser, error) {
		opened = true
		return &memoryWriter{}, nil
	}}

	if _, err := dest.Publish(filepath.Join(t.TempDir(), "missing"), "a.txt"); err == nil {
		t.Fatal("Publish of a missing staged file succeeded")
	}
	if opened {
		t.Fatal("a save location was asked for a file that does not exist")
	}
}
