package main

import (
	"os"
	"path/filepath"
	"testing"

	"tomorrow-drop/internal/ui"
)

// Files copied in the file manager are sent as the files themselves. Their
// paths come from the clipboard on desktop systems; a folder is not sent.
func TestCopiedFilesArePlannedAsFiles(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "relatorio.pdf")
	if err := os.WriteFile(file, []byte("pdf-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := ui.PlanFiles([]string{file})
	if err != nil || len(plan.Items) != 1 || plan.Total != int64(len("pdf-bytes")) {
		t.Fatalf("plan = %+v (%v), want the copied file with its size", plan, err)
	}
	if _, err := ui.PlanFiles([]string{dir}); err == nil {
		t.Fatalf("a folder copied in the file manager must not be sent")
	}
}
