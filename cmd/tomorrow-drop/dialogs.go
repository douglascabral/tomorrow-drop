package main

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/ui"
)

// onUI runs fn on the window thread. Goroutines use it to show dialogs.
func (g *gui) onUI(fn func()) {
	fyne.Do(fn)
}

// askBlocking shows a confirmation and waits for the answer. It returns false
// when the context ends first, so an answer after a timeout is never used.
func (g *gui) askBlocking(ctx context.Context, title, body, yes, no string) bool {
	answer := make(chan bool, 1)
	g.onUI(func() {
		dialog.ShowCustomConfirm(title, yes, no, widget.NewLabel(body), func(ok bool) {
			answer <- ok
		}, g.win)
	})
	select {
	case ok := <-answer:
		return ok
	case <-ctx.Done():
		return false
	}
}

// askOffer is the recipient's decision for every offer, including offers from
// trusted devices (PRD §13). Nothing is saved before the person accepts.
func (g *gui) askOffer(ctx context.Context, offer transfer.Offer) bool {
	from := g.deviceName(offer.Peer)
	v := ui.OfferView(offer, from)
	var b strings.Builder
	kind := g.tr.T("cli.kind." + string(v.Kind))
	fmt.Fprintf(&b, "%s %s · %d item(s) · %d bytes", g.tr.T("gui.receive.from"), from, v.Count, v.TotalBytes)
	if len(v.Names) > 0 {
		fmt.Fprintf(&b, "\n\n%s", strings.Join(v.Names, "\n"))
	}
	title := g.tr.T("gui.receive.title", kind)
	return g.askBlocking(ctx, title, b.String(), g.tr.T("gui.receive.accept"), g.tr.T("gui.receive.decline"))
}

// askPairing shows both fingerprints in full so the person can compare them
// on both devices (PRD §9.2).
func (g *gui) askPairing(local, remote identity.Fingerprint) bool {
	v := ui.PairingView(local, remote)
	body := g.tr.T("gui.pair.instructions") + "\n\n" +
		g.tr.T("gui.pair.local") + "\n" + v.Local + "\n\n" +
		g.tr.T("gui.pair.remote") + "\n" + v.Remote
	return g.askBlocking(g.ctx, g.tr.T("gui.pair.title"), body, g.tr.T("gui.pair.confirm"), g.tr.T("gui.pair.cancel"))
}

// askName asks for a friendly name for a newly paired device.
func (g *gui) askName(remote identity.Fingerprint) string {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(remote.Hex()[:8])
	answer := make(chan string, 1)
	g.onUI(func() {
		dialog.ShowCustomConfirm(g.tr.T("gui.pair.name_title"), g.tr.T("gui.pair.save"), g.tr.T("gui.pair.cancel"), entry, func(ok bool) {
			if ok {
				answer <- entry.Text
			} else {
				answer <- ""
			}
		}, g.win)
	})
	select {
	case name := <-answer:
		return name
	case <-g.ctx.Done():
		return ""
	}
}

func (g *gui) deviceName(key []byte) string {
	peers, err := g.store.List()
	if err != nil {
		return g.tr.T("gui.device.unknown")
	}
	for _, p := range peers {
		if string(p.Key) == string(key) {
			return p.Name
		}
	}
	fp, err := identity.FingerprintOf(key)
	if err != nil {
		return g.tr.T("gui.device.unknown")
	}
	return fp.Hex()[:8]
}

func (g *gui) notify(title, body string) {
	g.onUI(func() {
		dialog.ShowInformation(title, body, g.win)
	})
}

func (g *gui) showText(text string) {
	g.onUI(func() {
		entry := widget.NewMultiLineEntry()
		entry.SetText(text)
		entry.Wrapping = fyne.TextWrapWord
		dialog.ShowCustom(g.tr.T("gui.receive.text_title"), g.tr.T("gui.close"), entry, g.win)
	})
}
