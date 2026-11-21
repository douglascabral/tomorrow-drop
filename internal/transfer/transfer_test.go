package transfer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/protocol"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
)

const testTimeout = 30 * time.Second

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	t.Cleanup(cancel)
	return ctx
}

func newIdentity(t *testing.T) *identity.Identity {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func newStore(t *testing.T) *trust.Store {
	t.Helper()
	return trust.Open(filepath.Join(t.TempDir(), "trust.json"))
}

func tcpPair(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()
	type accepted struct {
		c   net.Conn
		err error
	}
	ch := make(chan accepted, 1)
	go func() {
		c, err := ln.Accept()
		ch <- accepted{c, err}
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	r := <-ch
	if r.err != nil {
		t.Fatal(r.err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = r.c.Close()
	})
	return client, r.c
}

// pairOnce runs the real pairing between a and b, so each trusts the other.
func pairOnce(t *testing.T, ctx context.Context, a, b *identity.Identity, sa, sb *trust.Store) {
	t.Helper()
	c1, c2 := tcpPair(t)
	ch := make(chan *transport.Session, 1)
	go func() {
		s, err := transport.Server(ctx, c2, b, sb)
		if err != nil {
			t.Errorf("server handshake: %v", err)
		}
		ch <- s
	}()
	ca, err := transport.Client(ctx, c1, a, nil, sa)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	cb := <-ch
	pa, err := ca.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	pb, err := cb.Pairing()
	if err != nil {
		t.Fatal(err)
	}
	pa.ConfirmLocal()
	pb.ConfirmLocal()
	done := make(chan error, 2)
	go func() { done <- transport.ExchangePairing(ctx, ca, pa) }()
	go func() { done <- transport.ExchangePairing(ctx, cb, pb) }()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatalf("confirmation exchange: %v", err)
		}
	}
	if err := trust.CommitPairing(sa, pa, "b"); err != nil {
		t.Fatal(err)
	}
	if err := trust.CommitPairing(sb, pb, "a"); err != nil {
		t.Fatal(err)
	}
	_ = ca.Close()
	_ = cb.Close()
}

// connectTrusted opens a new connection between two already-paired devices.
// It returns (sender, receiver) sessions, both in the trusted state.
func connectTrusted(t *testing.T, ctx context.Context, a, b *identity.Identity, sa, sb *trust.Store) (*transport.Session, *transport.Session) {
	t.Helper()
	c1, c2 := tcpPair(t)
	ch := make(chan *transport.Session, 1)
	go func() {
		s, err := transport.Server(ctx, c2, b, sb)
		if err != nil {
			t.Errorf("server handshake: %v", err)
		}
		ch <- s
	}()
	sender, err := transport.Client(ctx, c1, a, nil, sa)
	if err != nil {
		t.Fatalf("client handshake: %v", err)
	}
	receiver := <-ch
	if sender.State() != protocol.StateTrusted || receiver.State() != protocol.StateTrusted {
		t.Fatalf("paired devices are not trusted: %s / %s", sender.State(), receiver.State())
	}
	return sender, receiver
}

// pairedSessions pairs two new devices and returns a trusted connection between them.
func pairedSessions(t *testing.T, ctx context.Context) (*transport.Session, *transport.Session) {
	t.Helper()
	a, b := newIdentity(t), newIdentity(t)
	sa, sb := newStore(t), newStore(t)
	pairOnce(t, ctx, a, b, sa, sb)
	return connectTrusted(t, ctx, a, b, sa, sb)
}

func receiverFor(t *testing.T, dest string, decide Decision) *Receiver {
	t.Helper()
	r, err := NewReceiver(ReceiverConfig{
		Decide:      decide,
		StagingDir:  t.TempDir(),
		DestDir:     dest,
		Destination: DesktopDestination{Dir: dest},
	})
	if err != nil {
		t.Fatalf("NewReceiver: %v", err)
	}
	return r
}

func acceptAll(context.Context, Offer) bool { return true }

func randomFile(t *testing.T, dir, name string, size int) (string, [32]byte) {
	t.Helper()
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, sha256.Sum256(data)
}

