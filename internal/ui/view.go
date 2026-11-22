// Package ui holds the presentation logic of the graphical application. It
// does not import any toolkit: it turns core state into rows, labels and
// validated plans, so the rules can be tested without opening a window.
package ui

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/limits"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/trust"
)

const snippetRunes = 120

// DeviceRow is one line of the discovered-devices list (task 022).
type DeviceRow struct {
	Name    string
	Address string
	Trusted bool
	// StatusKey is the translation key of the status label.
	StatusKey string
	// Presence is the last reachability check of a paired device (presence.go).
	Presence discovery.PresenceState
	// Fingerprint is the hex fingerprint of a paired device, from the trust list.
	// It identifies the device without a connection, so an offline one can be
	// opened and unpaired. Empty for a device that is not paired.
	Fingerprint string
}

// CurrentPeers lists the paired devices that are shown, given the address each
// one last answered at (by hex fingerprint). The same device paired again, for
// example after a reinstall, has a new key and often another name, and the
// earlier pairing stays in the trust list with the same address. Only the latest
// pairing at an address is shown. Devices with no known address are different
// devices and are always shown. The order of peers is kept.
func CurrentPeers(peers []trust.Peer, addrs map[string]string) []trust.Peer {
	latest := map[string]trust.Peer{}
	for _, p := range peers {
		a := addrs[p.Fingerprint.Hex()]
		if a == "" {
			continue
		}
		if best, ok := latest[a]; !ok || p.AddedAt.After(best.AddedAt) {
			latest[a] = p
		}
	}
	out := make([]trust.Peer, 0, len(peers))
	for _, p := range peers {
		a := addrs[p.Fingerprint.Hex()]
		if a == "" || latest[a].Fingerprint == p.Fingerprint {
			out = append(out, p)
		}
	}
	return out
}

// DeviceRows orders discovered devices with trusted ones first, then by name.
func DeviceRows(devices []discovery.Device) []DeviceRow {
	rows := make([]DeviceRow, 0, len(devices))
	for _, d := range devices {
		status := "ui.status.discovered"
		if d.Trusted {
			status = "ui.status.trusted"
		}
		rows = append(rows, DeviceRow{Name: d.Name, Address: d.Addr.String(), Trusted: d.Trusted, StatusKey: status})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Trusted != rows[j].Trusted {
			return rows[i].Trusted
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// PairingScreen shows both complete fingerprints for cross comparison (task 047).
type PairingScreen struct {
	Local  string
	Remote string
}

// PairingView groups both fingerprints in the form people compare on screen.
func PairingView(local, remote identity.Fingerprint) PairingScreen {
	return PairingScreen{Local: local.Grouped(), Remote: remote.Grouped()}
}

// ProgressFraction is the share of the whole offer sent, between 0 and 1.
func ProgressFraction(p transfer.Progress) float64 {
	if p.TotalSize <= 0 {
		return 0
	}
	f := float64(p.TotalSent) / float64(p.TotalSize)
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// Preview is what the person sees before the clipboard is sent (task 050).
type Preview struct {
	Kind    transfer.Kind
	Text    string
	Snippet string
}

// ClipboardPreview validates clipboard text before anything is sent. Empty
// text, invalid UTF-8, oversized text and links with unsafe schemes are refused.
func ClipboardPreview(text string) (Preview, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return Preview{}, fmt.Errorf("clipboard is empty: %w", apperr.ErrMalformed)
	}
	if len(text) > limits.MaxTextBytes {
		return Preview{}, fmt.Errorf("clipboard text is larger than the limit: %w", apperr.ErrLimitExceeded)
	}
	if !utf8.ValidString(text) {
		return Preview{}, fmt.Errorf("clipboard text is not valid UTF-8: %w", apperr.ErrMalformed)
	}
	kind := transfer.KindText
	if transfer.ValidateLink(trimmed) == nil {
		kind = transfer.KindLink
	}
	return Preview{Kind: kind, Text: text, Snippet: snippet(text)}, nil
}

func snippet(s string) string {
	runes := []rune(s)
	if len(runes) <= snippetRunes {
		return s
	}
	return string(runes[:snippetRunes]) + "…"
}

// FilePlan lists the files of a send, validated before the offer (task 049).
type FilePlan struct {
	Items []PlanItem
	Total int64
}

// PlanItem is one file of a FilePlan.
type PlanItem struct {
	Name string
	Path string
	Size int64
}

// PlanFiles checks each path is a readable regular file with a valid name.
func PlanFiles(paths []string) (FilePlan, error) {
	var plan FilePlan
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return FilePlan{}, fmt.Errorf("read %s: %w", filepath.Base(p), err)
		}
		if !info.Mode().IsRegular() {
			return FilePlan{}, fmt.Errorf("%s is not a regular file: %w", filepath.Base(p), apperr.ErrMalformed)
		}
		name := filepath.Base(p)
		if err := transfer.ValidateName(name); err != nil {
			return FilePlan{}, err
		}
		plan.Items = append(plan.Items, PlanItem{Name: name, Path: p, Size: info.Size()})
		plan.Total += info.Size()
	}
	if len(plan.Items) == 0 {
		return FilePlan{}, fmt.Errorf("no files selected: %w", apperr.ErrMalformed)
	}
	return plan, nil
}

// OfferDialog summarizes an incoming request for the receive dialog (task 051).
type OfferDialog struct {
	Kind       transfer.Kind
	Count      int
	TotalBytes int64
	Names      []string
	Items      []OfferItem
	FromName   string
}

// OfferItem is one file of an incoming request, with its size.
type OfferItem struct {
	Name string
	Size int64
}

// OfferView summarizes an offer. fromName is the trusted device's name.
func OfferView(offer transfer.Offer, fromName string) OfferDialog {
	v := OfferDialog{Kind: offer.Kind, Count: len(offer.Items), FromName: fromName}
	for _, it := range offer.Items {
		v.TotalBytes += it.Size
		v.Items = append(v.Items, OfferItem{Name: it.Name, Size: it.Size})
		if it.Name != "" {
			v.Names = append(v.Names, it.Name)
		}
	}
	return v
}

// TrustedRow is one line of the trusted-devices screen (task 048).
type TrustedRow struct {
	Name        string
	ID          string
	Fingerprint string
}

// TrustedRows lists trusted devices with their full fingerprint for inspection.
func TrustedRows(peers []trust.Peer) []TrustedRow {
	rows := make([]TrustedRow, 0, len(peers))
	for _, p := range peers {
		rows = append(rows, TrustedRow{Name: p.Name, ID: p.Fingerprint.Hex()[:16], Fingerprint: p.Fingerprint.Grouped()})
	}
	return rows
}

// DeviceAge formats how long ago a device was last seen, for the device list.
func DeviceAge(seen, now time.Time) string {
	d := now.Sub(seen)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
}

// ensure netip is used by callers that build rows from addresses
var _ netip.AddrPort

// FormatBytes writes a size in the most readable unit, as shown on the screens.
func FormatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	f := float64(n) / unit
	i := 0
	for f >= unit && i < len(units)-1 {
		f /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

// FingerprintBlock breaks a grouped fingerprint (sixteen groups of four) into
// four lines of four groups, so it is read and compared in short pieces and does
// not depend on the text wrapping to fit.
func FingerprintBlock(grouped string) string {
	groups := strings.Fields(grouped)
	if len(groups) != 16 {
		return grouped
	}
	lines := make([]string, 0, 4)
	for i := 0; i < len(groups); i += 4 {
		lines = append(lines, strings.Join(groups[i:i+4], " "))
	}
	return strings.Join(lines, "\n")
}
