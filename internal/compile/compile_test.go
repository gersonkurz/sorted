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

	"functions": `#include <stdio.h>
void printNum(int n);
int calls = 0;
int add(int a, int b) { calls++; return a + b; }
int square(int x) { return x * x; }
int gcd(int a, int b) { while (b != 0) { int t = b; b = a % b; a = t; } return a; }
int isPrime(int n) {
	if (n < 2) return 0;
	for (int d = 2; d * d <= n; d++) if (n % d == 0) return 0;
	return 1;
}
char narrow(int v) { return v; }
int asChar(char c) { return c; }
void shout(char c) { putchar(c); putchar(c); putchar('\n'); }
void nothing(void) { return; }
int main() {
	printNum(add(2, 3));
	printNum(square(add(1, 2)));
	printNum(add(add(1, 2), add(5, 4)));
	printNum(square(3) + square(4));
	printNum(gcd(1071, 462));
	for (int i = 0; i < 20; i++) if (isPrime(i)) printNum(i);
	printNum(narrow(200));
	printNum(asChar(200) * 1000 + asChar(-129));
	shout(321);
	nothing();
	printNum(calls);
	return 0;
}
void printNum(int n) {
	if (n < 0) { putchar('-'); n = -n; }
	int p = 1;
	while (p <= n / 10) p *= 10;
	while (p > 0) { putchar('0' + n / p % 10); p /= 10; }
	putchar('\n');
}`,

	"single call": `#include <stdio.h>
int twice(int x) { return x + x; }
void line(void) { putchar('-'); putchar('\n'); }
int main() { putchar('A' + twice(3)); line(); }`,

	"functions calling functions": `#include <stdio.h>
void digit(int d) { putchar('0' + d); }
void number(int n) { if (n >= 10) digit(n / 10); digit(n % 10); }
void pair(int a, int b) { number(a); putchar(','); number(b); putchar('\n'); }
int fib(int n) { int a = 0, b = 1; while (n-- > 0) { int t = a + b; a = b; b = t; } return a; }
int main() {
	for (int i = 0; i < 12; i++) pair(i, fib(i));
	int s = 0;
	for (int i = 0; i < 5; i++) s += fib(i) * fib(i + 1);
	pair(s, fib(10) - fib(9));
}`,

	"values across calls": `#include <stdio.h>
int g;
int reset(void) { g = 66; return 0; }
int first(int a, int b) { return a; }
int second(int a, int b) { return b; }
int shadow(int x) { { int x = 7; g = x; } return x; }
int main() {
	putchar(first(g = 65, reset()));
	putchar((g = 65) + reset());
	putchar(second(reset(), g = 67));
	if ((g = 65) != reset() + 65) putchar('N'); else putchar('Y');
	if ((g = 65) <= reset() + 65) putchar('Y'); else putchar('N');
	if ((g = 65) < reset() + 66 || g == 0) putchar('Y'); else putchar('N');
	int s = shadow(80);
	putchar(s + g);
	putchar('\n');
}`,

	"strings": `#include <stdio.h>
char greeting[] = "Hello, " "World!\n";
int main() {
	for (int i = 0; greeting[i]; i++) putchar(greeting[i]);
	char local[] = "Sorted!\tsingt\n";
	int i = 0;
	while (local[i] != 0) { putchar(local[i]); i++; }
	char partial[8] = "abc";
	for (i = 0; i < 8; i++) if (partial[i]) putchar(partial[i]); else putchar('.');
	putchar('\n');
}`,

	"arrays": `#include <stdio.h>
int pn; int pp;
int squares[10];
int g[6] = {3, -1, 4, 1, -5};
int main() {
	for (int i = 0; i < 10; i++) squares[i] = i * i;
	int sum = 0;
	for (int i = 0; i < 10; i++) sum += squares[i];
	` + printNum("sum") + printNum("squares[3] + squares[9]") + `
	int a[5] = {5, 3, 8, 1, 9};
	for (int i = 0; i < 5; i++)
		for (int j = 0; j + 1 < 5 - i; j++)
			if (a[j] > a[j + 1]) { int t = a[j]; a[j] = a[j + 1]; a[j + 1] = t; }
	for (int i = 0; i < 5; i++) { putchar('0' + a[i]); putchar(' '); }
	putchar('\n');
	int k = 0;
	a[k++] = 7;
	a[k] += 10;
	a[k]++;
	++a[k + 1];
	int old = a[k + 2]--;
	` + printNum("a[0] * 10000 + a[1] * 100 + a[2]") + printNum("a[3] * 100 + old") + printNum("k") + `
	int m = 0;
	a[m++] += 5;
	a[m++]++;
	` + printNum("m * 100 + a[0] - a[1]") + `
	int idx[3] = {2, 0, 1};
	` + printNum("a[idx[0]] + a[idx[idx[2]]]") + `
	int total = 0;
	for (int i = 0; i < 6; i++) total = total * 10 + g[i] + 5;
	` + printNum("total") + `
}`,

	"sieve": `#include <stdio.h>
int pn; int pp;
int composite[60];
int main() {
	for (int i = 2; i < 60; i++) {
		if (composite[i]) continue;
		` + printNum("i") + `
		for (int j = i * i; j < 60; j += i) composite[j] = 1;
	}
}`,

	"itoa": `#include <stdio.h>
int main() {
	int numbers[4] = {41281927, 0, -2026, 7};
	for (int k = 0; k < 4; k++) {
		char buf[12];
		int n = numbers[k], len = 0, neg = n < 0;
		if (neg) n = -n;
		buf[len++] = '0' + n % 10;
		n /= 10;
		while (n > 0) { buf[len++] = '0' + n % 10; n /= 10; }
		if (neg) putchar('-');
		while (len > 0) putchar(buf[--len]);
		putchar('\n');
	}
}`,

	"chars": `#include <stdio.h>
int pn; int pp;
char gc = 300;
int main() {
	char c = 200;
	` + printNum("c") + printNum("gc") + `
	c = c + 100;
	` + printNum("c") + `
	char s[3] = {127, -128, 0};
	s[0]++;
	s[1]--;
	s[2] = 'A' * 3;
	` + printNum("s[0]") + printNum("s[1]") + printNum("s[2]") + `
	int i = 1000;
	c = i;
	` + printNum("c") + `
	char hi = 127, lo = -128;
	int a = hi++, b = lo--;
	` + printNum("a") + printNum("hi") + printNum("b") + printNum("lo") + `
	char e[3] = {127, -128, 5};
	int j = 0;
	int x = e[j]++, y = e[j + 1]--, z = e[2]++;
	` + printNum("x") + printNum("e[0]") + printNum("y") + printNum("e[1]") + printNum("z") + `
	hi = 127;
	int w = ++hi;
	` + printNum("w") + `
}`,

	"local arrays": `#include <stdio.h>
int main() {
	for (int k = 1; k <= 3; k++) {
		int z[3] = {k};
		putchar('0' + z[0] + z[1] + z[2]);
		z[1] = 5;
		char word[4] = "ab";
		putchar(word[0]); putchar(word[1]); putchar('0' + word[2] + word[3]);
		word[2] = 'X';
	}
	putchar('\n');
}`,

	"for loops": `#include <stdio.h>
int pn; int pp;
int main() {
	int sum = 0;
	for (int i = 1; i <= 10; i++) sum += i;
	` + printNum("sum") + `
	for (int i = 0; i < 3; ++i)
		for (int j = 0; j < 3; j++) { putchar('a' + i); putchar('0' + j); putchar(' '); }
	putchar('\n');
	int k;
	for (k = 10; k > 0; k -= 3) putchar('0' + k % 10);
	putchar('\n');
	for (;;) { k++; if (k > 5) break; }
	` + printNum("k") + `
}`,

	"break and continue": `#include <stdio.h>
int main() {
	for (int i = 0; i < 20; i++) {
		if (i % 2) continue;
		if (i > 12) break;
		putchar('0' + i % 10);
	}
	putchar('\n');
	int n = 0;
	while (1) {
		n++;
		if (n == 3) continue;
		if (n == 7) break;
		int m = 0;
		while (m < n) { m++; if (m == 2) continue; putchar('*'); }
		putchar('\n');
	}
}`,

	"logic": `#include <stdio.h>
int pn; int pp;
int calls;
int main() {
	int a = 3, b = 0, d = 0;
	` + printNum("a && b") + printNum("a || b") + printNum("!a") + printNum("!b") + printNum("!!a") +
		printNum("a > 2 && a < 4") + printNum("b || a == 3 && !b") + `
	if (d != 0 && 10 / d > 1) putchar('x'); else putchar('y');
	if (d == 0 || 10 / d > 1) putchar('y'); else putchar('x');
	int guarded = d != 0 && 10 / d > 1;
	` + printNum("guarded") + `
	int side = 0;
	b && (side = 1);
	a || (side = 2);
	a && (side = 3);
	` + printNum("side") + `
	if (!(a == 3) || !(b != 0)) putchar('k');
	if (!(a < 4 && b < 1)) putchar('x'); else putchar('k');
	putchar('\n');
}`,

	"increments": `#include <stdio.h>
int pn; int pp;
int main() {
	int a = 5;
	int b = a++;
	int c = ++a;
	int d = a--;
	int e = --a;
	` + printNum("a * 1000 + b * 100 + c * 10 + d") + printNum("e") + `
	a += 10; a -= 3; a *= 2; a /= 3; a %= 5;
	` + printNum("a") + `
	int x = 0;
	while (x++ < 3) putchar('0' + x);
	putchar('\n');
}`,

	"characters": `#include <stdio.h>
int pn; int pp;
int main() {
	putchar('H'); putchar('e'); putchar('l'); putchar('l'); putchar('o'); putchar(',');
	putchar(' '); putchar('W'); putchar('\x6f'); putchar('\162'); putchar('l'); putchar('d');
	putchar('!'); putchar('\t'); putchar('\''); putchar('\\'); putchar('\n');
	` + printNum("'\\xff'") + printNum("'\\0'") + printNum("'A' - 'a'") + `
}`,

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
		{`int f(int n) { return f(n); } int main() { f(1); }`, "1:23: recursion is not supported yet ('f' calls itself, directly or indirectly)"},
		{`int g(int n); int f(int n) { return g(n); } int g(int n) { return f(n); } int main() { f(1); }`, "1:67: recursion is not supported yet ('f' calls itself, directly or indirectly)"},
		{`int f(int n); int main() { f(1); }`, "1:28: 'f' is declared but never defined"},
		{`void f(void) { } int main() { int a = f(); }`, "1:39: 'f' returns nothing (void)"},
		{`int f(int n) { return f(n); } int main() { }`, ""},
		{`int a[3]; int main() { a[3] = 1; }`, "1:26: index 3 is out of range for 'a' (3 elements)"},
		{`int a[3]; int main() { putchar(a[-1]); }`, "1:34: index -1 is out of range for 'a' (3 elements)"},
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