func sha256File(t *testing.T, path string) [32]byte {
	t.Helper()
	data, err := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(data)
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// run executes a send and a receive concurrently and returns both outcomes.
func run(t *testing.T, ctx context.Context, sender *transport.Session, spec SendSpec, progress func(Progress), r *Receiver, receiver *transport.Session) (Result, error, Result, error) {
	t.Helper()
	type out struct {
		res Result
		err error
	}
	recvCh := make(chan out, 1)
	go func() {
		res, err := r.Serve(ctx, receiver)
		recvCh <- out{res, err}
	}()
	sres, serr := Send(ctx, sender, spec, progress)
	rr := <-recvCh
	return sres, serr, rr.res, rr.err
}

func TestTextTransferDeliversContentWithoutPersisting(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)

	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindText, Text: "hello, café ☕"}, nil, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if rres.Text != "hello, café ☕" {
		t.Fatalf("received text = %q", rres.Text)
	}
	if len(dirEntries(t, dest)) != 0 {
		t.Fatalf("text transfer left files in the destination")
	}
}

func TestLinkTransferAcceptsHTTPOnly(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	r := receiverFor(t, t.TempDir(), acceptAll)
	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindLink, Text: "https://example.com/x"}, nil, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if rres.Text != "https://example.com/x" {
		t.Fatalf("received link = %q", rres.Text)
	}
}

func TestSingleFileTransferPublishesIntactFile(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	path, want := randomFile(t, src, "photo.bin", 1<<20)
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)

	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if len(rres.Files) != 1 {
		t.Fatalf("published files = %v", rres.Files)
	}
	if got := sha256File(t, rres.Files[0]); got != want {
		t.Fatalf("received file differs from the sent file")
	}
}

func TestLargeFileStreamsWithoutFullBuffering(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	path, want := randomFile(t, src, "large.bin", 8<<20)
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)

	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if got := sha256File(t, rres.Files[0]); got != want {
		t.Fatalf("large file corrupted")
	}
}

func TestEmptyFileTransfer(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	path := filepath.Join(src, "empty.txt")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)
	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if info, _ := os.Stat(rres.Files[0]); info.Size() != 0 {
		t.Fatalf("published empty file has size %d", info.Size())
	}
}

func TestMultipleFilesReportMonotonicProgress(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	var paths []string
	var total int64
	for i, size := range []int{100 << 10, 0, 300 << 10} {
		p, _ := randomFile(t, src, string(rune('a'+i))+".bin", size)
		paths = append(paths, p)
		total += int64(size)
	}
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)

	var last int64 = -1
	progress := func(p Progress) {
		if p.TotalSent < last {
			t.Errorf("total progress went backwards: %d -> %d", last, p.TotalSent)
		}
		last = p.TotalSent
		if p.TotalSize != total {
			t.Errorf("TotalSize = %d, want %d", p.TotalSize, total)
		}
	}
	_, serr, rres, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: paths}, progress, r, receiver)
	if serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if len(rres.Files) != 3 {
		t.Fatalf("published %d files, want 3", len(rres.Files))
	}
	if last != total {
		t.Fatalf("final total progress = %d, want %d", last, total)
	}
}

func TestDeclinedTransferSavesNothing(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	path, _ := randomFile(t, src, "x.bin", 1024)
	dest := t.TempDir()
	r := receiverFor(t, dest, func(context.Context, Offer) bool { return false })

	_, serr, _, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
	if !errors.Is(serr, apperr.ErrDeclined) {
		t.Fatalf("sender error = %v, want ErrDeclined", serr)
	}
	if !errors.Is(rerr, apperr.ErrDeclined) {
		t.Fatalf("receiver error = %v, want ErrDeclined", rerr)
	}
	if len(dirEntries(t, dest)) != 0 {
		t.Fatalf("declined transfer saved files")
	}
}

func TestDecisionTimeoutDeclinesAutomatically(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	path, _ := randomFile(t, src, "x.bin", 1024)
	dest := t.TempDir()
	r := receiverFor(t, dest, func(c context.Context, _ Offer) bool {
		<-c.Done()
		return false
	})
	r.decisionTimeout = 200 * time.Millisecond

	_, serr, _, _ := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver)
	if !errors.Is(serr, apperr.ErrDeclined) {
		t.Fatalf("sender error after decision timeout = %v, want ErrDeclined", serr)
	}
	if len(dirEntries(t, dest)) != 0 {
		t.Fatalf("timed-out transfer saved files")
	}
}

