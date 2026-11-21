package main

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/ui"
)

// With no device found the list is not a blank area: the screen says it is
// looking and what to do. The message goes away once a device is listed.
func TestHomeExplainsAnEmptyList(t *testing.T) {
	g := newTestGUI(t)
	g.showHome()
	if g.emptyHint == nil || !g.emptyHint.Visible() {
		t.Fatalf("empty list shows no explanation")
	}
	texts := labelTexts(g.emptyHint)
	if !contains(texts, g.tr.T("gui.home.empty_title")) || !contains(texts, g.tr.T("gui.home.empty_hint")) {
		t.Fatalf("empty state texts = %v", texts)
	}

	now := time.Now()
	_ = g.reg.Observe(discovery.Announcement{Name: "laptop", Addr: netip.MustParseAddrPort("10.0.0.2:47521"), Seen: now}, now)
	g.refreshDevices()
	if g.emptyHint.Visible() {
		t.Fatalf("the empty-list message stays after a device is listed")
	}
}

// The logo is smaller on a phone, so the list gets the room.
func TestPhoneHomeHasACompactLogo(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = true
	g.showHome()
	logo := findImage(g.win.Content())
	if logo == nil {
		t.Fatalf("home has no logo")
	}
	if w := logo.MinSize().Width; w > 64 {
		t.Fatalf("phone logo is %v wide, want at most 64", w)
	}
}

// Send is available only when there is something to send, and it says "Send".
func TestSendButtonWaitsForSomethingToSend(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = false
	g.showSend("10.0.0.2:47521", true)
	send := findButton(g.win.Content(), g.tr.T("gui.send.send"))
	if send == nil {
		t.Fatalf("desktop send button has no label; buttons = %v", buttonLabels(g.win.Content()))
	}
	if !send.Disabled() {
		t.Fatalf("send is enabled with nothing to send")
	}
	entry := findMultiLineEntry(g.win.Content())
	if entry == nil {
		t.Fatalf("text screen has no text field")
	}
	entry.SetText("olá")
	if send.Disabled() {
		t.Fatalf("send stays disabled after writing a text")
	}
	entry.SetText("   ")
	if !send.Disabled() {
		t.Fatalf("send is enabled for a text of spaces")
	}
}

func TestSendFilesButtonStartsDisabled(t *testing.T) {
	g := newTestGUI(t)
	g.showSend("10.0.0.2:47521", false)
	send := findButton(g.win.Content(), g.tr.T("gui.send.send"))
	if send == nil || !send.Disabled() {
		t.Fatalf("send of files must start disabled")
	}
}

// Files dropped on the window are added; folders and other schemes are not.
func TestDroppedPathsKeepOnlyFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	web, err := storage.ParseURI("https://example.com/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	uris := []fyne.URI{
		storage.NewFileURI(file),
		storage.NewFileURI(dir),
		storage.NewFileURI(filepath.Join(dir, "missing.txt")),
		web,
	}
	got := droppedPaths(uris)
	if len(got) != 1 || got[0] != file {
		t.Fatalf("droppedPaths = %v, want only %s", got, file)
	}
}

// The settings picker names the languages for people, not by code.
func TestLanguagePickerNamesTheLanguages(t *testing.T) {
	var chosen string
	picker := languagePicker("pt-BR", "Automatic", func(code string) { chosen = code })
	if picker.Selected != "Português (Brasil)" {
		t.Fatalf("selected = %q, want the language name", picker.Selected)
	}
	want := []string{"Automatic", "Português (Brasil)", "English"}
	if len(picker.Options) != len(want) {
		t.Fatalf("options = %v, want %v", picker.Options, want)
	}
	for i, o := range want {
		if picker.Options[i] != o {
			t.Fatalf("options = %v, want %v", picker.Options, want)
		}
	}
	picker.SetSelected("English")
	if chosen != "en" {
		t.Fatalf("choosing English reported %q, want the code en", chosen)
	}
	picker.SetSelected("Automatic")
	if chosen != "auto" {
		t.Fatalf("choosing Automatic reported %q, want auto", chosen)
	}
}

