package main

import (
	"os"
	"strings"
	"testing"
)

// fakeShare makes takeShare see a share of the given content, once.
func fakeShare(t *testing.T, content shareContent, ok bool) {
	t.Helper()
	old := shareSource
	shareSource = func() (shareContent, bool) { return content, ok }
	t.Cleanup(func() { shareSource = old })
}

// A file shared from another app (a screenshot, for example) is copied into the
// application and waits on the home screen until a device is chosen.
func TestSharedFileWaitsForADevice(t *testing.T) {
	g := newTestGUI(t)
	fakeShare(t, shareContent{files: []sharedFile{{name: "Screenshot 2026-11-21.png", mime: "image/png", data: []byte("\x89PNG-data")}}}, true)
	g.takeShare()
	g.pendingMu.Lock()
	share := g.share
	g.pendingMu.Unlock()
	if share == nil || len(share.paths) != 1 {
		t.Fatalf("share = %+v, want one staged file", share)
	}
	data, err := os.ReadFile(share.paths[0]) //nolint:gosec // G304: file staged by the test
	if err != nil || string(data) != "\x89PNG-data" {
		t.Fatalf("staged file content = %q (%v)", data, err)
	}
	g.showHome()
	if !contains(labelTexts(g.win.Content()), g.tr.T("gui.share.banner", g.countFiles(1))) {
		t.Fatalf("home does not say what waits to be sent; texts = %v", labelTexts(g.win.Content()))
	}
}

// Text shared from another app is sent as text.
func TestSharedTextWaitsForADevice(t *testing.T) {
	g := newTestGUI(t)
	fakeShare(t, shareContent{text: "https://example.com/artigo"}, true)
	g.takeShare()
	g.pendingMu.Lock()
	share := g.share
	g.pendingMu.Unlock()
	if share == nil || share.text != "https://example.com/artigo" || len(share.paths) != 0 {
		t.Fatalf("share = %+v, want the text", share)
	}
}

// Nothing shared means nothing waits: opening the app normally changes nothing.
func TestNothingSharedChangesNothing(t *testing.T) {
	g := newTestGUI(t)
	fakeShare(t, shareContent{}, false)
	g.takeShare()
	if g.sharedWaiting() {
		t.Fatalf("a share waits although nothing was shared")
	}
}

// A share is sent once: choosing a device takes it off the screen.
func TestChoosingADeviceSendsTheShareOnce(t *testing.T) {
	g := newTestGUI(t)
	g.pendingMu.Lock()
	g.share = &pendingShare{text: "olá"}
	g.pendingMu.Unlock()
	g.sendShared("127.0.0.1:1")
	if g.sharedWaiting() {
		t.Fatalf("the share is still waiting after a device was chosen")
	}
	g.bg.Wait()
}

// Unsafe or missing names of shared files are replaced, so the file is always a
// normal file name.
func TestSharedNamesAreSafe(t *testing.T) {
	cases := map[[2]string]string{
		{"foto.jpg", "image/jpeg"}:    "foto.jpg",
		{"../../etc/x", "text/plain"}: "Compartilhado.bin",
		{"", "image/png"}:             "Compartilhado.png",
		{".oculto", "image/gif"}:      "Compartilhado.gif",
	}
	for in, want := range cases {
		if got := safeSharedName(in[0], in[1]); got != want {
			t.Errorf("safeSharedName(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
	if strings.ContainsAny(safeSharedName("a/b", ""), `/\:`) {
		t.Errorf("a name with a separator was kept")
	}
}
