package transfer

import (
	"os"
	"path/filepath"
	"testing"
)

// BenchmarkTransferLargeFile measures end-to-end throughput of a 64 MiB file
// over an authenticated loopback TCP connection (task 066). Run with:
//
//	go test -run='^$' -bench=BenchmarkTransferLargeFile -benchtime=1x ./internal/transfer/
func BenchmarkTransferLargeFile(b *testing.B) {
	ctx := testCtx(b)
	src := b.TempDir()
	path := filepath.Join(src, "large.bin")
	const size = 64 << 20
	if err := os.WriteFile(path, make([]byte, size), 0o600); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(size)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		sender, receiver := pairedSessions(b, ctx)
		dest := b.TempDir()
		r, err := NewReceiver(ReceiverConfig{Decide: acceptAll, StagingDir: b.TempDir(), DestDir: dest, Destination: DesktopDestination{Dir: dest}})
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		_, serr, _, rerr := run(b, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
		if serr != nil || rerr != nil {
			b.Fatalf("send=%v receive=%v", serr, rerr)
		}
	}
}

// TestManySequentialTransfersStayStable sends one file after another over new
// connections between the same paired devices. Every transfer must complete
// intact and no staging file may be left behind (task 066, stability).
func TestManySequentialTransfersStayStable(t *testing.T) {
	ctx := testCtx(t)
	parties := newPairedParties(t, ctx)
	staging := t.TempDir()
	dest := t.TempDir()
	r, err := NewReceiver(ReceiverConfig{Decide: acceptAll, StagingDir: staging, DestDir: dest, Destination: DesktopDestination{Dir: dest}})
	if err != nil {
		t.Fatal(err)
	}
	path, want := randomFile(t, t.TempDir(), "series.bin", 256<<10)

	for i := 0; i < 20; i++ {
		sender, receiver := parties.connect(t, ctx)
		_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
		if serr != nil || rerr != nil {
			t.Fatalf("transfer %d: send=%v receive=%v", i, serr, rerr)
		}
		if got := sha256File(t, rres.Files[0]); got != want {
			t.Fatalf("transfer %d delivered a different file", i)
		}
	}
	if n := len(dirEntries(t, staging)); n != 0 {
		t.Fatalf("%d staging files left after the series", n)
	}
}