// An address that happens to equal a constant shares its slot; a filler
// keeps the pool at its planned size, so the variable cells stay where the
// addresses point.
func TestAddressCollision(t *testing.T) {
	c := &compiler{pool: []int32{5}, addrs: []int{3}, exprs: map[valKind]*table{}}
	for _, k := range []valKind{vSum, vDiff, vProd, vRatio, vCond} {
		c.exprs[k] = &table{}
	}
	p := c.program()
	if c.fillers != 1 {
		t.Errorf("%d fillers, want 1", c.fillers)
	}
	// two slots planned (one constant, one address): cells start at 2, and
	// the address of variable cell 3 is 2 + 3 = 5, the constant's own slot.
	if fmt.Sprint(p.Data) != "[5 0]" {
		t.Errorf("pool %v, want [5 0] (5 shared, 0 as filler)", p.Data)
	}
}

// The same end to end: find a program whose address equals one of its own
// constants, then check it still runs like C.
func TestAddressCollisionRuns(t *testing.T) {
	for n := 1; n < 100; n++ {
		src := fmt.Sprintf(`#include <stdio.h>
int a[4];
int main() { int i = 2; a[i] = %d; putchar('A' + a[2] - %d); putchar(a[i] / %d + 'a'); putchar('\n'); }`, n, n, n)
		prog, err := cc.Parse(src)
		if err != nil {
			t.Fatal(err)
		}
		c, _, err := compileProgram(prog)
		if err != nil {
			t.Fatal(err)
		}
		if c.fillers == 0 {
			continue
		}
		want := runNative(t, src)
		for _, lang := range []render.Lang{render.English, render.German} {
			text := toSorted(t, src, lang)
			if got := runSorted(t, text); got != want {
				t.Errorf("n=%d lang %d: %q, want %q\n%s", n, lang, got, want, text)
			}
		}
		return
	}
	t.Fatal("no program with an address collision found")
}

