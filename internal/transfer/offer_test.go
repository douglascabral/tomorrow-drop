package transfer

import (
	"errors"
	"strings"
	"testing"

	"tomorrow-drop/internal/apperr"
	"tomorrow-drop/internal/limits"
)

func TestValidateNameAcceptsPlainNames(t *testing.T) {
	for _, name := range []string{"report.pdf", "photo 2026.jpg", "notes", "archive.tar.gz"} {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q): %v", name, err)
		}
	}
}

func TestValidateNameRejectsMaliciousNames(t *testing.T) {
	names := []string{
		"", ".", "..", "../evil", "a/b", `a\b`, "/etc/passwd", `C:\x`,
		".bashrc", "con:stream", "bad\x00name", "bad\nname",
		strings.Repeat("a", limits.MaxFileNameBytes+1),
	}
	for _, name := range names {
		if err := ValidateName(name); !errors.Is(err, apperr.ErrMalformed) {
			t.Errorf("ValidateName(%q) = %v, want ErrMalformed", name, err)
		}
	}
}

func TestValidateOfferFileLimits(t *testing.T) {
	ok := Offer{ID: newOfferID(), Kind: KindFile, Items: []Item{{Name: "a.txt", Size: 10}}}
	if err := ValidateOffer(ok); err != nil {
		t.Fatalf("valid offer rejected: %v", err)
	}

	empty := ok
	empty.Items = nil
	if err := ValidateOffer(empty); !errors.Is(err, apperr.ErrMalformed) {
		t.Errorf("offer without items = %v, want ErrMalformed", err)
	}

	many := ok
	many.Items = make([]Item, limits.MaxFilesPerOffer+1)
	for i := range many.Items {
		many.Items[i] = Item{Name: "f.txt", Size: 1}
	}
	if err := ValidateOffer(many); !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Errorf("too many files = %v, want ErrLimitExceeded", err)
	}

	negative := ok
	negative.Items = []Item{{Name: "a.txt", Size: -1}}
	if err := ValidateOffer(negative); !errors.Is(err, apperr.ErrMalformed) {
		t.Errorf("negative size = %v, want ErrMalformed", err)
	}

	huge := ok
	huge.Items = []Item{{Name: "a.bin", Size: limits.MaxFileBytes + 1}}
	if err := ValidateOffer(huge); !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Errorf("file above the size limit = %v, want ErrLimitExceeded", err)
	}

	total := ok
	total.Items = []Item{{Name: "a.bin", Size: limits.MaxFileBytes}, {Name: "b.bin", Size: limits.MaxFileBytes}, {Name: "c.bin", Size: limits.MaxFileBytes}, {Name: "d.bin", Size: limits.MaxFileBytes}, {Name: "e.bin", Size: limits.MaxFileBytes}}
	if err := ValidateOffer(total); !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Errorf("offer above the total limit = %v, want ErrLimitExceeded", err)
	}
}

func TestValidateOfferTextAndLinks(t *testing.T) {
	text := Offer{ID: newOfferID(), Kind: KindText, Items: []Item{{Size: 5}}}
	if err := ValidateOffer(text); err != nil {
		t.Fatalf("valid text rejected: %v", err)
	}
	tooLong := Offer{ID: newOfferID(), Kind: KindText, Items: []Item{{Size: limits.MaxTextBytes + 1}}}
	if err := ValidateOffer(tooLong); !errors.Is(err, apperr.ErrLimitExceeded) {
		t.Errorf("text above limit = %v, want ErrLimitExceeded", err)
	}
	multi := Offer{ID: newOfferID(), Kind: KindText, Items: []Item{{Size: 1}, {Size: 1}}}
	if err := ValidateOffer(multi); !errors.Is(err, apperr.ErrMalformed) {
		t.Errorf("text with two items = %v, want ErrMalformed", err)
	}
	named := Offer{ID: newOfferID(), Kind: KindText, Items: []Item{{Name: "x.txt", Size: 1}}}
	if err := ValidateOffer(named); !errors.Is(err, apperr.ErrMalformed) {
		t.Errorf("text with a file name = %v, want ErrMalformed", err)
	}
}

func TestValidateLinkAcceptsOnlyHTTP(t *testing.T) {
	for _, link := range []string{"https://example.com/a?b=c", "http://intranet/page"} {
		if err := ValidateLink(link); err != nil {
			t.Errorf("ValidateLink(%q): %v", link, err)
		}
	}
	for _, link := range []string{"file:///etc/passwd", "javascript:alert(1)", "mailto:a@b.c", "not a url", ""} {
		if err := ValidateLink(link); !errors.Is(err, apperr.ErrMalformed) {
			t.Errorf("ValidateLink(%q) = %v, want ErrMalformed", link, err)
		}
	}
}

func TestOfferIDFormat(t *testing.T) {
	id := newOfferID()
	if len(id) != 16 {
		t.Fatalf("offer ID %q has %d characters, want 16", id, len(id))
	}
	if newOfferID() == id {
		t.Fatalf("two offer IDs are equal")
	}
}
