package internal_test

import (
	"os/exec"
	"strings"
	"testing"
)

// The shared core must never depend on the graphical toolkit (AGENTS.md:
// "Never import Fyne in the shared core"). This test enforces the rule with
// the real dependency graph, not with a convention.
func TestCoreDoesNotDependOnFyne(t *testing.T) {
	for _, pkg := range listPackages(t) {
		deps := depsOf(t, pkg)
		for _, dep := range deps {
			if strings.HasPrefix(dep, "fyne.io/") {
				t.Fatalf("core package %s depends on %s", pkg, dep)
			}
		}
	}
}

func listPackages(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "./...").CombinedOutput()
	if err != nil {
		t.Fatalf("go list ./...: %v\n%s", err, out)
	}
	var pkgs []string
	for _, p := range strings.Fields(string(out)) {
		if strings.Contains(p, "/cmd/") {
			continue
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func depsOf(t *testing.T, pkg string) []string {
	t.Helper()
	out, err := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", pkg).CombinedOutput() //nolint:gosec // G204: fixed go tool arguments in a test
	if err != nil {
		t.Fatalf("go list -deps %s: %v\n%s", pkg, err, out)
	}
	return strings.Fields(string(out))
}