func TestTrustedDevicesStillNeedAcceptance(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	asked := false
	src := t.TempDir()
	path, _ := randomFile(t, src, "x.bin", 64)
	r := receiverFor(t, t.TempDir(), func(context.Context, Offer) bool {
		asked = true
		return true
	})
	if _, serr, _, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, receiver); serr != nil || rerr != nil {
		t.Fatalf("send=%v receive=%v", serr, rerr)
	}
	if !asked {
		t.Fatalf("trusted device was accepted without asking the recipient")
	}
}

func TestSenderDropAt99PercentLeavesNothingBehind(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	src := t.TempDir()
	const size = 4 << 20
	path, _ := randomFile(t, src, "big.bin", size)
	dest := t.TempDir()
	staging := t.TempDir()
	r, err := NewReceiver(ReceiverConfig{Decide: acceptAll, StagingDir: staging, DestDir: dest, Destination: DesktopDestination{Dir: dest}})
	if err != nil {
		t.Fatal(err)
	}

	progress := func(p Progress) {
		if p.ItemSent*100 >= p.ItemSize*99 {
			_ = sender.Close()
		}
	}
	_, serr, _, rerr := run(t, ctx, sender, SendSpec{Kind: KindFile, Paths: []string{path}}, progress, r, receiver)
	if serr == nil {
		t.Fatalf("sender succeeded after dropping the connection")
	}
	if rerr == nil {
		t.Fatalf("receiver accepted an interrupted transfer")
	}
	if n := len(dirEntries(t, dest)); n != 0 {
		t.Fatalf("interrupted transfer left %d files in the destination", n)
	}
	if n := len(dirEntries(t, staging)); n != 0 {
		t.Fatalf("interrupted transfer left %d partial files in staging", n)
	}
}

func TestMaliciousOfferNamesAreRejected(t *testing.T) {
	ctx := testCtx(t)
	for _, name := range []string{"../evil.sh", "/etc/passwd", ".bashrc", `a\b`} {
		t.Run(name, func(t *testing.T) {
			sender, receiver := pairedSessions(t, ctx)
			dest := t.TempDir()
			r := receiverFor(t, dest, acceptAll)
			go func() {
				_, _ = r.Serve(ctx, receiver)
			}()
			body := offerBody{ID: newOfferID(), Kind: KindFile, Items: []itemBody{{Name: name, Size: 3}}}
			if err := sender.Send(protocol.MsgTransferOffer, body); err != nil {
				t.Fatal(err)
			}
			reply, err := sender.Receive()
			if err != nil {
				t.Fatal(err)
			}
			if reply.Type != protocol.MsgTransferDecline {
				t.Fatalf("reply = %s, want DECLINE", reply.Type)
			}
			if len(dirEntries(t, dest)) != 0 {
				t.Fatalf("malicious offer wrote files")
			}
		})
	}
}

func TestCorruptedHashIsRejectedAndNothingPublished(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	dest := t.TempDir()
	staging := t.TempDir()
	r, _ := NewReceiver(ReceiverConfig{Decide: acceptAll, StagingDir: staging, DestDir: dest, Destination: DesktopDestination{Dir: dest}})

	recvErr := make(chan error, 1)
	go func() {
		_, err := r.Serve(ctx, receiver)
		recvErr <- err
	}()
	id := newOfferID()
	if err := sender.Send(protocol.MsgTransferOffer, offerBody{ID: id, Kind: KindFile, Items: []itemBody{{Name: "a.txt", Size: 5}}}); err != nil {
		t.Fatal(err)
	}
	if reply, err := sender.Receive(); err != nil || reply.Type != protocol.MsgTransferAccept {
		t.Fatalf("expected ACCEPT, got %v (%v)", reply.Type, err)
	}
	if err := sender.Send(protocol.MsgTransferData, dataBody{ID: id, Index: 0, Data: []byte("hello")}); err != nil {
		t.Fatal(err)
	}
	wrong := sha256.Sum256([]byte("other"))
	if err := sender.Send(protocol.MsgTransferEnd, endBody{ID: id, Index: 0, Size: 5, SHA256: hex.EncodeToString(wrong[:])}); err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(protocol.MsgTransferDone, idBody{ID: id}); err != nil {
		t.Fatal(err)
	}
	if err := <-recvErr; !errors.Is(err, apperr.ErrIntegrity) {
		t.Fatalf("receiver error = %v, want ErrIntegrity", err)
	}
	if len(dirEntries(t, dest)) != 0 || len(dirEntries(t, staging)) != 0 {
		t.Fatalf("corrupted transfer left files behind")
	}
}

