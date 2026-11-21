package transfer

import (
	"fmt"
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
