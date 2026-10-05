package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type result struct {
	code           int
	stdout, stderr string
}

func runCLI(args ...string) result {
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(""), &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

func TestVersion(t *testing.T) {
	r := runCLI("--version")
	if r.code != 0 || r.stdout != "sorted "+version+"\n" || r.stderr != "" {
		t.Errorf("%+v", r)
	}
}

func TestUsage(t *testing.T) {
	for name, args := range map[string][]string{
		"no arguments": nil,
		"two programs": {"a.s", "b.s"},
		"unknown flag": {"--bogus", "a.s"},
	} {
		t.Run(name, func(t *testing.T) {
			r := runCLI(args...)
			if r.code != 2 || r.stdout != "" || !strings.Contains(r.stderr, "usage: sorted") {
				t.Errorf("%+v", r)
			}
		})
	}
}

// normalise applies the golden-test rule: CRLF is LF and a final newline is
// ignored.
func normalise(s string) string {
	return strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// The samples print what Sorted.exe printed for them
// (testdata/golden/<sample>.out).
func TestSamples(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) {
			r := runCLI(filepath.Join("..", "..", "legacy", "sorted.win32", name+".s"))
			want, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name+".out"))
			if err != nil {
				t.Fatal(err)
			}
			if r.code != 0 || r.stderr != "" || normalise(r.stdout) != normalise(string(want)) {
				t.Errorf("%+v, want stdout %q", r, want)
			}
		})
	}
}

func golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// A missing program: the preprocessor's message has no newline, then main
// reports the program as unintelligible. Captured from Sorted.exe as
// "Sorted.exe /Snothere.s" (testdata/golden/missing.out); the name is
// printed as given, so the capture's name is replaced by this test's path.
func TestMissingFile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "nothere.s")
	r := runCLI(name)
	want := strings.ReplaceAll(golden(t, "missing.out"), "nothere.s", name)
	if r.code != 1 || r.stdout != want || r.stderr != "" {
		t.Errorf("%+v, want stdout %q", r, want)
	}
}

// A parse failure: Parse prints its own line, then main reports the program
// as unintelligible. Captured from Sorted.exe as "Sorted.exe /Sduplicate.s"
// (testdata/golden/duplicate.out).
func TestParseFailure(t *testing.T) {
	name := filepath.Join("..", "..", "testdata", "golden", "duplicate.s")
	r := runCLI(name)
	want := strings.ReplaceAll(golden(t, "duplicate.out"), "duplicate.s is", name+" is")
	if r.code != 1 || r.stdout != want || r.stderr != "" {
		t.Errorf("%+v, want stdout %q", r, want)
	}
}

func writeProgram(t *testing.T, src string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "prog.s")
	if err := os.WriteFile(name, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return name
}

// A run-time error (undefined behaviour in the original, port-defined here)
// keeps the output so far on stdout and reports on stderr.
func TestRuntimeError(t *testing.T) {
	name := writeProgram(t, `This code uses the numbers seventy, sixtyfive, and eighty.
This code does never go anywhere.
This code writes the first ordered difference as a english cardinal.
This code cannot read.
This code does not use any sums.
This code does not use any conditions.
This code does not use any labels.
This code uses the ordered difference between the first number and the second number.
This code assigns the third number to the second number.
This code does not use any products.
This code implements the first output, the first assignment, and the first output.
This code does not use any ratios.
This code does not use any logical operations.
Cool.
`)
	r := runCLI(name)
	if r.code != 1 || r.stdout != "five\n" || r.stderr != "sorted: runtime error: cannot write a negative number as a cardinal\n" {
		t.Errorf("%+v", r)
	}
}
