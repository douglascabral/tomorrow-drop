package main

import (
	"bytes"
	"net/netip"
	"strings"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/i18n"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/trust"
	"tomorrow-drop/internal/ui"
)

// offerSize is the total size of an offer.
func offerSize(offer transfer.Offer) int64 {
	var total int64
	for _, it := range offer.Items {
		total += it.Size
	}
	return total
}

// offerWhat says what is offered: "2 files", "a text" or "a link".
func offerWhat(tr *i18n.Bundle, offer transfer.Offer) string {
	switch offer.Kind {
	case transfer.KindText:
		return tr.T("cli.what_text")
	case transfer.KindLink:
		return tr.T("cli.what_link")
	}
	return tr.Count("count.file", len(offer.Items))
}

// offerText is what the person reads before accepting: who sends, what, how
// much, and for files each name and size.
func offerText(tr *i18n.Bundle, offer transfer.Offer, from string) string {
	switch offer.Kind {
	case transfer.KindText:
		return tr.T("cli.offer_text", from)
	case transfer.KindLink:
		return tr.T("cli.offer_link", from)
	}
	lines := []string{tr.T("cli.offer_files", from, offerWhat(tr, offer), ui.FormatBytes(offerSize(offer)))}
	for _, it := range offer.Items {
		lines = append(lines, tr.T("cli.offer_item", it.Name, ui.FormatBytes(it.Size)))
	}
	return strings.Join(lines, "\n")
}

// peerLabel is how a sender is called: the name given when pairing, or the
// start of its fingerprint when it is not in the list.
func peerLabel(peers []trust.Peer, key []byte) string {
	for _, p := range peers {
		if bytes.Equal(p.Key, key) {
			return p.Name
		}
	}
	fp, err := identity.FingerprintOf(key)
	if err != nil {
		return "?"
	}
	return fp.Hex()[:8]
}

// progressLine reads "file: 42% (1.0 MB of 2.4 MB)".
func progressLine(tr *i18n.Bundle, name string, sent, total int64) string {
	percent := 0
	if total > 0 {
		percent = int(sent * 100 / total)
	}
	return tr.T("cli.progress", name, percent, ui.FormatBytes(sent), ui.FormatBytes(total))
}

// withoutSelf drops this device from a list of discovered devices: it answers
// its own query, and it is not one to send to.
func withoutSelf(devices []discovery.Device, local []netip.Addr, port int) []discovery.Device {
	var out []discovery.Device
	for _, d := range devices {
		if discovery.IsOwnAnnouncement(discovery.Announcement{Name: d.Name, Addr: d.Addr}, local, port) {
			continue
		}
		out = append(out, d)
	}
	return out
}

// indentBlock shows a fingerprint in four short lines, indented.
func indentBlock(grouped string) string {
	lines := strings.Split(ui.FingerprintBlock(grouped), "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}
