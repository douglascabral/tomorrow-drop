package main

import (
	"strings"
	"testing"
)

// Regression: an empty device list was sent as null, which the interface could
// not decode, so it showed no address and no fingerprint. Lists are always arrays.
func TestStateSendsEmptyListsAsArrays(t *testing.T) {
	c, err := newCore(t.TempDir(), t.TempDir(), "iPad test", "en")
	if err != nil {
		t.Fatal(err)
	}
	js := c.toJSON()
	for _, key := range []string{`"devices":[]`, `"paired":[]`, `"jobs":[]`} {
		if !strings.Contains(js, key) {
			t.Fatalf("state lacks %s: %s", key, js)
		}
	}
	if strings.Contains(js, "null") {
		t.Fatalf("state has a null: %s", js)
	}
}
