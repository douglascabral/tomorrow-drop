package main

import (
	"bytes"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/transfer"
)

// sharedFile is a file another app shared with Tomorrow Drop.
type sharedFile struct {
	name string
	mime string
	data []byte
}

// shareContent is what another app shared: files, or a text.
type shareContent struct {
	files []sharedFile
	text  string
}

// pendingShare is a share waiting for the person to choose the device it goes to.
type pendingShare struct {
	paths []string
	text  string
}

// shareSource reads what was shared. Tests replace it.
var shareSource = readShare

// takeShare collects what another app shared with the application when it comes
// to the foreground. Files are copied into the application's folder, and the
// share waits on the home screen until a device is chosen.
func (g *gui) takeShare() {
	content, ok := shareSource()
	if !ok {
		return
	}
	share := &pendingShare{text: content.text}
	for _, f := range content.files {
		if int64(len(f.data)) > limits.MaxFileBytes {
			g.notify(g.tr.T("gui.share.title"), g.tr.T(apperr.MessageKey(apperr.ErrLimitExceeded)))
			continue
		}
		path, err := stagePicked(bytes.NewReader(f.data), safeSharedName(f.name, f.mime), filepath.Join(g.home, "shared"))
		if err != nil {
			g.notify(g.tr.T("gui.share.title"), g.tr.T(apperr.MessageKey(err)))
			continue
		}
		share.paths = append(share.paths, path)
	}
	if len(share.paths) == 0 && share.text == "" {
		return
	}
	g.pendingMu.Lock()
	g.share = share
	g.pendingMu.Unlock()
	g.onUI(g.showHome)
}

// safeSharedName is the name of a shared file, checked like any file name. An
// app may give no name, or one with path separators; then a name is made up
// from the type.
func safeSharedName(name, mime string) string {
	name = strings.TrimSpace(name)
	if name != "" && !strings.ContainsAny(name, `/\:`) && !strings.HasPrefix(name, ".") && len(name) <= 200 {
		return name
	}
	ext := clipboardExt(mime)
	if ext == "" {
		ext = ".bin"
	}
	return "Compartilhado" + ext
}

// shareBanner says what is waiting to be sent, on the home screen.
func (g *gui) shareBanner(share *pendingShare) fyne.CanvasObject {
	what := g.tr.T("gui.share.text_word")
	if len(share.paths) > 0 {
		what = g.countFiles(len(share.paths))
	}
	return container.NewVBox(
		g.compact(newLabelStyled(g.tr.T("gui.share.banner", what), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
	)
}

// sendShared sends the waiting share to the device chosen. The share is taken
// off the screen as it goes, so it is not sent twice.
func (g *gui) sendShared(addr string) {
	g.pendingMu.Lock()
	share := g.share
	g.share = nil
	g.pendingMu.Unlock()
	if share == nil {
		return
	}
	spec := transfer.SendSpec{Kind: transfer.KindFile, Paths: share.paths}
	if len(share.paths) == 0 {
		spec = transfer.SendSpec{Kind: transfer.KindText, Text: share.text}
	}
	g.startSend(addr, spec, g.tr.T("gui.share.title"))
}

// sharedWaiting reports whether a share waits for a device.
func (g *gui) sharedWaiting() bool {
	g.pendingMu.Lock()
	defer g.pendingMu.Unlock()
	return g.share != nil
}
