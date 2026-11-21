package discovery

import (
	"errors"
	"testing"

	"tomorrow-drop/internal/apperr"
)

func TestParseEndpointAcceptsIPv4AndIPv6(t *testing.T) {
	cases := map[string]string{
		"192.168.1.20:7000": "192.168.1.20:7000",
		" 10.0.0.5:9000 ":   "10.0.0.5:9000",
		"[fd00::1]:7000":    "[fd00::1]:7000",
	}
	for input, want := range cases {
		ep, err := ParseEndpoint(input)
		if err != nil {
			t.Errorf("ParseEndpoint(%q): %v", input, err)
			continue
		}
		if ep.String() != want {
			t.Errorf("ParseEndpoint(%q) = %s, want %s", input, ep, want)
		}
	}
}

func TestParseEndpointRejectsInvalidInput(t *testing.T) {
	cases := []string{
		"",
		"192.168.1.20",
		"192.168.1.20:0",
		"192.168.1.20:70000",
		"192.168.1.20:port",
		"example.com:7000",
		"0.0.0.0:7000",
		"224.0.0.1:7000",
		"255.255.255.255:7000",
		"[ff02::1]:7000",
		"192.168.1.20:7000; drop",
	}
	for _, input := range cases {
		if _, err := ParseEndpoint(input); !errors.Is(err, apperr.ErrMalformed) {
			t.Errorf("ParseEndpoint(%q) = %v, want ErrMalformed", input, err)
		}
	}
}
