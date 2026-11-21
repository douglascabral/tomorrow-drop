package main

import (
	"bytes"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tomorrow-drop/internal/discovery"
	"tomorrow-drop/internal/transfer"
	"tomorrow-drop/internal/trust"
)

func runIn(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errb bytes.Buffer
	code = run(args, &out, &errb, portugueseEnv(t))
	return code, out.String(), errb.String()
}

// The help lists every command; it used to list only version and config.
func TestHelpListsEveryCommand(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		code, out, errOut := runIn(t, arg)
		if code != 0 {
			t.Fatalf("tdrop %s exited %d (stderr %q), want 0", arg, code, errOut)
		}
		for _, cmd := range []string{"discover", "pair", "send", "receive", "daemon", "trust", "identity", "config", "version"} {
			if !strings.Contains(out, cmd) {
				t.Errorf("tdrop %s does not mention %q:\n%s", arg, cmd, out)
			}
		}
	}
}

func TestUnknownCommandPointsToHelp(t *testing.T) {
	code, out, errOut := runIn(t, "foo")
	if code != 2 || out != "" {
		t.Fatalf("exit %d, stdout %q; want 2 and nothing on stdout", code, out)
	}
	if !strings.Contains(errOut, "foo") || !strings.Contains(errOut, "tdrop help") {
		t.Fatalf("stderr = %q, want the command named and a pointer to tdrop help", errOut)
	}
}

// A mistake in a command shows how to use that command, not a generic line.
func TestMisuseShowsTheUsageOfTheCommand(t *testing.T) {
	cases := map[string][]string{
		"tdrop send":     {"send"},
		"tdrop pair":     {"pair"},
		"tdrop trust":    {"trust"},
		"tdrop identity": {"identity"},
		"tdrop discover": {"discover", "extra"},
		"tdrop receive":  {"receive", "extra"},
	}
	for want, args := range cases {
		code, _, errOut := runIn(t, args...)
		if code != 2 {
			t.Errorf("%v exited %d, want 2", args, code)
		}
		if !strings.Contains(errOut, want) {
			t.Errorf("%v: stderr = %q, want the usage of %q", args, errOut, want)
		}
	}
}

// Regression: "tdrop send a.txt --to X" said --to was missing, because the
// flags had to come before the files. The documented form works.
func TestFlagsMayFollowTheArguments(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, errOut := runIn(t, "send", file, "--to", "127.0.0.1:1")
	if strings.Contains(errOut, "--to") {
		t.Fatalf("--to after the file was not understood: %q", errOut)
	}
}

// Regression: a device that is off showed "An unexpected error occurred".
func TestUnreachableDeviceIsExplained(t *testing.T) {
	code, _, errOut := runIn(t, "pair", "127.0.0.1:1")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if !strings.Contains(errOut, "alcançar") || strings.Contains(errOut, "inesperado") {
		t.Fatalf("stderr = %q, want the unreachable message", errOut)
	}
}

// The person who receives sees what is offered, and by whom, before accepting.
func TestOfferTextListsTheFilesAndTheSender(t *testing.T) {
	tr := portugueseEnv(t).Tr
	offer := transfer.Offer{Kind: transfer.KindFile, Items: []transfer.Item{
		{Name: "relatório.pdf", Size: 2400000}, {Name: "foto.jpg", Size: 800000},
	}}
	text := offerText(tr, offer, "laptop")
	for _, want := range []string{"laptop", "2 arquivos", "3.1 MB", "relatório.pdf", "2.3 MB", "foto.jpg"} {
		if !strings.Contains(text, want) {
			t.Errorf("offer text lacks %q:\n%s", want, text)
		}
	}
	single := offerText(tr, transfer.Offer{Kind: transfer.KindFile, Items: []transfer.Item{{Name: "a", Size: 1}}}, "laptop")
	if !strings.Contains(single, "1 arquivo") || strings.Contains(single, "1 arquivos") {
		t.Errorf("one file reads wrong:\n%s", single)
	}
	if text := offerText(tr, transfer.Offer{Kind: transfer.KindText, Items: []transfer.Item{{Size: 12}}}, "laptop"); !strings.Contains(text, "texto") {
		t.Errorf("a text offer does not say it is a text:\n%s", text)
	}
}

// The sender is called by the name given when pairing; a fingerprint of 64
// characters says nothing to a person.
func TestSenderIsShownByItsPairedName(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	peers := []trust.Peer{{Name: "laptop", Key: key}}
	if got := peerLabel(peers, key); got != "laptop" {
		t.Fatalf("peerLabel = %q, want the paired name", got)
	}
	if got := peerLabel(nil, key); len(got) != 8 {
		t.Fatalf("peerLabel for an unknown key = %q, want the start of its fingerprint", got)
	}
}

func TestProgressLineIsReadable(t *testing.T) {
	tr := portugueseEnv(t).Tr
	line := progressLine(tr, "p.bin", 1048576, 2097152)
	for _, want := range []string{"p.bin", "50%", "1.0 MB", "2.0 MB"} {
		if !strings.Contains(line, want) {
			t.Errorf("progress line %q lacks %q", line, want)
		}
	}
	if strings.Contains(line, "bytes") {
		t.Errorf("progress line shows raw bytes: %q", line)
	}
}

// Like the application, the list of devices does not include this device.
func TestDiscoverHidesThisDevice(t *testing.T) {
	local := []netip.Addr{netip.MustParseAddr("10.0.0.9")}
	devices := []discovery.Device{
		{Name: "this-mac", Addr: netip.MustParseAddrPort("10.0.0.9:47521")},
		{Name: "phone", Addr: netip.MustParseAddrPort("10.0.0.3:47521")},
	}
	got := withoutSelf(devices, local, 47521)
	if len(got) != 1 || got[0].Name != "phone" {
		t.Fatalf("withoutSelf = %+v, want only the phone", got)
	}
}

// The words match the application: a device is paired.
func TestCommandOutputSaysPaired(t *testing.T) {
	_, out, _ := runIn(t, "trust", "list")
	if !strings.Contains(out, "pareado") || strings.Contains(out, "confiável") {
		t.Fatalf("trust list says %q, want 'pareado' like the application", out)
	}
}
