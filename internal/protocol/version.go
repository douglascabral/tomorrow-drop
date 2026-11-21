package protocol

import (
	"fmt"

	"tomorrow-drop/internal/apperr"
)

// Version is the version of the transfer protocol.
type Version uint16

// CurrentVersion is the version implemented by this build.
const CurrentVersion Version = 1

// SupportedVersions lists the versions this build accepts.
var SupportedVersions = []Version{CurrentVersion}

// Negotiate chooses the highest version present in both local and remote.
func Negotiate(local, remote []Version) (Version, error) {
	best := Version(0)
	found := false
	for _, l := range local {
		for _, r := range remote {
			if l == r && (!found || l > best) {
				best = l
				found = true
			}
		}
	}
	if !found {
		return 0, fmt.Errorf("no common version: %w", apperr.ErrVersionMismatch)
	}
	return best, nil
}
