package main

import (
	"context"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/ui"
)

// onUI runs fn on the window thread. Goroutines use it to change the screen.
func (g *gui) onUI(fn func()) {
	fyne.Do(fn)
}

// askScreen shows a decision screen and waits for the answer. It returns false
// when the context ends first, so a late answer is never used.
func (g *gui) askScreen(ctx context.Context, title string, body fyne.CanvasObject, yes, no string) bool {
	answer := make(chan bool, 1)
	decide := func(v bool) {
		select {
		case answer <- v:
		default:
		}
	}
	yesBtn := widget.NewButton(yes, func() { decide(true) })
	yesBtn.Importance = widget.HighImportance
	noBtn := widget.NewButton(no, func() { decide(false) })
	noBtn.Importance = widget.MediumImportance
	// Going back is declining: the other device is waiting for an answer.
	g.show(container.NewBorder(g.headerWith(title, func() { decide(false) }), container.NewGridWithColumns(2, noBtn, yesBtn), nil, nil, scroll(body)))
	select {
	case ok := <-answer:
		g.showHome()
		return ok
	case <-ctx.Done():
		return false
	}
}

// askOffer is the recipient's decision for every offer, including offers from
// trusted devices (PRD §13), shown as the request screen (layout 9).
func (g *gui) askOffer(ctx context.Context, offer transfer.Offer) bool {
	from := g.deviceName(offer.Peer)
	v := ui.OfferView(offer, from)
	items := container.NewVBox()
	for _, it := range v.Items {
		name := it.Name
		if name == "" {
			name = g.tr.T("cli.kind." + string(v.Kind))
		}
		items.Add(card(container.NewBorder(nil, nil, widget.NewIcon(theme.DocumentIcon()), nil,
			g.compact(container.NewVBox(newNameLabel(name), newLabel(ui.FormatBytes(it.Size)))))))
	}
	head := newLabelStyled(g.tr.T("gui.receive.from_device", from), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	body := container.NewVBox(head,
		g.compact(newLabelStyled(g.tr.T("gui.receive.summary", g.countItems(v.Count), ui.FormatBytes(v.TotalBytes)), fyne.TextAlignCenter, fyne.TextStyle{})),
		items)
	title := g.tr.T("gui.receive.title")
	accepted := g.askScreen(ctx, title, body, g.tr.T("gui.receive.accept"), g.tr.T("gui.receive.decline"))
	if accepted {
		g.beginReceiveMeter(from, v.Count, v.TotalBytes)
	}
	return accepted
}

// beginReceiveMeter shows the receiving progress screen (layout 10).
func (g *gui) beginReceiveMeter(from string, count int, total int64) {
	m := newMeter(g, g.tr.T("gui.receive.receiving_from", from), g.countFiles(count), g.cancelOperations)
	g.receiving.Store(true)
	g.recvStart = time.Now()
	g.onUI(func() { g.recvMeter = m })
	g.show(m.screen)
}

// onReceiveProgress feeds the receiving meter, if one is on screen.
func (g *gui) onReceiveProgress(p transfer.ReceiveProgress) {
	g.onUI(func() {
		if g.recvMeter == nil || p.Total == 0 {
			return
		}
		fraction := float64(p.Received) / float64(p.Total)
		g.recvMeter.update(fraction, p.Name, p.Received, p.Total, time.Since(g.recvStart))
	})
}

// askPairing shows both complete fingerprints so the person can compare them
// on both devices (PRD §9.2), as the pairing screen (layout 2).
func (g *gui) askPairing(local, remote identity.Fingerprint) bool {
	v := ui.PairingView(local, remote)
	head := newLabelStyled(g.tr.T("gui.pair.instructions"), fyne.TextAlignCenter, fyne.TextStyle{})
	localCard := card(g.compact(container.NewVBox(newLabelStyled(g.tr.T("gui.pair.local"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newLabelStyled(ui.FingerprintBlock(v.Local), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))))
	remoteCard := card(g.compact(container.NewVBox(newLabelStyled(g.tr.T("gui.pair.remote"), fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		newLabelStyled(ui.FingerprintBlock(v.Remote), fyne.TextAlignLeading, fyne.TextStyle{Monospace: true}))))
	body := container.NewVBox(head, localCard, remoteCard)
	return g.askScreen(g.ctx, g.tr.T("gui.pair.title"), body, g.tr.T("gui.pair.confirm"), g.tr.T("gui.pair.cancel"))
}

// askName asks for a friendly name for a newly paired device.
func (g *gui) askName(remote identity.Fingerprint) string {
	entry := widget.NewEntry()
	entry.SetPlaceHolder(remote.Hex()[:8])
	// Start from the name the device announces, so most people only confirm.
	if suggestion := g.nameSuggestion(remote); suggestion != remote.Hex()[:8] {
		entry.SetText(suggestion)
	}
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

// nameSuggestion is where the name of a new device starts: the name it
// announces, when that is a name and not an address, or the start of its
// fingerprint.
func (g *gui) nameSuggestion(remote identity.Fingerprint) string {
	hint := strings.TrimSpace(g.pairHint)
	if hint == "" {
		return remote.Hex()[:8]
	}
	if _, err := manualEndpoint(hint); err == nil {
		return remote.Hex()[:8]
	}
	return hint
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

// showText shows received text in the completion screen. It is not saved.
func (g *gui) showText(text string) {
	entry := widget.NewMultiLineEntry()
	entry.SetText(text)
	entry.Wrapping = fyne.TextWrapWord
	entry.Disable()
	copy := widget.NewButtonWithIcon(g.tr.T("gui.receive.copy"), theme.ContentCopyIcon(), func() {
		g.app.Clipboard().SetContent(text)
	})
	copy.Importance = widget.HighImportance
	body := container.NewVBox(card(entry), copy)
	g.show(container.NewBorder(g.header(g.tr.T("gui.receive.text_title")), nil, nil, nil, scroll(body)))
}
