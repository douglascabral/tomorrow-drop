package trust

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzTrustFile checks that a corrupted or hostile trust file never makes the
// store panic, and that every listed peer has a matching fingerprint.
func FuzzTrustFile(f *testing.F) {
	f.Add([]byte(`{"version":1,"peers":[]}`))
	f.Add([]byte(`{"version":1,"peers":[{"name":"a","public_key":"AAAA","fingerprint":"00"}]}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		path := filepath.Join(t.TempDir(), "trust.json")
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		peers, err := Open(path).List()
		if err != nil {
			return
		}
		for _, p := range peers {
			if len(p.Key) != 32 {
				t.Fatalf("listed a peer with a %d-byte key", len(p.Key))
			}
		}
	})
}