func TestDataBeyondDeclaredSizeIsRejected(t *testing.T) {
	ctx := testCtx(t)
	sender, receiver := pairedSessions(t, ctx)
	dest := t.TempDir()
	r := receiverFor(t, dest, acceptAll)
	recvErr := make(chan error, 1)
	go func() {
		_, err := r.Serve(ctx, receiver)
		recvErr <- err
	}()
	id := newOfferID()
	_ = sender.Send(protocol.MsgTransferOffer, offerBody{ID: id, Kind: KindFile, Items: []itemBody{{Name: "a.txt", Size: 3}}})
	_, _ = sender.Receive()
	_ = sender.Send(protocol.MsgTransferData, dataBody{ID: id, Index: 0, Data: []byte("hello")})
	if err := <-recvErr; !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("data above declared size = %v, want ErrMalformed", err)
	}
	if len(dirEntries(t, dest)) != 0 {
		t.Fatalf("oversized transfer wrote files")
	}
}

func TestUntrustedSessionCannotReceive(t *testing.T) {
	ctx := testCtx(t)
	a, b := newIdentity(t), newIdentity(t)
	c1, c2 := tcpPair(t)
	ch := make(chan *transport.Session, 1)
	go func() {
		s, _ := transport.Server(ctx, c2, b, newStore(t))
		ch <- s
	}()
	_, err := transport.Client(ctx, c1, a, nil, newStore(t))
	if err != nil {
		t.Fatal(err)
	}
	provisional := <-ch
	r := receiverFor(t, t.TempDir(), acceptAll)
	if _, err := r.Serve(ctx, provisional); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("Serve on an untrusted session = %v, want ErrUnauthorized", err)
	}
}

func TestSenderRejectsMissingFileBeforeOffering(t *testing.T) {
	ctx := testCtx(t)
	sender, _ := pairedSessions(t, ctx)
	_, err := Send(ctx, sender, SendSpec{Kind: KindFile, Paths: []string{filepath.Join(t.TempDir(), "nope.txt")}}, nil)
	if err == nil {
		t.Fatalf("Send of a missing file succeeded")
	}
}

func TestBusyReceiverDeclinesExtraTransfers(t *testing.T) {
	ctx := testCtx(t)
	a, b := newIdentity(t), newIdentity(t)
	sa, sb := newStore(t), newStore(t)
	pairOnce(t, ctx, a, b, sa, sb)

	release := make(chan struct{})
	r, err := NewReceiver(ReceiverConfig{
		Decide:        func(context.Context, Offer) bool { <-release; return true },
		StagingDir:    t.TempDir(),
		DestDir:       t.TempDir(),
		Destination:   DesktopDestination{Dir: t.TempDir()},
		MaxConcurrent: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	src := t.TempDir()
	path, _ := randomFile(t, src, "x.bin", 16)
	s1, r1 := connectTrusted(t, ctx, a, b, sa, sb)
	first := make(chan error, 1)
	go func() {
		_, err := r.Serve(ctx, r1)
		first <- err
	}()
	firstSend := make(chan error, 1)
	go func() {
		_, err := Send(ctx, s1, SendSpec{Kind: KindFile, Paths: []string{path}}, nil)
		firstSend <- err
	}()
	waitFor(t, func() bool { return r.active() == 1 })

	s2, r2 := connectTrusted(t, ctx, a, b, sa, sb)
	_, serr, _, _ := run(t, ctx, s2, SendSpec{Kind: KindFile, Paths: []string{path}}, nil, r, r2)
	if !errors.Is(serr, apperr.ErrBusy) {
		t.Fatalf("second transfer = %v, want ErrBusy", serr)
	}
	close(release)
	if err := <-firstSend; err != nil {
		t.Fatalf("first transfer: %v", err)
	}
	if err := <-first; err != nil {
		t.Fatalf("first receive: %v", err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not reached in time")
}

var _ = bytes.Equal
