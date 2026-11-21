package pairing

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"tomorrow-drop/internal/identity"
	"tomorrow-drop/internal/transport"
	"tomorrow-drop/internal/trust"
)

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

func ctxTest(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func yes(identity.Fingerprint, identity.Fingerprint) bool { return true }
func no(identity.Fingerprint, identity.Fingerprint) bool  { return false }

// runPair pairs two devices. decideA and decideB are the two people's
// confirmations. It returns the errors of each side.
func runPair(t *testing.T, decideA, decideB func(identity.Fingerprint, identity.Fingerprint) bool) (idA, idB *identity.Identity, storeA, storeB *trust.Store, errA, errB error) {
	t.Helper()
	ctx := ctxTest(t)
	idA, _ = identity.Generate()
	idB, _ = identity.Generate()
	storeA = trust.Open(filepath.Join(t.TempDir(), "a.json"))
	storeB = trust.Open(filepath.Join(t.TempDir(), "b.json"))

	c1, c2 := tcpPair(t)
	ch := make(chan *transport.Session, 1)
	go func() {
		s, _ := transport.Server(ctx, c2, idB, storeB)
		ch <- s
	}()
	sa, err := transport.Client(ctx, c1, idA, nil, storeA)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	sb := <-ch
	// Each side reports on its own channel, so results cannot swap sides.
	doneA := make(chan error, 1)
	doneB := make(chan error, 1)
	go func() {
		_, err := Run(ctx, sa, idA, storeA, Prompts{Confirm: decideA, Name: func(identity.Fingerprint) string { return "other" }})
		doneA <- err
	}()
	go func() {
		_, err := Run(ctx, sb, idB, storeB, Prompts{Confirm: decideB, Name: func(identity.Fingerprint) string { return "other" }})
		doneB <- err
	}()
	errA = <-doneA
	errB = <-doneB
	return
}

func TestMutualConfirmationTrustsBothSides(t *testing.T) {
	idA, idB, storeA, storeB, errA, errB := runPair(t, yes, yes)
	if errA != nil || errB != nil {
		t.Fatalf("pairing errors: %v / %v", errA, errB)
	}
	if !storeA.IsTrusted(idB.PublicKey()) || !storeB.IsTrusted(idA.PublicKey()) {
		t.Fatalf("pairing did not store trust on both sides")
	}
}

func TestDeclinedComparisonStoresNothing(t *testing.T) {
	idA, idB, storeA, storeB, errA, _ := runPair(t, yes, no)
	if errA == nil {
		t.Fatalf("pairing succeeded although one person declined")
	}
	if storeA.IsTrusted(idB.PublicKey()) || storeB.IsTrusted(idA.PublicKey()) {
		t.Fatalf("trust was stored after a declined comparison")
	}
}

func TestCancelledErrorIsRecognizable(t *testing.T) {
	_, _, _, _, errA, errB := runPair(t, no, yes)
	if !errors.Is(errA, ErrCancelled) {
		t.Fatalf("declining side error = %v, want ErrCancelled", errA)
	}
	if errB == nil {
		t.Fatalf("the other side must not finish pairing alone")
	}
}