// findMultiLineEntry returns the first multi-line text field under obj, or nil.
func findMultiLineEntry(obj fyne.CanvasObject) *widget.Entry {
	switch o := obj.(type) {
	case *widget.Entry:
		if o.MultiLine {
			return o
		}
	case *container.Scroll:
		return findMultiLineEntry(o.Content)
	case *container.ThemeOverride:
		return findMultiLineEntry(o.Content)
	case *fyne.Container:
		for _, c := range o.Objects {
			if e := findMultiLineEntry(c); e != nil {
				return e
			}
		}
	}
	return nil
}

// On a phone the padding that makes buttons easy to tap would also make every
// line of text about 65 dp high. Text is laid out compactly; controls keep the
// large size. Desktop is not touched.
func TestPhoneTextIsCompactAndControlsStayLarge(t *testing.T) {
	g := newTestGUI(t)
	g.app.Settings().SetTheme(themeFor(true))
	g.mobile = true

	line := g.compact(newLabel("Tomorrow Drop"))
	if h := line.MinSize().Height; h > 44 {
		t.Fatalf("a line of text is %v high on a phone, want at most 44", h)
	}
	if h := widget.NewButton("Enviar", nil).MinSize().Height; h < 64 {
		t.Fatalf("a button is %v high, want at least 64", h)
	}
	if h := g.newDeviceRow().MinSize().Height; h < 72 {
		t.Fatalf("a device row is %v high, want at least 72 to tap", h)
	}

	g.mobile = false
	label := newLabel("x")
	if g.compact(label) != fyne.CanvasObject(label) {
		t.Fatalf("desktop text was wrapped; it must stay as it is")
	}
}

// "Open folder" cannot work on a phone: the file was saved where the person
// chose, through the system dialog, and there is no folder to open from the app.
func TestOpenFolderIsOnlyOfferedOnDesktop(t *testing.T) {
	g := newTestGUI(t)
	g.mobile = false
	g.showCompleted("Recebido", "x", []string{"/tmp/a.pdf"})
	if !contains(buttonLabels(g.win.Content()), g.tr.T("gui.done.open_folder")) {
		t.Fatalf("desktop result lacks Open folder")
	}
	g.mobile = true
	g.showCompleted("Recebido", "x", []string{"/tmp/a.pdf"})
	if contains(buttonLabels(g.win.Content()), g.tr.T("gui.done.open_folder")) {
		t.Fatalf("a phone offers Open folder, which cannot work there")
	}
}

// A long name or file name is cut with an ellipsis, so the row keeps its height.
func TestRowAndFileNamesAreTruncatedNotWrapped(t *testing.T) {
	g := newTestGUI(t)
	parts := g.rowParts[g.newDeviceRow()]
	if parts.name.Wrapping != fyne.TextWrapOff || parts.name.Truncation != fyne.TextTruncateEllipsis {
		t.Fatalf("row name must be cut with an ellipsis, got wrapping %v truncation %v", parts.name.Wrapping, parts.name.Truncation)
	}
	if l := newNameLabel("x"); l.Wrapping != fyne.TextWrapOff || l.Truncation != fyne.TextTruncateEllipsis {
		t.Fatalf("newNameLabel must cut with an ellipsis")
	}
}

