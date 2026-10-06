package compile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gersonkurz/sorted/internal/cc"
	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/render"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// toSorted compiles C source and writes it as Sorted! text in lang.
func toSorted(t *testing.T, src string, lang render.Lang) string {
	t.Helper()
	prog, err := cc.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	p, err := Compile(prog)
	if err != nil {
		t.Fatal(err)
	}
	text, _, err := render.Compose(p, lang)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// runSorted runs Sorted! text the way the CLI does: parse, then interpret.
func runSorted(t *testing.T, text string) string {
	t.Helper()
	p, err := syntax.Parse(syntax.Filter([]byte(text)))
	if err != nil {
		t.Fatalf("%v in:\n%s", err, text)
	}
	var out bytes.Buffer
	if err := interp.Run(p, strings.NewReader(""), &out, 10000000); err != nil {
		t.Fatalf("%v; output so far %q", err, out.String())
	}
	return out.String()
}

// runNative compiles C with the host compiler and runs it. It skips the
// test when there is no C compiler.
func runNative(t *testing.T, src string) string {
	t.Helper()
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler (cc) on PATH")
	}
	dir := t.TempDir()
	file, bin := filepath.Join(dir, "prog.c"), filepath.Join(dir, "prog")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(compiler, "-std=c17", "-w", "-o", bin, file).CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s", err, out)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		if _, exit := err.(*exec.ExitError); !exit {
			t.Fatal(err)
		}
	}
	return nativeText(string(out))
}

