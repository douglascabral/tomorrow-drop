package trust

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"path/filepath"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/protocol"
)

// pair creates both sides of a pairing over the same session (binding) and version.
func pair(t *testing.T, binding []byte, version protocol.Version) (alice, bob *Pairing, keyA, keyB ed25519.PublicKey) {
	t.Helper()
	pubA, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubB, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	alice, err = NewPairing(Initiator, pubA, pubB, binding, version)
	if err != nil {
		t.Fatalf("NewPairing alice: %v", err)
	}
	bob, err = NewPairing(Responder, pubB, pubA, binding, version)
	if err != nil {
		t.Fatalf("NewPairing bob: %v", err)
	}
	return alice, bob, pubA, pubB
}

func session(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestMutualPairingIsComplete(t *testing.T) {
	alice, bob, _, _ := pair(t, session(1), protocol.CurrentVersion)
	alice.ConfirmLocal()
	bob.ConfirmLocal()

	tagA, err := alice.LocalTag()
	if err != nil {
		t.Fatalf("LocalTag alice: %v", err)
	}
	tagB, err := bob.LocalTag()
	if err != nil {
		t.Fatalf("LocalTag bob: %v", err)
	}
	if err := bob.VerifyRemote(tagA); err != nil {
		t.Fatalf("bob rejected alice's confirmation: %v", err)
	}
	if err := alice.VerifyRemote(tagB); err != nil {
		t.Fatalf("alice rejected bob's confirmation: %v", err)
	}
	if !alice.Complete() || !bob.Complete() {
		t.Fatalf("mutual pairing is not complete")
	}
}

func TestLocalTagRequiresExplicitConfirmation(t *testing.T) {
	alice, _, _, _ := pair(t, session(1), protocol.CurrentVersion)
	if _, err := alice.LocalTag(); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("LocalTag without confirmation = %v, want ErrUnauthorized", err)
	}
}

func TestUnilateralConfirmationIsNotComplete(t *testing.T) {
	alice, bob, _, _ := pair(t, session(1), protocol.CurrentVersion)
	alice.ConfirmLocal()
	tagA, _ := alice.LocalTag()

	if err := bob.VerifyRemote(tagA); err != nil {
		t.Fatalf("verification of alice's confirmation failed: %v", err)
	}
	if bob.Complete() {
		t.Fatalf("bob completed without confirming locally")
	}
	if alice.Complete() {
		t.Fatalf("alice completed without verifying bob's confirmation")
	}
}

func TestReflectedOwnConfirmationIsRejected(t *testing.T) {
	alice, _, _, _ := pair(t, session(1), protocol.CurrentVersion)
	alice.ConfirmLocal()
	tagA, _ := alice.LocalTag()
	if err := alice.VerifyRemote(tagA); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("reflection accepted: %v", err)
	}
}

func TestConfirmationCannotBeReusedInAnotherSession(t *testing.T) {
	session1Alice, _, _, _ := pair(t, session(1), protocol.CurrentVersion)
	_, session2Bob, _, _ := pair(t, session(2), protocol.CurrentVersion)
	session1Alice.ConfirmLocal()
	tagSession1, _ := session1Alice.LocalTag()

	if err := session2Bob.VerifyRemote(tagSession1); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("confirmation from another session was accepted: %v", err)
	}
}

func TestConfirmationCannotBeAttributedToAnotherPeer(t *testing.T) {
	alice, bob, _, _ := pair(t, session(1), protocol.CurrentVersion)
	alice.ConfirmLocal()
	tagA, _ := alice.LocalTag()

	otherKey, _, _ := ed25519.GenerateKey(rand.Reader)
	impostor, err := NewPairing(Responder, bob.local, otherKey, session(1), protocol.CurrentVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := impostor.VerifyRemote(tagA); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("confirmation attributed to another key was accepted: %v", err)
	}
}

func TestConfirmationCannotBeReusedWithAnotherVersion(t *testing.T) {
	alice, _, _, _ := pair(t, session(1), protocol.CurrentVersion)
	_, bobOtherVersion, _, _ := pair(t, session(1), protocol.CurrentVersion+1)
	alice.ConfirmLocal()
	tagA, _ := alice.LocalTag()
	if err := bobOtherVersion.VerifyRemote(tagA); err == nil {
		t.Fatalf("confirmation was accepted with a different protocol version (downgrade)")
	}
}

func TestNewPairingValidatesInputs(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	other, _, _ := ed25519.GenerateKey(rand.Reader)

	if _, err := NewPairing(Initiator, pub, other, session(1)[:16], protocol.CurrentVersion); err == nil {
		t.Errorf("short binding was accepted")
	}
	if _, err := NewPairing(Initiator, pub, pub, session(1), protocol.CurrentVersion); err == nil {
		t.Errorf("pairing with itself was accepted")
	}
	if _, err := NewPairing(Role(0), pub, other, session(1), protocol.CurrentVersion); err == nil {
		t.Errorf("invalid role was accepted")
	}
}

func TestCommitPairingOnlyAfterMutualConfirmation(t *testing.T) {
	alice, bob, _, _ := pair(t, session(1), protocol.CurrentVersion)
	aliceStore := Open(filepath.Join(t.TempDir(), "a.json"))

	alice.ConfirmLocal()
	tagA, _ := alice.LocalTag()
	_ = bob.VerifyRemote(tagA)

	if err := CommitPairing(aliceStore, alice, "bob"); !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("commit without mutual confirmation = %v, want ErrUnauthorized", err)
	}
	if peers, _ := aliceStore.List(); len(peers) != 0 {
		t.Fatalf("trust was persisted without mutual confirmation")
	}
}

func TestCommitPairingPersistsWhenComplete(t *testing.T) {
	alice, bob, _, pubB := pair(t, session(1), protocol.CurrentVersion)
	aliceStore := Open(filepath.Join(t.TempDir(), "a.json"))

	alice.ConfirmLocal()
	bob.ConfirmLocal()
	tagA, _ := alice.LocalTag()
	tagB, _ := bob.LocalTag()
	_ = bob.VerifyRemote(tagA)
	_ = alice.VerifyRemote(tagB)

	if err := CommitPairing(aliceStore, alice, "bob"); err != nil {
		t.Fatalf("CommitPairing: %v", err)
	}
	if !aliceStore.IsTrusted(pubB) {
		t.Fatalf("the peer's key was not trusted after a complete pairing")
	}
}
