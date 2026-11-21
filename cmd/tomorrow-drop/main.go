// Command tomorrow-drop is the Tomorrow Drop graphical application.
//
// The interface is built with Fyne. Presentation rules live in internal/ui;
// protocol, trust and transfer rules live in the shared core, which this
// package uses but never implements.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"

	"tomorrow-drop/assets"
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

	a := app.NewWithID("org.tomorrowdrop.app")
	a.Settings().SetTheme(darkTheme{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, err := newGUI(ctx, tr, defaultHome(), path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g.app = a
	g.win = a.NewWindow(tr.T("app.nome"))
	g.win.Resize(fyne.NewSize(420, 640))

	// Discovery and incoming transfers follow the window: foreground means
	// available, background means not available (PRD §17.2, §18).
	a.Lifecycle().SetOnEnteredForeground(func() { g.setAvailable(true) })
	a.Lifecycle().SetOnExitedForeground(func() { g.setAvailable(false) })

	g.showHome()
	g.win.ShowAndRun()
}

// defaultHome is the per-user directory for identity, trust and staging files.
func defaultHome() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "tomorrow-drop")
	}
	return filepath.Join(dir, "tomorrow-drop")
}

// logoImage shows the two-mountains-and-sun logo at the given size.
func logoImage(size float32) *canvas.Image {
	res := fyne.NewStaticResource("logo.svg", assets.LogoSVG())
	img := canvas.NewImageFromResource(res)
	img.SetMinSize(fyne.NewSize(size, size))
	img.FillMode = canvas.ImageFillContain
	return img
}