// This device's fingerprint is one tap away, right below its address on the home
// screen. Only a button is shown, never the start of the fingerprint: seeing just
// the beginning would invite people to compare just the beginning.
func TestHomeOffersThisDevicesFingerprintBelowItsAddress(t *testing.T) {
	g := newTestGUI(t)
	g.addrs = func() []netip.Addr { return []netip.Addr{netip.MustParseAddr("192.168.1.42")} }
	g.showHome()

	address := g.tr.T("gui.home.this_device", "192.168.1.42")
	view := g.tr.T("gui.fingerprint.view")
	texts := labelTexts(g.win.Content())
	iAddr, iView := indexOf(texts, address), indexOf(texts, view)
	if iAddr < 0 || iView < 0 {
		t.Fatalf("home lacks the address or the fingerprint button; texts = %v", texts)
	}
	if iView != iAddr+1 {
		t.Fatalf("the fingerprint button is not right below the address: %v", texts)
	}
	assertNoPartialFingerprint(t, g, texts)

	findButton(g.win.Content(), view).OnTapped()
	waitFor(t, func() bool { return g.win.Canvas().Overlays().Top() != nil }, "the fingerprint is not shown after a tap")
}

// What a tap shows is the whole fingerprint, in four lines.
func TestFingerprintViewShowsTheWholeFingerprint(t *testing.T) {
	g := newTestGUI(t)
	texts := labelTexts(g.fingerprintContent())
	whole := ui.FingerprintBlock(g.id.Fingerprint().Grouped())
	if !contains(texts, whole) {
		t.Fatalf("the view does not show the whole fingerprint; texts = %v", texts)
	}
	if got := strings.Join(strings.Fields(whole), ""); got != g.id.Fingerprint().Hex() {
		t.Fatalf("the fingerprint shown is not this device's")
	}
}

// assertNoPartialFingerprint fails when a text shows part of this device's
// fingerprint: the first group would be enough to be tempted to stop there.
func assertNoPartialFingerprint(t *testing.T, g *gui, texts []string) {
	t.Helper()
	first := strings.Fields(g.id.Fingerprint().Grouped())[0]
	for _, text := range texts {
		if strings.Contains(text, first) {
			t.Fatalf("a screen shows the start of the fingerprint: %q", text)
		}
	}
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func waitFor(t *testing.T, ok func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(msg)
}

// On the screen of a paired device, whose address is shown, the same button is
// right below that address.
func TestPairedDeviceScreenOffersThisDevicesFingerprintBelowTheAddress(t *testing.T) {
	g := newTestGUI(t)
	ep := pairedDevice(t, g)
	g.openDevice(ui.DeviceRow{Name: "x", Address: ep.String(), StatusKey: "ui.status.discovered"})
	waitForButton(t, g, g.tr.T("gui.device.send_files"))
	g.bg.Wait() // the screen is still being laid out by the goroutine that opened it

	texts := labelTexts(g.win.Content())
	iAddr, iView := indexOf(texts, ep.String()), indexOf(texts, g.tr.T("gui.fingerprint.view"))
	if iAddr < 0 || iView < 0 {
		t.Fatalf("device screen lacks the address or the fingerprint button; texts = %v", texts)
	}
	if iView != iAddr+1 {
		t.Fatalf("the fingerprint button is not right below the address: %v", texts)
	}
	assertNoPartialFingerprint(t, g, texts)
	findButton(g.win.Content(), g.tr.T("gui.fingerprint.view")).OnTapped()
	waitFor(t, func() bool { return g.win.Canvas().Overlays().Top() != nil }, "the fingerprint is not shown after a tap")
}

// The device screen offers sending the clipboard with the other send actions.
func TestDeviceScreenOffersSendClipboard(t *testing.T) {
	g := newTestGUI(t)
	row := ui.DeviceRow{Name: "laptop", Address: "10.0.0.2:47521", Trusted: true, StatusKey: "ui.status.trusted"}
	g.showTrustedDevice(row)
	labels := buttonLabels(g.win.Content())
	for _, key := range []string{"gui.device.send_files", "gui.device.send_text", "gui.send.clipboard", "gui.home.receive"} {
		if !contains(labels, g.tr.T(key)) {
			t.Errorf("device screen lacks %q; buttons = %v", key, labels)
		}
	}
}
