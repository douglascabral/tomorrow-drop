package protocol

import (
	"fmt"

	"tomorrow-drop/internal/apperr"
)

// Version é a versão do protocolo de transferência.
type Version uint16

// CurrentVersion é a versão implementada por este build.
const CurrentVersion Version = 1

// SupportedVersions lista as versões que este build aceita.
var SupportedVersions = []Version{CurrentVersion}

// Negotiate escolhe a maior versão presente em local e remote.
func Negotiate(local, remote []Version) (Version, error) {
	melhor := Version(0)
	encontrada := false
	for _, l := range local {
		for _, r := range remote {
			if l == r && (!encontrada || l > melhor) {
				melhor = l
				encontrada = true
			}
		}
	}
	if !encontrada {
		return 0, fmt.Errorf("sem versão comum: %w", apperr.ErrVersionMismatch)
	}
	return melhor, nil
}
