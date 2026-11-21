package discovery

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

func TestDiscoverRoundTrip(t *testing.T) {
	query, err := NewDiscover()
	if err != nil {
		t.Fatalf("NewDiscover: %v", err)
	}
	data, err := Encode(query)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got != query {
		t.Fatalf("round trip = %+v, want %+v", got, query)
	}
}

func TestAnnounceEchoesQueryNonce(t *testing.T) {
	query, _ := NewDiscover()
	answer, err := NewAnnounce(query, "living room", 7000)
	if err != nil {
		t.Fatalf("NewAnnounce: %v", err)
	}
	if answer.Nonce != query.Nonce || answer.Type != TypeAnnounce || answer.Port != 7000 {
		t.Fatalf("announce = %+v, does not echo the query", answer)
	}
}

func TestNoncesAreUnique(t *testing.T) {
	a, _ := NewDiscover()
	b, _ := NewDiscover()
	if a.Nonce == b.Nonce {
		t.Fatalf("two queries share a nonce")
	}
}

func TestDecodeRejectsMalformedInput(t *testing.T) {
	valid := `{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef"}`
	cases := map[string]string{
		"not json":           `{nope`,
		"unknown field":      `{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef","ip":"10.0.0.1"}`,
		"wrong version":      `{"v":2,"type":"DISCOVER","nonce":"0123456789abcdef"}`,
		"unknown type":       `{"v":1,"type":"HELLO","nonce":"0123456789abcdef"}`,
		"short nonce":        `{"v":1,"type":"DISCOVER","nonce":"abcd"}`,
		"non-hex nonce":      `{"v":1,"type":"DISCOVER","nonce":"zzzzzzzzzzzzzzzz"}`,
		"discover with port": `{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef","port":7000}`,
		"announce no name":   `{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","port":7000}`,
		"announce bad port":  `{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"a","port":70000}`,
		"announce ctrl name": `{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"a\u0007b","port":7000}`,
		"trailing data":      valid + ` {}`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode([]byte(input)); !errors.Is(err, apperr.ErrMalformed) {
				t.Fatalf("Decode = %v, want ErrMalformed", err)
			}
		})
	}
}

func TestDecodeAcceptsValidDiscover(t *testing.T) {
	if _, err := Decode([]byte(`{"v":1,"type":"DISCOVER","nonce":"0123456789abcdef"}`)); err != nil {
		t.Fatalf("valid DISCOVER rejected: %v", err)
	}
}

func TestDecodeRejectsOversizedDatagram(t *testing.T) {
	big := strings.Repeat("a", limits.MaxDiscoveryDatagram)
	input := `{"v":1,"type":"ANNOUNCE","nonce":"0123456789abcdef","name":"` + big + `","port":7000}`
	if _, err := Decode([]byte(input)); !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Fatalf("oversized datagram = %v, want ErrLimitExceeded", err)
	}
}

func TestAnnounceContainsNoSensitiveFields(t *testing.T) {
	query, _ := NewDiscover()
	answer, _ := NewAnnounce(query, "laptop", 7000)
	data, _ := Encode(answer)
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for field := range fields {
		switch field {
		case "v", "type", "nonce", "name", "port":
		default:
			t.Fatalf("announce carries unexpected field %q", field)
		}
	}
}
