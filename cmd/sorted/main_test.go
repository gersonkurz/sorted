package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
// runC compiles a C file with the host compiler and returns what the program
// prints. It skips the test when there is no C compiler.
func runC(t *testing.T, file string) string {
	t.Helper()
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler (cc) on PATH")
	}
	bin := filepath.Join(t.TempDir(), "prog")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if out, err := exec.Command(compiler, "-std=c17", "-w", "-o", bin, file).CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("running %s: %v", file, err)
	}
	return string(out)
}

func normalise(s string) string {
	return strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
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

// A file that cannot be written is reported on stderr and fails the exit
// code; the original silently skips it, and the program runs either way.
func TestUnwritableOutputFile(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "missing-dir", "out.c")
	r := runCLI("--to-c", bad, filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s"))
	if r.code != 1 || normalise(r.stdout) != "Hello, World." || !strings.HasPrefix(r.stderr, "sorted: ") {
		t.Errorf("%+v", r)
	}
}

// TestGolden is the end-to-end suite: each sample runs through the CLI with
// --to-c and --dump, and its output and table dump must equal what
// Sorted.exe produced (testdata/golden, see README.md there). The C
// translation, compiled, must print the same output too (where there is a C
// compiler); it is not compared with the original's /C captures, which do
// not (see internal/emit).
// The diagnostics captures are checked by TestMissingFile and
// TestParseFailure.
func TestGolden(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			cFile, dumpFile := filepath.Join(dir, name+".c"), filepath.Join(dir, name+".dump")
			r := runCLI("--to-c", cFile, "--dump", dumpFile, filepath.Join("..", "..", "legacy", "sorted.win32", name+".s"))
			if r.code != 0 || r.stderr != "" {
				t.Fatalf("%+v", r)
			}
			if got, want := normalise(r.stdout), normalise(golden(t, name+".out")); got != want {
				t.Errorf("output %q, want %q", got, want)
			}
			got, err := os.ReadFile(dumpFile)
			if err != nil {
				t.Fatal(err)
			}
			if want := golden(t, name+".dump"); string(got) != want {
				t.Errorf("%s.dump:\n%s\nwant:\n%s", name, got, want)
			}
			if got, want := normalise(runC(t, cFile)), normalise(golden(t, name+".out")); got != want {
				t.Errorf("--to-c output %q, want %q", got, want)
			}
		})
	}
}

// The examples: the committed Sorted! versions are what --from-c writes
// today (run "just examples" after changing the compiler), English and
// German print the same, and that is what the C program prints compiled
// natively (where there is a C compiler).
func TestExamples(t *testing.T) {
	sources, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.c"))
	if err != nil || len(sources) == 0 {
		t.Fatalf("no examples: %v", err)
	}
	for _, src := range sources {
		base := strings.TrimSuffix(src, ".c")
		t.Run(filepath.Base(base), func(t *testing.T) {
			var printed []string
			for _, v := range []struct{ file, lang string }{{base + ".s", "en"}, {base + ".de.s", "de"}} {
				committed, err := os.ReadFile(v.file)
				if err != nil {
					t.Fatal(err)
				}
				r := runCLI("--from-c", src, "--lang", v.lang)
				if r.code != 0 || r.stdout != strings.ReplaceAll(string(committed), "\r\n", "\n") {
					t.Fatalf("%s is not what --from-c writes now (run: just examples): %+v", v.file, r.stderr)
				}
				run := runCLI(v.file)
				if run.code != 0 {
					t.Fatalf("running %s: %+v", v.file, run)
				}
				printed = append(printed, run.stdout)
			}
			if printed[0] != printed[1] {
				t.Errorf("English and German print differently")
			}
			if native := runC(t, src); printed[0] != native {
				t.Errorf("Sorted! printed %q..., C printed %q...", printed[0][:min(80, len(printed[0]))], native[:min(80, len(native))])
			}
		})
	}
}

// --lang prints the program instead of running it; the printed program runs
// like the original.
func TestLang(t *testing.T) {
	r := runCLI("--lang", "de", filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s"))
	if r.code != 0 || r.stderr != "" || !strings.HasPrefix(r.stdout, "Dieses Programm benutzt die Zahlen") || !strings.HasSuffix(r.stdout, "Hervorragend.\n") {
		t.Fatalf("%+v", r)
	}
	r = runCLI(writeProgram(t, r.stdout))
	if r.code != 0 || normalise(r.stdout) != "Hello, World." {
		t.Errorf("running the German version: %+v", r)
	}
	if r := runCLI("--lang", "fr", "hello.s"); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("unknown language: %+v", r)
	}
}