// Constant indices are plain cells: no pointer cells, no address numbers.
func TestConstantIndex(t *testing.T) {
	p, err := compileC(t, `int a[3]; int main() { a[0] = 5; a[2] = a[0] + 1; putchar(a[2]); }`)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range p.Entries(syntax.Assigns) {
		for _, op := range s.Ops {
			if op.Type&syntax.Indirect != 0 {
				t.Errorf("indirect operand %v in %v", op, s)
			}
		}
	}
	if fmt.Sprint(p.Data) != "[5 1]" {
		t.Errorf("numbers %v, want [5 1] (no addresses)", p.Data)
	}
}

// A program must fit Sorted!'s 193719 cells: declared numbers, variables and
// temporaries together (#19). The array below takes the rest after 'A' (one
// declared number) and the output cell.
func TestMemoryLimit(t *testing.T) {
	var e *Error
	fits := fmt.Sprintf("int a[%d]; int main() { putchar(65); }", memory-2)
	if _, err := compileC(t, fits); err != nil {
		t.Errorf("exactly %d cells: %v", memory, err)
	}
	// Variables alone may not exceed the memory, and counting them cannot
	// overflow, even with lengths near the int32 limit (a 32-bit int would
	// wrap here without the early check).
	huge := "int a[2147483647];\nint b[2];\nint main() { putchar(65); }"
	if _, err := compileC(t, huge); !errors.As(err, &e) || err.Error() != fmt.Sprintf("1:5: the program needs more than %d memory cells, which is all Sorted! has", memory) {
		t.Errorf("huge array: %v", err)
	}
	// The last cell is reachable: write it, read it back, print it.
	last := fmt.Sprintf("int a[%d]; int main() { a[%d] = 65; putchar(a[%d]); }", memory-2, memory-3, memory-3)
	if got := runSorted(t, toSorted(t, last, render.English)); got != "A" {
		t.Errorf("last cell: %q, want \"A\"", got)
	}
	tooBig := fmt.Sprintf("int a[%d]; int main() { putchar(65); }", memory-1)
	_, err := compileC(t, tooBig)
	if !errors.As(err, &e) || err.Error() != fmt.Sprintf("1:1: the program needs %d memory cells; Sorted! has %d", memory+1, memory) {
		t.Errorf("one cell too many: %v", err)
	}
}
