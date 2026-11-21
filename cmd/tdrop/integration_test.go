package main

import (
	"bytes"
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// cliHome returns a fresh home directory for one simulated device.
func cliHome(t *testing.T) env {
	t.Helper()
	base := t.TempDir()
	e := portugueseEnv(t)
	e.Home = filepath.Join(base, "home")
	e.ConfigPath = filepath.Join(base, "settings.json")
	e.DiscoveryAddr = ""
	return e
}

type result struct {
	code           int
	stdout, stderr string
}

func exec(t *testing.T, ctx context.Context, e env, stdin string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	e.In = strings.NewReader(stdin)
	code := runCtx(ctx, args, &out, &errb, e)
	return result{code, out.String(), errb.String()}
}

// execAsync runs a command in the background, reporting when it has bound its listener.
func execAsync(t *testing.T, ctx context.Context, e env, stdin string, args ...string) (<-chan result, <-chan netip.AddrPort) {
	t.Helper()
	ready := make(chan netip.AddrPort, 1)
	e.Ready = func(a netip.AddrPort) { ready <- a }
	done := make(chan result, 1)
	go func() {
		done <- exec(t, ctx, e, stdin, args...)
	}()
	return done, ready
}

func waitResult(t *testing.T, ch <-chan result) result {
	t.Helper()
	select {
	case r := <-ch:
		return r
	case <-time.After(20 * time.Second):
		t.Fatalf("command did not finish")
		return result{}
	}
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestIdentityShowPrintsFingerprintOnly(t *testing.T) {
	e := cliHome(t)
	r := exec(t, testCtx(t), e, "", "identity", "show")
	if r.code != 0 {
		t.Fatalf("identity show = %d (stderr: %s)", r.code, r.stderr)
	}
	if !strings.Contains(r.stdout, "identidade") {
		t.Fatalf("output = %q, want the identity line", r.stdout)
	}
	data, err := os.ReadFile(filepath.Join(e.Home, "identity.key")) //nolint:gosec // G304: test home inside t.TempDir()
	if err != nil {
		t.Fatalf("identity file not created: %v", err)
	}
	if strings.Contains(r.stdout, "PRIVATE") || strings.Contains(r.stdout, string(data[:20])) {
		t.Fatalf("identity show leaked private key material")
	}
}

func TestIdentityShowIsStableAcrossRuns(t *testing.T) {
	e := cliHome(t)
	first := exec(t, testCtx(t), e, "", "identity", "show")
	second := exec(t, testCtx(t), e, "", "identity", "show")
	if first.stdout != second.stdout {
		t.Fatalf("identity changed between runs: %q vs %q", first.stdout, second.stdout)
	}
}

func TestTrustListEmptyAndRemoveUnknown(t *testing.T) {
	e := cliHome(t)
	if r := exec(t, testCtx(t), e, "", "trust", "list"); r.code != 0 {
		t.Fatalf("trust list on empty store = %d", r.code)
	}
	r := exec(t, testCtx(t), e, "", "trust", "remove", "0123456789abcdef")
	if r.code != 1 {
		t.Fatalf("trust remove unknown = %d, want 1", r.code)
	}
}

func TestSendWithoutToIsUsageError(t *testing.T) {
	e := cliHome(t)
	r := exec(t, testCtx(t), e, "", "send", "file.txt")
	if r.code != 2 {
		t.Fatalf("send without --to = %d, want 2", r.code)
	}
}

func TestSendToClosedEndpointFailsWithoutTouchingStdout(t *testing.T) {
	e := cliHome(t)
	r := exec(t, testCtx(t), e, "", "send", "--to", "127.0.0.1:9", "file.txt")
	if r.code == 0 {
		t.Fatalf("send to a closed port succeeded")
	}
	if r.stdout != "" {
		t.Fatalf("stdout should be empty on failure, got %q", r.stdout)
	}
}

func TestCanceledContextExitsWith130(t *testing.T) {
	e := cliHome(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := exec(t, ctx, e, "", "send", "--to", "127.0.0.1:9", "file.txt")
	if r.code != 130 {
		t.Fatalf("canceled send = %d, want 130", r.code)
	}
}

// pairDevices runs receive on one device and pair on the other, with both
// users confirming the fingerprints. It returns the receiver's address.
func pairDevices(t *testing.T, ctx context.Context, sender, receiver env) netip.AddrPort {
	t.Helper()
	recvDone, recvReady := execAsync(t, ctx, receiver, "y\nsender-laptop\n", "receive", "--listen", "127.0.0.1:0", "--out", t.TempDir())
	addr := <-recvReady

	pairDone := make(chan result, 1)
	go func() {
		pairDone <- exec(t, ctx, sender, "y\nreceiver-desk\n", "pair", addr.String())
	}()
	p := waitResult(t, pairDone)
	rv := waitResult(t, recvDone)
	if p.code != 0 {
		t.Fatalf("pair = %d (stdout: %s, stderr: %s)", p.code, p.stdout, p.stderr)
	}
	if rv.code != 0 {
		t.Fatalf("receive pairing = %d (stderr: %s)", rv.code, rv.stderr)
	}
	return addr
}

func TestPairThenTrustListShowsPeer(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	pairDevices(t, ctx, a, b)

	r := exec(t, ctx, a, "", "trust", "list")
	if r.code != 0 || !strings.Contains(r.stdout, "receiver-desk") {
		t.Fatalf("trust list after pairing = %d %q", r.code, r.stdout)
	}
}

func TestSendAfterPairingDeliversFile(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	addr := pairDevices(t, ctx, a, b)

	src := t.TempDir()
	path := filepath.Join(src, "notes.txt")
	if err := os.WriteFile(path, []byte("hello from a"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	recvDone, recvReady := execAsync(t, ctx, b, "y\n", "receive", "--listen", "127.0.0.1:0", "--out", out)
	recvAddr := <-recvReady
	_ = addr

	send := exec(t, ctx, a, "", "send", "--to", recvAddr.String(), path)
	rv := waitResult(t, recvDone)
	if send.code != 0 {
		t.Fatalf("send = %d (stderr: %s)", send.code, send.stderr)
	}
	if rv.code != 0 {
		t.Fatalf("receive = %d (stderr: %s)", rv.code, rv.stderr)
	}
	got, err := os.ReadFile(filepath.Join(out, "notes.txt")) //nolint:gosec // G304: test output directory
	if err != nil || string(got) != "hello from a" {
		t.Fatalf("received file = %q (%v)", got, err)
	}
}

func TestSendDeclinedExitsThreeAndSavesNothing(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	pairDevices(t, ctx, a, b)

	path := filepath.Join(t.TempDir(), "x.txt")
	_ = os.WriteFile(path, []byte("no"), 0o600)
	out := t.TempDir()
	recvDone, recvReady := execAsync(t, ctx, b, "n\n", "receive", "--listen", "127.0.0.1:0", "--out", out)
	recvAddr := <-recvReady

	send := exec(t, ctx, a, "", "send", "--to", recvAddr.String(), path)
	_ = waitResult(t, recvDone)
	if send.code != 3 {
		t.Fatalf("declined send = %d, want 3", send.code)
	}
	if entries, _ := os.ReadDir(out); len(entries) != 0 {
		t.Fatalf("declined transfer saved files")
	}
}

func TestSendToUntrustedDeviceRequiresPairing(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	recvDone, recvReady := execAsync(t, ctx, b, "n\n", "receive", "--listen", "127.0.0.1:0", "--out", t.TempDir())
	recvAddr := <-recvReady

	path := filepath.Join(t.TempDir(), "x.txt")
	_ = os.WriteFile(path, []byte("x"), 0o600)
	send := exec(t, ctx, a, "", "send", "--to", recvAddr.String(), path)
	_ = waitResult(t, recvDone)
	if send.code != 3 {
		t.Fatalf("send to an untrusted device = %d, want 3", send.code)
	}
	if !strings.Contains(send.stderr, "tdrop pair") {
		t.Fatalf("stderr should suggest pairing, got %q", send.stderr)
	}
}

func TestSendStdinSendsText(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	pairDevices(t, ctx, a, b)

	recvDone, recvReady := execAsync(t, ctx, b, "y\n", "receive", "--listen", "127.0.0.1:0", "--out", t.TempDir())
	recvAddr := <-recvReady
	send := exec(t, ctx, a, "text from stdin", "send", "--stdin", "--to", recvAddr.String())
	rv := waitResult(t, recvDone)
	if send.code != 0 {
		t.Fatalf("send --stdin = %d (stderr: %s)", send.code, send.stderr)
	}
	if rv.code != 0 || !strings.Contains(rv.stdout, "text from stdin") {
		t.Fatalf("receiver did not print the text: %d %q", rv.code, rv.stdout)
	}
}

func TestDaemonAutoAcceptsOnlyTrustedWhenConfigured(t *testing.T) {
	ctx, cancel := context.WithCancel(testCtx(t))
	a, b := cliHome(t), cliHome(t)
	pairDevices(t, ctx, a, b)

	out := t.TempDir()
	daemonDone, daemonReady := execAsync(t, ctx, b, "", "daemon", "--listen", "127.0.0.1:0", "--out", out, "--auto-accept-trusted")
	daemonAddr := <-daemonReady

	path := filepath.Join(t.TempDir(), "auto.txt")
	_ = os.WriteFile(path, []byte("auto"), 0o600)
	send := exec(t, ctx, a, "", "send", "--to", daemonAddr.String(), path)
	if send.code != 0 {
		t.Fatalf("send to daemon = %d (stderr: %s)", send.code, send.stderr)
	}
	if _, err := os.Stat(filepath.Join(out, "auto.txt")); err != nil {
		t.Fatalf("daemon did not save the file without prompting: %v", err)
	}
	cancel()
	if r := waitResult(t, daemonDone); r.code != 130 {
		t.Fatalf("daemon stopped with %d, want 130", r.code)
	}
}

func TestDaemonPromptsForUntrustedDevicesEvenWithAutoAccept(t *testing.T) {
	ctx, cancel := context.WithCancel(testCtx(t))
	defer cancel()
	a, b := cliHome(t), cliHome(t)
	daemonDone, daemonReady := execAsync(t, ctx, b, "", "daemon", "--listen", "127.0.0.1:0", "--out", t.TempDir(), "--auto-accept-trusted")
	daemonAddr := <-daemonReady

	path := filepath.Join(t.TempDir(), "x.txt")
	_ = os.WriteFile(path, []byte("x"), 0o600)
	send := exec(t, ctx, a, "", "send", "--to", daemonAddr.String(), path)
	if send.code == 0 {
		t.Fatalf("untrusted device was accepted by the daemon")
	}
	cancel()
	_ = waitResult(t, daemonDone)
}

func TestDiscoverFindsAnnouncedDevice(t *testing.T) {
	ctx, cancel := context.WithCancel(testCtx(t))
	defer cancel()
	receiverEnv := cliHome(t)
	receiverEnv.DiscoveryAddr = "127.0.0.1:0"
	discAddr := make(chan netip.AddrPort, 1)
	receiverEnv.ReadyDiscovery = func(a netip.AddrPort) { discAddr <- a }

	recvDone, recvReady := execAsync(t, ctx, receiverEnv, "", "receive", "--listen", "127.0.0.1:0", "--name", "living-room", "--out", t.TempDir())
	<-recvReady
	target := <-discAddr

	scanner := cliHome(t)
	var r result
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		r = exec(t, ctx, scanner, "", "discover", "--target", target.String())
		if strings.Contains(r.stdout, "living-room") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	cancel()
	_ = waitResult(t, recvDone)
	if r.code != 0 || !strings.Contains(r.stdout, "living-room") {
		t.Fatalf("discover = %d %q, want the announced device", r.code, r.stdout)
	}
}

func TestDiscoverWithoutDevicesSaysSo(t *testing.T) {
	ctx := testCtx(t)
	e := cliHome(t)
	r := exec(t, ctx, e, "", "discover", "--target", "127.0.0.1:9")
	if r.code != 0 || !strings.Contains(r.stdout, "Nenhum dispositivo") {
		t.Fatalf("discover with no devices = %d %q", r.code, r.stdout)
	}
}

func TestSendWritesProgressToStderrAndResultToStdout(t *testing.T) {
	ctx := testCtx(t)
	a, b := cliHome(t), cliHome(t)
	pairDevices(t, ctx, a, b)

	path := filepath.Join(t.TempDir(), "p.bin")
	_ = os.WriteFile(path, make([]byte, 2<<20), 0o600)
	recvDone, recvReady := execAsync(t, ctx, b, "y\n", "receive", "--listen", "127.0.0.1:0", "--out", t.TempDir())
	recvAddr := <-recvReady
	send := exec(t, ctx, a, "", "send", "--to", recvAddr.String(), path)
	_ = waitResult(t, recvDone)
	if send.code != 0 || !strings.Contains(send.stdout, "1 arquivo(s) enviado(s)") {
		t.Fatalf("send stdout = %d %q", send.code, send.stdout)
	}
	if !strings.Contains(send.stderr, "p.bin") || strings.Contains(send.stdout, "bytes") {
		t.Fatalf("progress must go to stderr: stdout=%q stderr=%q", send.stdout, send.stderr)
	}
}
