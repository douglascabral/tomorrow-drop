package ui

import (
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/trust"
)

func TestDeviceRowsMarkTrustedAndDiscovered(t *testing.T) {
	devices := []discovery.Device{
		{Name: "laptop", Addr: netip.MustParseAddrPort("10.0.0.2:47521"), Trusted: true},
		{Name: "phone", Addr: netip.MustParseAddrPort("10.0.0.3:47521")},
	}
	rows := DeviceRows(devices)
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	if !rows[0].Trusted || rows[0].StatusKey != "ui.status.trusted" {
		t.Errorf("laptop row = %+v, want trusted", rows[0])
	}
	if rows[1].Trusted || rows[1].StatusKey != "ui.status.discovered" {
		t.Errorf("phone row = %+v, want discovered", rows[1])
	}
	if rows[0].Address != "10.0.0.2:47521" {
		t.Errorf("address = %q", rows[0].Address)
	}
}

func TestPairingViewShowsBothFingerprintsInFull(t *testing.T) {
	local, _ := identity.Generate()
	remote, _ := identity.Generate()
	v := PairingView(local.Fingerprint(), remote.Fingerprint())
	if strings.ReplaceAll(v.Local, " ", "") != local.Fingerprint().Hex() {
		t.Fatalf("local fingerprint truncated or wrong")
	}
	if strings.ReplaceAll(v.Remote, " ", "") != remote.Fingerprint().Hex() {
		t.Fatalf("remote fingerprint truncated or wrong")
	}
}

func TestProgressFractionIsBounded(t *testing.T) {
	cases := []struct {
		sent, total int64
		want        float64
	}{
		{0, 100, 0},
		{50, 100, 0.5},
		{100, 100, 1},
		{0, 0, 0},
		{120, 100, 1},
	}
	for _, c := range cases {
		got := ProgressFraction(transfer.Progress{TotalSent: c.sent, TotalSize: c.total})
		if got != c.want {
			t.Errorf("ProgressFraction(%d/%d) = %v, want %v", c.sent, c.total, got, c.want)
		}
	}
}

func TestClipboardPreviewClassifiesText(t *testing.T) {
	p, err := ClipboardPreview("hello world")
	if err != nil || p.Kind != transfer.KindText {
		t.Fatalf("text preview = %+v (%v)", p, err)
	}
	p, err = ClipboardPreview("https://example.com/page")
	if err != nil || p.Kind != transfer.KindLink {
		t.Fatalf("link preview = %+v (%v)", p, err)
	}
}

func TestClipboardPreviewRejectsEmpty(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t"} {
		if _, err := ClipboardPreview(in); err == nil {
			t.Errorf("ClipboardPreview(%q) accepted empty content", in)
		}
	}
}

// Text that merely looks like a dangerous URL is still text. It must never be
// classified as a link, because only http and https links are links.
func TestClipboardPreviewNeverClassifiesUnsafeSchemesAsLinks(t *testing.T) {
	for _, in := range []string{"file:///etc/passwd", "javascript:alert(1)", "mailto:a@b.c"} {
		p, err := ClipboardPreview(in)
		if err != nil {
			t.Errorf("ClipboardPreview(%q) refused plain text: %v", in, err)
			continue
		}
		if p.Kind == transfer.KindLink {
			t.Errorf("ClipboardPreview(%q) classified an unsafe scheme as a link", in)
		}
	}
}

func TestClipboardPreviewTruncatesLongSnippets(t *testing.T) {
	long := strings.Repeat("a", 500)
	p, err := ClipboardPreview(long)
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(p.Snippet)) > snippetRunes+1 {
		t.Fatalf("snippet has %d runes, want at most %d", len([]rune(p.Snippet)), snippetRunes)
	}
	if p.Text != long {
		t.Fatalf("preview lost the full content")
	}
}

func TestPlanFilesListsItemsAndTotal(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.bin")
	_ = os.WriteFile(a, make([]byte, 10), 0o600)
	_ = os.WriteFile(b, make([]byte, 30), 0o600)

	plan, err := PlanFiles([]string{a, b})
	if err != nil {
		t.Fatalf("PlanFiles: %v", err)
	}
	if len(plan.Items) != 2 || plan.Total != 40 {
		t.Fatalf("plan = %+v, want 2 items totalling 40 bytes", plan)
	}
	if plan.Items[0].Name != "a.txt" {
		t.Fatalf("first item = %q", plan.Items[0].Name)
	}
}

func TestPlanFilesRejectsMissingAndDirectories(t *testing.T) {
	if _, err := PlanFiles([]string{filepath.Join(t.TempDir(), "missing")}); err == nil {
		t.Fatalf("missing file accepted")
	}
	if _, err := PlanFiles([]string{t.TempDir()}); err == nil {
		t.Fatalf("directory accepted as a file")
	}
}

func TestOfferViewSummarizesIncomingRequest(t *testing.T) {
	peer, _ := identity.Generate()
	offer := transfer.Offer{
		ID:    "0123456789abcdef",
		Kind:  transfer.KindFile,
		Items: []transfer.Item{{Name: "report.pdf", Size: 1024}, {Name: "photo.jpg", Size: 2048}},
		Peer:  peer.PublicKey(),
	}
	v := OfferView(offer, "laptop")
	if v.Count != 2 || v.TotalBytes != 3072 {
		t.Fatalf("view = %+v", v)
	}
	if v.FromName != "laptop" {
		t.Fatalf("from = %q", v.FromName)
	}
	if len(v.Names) != 2 || v.Names[0] != "report.pdf" {
		t.Fatalf("names = %v", v.Names)
	}
}

func TestTrustedRowsShowInspectableFingerprint(t *testing.T) {
	id, _ := identity.Generate()
	peers := []trust.Peer{{Name: "laptop", Key: id.PublicKey(), Fingerprint: id.Fingerprint()}}
	rows := TrustedRows(peers)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Name != "laptop" || strings.ReplaceAll(rows[0].Fingerprint, " ", "") != id.Fingerprint().Hex() {
		t.Fatalf("row = %+v, want the full fingerprint", rows[0])
	}
	if rows[0].ID != id.Fingerprint().Hex()[:16] {
		t.Fatalf("ID = %q, want the 16-character identifier used by trust remove", rows[0].ID)
	}
}

func TestDeviceAgeLabelStaysUnderMinute(t *testing.T) {
	now := time.Unix(1000, 0)
	if got := DeviceAge(now.Add(-5*time.Second), now); got != "5s" {
		t.Fatalf("DeviceAge = %q, want 5s", got)
	}
}
