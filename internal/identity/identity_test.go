package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func newStore(t *testing.T) Store {
	t.Helper()
	return Store{Dir: filepath.Join(t.TempDir(), "tdrop")}
}

func TestGenerateProducesConsistentEd25519Key(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if n := len(id.PublicKey()); n != 32 {
		t.Fatalf("len(PublicKey) = %d, want 32", n)
	}
	if !id.PublicKey().Equal(id.priv.Public()) {
		t.Fatalf("public key does not match the private key")
	}
}

func TestGenerateProducesDifferentIdentities(t *testing.T) {
	a, _ := Generate()
	b, _ := Generate()
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatalf("two generated identities are equal")
	}
}

func TestCreatePersistsAndIdentityIsReloaded(t *testing.T) {
	store := newStore(t)
	created, err := store.Create()
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if created.Fingerprint() != loaded.Fingerprint() {
		t.Fatalf("reloaded identity differs from the created one")
	}
}

func TestCreateAppliesRestrictivePermissions(t *testing.T) {
	store := newStore(t)
	if _, err := store.Create(); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(store.Dir, FileName))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("file permission = %o, want 600", info.Mode().Perm())
		}
		dirInfo, err := os.Stat(store.Dir)
		if err != nil {
			t.Fatal(err)
		}
		if dirInfo.Mode().Perm() != 0o700 {
			t.Fatalf("directory permission = %o, want 700", dirInfo.Mode().Perm())
		}
	}
}

func TestLoadMissingReturnsNotExist(t *testing.T) {
	_, err := newStore(t).Load()
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load missing = %v, want os.ErrNotExist", err)
	}
}

func TestCreateDoesNotOverwriteExistingIdentity(t *testing.T) {
	store := newStore(t)
	if _, err := store.Create(); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(filepath.Join(store.Dir, FileName)) //nolint:gosec // G304: test path inside t.TempDir()

	_, err := store.Create()
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("duplicate Create = %v, want os.ErrExist", err)
	}
	after, _ := os.ReadFile(filepath.Join(store.Dir, FileName)) //nolint:gosec // G304: test path inside t.TempDir()
	if string(before) != string(after) {
		t.Fatalf("Create silently overwrote the identity")
	}
}

func TestDeliberateRegenerationReplacesIdentity(t *testing.T) {
	store := newStore(t)
	original, err := store.Create()
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := store.Replace()
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if original.Fingerprint() == replacement.Fingerprint() {
		t.Fatalf("Replace kept the old identity")
	}
	reloaded, _ := store.Load()
	if reloaded.Fingerprint() != replacement.Fingerprint() {
		t.Fatalf("Replace did not persist the new identity")
	}
}

func TestLoadOrCreateCreatesWhenMissingAndReusesAfterwards(t *testing.T) {
	store := newStore(t)
	first, err := store.LoadOrCreate()
	if err != nil {
		t.Fatalf("first LoadOrCreate: %v", err)
	}
	second, err := store.LoadOrCreate()
	if err != nil {
		t.Fatalf("next LoadOrCreate: %v", err)
	}
	if first.Fingerprint() != second.Fingerprint() {
		t.Fatalf("LoadOrCreate generated a new identity instead of reusing it")
	}
}

func TestLoadOrCreateDoesNotOverwriteCorruptedFile(t *testing.T) {
	store := newStore(t)
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(store.Dir, FileName)
	if err := os.WriteFile(path, []byte("garbage"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := store.LoadOrCreate()
	if !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("LoadOrCreate corrupted = %v, want ErrMalformed", err)
	}
	data, _ := os.ReadFile(path) //nolint:gosec // G304: test path inside t.TempDir()
	if string(data) != "garbage" {
		t.Fatalf("corrupted file was silently overwritten")
	}
}

func TestLoadRejectsCorruptedFiles(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecDER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	ecPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecDER})

	cases := map[string][]byte{
		"empty":           {},
		"garbage":         []byte("not pem"),
		"wrong type":      pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("x")}),
		"truncated":       []byte("-----BEGIN PRIVATE KEY-----\nAAAA\n"),
		"ecdsa algorithm": ecPEM,
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			store := newStore(t)
			if err := os.MkdirAll(store.Dir, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(store.Dir, FileName), data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(); !errors.Is(err, apperr.ErrMalformed) {
				t.Fatalf("Load = %v, want ErrMalformed", err)
			}
		})
	}
}

func TestFingerprintHas64HexCharacters(t *testing.T) {
	id, _ := Generate()
	hexStr := id.Fingerprint().Hex()
	if len(hexStr) != 64 {
		t.Fatalf("Hex() has %d characters, want 64", len(hexStr))
	}
	if _, err := hex.DecodeString(hexStr); err != nil {
		t.Fatalf("Hex() is not hexadecimal: %v", err)
	}
}

func TestFingerprintIsSHA256OfCanonicalPublicKey(t *testing.T) {
	// SHA-256 of 32 zero bytes, reference value checked independently.
	fp, err := FingerprintOf(make([]byte, 32))
	if err != nil {
		t.Fatalf("FingerprintOf: %v", err)
	}
	const want = "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925"
	if fp.Hex() != want {
		t.Fatalf("Hex() = %s, want %s", fp.Hex(), want)
	}
}

func TestFingerprintRejectsWrongKeySize(t *testing.T) {
	if _, err := FingerprintOf(make([]byte, 31)); !errors.Is(err, apperr.ErrMalformed) {
		t.Fatalf("FingerprintOf 31 bytes = %v, want ErrMalformed", err)
	}
}

func TestFingerprintGroupsWithoutTruncation(t *testing.T) {
	fp, _ := FingerprintOf(make([]byte, 32))
	groups := strings.Split(fp.Grouped(), " ")
	if len(groups) != 16 {
		t.Fatalf("Grouped() has %d groups, want 16", len(groups))
	}
	for _, g := range groups {
		if len(g) != 4 {
			t.Fatalf("group %q has %d characters, want 4", g, len(g))
		}
	}
	if strings.ReplaceAll(fp.Grouped(), " ", "") != fp.Hex() {
		t.Fatalf("Grouped() does not preserve all characters of Hex()")
	}
}

func TestFormattingDoesNotRevealPrivateKey(t *testing.T) {
	id, _ := Generate()
	seedHex := hex.EncodeToString(id.priv.Seed())
	outputs := []string{
		fmt.Sprintf("%v", id),
		fmt.Sprintf("%+v", id),
		fmt.Sprintf("%#v", id),
	}
	for _, s := range outputs {
		if strings.Contains(s, seedHex) {
			t.Fatalf("formatting exposed the private key: %s", s)
		}
	}
}
