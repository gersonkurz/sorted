package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gersonkurz/sorted/internal/render"
)

type result struct {
	code           int
	stdout, stderr string
}

func runCLI(args ...string) result { return runCLIIn("", args...) }

// The tests pick English wherever Sorted! would pick a language at random
// (the help, --from-c without --lang); the tests of the random pick put it
// back.
func TestMain(m *testing.M) {
	pickLang = func() render.Lang { return render.English }
	os.Exit(m.Run())
}

// runCLIIn runs the command line with stdin.
func runCLIIn(stdin string, args ...string) result {
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

// A C program that reads compiles to Very Sorted!, which runs it on stdin,
// and --lang translates it with its marker.
func TestFromCVery(t *testing.T) {
	cFile := filepath.Join(t.TempDir(), "rev.c")
	src := "#include <stdio.h>\nint main() { char s[64]; int n = 0, c; while ((c = getchar()) != -1 && c != 10) s[n++] = c; while (n > 0) putchar(s[--n]); putchar(10); }\n"
	if err := os.WriteFile(cFile, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	r := runCLI("--from-c", cFile, "--english")
	if r.code != 0 || !strings.HasSuffix(r.stdout, "This code is very cool.\n") || !strings.Contains(r.stdout, "the first input") {
		t.Fatalf("%+v", r)
	}
	prog := writeProgram(t, r.stdout)
	if run := runCLIIn("Ordinata non errant\n", prog); run.code != 0 || run.stdout != "tnarre non atanidrO\n" {
		t.Errorf("running it: %+v", run)
	}
	if de := runCLI("--deutsch", prog); de.code != 0 || !strings.HasSuffix(de.stdout, "Dieses Programm ist ganz hervorragend.\n") || !strings.Contains(de.stdout, "die erste Eingabe") {
		t.Errorf("--deutsch: %+v", de)
	}
}

// A Very Sorted! program in UTF-8 German runs, prints German in UTF-8, and
// translates to English and back (#28).
func TestVeryUTF8(t *testing.T) {
	prog := writeProgram(t, `Dieses Programm benutzt die Zahlen fünf, zwölf, und dreißig.
Dieses Programm geht nirgendwo hin.
Dieses Programm schreibt die dritte Zahl als eine deutsche Ordinalzahl.
Dieses Programm kann nicht lesen.
Dieses Programm benutzt keine Summen.
Dieses Programm benutzt keine Bedingungen.
Dieses Programm benutzt keine Sprungziele.
Dieses Programm benutzt keine geordneten Differenzen.
Dieses Programm benutzt keine Zuweisungen.
Dieses Programm benutzt keine Produkte.
Dieses Programm implementiert die erste Ausgabe.
Dieses Programm benutzt keine Verhältnisse.
Dieses Programm ist unlogisch.
Dieses Programm ist ganz hervorragend.
`)
	if r := runCLI(prog); r.code != 0 || r.stdout != "dreißigste\n" {
		t.Errorf("running it: %+v", r)
	}
	en := runCLI("--english", prog)
	if en.code != 0 || !strings.Contains(en.stdout, "five,") || !strings.HasSuffix(en.stdout, "This code is very cool.\n") {
		t.Fatalf("--english: %+v", en)
	}
	if de := runCLI("--deutsch", writeProgram(t, en.stdout)); de.code != 0 || !strings.Contains(de.stdout, "fünf,") || !strings.Contains(de.stdout, "Verhältnisse") {
		t.Errorf("--deutsch: %+v", de)
	}
}

// A C program with a bitwise operator compiles to Very Sorted! NANDs and
// runs (#26).
func TestFromCNand(t *testing.T) {
	cFile := filepath.Join(t.TempDir(), "xor.c")
	if err := os.WriteFile(cFile, []byte("#include <stdio.h>\nint main() { int x = 65, y = 3; putchar(x ^ y); putchar(10); }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, lang := range []string{"--english", "--deutsch"} {
		r := runCLI("--from-c", cFile, lang)
		text := strings.Join(strings.Fields(r.stdout), " ")
		if r.code != 0 || !strings.Contains(text, "of not both") && !strings.Contains(text, "von nicht beiden,") {
			t.Fatalf("%s: %+v", lang, r)
		}
		if run := runCLI(writeProgram(t, r.stdout)); run.code != 0 || run.stdout != "B\n" {
			t.Errorf("%s: running it: %+v", lang, run)
		}
	}
}

// --version prints the newest dialect's marker, and the build unless it is
// exactly that dialect's release tag (#39).
func TestVersion(t *testing.T) {
	r := runCLI("--version")
	if r.code != 0 || r.stdout != versionLine(version)+"\n" || r.stderr != "" {
		t.Errorf("%+v", r)
	}
	for build, want := range map[string]string{
		"very-very-cool":       "This code is very very cool.",
		"dev":                  "This code is very very cool. (dev)",
		"a105f41-dirty":        "This code is very very cool. (a105f41-dirty)",
		"very-very-cool-dirty": "This code is very very cool. (very-very-cool-dirty)",
		"very-cool":            "This code is very very cool. (very-cool)",
		"cool":                 "This code is very very cool. (cool)",
	} {
		if got := versionLine(build); got != want {
			t.Errorf("%s: %q, want %q", build, got, want)
		}
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
	return runCIn(t, file, "")
}

// runCIn compiles and runs a C file natively, with stdin.
func runCIn(t *testing.T, file, stdin string) string {
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
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
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
// today (run "just examples" after changing the compiler), English, German,
// Italian, French, Portuguese, Japanese and Mandarin (hanzi and pinyin) print the same, and that is what the C program prints compiled
// natively (where there is a C compiler).
func TestExamples(t *testing.T) {
	sources, err := filepath.Glob(filepath.Join("..", "..", "examples", "*.c"))
	if err != nil || len(sources) == 0 {
		t.Fatalf("no examples: %v", err)
	}
	for _, src := range sources {
		base := strings.TrimSuffix(src, ".c")
		t.Run(filepath.Base(base), func(t *testing.T) {
			// An example that reads gets <name>.in as its input.
			stdin, err := os.ReadFile(base + ".in")
			if err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatal(err)
			}
			var printed []string
			for _, v := range []struct{ file, lang string }{{base + ".s", "en"}, {base + ".de.s", "de"}, {base + ".it.s", "it"}, {base + ".fr.s", "fr"}, {base + ".pt.s", "pt"}, {base + ".ja.s", "ja"}, {base + ".zh.s", "zh"}, {base + ".pinyin.s", "pinyin"}} {
				committed, err := os.ReadFile(v.file)
				if err != nil {
					t.Fatal(err)
				}
				r := runCLI("--from-c", src, "--lang", v.lang)
				if r.code != 0 || r.stdout != strings.ReplaceAll(string(committed), "\r\n", "\n") {
					t.Fatalf("%s is not what --from-c writes now (run: just examples): %+v", v.file, r.stderr)
				}
				run := runCLIIn(string(stdin), v.file)
				if run.code != 0 {
					t.Fatalf("running %s: %+v", v.file, run)
				}
				printed = append(printed, run.stdout)
			}
			for i, p := range printed {
				if p != printed[0] {
					t.Errorf("language %d prints differently", i)
				}
			}
			if native := runCIn(t, src, string(stdin)); printed[0] != native {
				t.Errorf("Sorted! printed %q..., C printed %q...", printed[0][:min(80, len(printed[0]))], native[:min(80, len(native))])
			}
		})
	}
}

// --lang prints the program instead of running it; the printed program runs
// like the original.
// A language may be named in any language Sorted! speaks or will, ignoring
// case and accents, with --lang or as a flag of its own.
func TestLanguageNames(t *testing.T) {
	hello := filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s")
	const en, de, it, fr, pt, ja, zh, py = "This code uses the numbers", "Dieses Programm benutzt die Zahlen", "Questo programma usa i numeri", "Ce programme utilise les nombres", "Este programa usa os números", "Kono puroguramu wa kazu", "这个程序使用数字", "Zhège chéngxù shǐyòng shùzì"
	for _, tt := range []struct {
		args []string
		head string
	}{
		{[]string{"--中文"}, zh}, {[]string{"--lang", "Chinesisch"}, zh}, {[]string{"--mandarin"}, zh}, {[]string{"--chinois"}, zh},
		{[]string{"--汉语"}, zh}, {[]string{"--putonghua"}, zh}, {[]string{"--zh"}, zh}, {[]string{"--中国語"}, zh},
		{[]string{"--pinyin"}, py}, {[]string{"--拼音"}, py}, {[]string{"--lang", "pīnyīn"}, py}, {[]string{"--zh-latn"}, py}, {[]string{"--ZH-Latn"}, py},
		{[]string{"--nihongo"}, ja}, {[]string{"--日本語"}, ja}, {[]string{"--lang", "Japanisch"}, ja}, {[]string{"--japonais"}, ja},
		{[]string{"--giapponese"}, ja}, {[]string{"--japones"}, ja}, {[]string{"--Japanese"}, ja}, {[]string{"--lang", "riyu"}, ja}, {[]string{"--ja"}, ja},
		{[]string{"--brasileiro"}, pt}, {[]string{"--lang", "português"}, pt}, {[]string{"--portugues"}, pt},
		{[]string{"--Portugiesisch"}, pt}, {[]string{"--portugais"}, pt}, {[]string{"--portoghese"}, pt}, {[]string{"--Brazilian"}, pt},
		{[]string{"--ポルトガル語"}, pt}, {[]string{"--lang", "putaoyayu"}, pt}, {[]string{"--pt"}, pt}, {[]string{"--lang", "portugue\u0302s"}, pt},
		{[]string{"--vaudois"}, fr}, {[]string{"--lang", "Französisch"}, fr}, {[]string{"--franzoesisch"}, fr},
		{[]string{"--français"}, fr}, {[]string{"--francese"}, fr}, {[]string{"--French"}, fr}, {[]string{"--Waadtlaendisch"}, fr},
		{[]string{"--フランス語"}, fr}, {[]string{"--lang", "fayu"}, fr}, {[]string{"--fr"}, fr},
		{[]string{"--italiano"}, it}, {[]string{"--lang", "Italienisch"}, it}, {[]string{"--italien"}, it},
		{[]string{"--Italian"}, it}, {[]string{"--イタリア語"}, it}, {[]string{"--lang", "yidaliyu"}, it}, {[]string{"--it"}, it},
		{[]string{"--lang", "Englisch"}, en}, {[]string{"--anglais"}, en}, {[]string{"--INGLESE"}, en},
		{[]string{"--lang=inglês"}, en}, {[]string{"--ingles"}, en}, {[]string{"--英語"}, en},
		{[]string{"--lang", "yingyu"}, en}, {[]string{"-eigo"}, en}, {[]string{"--en"}, en},
		{[]string{"--deutsch"}, de}, {[]string{"--lang", "German"}, de}, {[]string{"--allemand"}, de},
		{[]string{"--alemao"}, de}, {[]string{"--lang", "德语"}, de}, {[]string{"--déyǔ"}, de},
		{[]string{"--deutsch", "--german"}, de},
	} {
		r := runCLI(append(tt.args, hello)...)
		if r.code != 0 || !strings.HasPrefix(r.stdout, tt.head) {
			t.Errorf("%v: %+v", tt.args, r)
		}
	}
	if r := runCLI("--deutsch", "--anglais", hello); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("two languages: %+v", r)
	}
	if r := runCLI("--lang", "Deutsch", "--", "--english"); r.code != 1 || !strings.Contains(r.stdout, "--english is not intelligible") {
		t.Errorf("after --, a name is a file: %+v", r)
	}
	// Decomposed accents and kana name the same language as precomposed ones.
	for _, args := range [][]string{{"--lang", "ingle\u0302s"}, {"--ingle\u0302s"}, {"--lang", "yi\u0304ngyu\u030c"}, {"--englisch"}} {
		if r := runCLI(append(args, hello)...); r.code != 0 || !strings.HasPrefix(r.stdout, en) {
			t.Errorf("%q: %+v", args, r)
		}
	}
	for _, args := range [][]string{{"--lang", "\u30c8\u3099イツ語"}, {"--alema\u0303o"}, {"--de\u0301yu\u030c"}} {
		if r := runCLI(append(args, hello)...); r.code != 0 || !strings.HasPrefix(r.stdout, de) {
			t.Errorf("%q: %+v", args, r)
		}
	}
	if r := runCLI("--lang", "Franzo\u0308sisch", hello); r.code != 0 || !strings.HasPrefix(r.stdout, fr) {
		t.Errorf("decomposed umlaut: %+v", r)
	}
	if r := runCLI("--lang", "\u30db\u309aルトカ\u3099ル語", hello); r.code != 0 || !strings.HasPrefix(r.stdout, pt) {
		t.Errorf("decomposed kana: %+v", r)
	}
	// A name is only a flag where a flag can be: not as an option's value,
	// nor after the program. These write files, so they run in a temporary
	// directory on a copy of hello.s, which a rejected command must leave
	// alone.
	src, err := os.ReadFile(hello)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if err := os.WriteFile("prog.s", src, 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI("--dump", "--english", "prog.s"); r.code != 0 {
		t.Errorf("--dump to a file named --english: %+v", r)
	} else if _, err := os.Stat("--english"); err != nil {
		t.Errorf("the dump was not written to --english: %v", err)
	}
	if r := runCLI("--to-c", "--english", "prog.s", "prog.s"); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("--to-c --english with two programs: %+v", r)
	}
	if got, _ := os.ReadFile("prog.s"); string(got) != string(src) {
		t.Errorf("prog.s was changed by a rejected command")
	}
	if r := runCLI("--dump=--english", "--deutsch", "prog.s"); r.code != 0 || !strings.HasPrefix(r.stdout, de) {
		t.Errorf("--dump=NAME and a language: %+v", r)
	}
	if r := runCLI("prog.s", "--deutsch"); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("a name after the program: %+v", r)
	}
	if r := runCLI("--version", "--deutsch", "prog.s"); r.code != 0 || r.stdout != versionLine(version)+"\n" {
		t.Errorf("a bool flag before a name: %+v", r)
	}
	if r := runCLI("--help", "--deutsch"); !strings.HasPrefix(r.stderr, "Aufruf: sorted [--dump DATEI]") || !strings.Contains(r.stderr, "NAME ist jede Sprache") {
		t.Errorf("help in German: %q", r.stderr)
	}
	if r := runCLI("--lang", "fr", "--help"); !strings.HasPrefix(r.stderr, "usage : sorted [--dump FICHIER]") || !strings.Contains(r.stderr, "voilà") {
		t.Errorf("help in French: %q", r.stderr)
	}
	if r := runCLI("--中文", "--bogus"); !strings.Contains(r.stderr, "用法：sorted") {
		t.Errorf("usage error in Mandarin: %q", r.stderr)
	}
	if r := runCLI("--help"); !strings.Contains(r.stderr, "en: English, Englisch, anglais") || !strings.Contains(r.stderr, "zh: Mandarin") || !strings.Contains(r.stderr, "zh-latn: Pinyin") || strings.Contains(r.stderr, "not yet") {
		t.Errorf("help: %q", r.stderr)
	}
}

// keys folds case and accents however they are encoded, and keeps kana
// voicing, which is no accent.
func TestKeys(t *testing.T) {
	same := [][2]string{{"Français", "francais"}, {"franc\u0327ais", "FRANCAIS"}, {"Straße", "strasse"},
		{"Französisch", "franzoesisch"}, {"Franzo\u0308sisch", "franzosisch"}, {"ドイツ語", "\u30c8\u3099イツ語"}, {"yīngyǔ", "YINGYU"}}
	for _, p := range same {
		a, b := keys(p[0]), keys(p[1])
		if a[0] != b[0] && a[1] != b[1] {
			t.Errorf("%q and %q differ: %q, %q", p[0], p[1], a, b)
		}
	}
	if keys("ド") == keys("ト") {
		t.Errorf("voicing dropped: %q", keys("ド"))
	}
}

func TestLang(t *testing.T) {
	r := runCLI("--lang", "de", filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s"))
	if r.code != 0 || r.stderr != "" || !strings.HasPrefix(r.stdout, "Dieses Programm benutzt die Zahlen") || !strings.HasSuffix(r.stdout, "Hervorragend.\n") {
		t.Fatalf("%+v", r)
	}
	r = runCLI(writeProgram(t, r.stdout))
	if r.code != 0 || normalise(r.stdout) != "Hello, World." {
		t.Errorf("running the German version: %+v", r)
	}
	hello := filepath.Join("..", "..", "legacy", "sorted.win32", "hello.s")
	for flag, head := range map[string]string{"--english": "This code uses the numbers", "--german": "Dieses Programm benutzt die Zahlen"} {
		if r := runCLI(flag, hello); r.code != 0 || !strings.HasPrefix(r.stdout, head) {
			t.Errorf("%s: %+v", flag, r)
		}
	}
	if r := runCLI("--english", "--lang", "de", hello); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("two choices: %+v", r)
	}
	if r := runCLI("--lang", "klingon", "hello.s"); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
		t.Errorf("unknown language: %+v", r)
	}
	// Every language Sorted! names is spoken now; one that is named but not
	// spoken yet is a usage error of its own.
	defer func(ls []language) { languages = ls }(languages)
	languages = append(append([]language{}, languages...), language{"qya", []string{"Quenya"}, 0, false})
	if r := runCLI("--lang", "quenya", "hello.s"); r.code != 2 || r.stderr != "sorted: Sorted! does not speak quenya yet\n" {
		t.Errorf("a language not spoken yet: %+v", r)
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
	const en, de, it, fr, pt, ja, zh, py = "This code uses the numbers", "Dieses Programm benutzt die Zahlen", "Questo programma usa i numeri", "Ce programme utilise les nombres", "Este programa usa os números", "Kono puroguramu wa kazu", "这个程序使用数字", "Zhège chéngxù shǐyòng shùzì"
	for _, tt := range []struct {
		flags []string
		head  string
	}{
		{[]string{"--lang", "ja"}, ja}, {[]string{"--nihongo"}, ja},
		{[]string{"--lang", "zh"}, zh}, {[]string{"--中文"}, zh}, {[]string{"--pinyin"}, py},
		{[]string{"--lang", "pt"}, pt}, {[]string{"--brasileiro"}, pt},
		{[]string{"--lang", "fr"}, fr}, {[]string{"--vaudois"}, fr},
		{[]string{"--lang", "en"}, en}, {[]string{"--english"}, en},
		{[]string{"--lang", "de"}, de}, {[]string{"--german"}, de},
		{[]string{"--lang", "it"}, it}, {[]string{"--italiano"}, it},
	} {
		r := runCLI(append([]string{"--from-c", cFile}, tt.flags...)...)
		if r.code != 0 || r.stderr != "" || !strings.HasPrefix(r.stdout, tt.head) {
			t.Fatalf("%v: %+v", tt.flags, r)
		}
		if run := runCLI(writeProgram(t, r.stdout)); run.code != 0 || run.stdout != "HIJ\n" {
			t.Errorf("%v: running the compiled program: %+v", tt.flags, run)
		}
	}
	// Without a choice, the language is whatever pickLang picks...
	defer func(pick func() render.Lang) { pickLang = pick }(pickLang)
	for lang, head := range map[render.Lang]string{render.English: en, render.German: de, render.Italian: it, render.French: fr, render.Portuguese: pt, render.Japanese: ja, render.Mandarin: zh, render.Pinyin: py} {
		pickLang = func() render.Lang { return lang }
		if r := runCLI("--from-c", cFile); r.code != 0 || !strings.HasPrefix(r.stdout, head) {
			t.Errorf("picked %d: %+v", lang, r)
		}
	}
	// ...and pickLang picks any: 300 runs show all eight, unless chance is
	// against it about 3 times in 10^17.
	pickLang = defaultPick
	seen := map[string]bool{}
	for range 300 {
		r := runCLI("--from-c", cFile)
		for _, head := range []string{en, de, it, fr, pt, ja, zh, py} {
			if strings.HasPrefix(r.stdout, head) {
				seen[head] = true
			}
		}
	}
	if len(seen) != 8 {
		t.Errorf("languages picked: %v", seen)
	}
	pickLang = func() render.Lang { return render.English }
	// One choice at most; the help names neither (a random one, English
	// here).
	for _, flags := range [][]string{{"--english", "--german"}, {"--lang", "en", "--german"}, {"--lang", "de", "--english"}} {
		if r := runCLI(append([]string{"--from-c", cFile}, flags...)...); r.code != 2 || !strings.Contains(r.stderr, "usage: sorted") {
			t.Errorf("%v: %+v", flags, r)
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
	if err := os.WriteFile(bad, []byte("int main() {\n  int a;\n  a = sizeof a;\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runCLI("--from-c", bad); r.code != 1 || r.stdout != "" || r.stderr != "sorted: "+bad+":3:7: not supported in Sorted! (yet): 'sizeof'\n" {
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

// The help is in every language Sorted! speaks, each with every flag and
// the table of names, and in one picked at random unless one is named.
func TestHelpLanguages(t *testing.T) {
	for _, l := range languages {
		h, ok := helps[l.lang]
		if !ok {
			t.Errorf("no help in %s", l.code)
			continue
		}
		for _, flag := range []string{"--dump", "--from-c", "--lang", "--to-c", "--version", " sorted "} {
			if !strings.Contains(h, flag) {
				t.Errorf("%s: no %s", l.code, flag)
			}
		}
		if lines := strings.Count(h, "\n"); lines != 8 {
			t.Errorf("%s: %d lines", l.code, lines)
		}
		r := runCLI("--help", "--lang", l.code)
		if r.code != 2 || !strings.HasPrefix(r.stderr, h) || !strings.HasSuffix(r.stderr, languageTable()) {
			t.Errorf("%s: %+v", l.code, r)
		}
	}
	defer func(pick func() render.Lang) { pickLang = pick }(pickLang)
	pickLang = defaultPick
	seen := map[string]bool{}
	for range 300 {
		r := runCLI("--help")
		seen[r.stderr[:strings.Index(r.stderr, "\n")]] = true
	}
	if len(seen) != len(helps) {
		t.Errorf("help languages picked: %d of %d", len(seen), len(helps))
	}
}
