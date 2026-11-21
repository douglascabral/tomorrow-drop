// Command tomorrow-drop is the Tomorrow Drop graphical application.
//
// The Fyne interface will be integrated in task 045. Until then the binary
// only reports that the GUI is not available yet, in the chosen language.
package main

import (
	"fmt"
	"os"

	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/settings"
)

func main() {
	path, _ := settings.DefaultPath()
	cfg, err := settings.Load(path)
	if err != nil {
		cfg = settings.Default
	}
	tr, err := i18n.Load(i18n.Resolve(cfg.Language, i18n.DetectSystem()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, tr.T("gui.indisponivel"))
	os.Exit(1)
}
