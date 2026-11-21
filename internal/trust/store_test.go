package trust

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/identity"
)

func newKey(t *testing.T) *identity.Identity {
	t.Helper()
	id, err := identity.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return id
}

func newPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "confiancas", "trust.json")
}

func TestEmptyStoreTrustsNobody(t *testing.T) {
	store := Open(newPath(t))
	if store.IsTrusted(newKey(t).PublicKey()) {
		t.Fatalf("empty store trusted a key")
	}
	peers, err := store.List()
	if err != nil || len(peers) != 0 {
		t.Fatalf("List = %v (%v), want empty list", peers, err)
	}
}

func TestAddTrustsAndPersistsAcrossOpens(t *testing.T) {
	path := newPath(t)
	key := newKey(t)

	if err := Open(path).add(key.PublicKey(), "laptop"); err != nil {
		t.Fatalf("add: %v", err)
	}
	reopened := Open(path)
	if !reopened.IsTrusted(key.PublicKey()) {
		t.Fatalf("trust did not persist after reopening the store")
	}
	peers, _ := reopened.List()
	if len(peers) != 1 || peers[0].Name != "laptop" || peers[0].Fingerprint != key.Fingerprint() {
		t.Fatalf("persisted record is incorrect: %+v", peers)
	}
}

func TestAddRejectsDuplicate(t *testing.T) {
	store := Open(newPath(t))
	key := newKey(t)
	if err := store.add(key.PublicKey(), "a"); err != nil {
		t.Fatal(err)
	}
	if err := store.add(key.PublicKey(), "b"); !errors.Is(err, ErrAlreadyTrusted) {
		t.Fatalf("duplicate add = %v, want ErrAlreadyTrusted", err)
	}
}

func TestRemoveRevokesTrust(t *testing.T) {
	store := Open(newPath(t))
	key := newKey(t)
	if err := store.add(key.PublicKey(), "phone"); err != nil {
		t.Fatal(err)
	}
	if err := store.Remove(key.Fingerprint()); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if store.IsTrusted(key.PublicKey()) {
		t.Fatalf("revoked key is still trusted")
	}
	if err := store.Remove(key.Fingerprint()); !errors.Is(err, ErrNotTrusted) {
		t.Fatalf("Remove of unknown key = %v, want ErrNotTrusted", err)
	}
}

func TestRenameTrustedDevice(t *testing.T) {
	store := Open(newPath(t))
	key := newKey(t)
	_ = store.add(key.PublicKey(), "old")
	if err := store.Rename(key.Fingerprint(), "new name"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	peers, _ := store.List()
	if peers[0].Name != "new name" {
		t.Fatalf("name = %q, want %q", peers[0].Name, "new name")
	}
}

func TestValidateNameRejectsInvalidNames(t *testing.T) {
	cases := map[string]string{
		"empty":       "",
		"only spaces": "   ",
		"line break":  "a\nb",
		"too long":    strings.Repeat("x", 65),
	}
	for name, value := range cases {
		if err := ValidateName(value); !errors.Is(err, apperr.ErrMalformed) {
			t.Errorf("%s: ValidateName = %v, want ErrMalformed", name, err)
		}
	}
	if err := ValidateName("Living room laptop"); err != nil {
		t.Errorf("valid name rejected: %v", err)
	}
}

func TestCorruptedFileFailsClosedAndIsNotOverwritten(t *testing.T) {
	path := newPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := Open(path)
	key := newKey(t)

	if store.IsTrusted(key.PublicKey()) {
		t.Fatalf("corrupted store should fail closed (no trust)")
	}
	if _, err := store.List(); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("List corrupted = %v, want ErrMalformed", err)
	}
	if err := store.add(key.PublicKey(), "x"); err == nil {
		t.Fatalf("add on a corrupted file did not fail")
	}
	data, _ := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	if string(data) != "{corrupted" {
		t.Fatalf("corrupted file was silently overwritten")
	}
}

func TestRecordWithTamperedFingerprintIsRejected(t *testing.T) {
	path := newPath(t)
	keyA := newKey(t)
	keyB := newKey(t)
	if err := Open(path).add(keyA.PublicKey(), "a"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path) //nolint:gosec // G304: path inside t.TempDir() of this test
	swapped := bytes.Replace(data, []byte(keyA.Fingerprint().Hex()), []byte(keyB.Fingerprint().Hex()), 1)
	if err := os.WriteFile(path, swapped, 0o600); err != nil { //nolint:gosec // G703: path inside t.TempDir() of this test
		t.Fatal(err)
	}
	if _, err := Open(path).List(); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("tampered fingerprint = %v, want ErrMalformed", err)
	}
}

func TestStorePermissionsAreRestrictive(t *testing.T) {
	path := newPath(t)
	if err := Open(path).add(newKey(t).PublicKey(), "a"); err != nil {
		t.Fatal(err)
	}
	// Windows does not expose POSIX permission bits through os.FileMode; the assertion applies to the other systems.
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("store permission = %o, want 600", info.Mode().Perm())
		}
		dirInfo, _ := os.Stat(filepath.Dir(path))
		if dirInfo.Mode().Perm() != 0o700 {
			t.Fatalf("directory permission = %o, want 700", dirInfo.Mode().Perm())
		}
	}
}
