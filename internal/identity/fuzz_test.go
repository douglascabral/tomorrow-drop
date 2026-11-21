package identity

import "testing"

// FuzzDecodeIdentity checks that no file content can make the private-key
// decoder panic, and that accepted keys are exactly Ed25519 key sizes.
func FuzzDecodeIdentity(f *testing.F) {
	id, err := Generate()
	if err != nil {
		f.Fatal(err)
	}
	seed, err := encode(id)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Add([]byte("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := decode(data)
		if err != nil {
			return
		}
		if len(got.PublicKey()) != 32 {
			t.Fatalf("accepted a key whose public part is %d bytes", len(got.PublicKey()))
		}
	})
}