// --from-c compiles C into Sorted! and prints it; the printed program runs.
func TestFromC(t *testing.T) {
	dir := t.TempDir()
	cFile := filepath.Join(dir, "hi.c")
	src := "#include <stdio.h>\nint main() { int i = 0; while (i < 3) { putchar(72 + i); i = i + 1; } putchar(10); }\n"
	if err := os.WriteFile(cFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	for lang, head := range map[string]string{"": "This code uses the numbers", "de": "Dieses Programm benutzt die Zahlen"} {
		args := []string{"--from-c", cFile}
		if lang != "" {
			args = append(args, "--lang", lang)
		}
		r := runCLI(args...)
		if r.code != 0 || r.stderr != "" || !strings.HasPrefix(r.stdout, head) {
			t.Fatalf("lang %q: %+v", lang, r)
		}
		if run := runCLI(writeProgram(t, r.stdout)); run.code != 0 || run.stdout != "HIJ\n" {
			t.Errorf("lang %q: running the compiled program: %+v", lang, run)
		}
	}
	// --dump and --to-c describe the compiled program as any Sorted!
	// interpreter sees it: the same as loading the printed program.
	r := runCLI("--from-c", cFile, "--dump", filepath.Join(dir, "a.dump"), "--to-c", filepath.Join(dir, "a.c"))
	if r.code != 0 {
		t.Fatalf("%+v", r)
	}
	if r := runCLI("--dump", filepath.Join(dir, "b.dump"), "--to-c", filepath.Join(dir, "b.c"), writeProgram(t, r.stdout)); r.code != 0 {
		t.Fatalf("%+v", r)
	}
	for _, pair := range [][2]string{{"a.dump", "b.dump"}, {"a.c", "b.c"}} {
		a, errA := os.ReadFile(filepath.Join(dir, pair[0]))
		b, errB := os.ReadFile(filepath.Join(dir, pair[1]))
		if errA != nil || errB != nil || len(a) == 0 || string(a) != string(b) {
			t.Errorf("%s and %s differ (%v, %v):\n%s\n---\n%s", pair[0], pair[1], errA, errB, a, b)
		}
	}
	if c, err := os.ReadFile(filepath.Join(dir, "a.c")); err != nil || !strings.HasPrefix(string(c), "/* Written by sorted --to-c") {
		t.Errorf("--to-c wrote %q, %v", c, err)
	}
}

// failingWriter fails every write, like a full disk behind a redirect.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

// A program that cannot be printed is a failure, not a truncated success.
func TestPrintFailure(t *testing.T) {
	cFile := filepath.Join(t.TempDir(), "x.c")
	if err := os.WriteFile(cFile, []byte("int main() { putchar(65); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, args := range map[string][]string{
		"--from-c": {"--from-c", cFile},
		"--lang":   {"--lang", "de", filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s")},
	} {
		var stderr bytes.Buffer
		if code := run(args, strings.NewReader(""), failingWriter{}, &stderr); code != 1 || stderr.String() != "sorted: disk full\n" {
			t.Errorf("%s: exit %d, stderr %q", name, code, stderr.String())
		}
	}
}

func TestFromCErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.c")
	if err := os.WriteFile(bad, []byte("int main() {\n  int a;\n  a = a ? 1 : 2;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI("--from-c", bad); r.code != 1 || r.stdout != "" || r.stderr != "sorted: "+bad+":3:9: not supported in Sorted! (yet): ?:\n" {
		t.Errorf("syntax: %+v", r)
	}
	big := filepath.Join(dir, "big.c")
	if err := os.WriteFile(big, []byte("int main() { putchar(1 << 32); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI("--from-c", big); r.code != 1 || !strings.Contains(r.stderr, big+":1:27: shift count 32 is out of range") {
		t.Errorf("lowering: %+v", r)
	}
	if r := runCLI("--from-c", filepath.Join(dir, "nothere.c")); r.code != 1 || !strings.HasPrefix(r.stderr, "sorted: ") {
		t.Errorf("missing file: %+v", r)
	}
	if r := runCLI("--from-c", bad, "extra.s"); r.code != 2 || !strings.Contains(r.stderr, "sorted --from-c PROGRAM.c") {
		t.Errorf("extra argument: %+v", r)
	}
}
