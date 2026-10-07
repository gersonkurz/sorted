package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCrossCompile builds the tool for every platform the justfile's
// `platforms` lists, which `just package` ships. It only runs when
// SORTED_CROSS is set (`just cross`), since it builds the tool many times.
func TestCrossCompile(t *testing.T) {
	if os.Getenv("SORTED_CROSS") == "" {
		t.Skip("set SORTED_CROSS (just cross) to build for every platform")
	}
	justfile, err := os.ReadFile(filepath.Join("..", "..", "justfile"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^platforms := "([^"]*)"`).FindSubmatch(justfile)
	if m == nil {
		t.Fatal("no platforms line in the justfile")
	}
	out := filepath.Join(t.TempDir(), "sorted")
	for _, platform := range strings.Fields(string(m[1])) {
		goos, goarch, ok := strings.Cut(platform, "/")
		if !ok {
			t.Fatalf("platform %q is not GOOS/GOARCH", platform)
		}
		cmd := exec.Command("go", "build", "-buildvcs=false", "-o", out, ".")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+goos, "GOARCH="+goarch)
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: %v\n%s", platform, err, b)
		}
	}
}
