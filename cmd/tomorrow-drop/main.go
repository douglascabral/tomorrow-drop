// Command tomorrow-drop is the Tomorrow Drop graphical application.
//
// The interface is built with Fyne. Presentation rules live in internal/ui;
// protocol, trust and transfer rules live in the shared core, which this
// package uses but never implements.
package main

import (
	"context"
	"fmt"
	"io"
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
	a := app.NewWithID("org.tomorrowdrop.app")
	a.Settings().SetTheme(themeFor(fyne.CurrentDevice().IsMobile()))

	home := dataDirFrom(os.UserConfigDir, func() string { return appStorageDir(a) })
	if err := migrateHome(filepath.Join(os.TempDir(), "tomorrow-drop"), home); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	path := filepath.Join(home, "settings.json")
	cfg, err := settings.Load(path)
	if err != nil {
		cfg = settings.Default
	}
	tr, err := i18n.Load(i18n.Resolve(cfg.Language, i18n.DetectSystem()))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// On phones the platform chooses where each received file is saved; the
	// desktop keeps saving into Downloads (PRD §14.1).
	var g *gui
	ask := func(name string) (io.WriteCloser, error) { return g.saveLocation(name) }
	g, err = newGUI(ctx, tr, home, path, receiveDestination(fyne.CurrentDevice().IsMobile(), ask))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g.app = a
	// Phones keep the screen on while the app is open (see gui.keepScreenOn).
	g.screenOn = func(on bool) { setScreenAwake(g.win, on) }
	g.mobile = fyne.CurrentDevice().IsMobile()
	g.win = a.NewWindow(tr.T("app.nome"))
	g.win.Resize(fyne.NewSize(420, 640))

	// Discovery and incoming transfers follow the window: foreground means
	// available, background means not available (PRD §17.2, §18).
	a.Lifecycle().SetOnEnteredForeground(func() { g.setAvailable(true) })
	a.Lifecycle().SetOnExitedForeground(func() { g.setAvailable(false) })

	g.showHome()
	g.win.ShowAndRun()
}

// appStorageDir is the folder that the platform gives the application for its
// own files, or "" when there is none.
func appStorageDir(a fyne.App) string {
	uri := a.Storage().RootURI()
	if uri == nil {
		return ""
	}
	return uri.Path()
}

// logoImage shows the two-mountains-and-sun logo at the given size.
func logoImage(size float32) *canvas.Image {
	res := fyne.NewStaticResource("logo.svg", assets.LogoSVG())
	img := canvas.NewImageFromResource(res)
	img.SetMinSize(fyne.NewSize(size, size))
	img.FillMode = canvas.ImageFillContain
	return img
}