// nativeText undoes the newline translation of a text-mode stdout: on
// Windows, putchar(10) arrives as CRLF, while Sorted! writes LF. This is the
// golden-test rule (CRLF is LF) applied to native output.
func nativeText(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

func TestNativeText(t *testing.T) {
	if got := nativeText("Hi!\r\n1\r\n\r\n2\n\r"); got != "Hi!\n1\n\n2\n\r" {
		t.Errorf("nativeText: %q", got)
	}
}

// printNum is C code that prints the int expression e in decimal, then a
// newline, using the scratch variables pn and pp (no arrays, no functions).
func printNum(e string) string {
	return fmt.Sprintf(`pn = %s;
	if (pn < 0) { putchar(45); pn = -pn; }
	pp = 1;
	while (pp <= pn / 10) pp = pp * 10;
	while (pp > 0) { putchar(48 + pn / pp %% 10); pp = pp / 10; }
	putchar(10);
	`, e)
}

// The differential suite: every program prints the same natively and as
// Sorted!, in English and in German.
var programs = map[string]string{
	"hello": `#include <stdio.h>
int main() { putchar(72); putchar(105); putchar(33); putchar(10); return 0; }`,

	"arithmetic": `#include <stdio.h>
int pn; int pp;
int main() {
	int a = 17, b = 5;
	` + printNum("a + b") + printNum("a - b") + printNum("b - a") + printNum("a * b") +
		printNum("a / b") + printNum("a % b") + printNum("-a / b") + printNum("-a % b") +
		printNum("a % -b") + printNum("-a / -b") + printNum("-(a - 3) * (b + 1) - 2") + `
	return 0;
}`,

	"comparisons": `#include <stdio.h>
int pn; int pp;
int main() {
	int x = 3, y = 5;
	` + printNum("x < y") + printNum("y < x") + printNum("x <= x") + printNum("y <= x") +
		printNum("x > y") + printNum("y >= y") + printNum("x == 3") + printNum("x != 3") +
		printNum("(x < y) + (x == 3) * 10 + (y != 5) * 100") + `
}`,

	"globals and scopes": `#include <stdio.h>
int g = 7; int h = -4; int z;
int pn; int pp;
int main() {
	int x = 1;
	{ int x = 2; ` + printNum("x") + `}
	` + printNum("x") + printNum("g + h + z") + `
	{ int g = 100; ` + printNum("g") + `}
	` + printNum("g") + `
}`,

	"assignments": `#include <stdio.h>
int pn; int pp;
int main() {
	int a; int b; int c;
	a = b = c = 9;
	` + printNum("a + b + c") + `
	a = (b = 3) + 1;
	` + printNum("a * 10 + b") + `
	int n = 5;
	while ((n = n - 1) > 0) putchar(48 + n);
	putchar(10);
}`,

	"fizzbuzz": `#include <stdio.h>
int pn; int pp;
int main() {
	int i = 1;
	while (i <= 15) {
		if (i % 15 == 0) { putchar(70); putchar(66); putchar(10); }
		else if (i % 3 == 0) { putchar(70); putchar(10); }
		else if (i % 5 == 0) { putchar(66); putchar(10); }
		else { ` + printNum("i") + ` }
		i = i + 1;
	}
}`,

	"primes": `#include <stdio.h>
int pn; int pp;
int main() {
	int n = 2;
	while (n < 60) {
		int d = 2; int prime = 1;
		while (d * d <= n) { if (n % d == 0) prime = 0; d = d + 1; }
		if (prime) { ` + printNum("n") + `}
		n = n + 1;
	}
}`,

	"fibonacci": `#include <stdio.h>
int pn; int pp;
int main() {
	int a = 0, b = 1;
	while (a < 100000) { ` + printNum("a") + ` b = a + b; a = b - a; }
}`,

	"early return": `#include <stdio.h>
int main() {
	int i = 0;
	while (1) {
		putchar(65 + i);
		if (i == 4) { putchar(10); return 0; }
		i = i + 1;
	}
	putchar(63);
}`,

	"nothing": `int main() { }`,

	"conditions": `#include <stdio.h>
int main() {
	int a = 3, b = 4, i = 0;
	if (a != b) putchar(49); else putchar(48);
	if (a != 3) putchar(49); else putchar(48);
	if (a <= b) putchar(49); else putchar(48);
	if (b <= a) putchar(49); else putchar(48);
	if (a < b) putchar(49);
	if (b == 4) putchar(49);
	if (a - 3) putchar(49); else putchar(48);
	a + b; 7;
	while (i != 3) { putchar(97 + i); i = i + 1; }
	{ return 0; }
	putchar(33);
}`,
}

func TestDifferential(t *testing.T) {
	for name, src := range programs {
		t.Run(name, func(t *testing.T) {
			want := runNative(t, src)
			for _, lang := range []render.Lang{render.English, render.German} {
				text := toSorted(t, src, lang)
				if got := runSorted(t, text); got != want {
					t.Errorf("lang %d: Sorted! printed %q, C printed %q\n%s", lang, got, want, text)
				}
			}
		})
	}
}

func compileC(t *testing.T, src string) (*syntax.Program, error) {
	t.Helper()
	prog, err := cc.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return Compile(prog)
}

// "Thou shalt not have the same cardinal more than once": every constant is
// declared once, and equal expressions share one entry.
func TestSharing(t *testing.T) {
	p, err := compileC(t, `int main() { int a; int b; a = 5 + 3; b = 5 + 3; a = 3 + 5; b = a * 5; }`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(p.Data) != "[5 3]" {
		t.Errorf("numbers %v, want [5 3]", p.Data)
	}
	if n := p.Tables[syntax.Sums].Count; n != 2 { // 5+3 and 3+5
		t.Errorf("%d sums, want 2", n)
	}
	if n := p.Tables[syntax.Assigns].Count; n != 4 { // a=5+3, b=5+3, a=3+5, b=a*5
		t.Errorf("%d assignments, want 4", n)
	}
	if n := p.Tables[syntax.Statements].Count; n != 4 {
		t.Errorf("%d statements, want 4", n)
	}
}

// Negative constants are 0 - n; they need zero and the magnitude.
func TestNegativeConstant(t *testing.T) {
	p, err := compileC(t, `int g = -7; int main() { putchar(-3 + 0 * g); }`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(p.Data) != "[0 7 3]" {
		t.Errorf("numbers %v, want [0 7 3]", p.Data)
	}
}

// An empty main still needs one statement: a label.
func TestEmptyMain(t *testing.T) {
	p, err := compileC(t, `int main() { }`)
	if err != nil {
		t.Fatal(err)
	}
	if p.LabelsCount != 1 || p.Tables[syntax.Statements].Count != 1 {
		t.Errorf("labels %d, statements %v", p.LabelsCount, p.Tables[syntax.Statements])
	}
}

// A final return just ends the program; an earlier one jumps to the end.
func TestReturn(t *testing.T) {
	p, err := compileC(t, `int main() { putchar(65); return 0; }`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Tables[syntax.Jumps].Count != 0 || p.LabelsCount != 0 {
		t.Errorf("final return: %d jumps, %d labels", p.Tables[syntax.Jumps].Count, p.LabelsCount)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct{ src, want string }{
		{`int main() { int a = 1000000000; }`, "1:22: constants above 999999999 are not supported yet (1000000000)"},
		{`int main() { int a = -2147483647 - 1; }`, "1:22: constants above 999999999 are not supported yet (2147483647)"},
		{`int g = -1000000000; int main() { }`, "1:5: constants above 999999999 are not supported yet (1000000000)"},
		{`int main() { int a = -999999999; putchar(a); }`, ""},
		{`int main() { int a = putchar(65); }`, "1:22: using the result of putchar is not supported yet"},
	}
	for _, tt := range tests {
		_, err := compileC(t, tt.src)
		if tt.want == "" {
			if err != nil {
				t.Errorf("%s: %v", tt.src, err)
			}
			continue
		}
		var e *Error
		if !errors.As(err, &e) || err.Error() != tt.want {
			t.Errorf("%s\n got %v\nwant %s", tt.src, err, tt.want)
		}
	}
}
