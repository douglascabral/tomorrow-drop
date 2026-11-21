package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Destination stores completed files. The desktop implementation saves into a
// folder. Android and iOS implement the same interface with their native
// storage APIs (tasks 058 and 035).
type Destination interface {
	// Publish moves the staged file into the destination under name and returns
	// the final location. It must never overwrite an existing file.
	Publish(stagedPath, name string) (string, error)
}

// DesktopDestination saves files into Dir, usually the user's Downloads folder.
type DesktopDestination struct {
	Dir string
}

// Publish implements Destination.
func (d DesktopDestination) Publish(stagedPath, name string) (string, error) {
	return publishExclusive(stagedPath, d.Dir, name)
}

// DefaultDownloadsDir returns the user's Downloads folder, the default
// destination on desktop (PRD §14.1). The folder is not created here: it is
// expected to exist, and publishing fails clearly when it does not.
func DefaultDownloadsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, "Downloads"), nil
}

// WriterDestination saves files through a writer the platform provides, such
// as the Android storage picker or the iOS Files export. The core stays free of
// UI code: the application supplies Open, which asks the user where to save.
type WriterDestination struct {
	// Open returns a writer for a new file called name. It must not overwrite
	// an existing file. Returning an error cancels the publication.
	Open func(name string) (io.WriteCloser, error)
}

// Publish implements Destination. It opens the staged file first, so a missing
// file never reaches the user with a save prompt.
func (d WriterDestination) Publish(stagedPath, name string) (string, error) {
	src, err := os.Open(stagedPath) //nolint:gosec // G304: stagedPath is created by the receiver in its own staging directory.
	if err != nil {
		return "", fmt.Errorf("open staged file: %w", err)
	}
	defer func() { _ = src.Close() }()

	dst, err := d.Open(name)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return "", fmt.Errorf("write %s: %w", name, err)
	}
	if err := dst.Close(); err != nil {
		return "", fmt.Errorf("finish %s: %w", name, err)
	}
	return name, nil
}
