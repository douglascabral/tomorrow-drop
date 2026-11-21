package transfer

import (
	"strings"
	"testing"
)

// FuzzValidateName checks that an accepted file name never contains a path
// separator, a stream or drive marker, a control character, or a hidden name.
func FuzzValidateName(f *testing.F) {
	f.Add("report.pdf")
	f.Add("../etc")
	f.Add("a/b")
	f.Add(".hidden")
	f.Fuzz(func(t *testing.T, name string) {
		if err := ValidateName(name); err != nil {
			return
		}
		if strings.ContainsAny(name, `/\:`) || strings.HasPrefix(name, ".") || name == "" {
			t.Fatalf("accepted unsafe name %q", name)
		}
		for _, r := range name {
			if r < 0x20 || r == 0x7f {
				t.Fatalf("accepted a control character in %q", name)
			}
		}
	})
}

// FuzzValidateLink checks that an accepted link is always http or https.
func FuzzValidateLink(f *testing.F) {
	f.Add("https://example.com/x")
	f.Add("file:///etc/passwd")
	f.Add("javascript:alert(1)")
	f.Fuzz(func(t *testing.T, link string) {
		if err := ValidateLink(link); err != nil {
			return
		}
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") {
			t.Fatalf("accepted a non-http link %q", link)
		}
	})
}
