package discovery

import (
	"testing"
)

// FuzzDecodeDatagram checks that an accepted datagram re-encodes to a message
// that decodes to the same value: the validation is stable, not lenient.
func FuzzDecodeDatagram(f *testing.F) {
	f.Add([]byte(`{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef"}`))
	f.Add([]byte(`{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"desk","port":47521}`))
	f.Add([]byte(`{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"x","port":0}`))
	f.Add([]byte(`garbage`))
	f.Fuzz(func(t *testing.T, data []byte) {
		m, err := Decode(data)
		if err != nil {
			return
		}
		out, err := Encode(m)
		if err != nil {
			t.Fatalf("accepted a message that cannot be encoded: %v", err)
		}
		again, err := Decode(out)
		if err != nil || again != m {
			t.Fatalf("round trip changed the message: %+v -> %+v (%v)", m, again, err)
		}
	})
}

// FuzzParseEndpoint checks that an accepted endpoint is never unspecified,
// multicast or port zero, whatever the input.
func FuzzParseEndpoint(f *testing.F) {
	f.Add("192.168.1.20:47521")
	f.Add("[fd00::1]:7000")
	f.Add("0.0.0.0:7000")
	f.Add("example.com:80")
	f.Fuzz(func(t *testing.T, s string) {
		ep, err := ParseEndpoint(s)
		if err != nil {
			return
		}
		if ep.Port() == 0 || ep.Addr().IsUnspecified() || ep.Addr().IsMulticast() {
			t.Fatalf("accepted an unusable endpoint %s from %q", ep, s)
		}
	})
}
