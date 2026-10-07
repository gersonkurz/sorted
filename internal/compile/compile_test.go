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
	"github.com/gersonkurz/sorted/internal/emit"
	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/render"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// toSorted compiles C source and writes it as Sorted! text in lang.
func toSorted(t *testing.T, src string, lang render.Lang) string {
	t.Helper()
	return toSortedAs(t, src, lang, false)
}

// toSortedAs is toSorted, as Very Sorted! when very is set.
func toSortedAs(t *testing.T, src string, lang render.Lang, very bool) string {
	t.Helper()
	prog, err := cc.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	_, p, err := compileProgram(prog, very)
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
func runSorted(t *testing.T, text string) string { t.Helper(); return runSortedIn(t, text, "") }

// runSortedIn runs a Sorted! program with stdin.
func runSortedIn(t *testing.T, text, stdin string) string {
	t.Helper()
	p, err := syntax.Parse([]byte(text))
	if err != nil {
		t.Fatalf("%v in:\n%s", err, text)
	}
	var out bytes.Buffer
	if err := interp.Run(p, strings.NewReader(stdin), &out, 10000000); err != nil {
		t.Fatalf("%v; output so far %q", err, out.String())
	}
	return out.String()
}

// runNative compiles C with the host compiler and runs it. It skips the
// test when there is no C compiler.
func runNative(t *testing.T, src string) string { t.Helper(); return runNativeIn(t, src, "") }

// runNativeIn compiles C with the host compiler and runs it with stdin.
func runNativeIn(t *testing.T, src, stdin string) string {
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
	cmd := exec.Command(bin)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
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

	"bitwise": `#include <stdio.h>
int pn; int pp;
int v[7] = {0, 1, 12, -1, -12, 2147483647, -2147483647};
int sum(int a, int b) { return (a ^ b) + ((a & b) << 1); }
int main() {
	for (int i = 0; i < 7; i++)
		for (int j = 0; j < 7; j++) {
			int a = v[i], b = v[j];
			` + printNum("(a & b) % 1000") + printNum("(a | b) % 1000") + printNum("(a ^ b) % 1000") + `
		}
	for (int i = 0; i < 7; i++) {
		int a = v[i];
		` + printNum("~a / 3") + printNum("a >> 3") + printNum("a >> 31") + printNum("a >> 0") + printNum("a & 255") +
		printNum("255 & a") + printNum("a & 0") + printNum("(a & 1073741823) % 1000") + `
	}
	for (int n = 0; n < 31; n += 3) {
		` + printNum("1 << n") + printNum("(5 << n / 2) >> n / 3") + printNum("-12345 >> n") + printNum("(1 << 30) >> n") + `
	}
	` + printNum("(1 << 4) | 3") + printNum("0xF0 ^ 0x3C") + printNum("~0") + printNum("-17 >> 2") + printNum("5 << 2") + `
	` + printNum("sum(1234, 4321)") + printNum("sum(-77, 7)") + `
	int x = 0x5A;
	x &= 0x0F; ` + printNum("x") + `
	x |= 0x30; ` + printNum("x") + `
	x ^= 0xFF; ` + printNum("x") + `
	x <<= 3; ` + printNum("x") + `
	x >>= 2; ` + printNum("x") + `
	int s = 5;
	x = 1000; x >>= s; ` + printNum("x") + `
	x <<= s - 2; ` + printNum("x") + `
	char c = 100;
	c <<= 1; ` + printNum("c") + `
	c = 0x7F; c ^= 0xFF; ` + printNum("c") + `
	int a[4] = {1, 2, 3, 4};
	int k = 1;
	a[k++] |= 8; a[k] <<= s; a[3] ^= a[1];
	` + printNum("a[0] * 1000000 + a[1] * 10000 + a[2] * 10 + a[3]") + `
	` + printNum("2147483647") + printNum("2000000000") + printNum("1234567890") + printNum("-1000000000") + printNum("1000000") + `
	int big = -2147483647 - 1;
	` + printNum("~2147483647 / 1000") + printNum("(~2147483647 == big) + (big >> 3) / 1000") + `
	` + printNum("big / 1000") + printNum("big + 2147483647") + printNum("(big >> 31) + (big & 7)") + `
	if ((x & 1) || (x | 2) == 3) putchar('Y'); else putchar('N');
	putchar('\n');
}`,

	"recursion": `#include <stdio.h>
int m[30];
int calls;
int fact(int n) { if (n <= 1) return 1; return n * fact(n - 1); }
int fib(int n) { calls++; if (n < 2) return n; return fib(n - 1) + fib(n - 2); }
int fibm(int n) {
	if (n < 2) return n;
	if (m[n]) return m[n];
	m[n] = fibm(n - 1) + fibm(n - 2);
	return m[n];
}
void print(int n) { if (n < 0) { putchar('-'); n = -n; } if (n >= 10) print(n / 10); putchar('0' + n % 10); }
void line(int n) { print(n); putchar('\n'); }
int isOdd(int n);
int isEven(int n) { if (n == 0) return 1; return isOdd(n - 1); }
int isOdd(int n) { if (n == 0) return 0; return isEven(n - 1); }
int ack(int m, int n) {
	if (m == 0) return n + 1;
	if (n == 0) return ack(m - 1, 1);
	return ack(m - 1, ack(m, n - 1));
}
int hanoi(int n, int from, int to, int via) {
	if (n == 0) return 0;
	int moves = hanoi(n - 1, from, via, to);
	moves++;
	return moves + hanoi(n - 1, via, to, from);
}
int bitsSet(int x) { if (x == 0) return 0; return (x & 1) + bitsSet(x >> 1); }
int swap(int a, int b, int k) { if (k == 0) return a * 10 + b; return swap(b, a, k - 1); }
char up(char c, int k) { if (k == 0) return c; return up(c + 1, k - 1); }
int depthSum(int n) {
	int local[3];
	local[0] = n; local[1] = n * 2; local[2] = n * 3;
	int below = 0;
	if (n > 0) below = depthSum(n - 1);
	return local[0] + local[1] + local[2] + below;
}
int perm[4]; int used[4]; int count;
void permute(int k) {
	if (k == 4) {
		count++;
		if (count % 7 == 1) { for (int i = 0; i < 4; i++) putchar('a' + perm[i]); putchar(' '); }
		return;
	}
	for (int i = 0; i < 4; i++)
		if (!used[i]) { used[i] = 1; perm[k] = i; permute(k + 1); used[i] = 0; }
}
int main() {
	line(fact(10)); line(fact(1));
	line(fib(15)); line(calls);
	line(fibm(29));
	line(-12345);
	for (int i = 0; i < 6; i++) line(isEven(i) * 10 + isOdd(i));
	line(ack(2, 3)); line(ack(3, 2));
	line(hanoi(7, 1, 3, 2));
	line(swap(1, 2, 1) * 100 + swap(1, 2, 4));
	line(bitsSet(255) * 100 + bitsSet(1023));
	putchar(up('a', 25)); putchar(up(120, 10)); putchar('\n');
	line(depthSum(10));
	line(fact(fact(3)) + fib(fact(3)));
	if (fib(10) > fact(4) && isEven(fib(6))) line(1); else line(0);
	permute(0); putchar('\n'); line(count);
}`,

	"macros": `#include <stdio.h>
#define N 8
#define LAST (N - 1)
#define SQ(x) ((x) * (x))
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define BIT(k) (1 << (k))
#define FLAGS (BIT(0) | BIT(3) | BIT(5))
#define PUT(c) putchar(c)
#define NL PUT('\n')
#define TIMES(n, body) for (int i_ = 0; i_ < (n); i_++) { body; }
#define PRINT_DIGIT(d) PUT('0' + (d) % 10)
#define EMPTY
#define TWICE(f, x) f(f(x))
#define IGNORE(x) 7
#define PAIR(a, b) a + b
#define N2 /*
      */ (2)
int table[N * 2];
int primes[] = {2, 3, 5, 7, 11, SQ(4) - 3};
int flags = FLAGS, neg = -N * 2, big = BIT(30) + (BIT(30) - 1), small = -2147483647 - 1;
char hello[N] = {'H', 'i', '!' EMPTY};
int inc(int x) { return x + 1; }
int A = 65, F = 66;
#define A F
#define F(x) A
int main() {
	PUT(A(0)); PRINT_DIGIT(IGNORE(PAIR(1))); PRINT_DIGIT(N2); NL;
	for (int i = 0; i < N * 2; i++) table[i] = SQ(i - LAST);
	for (int i = 0; i < N * 2; i++) { PRINT_DIGIT(table[i] / 10); PRINT_DIGIT(table[i]); PUT(' '); }
	NL;
	for (int i = 0; i < 6; i++) { PRINT_DIGIT(primes[i] / 10); PRINT_DIGIT(primes[i]); PUT(' '); }
	NL;
	PRINT_DIGIT(flags / 10); PRINT_DIGIT(flags); NL;
	if (flags & BIT(3)) PUT('Y'); else PUT('N');
	if (flags & BIT(4)) PUT('Y'); else PUT('N');
	NL;
	PRINT_DIGIT(-neg / 10); PRINT_DIGIT(-neg); NL;
	if (big == 2147483647 && small < -2147483647 && small + 1 == -2147483647) PUT('B'); NL;
	for (int i = 0; hello[i]; i++) PUT(hello[i]);
	NL;
	TIMES(3, PUT('*'));
	NL;
	PRINT_DIGIT(TWICE(inc, 5)); PRINT_DIGIT(SQ(inc(2))); NL;
#undef N
#define N 3
	PRINT_DIGIT(N); NL;
	return 0;
}`,

	"pointers": `#include <stdio.h>
int pn; int pp;
int g[5] = {5, 4, 3, 2, 1};
int *gp = &g[1];
int *gend = g + 5;
int *gmid = 2 + g;
int *gback = g + 4 - 1;
char *greeting = "Hi, pointers!";
char *names[] = {"zero", "one", "two"};
int counter;
int *where = &counter;
char buf[32];
void say(char *s) { while (*s) putchar(*s++); }
void line(char *s) { say(s); putchar('\n'); }
int length(char *s) { char *p = s; while (*p) p++; return p - s; }
void copy(char *to, char *from) { while ((*to++ = *from++) != 0) ; }
void reverse(char *s) { char *e = s + length(s) - 1; while (s < e) { char t = *s; *s++ = *e; *e-- = t; } }
void swap(int *a, int *b) { int t = *a; *a = *b; *b = t; }
int sum(int a[], int n) { int s = 0; for (int i = 0; i < n; i++) s += a[i]; return s; }
int *largest(int *a, int n) { int *best = a; for (int i = 1; i < n; i++) if (a[i] > *best) best = a + i; return best; }
void bump(int **pp) { (**pp)++; *pp = *pp + 1; }
void fill(int *a, int n) { if (n == 0) return; *a = n * n; fill(a + 1, n - 1); }
int rlen(char *s) { if (!*s) return 0; return 1 + rlen(s + 1); }
int main() {
	line(greeting);
	line(names[2]);
	putchar(names[1][1]); putchar('\n');
	` + printNum("length(greeting) * 100 + rlen(names[0])") + `
	copy(buf, "copied"); line(buf);
	reverse(buf); line(buf);
	int x = 3, y = 4;
	swap(&x, &y);
	` + printNum("x * 10 + y") + printNum("sum(g, 5)") + printNum("*largest(g, 5) * 10 + (largest(g, 5) - g)") + `
	*largest(g, 5) = 0;
	` + printNum("sum(g, 5)") + `
	int *p = g;
	p += 2; *p = 9; p[1] = 8; *(p - 1) += 100;
	` + printNum("g[1] * 100 + g[2] * 10 + g[3]") + printNum("*gp") + `
	bump(&p);
	` + printNum("*p * 100 + g[2]") + `
	*where = 42;
	` + printNum("counter") + `
	char c = 'A'; char *cp = &c; *cp += 200;
	` + printNum("c") + printNum("\"abc\"[1]") + `
	char z = 127; char *zp = &z; int old = (*zp)++;
	` + printNum("old * 1000 + z") + `
	int *ap = &*gp; int *a2 = &g[2]; int j = 3; int *aj = &g[j];
	int two[2] = {1, 2}; int *twoEnd = &two[2];
	if (twoEnd == two + 2 && twoEnd - two == 2 && &g[5] == gend) line("one past the end ok");
	*&g[j] = 5; *&g[0] += 1; *&*ap = 6; *&j = j + 1;
	` + printNum("(aj - a2) * 1000 + (a2 - ap) * 100 + (g[3] - g[1]) * 10 + j") + `
	int arr[3] = {1, 2, 3}; int *q = arr; int first = *q++;
	` + printNum("first * 10 + *q") + `
	if (p != 0 && p > g && p - g == 3 && gend - g == 5 && gmid - g == 2 && gback - gmid == 1) line("compare ok");
	int *null = 0; if (!null) line("null ok");
	char *s = buf; while (*s) s++;
	` + printNum("s - buf") + `
	char **np = names; np++; line(*np); line(np[1]);
	fill(g, 5);
	for (int *i = g; i < gend; i++) { ` + printNum("*i") + ` }
	int k = 1;
	g[k++] = 7; *(g + k++) += 1; (*q)++; q[-1]--;
	` + printNum("g[1] * 10000 + g[2] * 100 + k * 10 + arr[1] - arr[0]") + `
	return 0;
}`,

	"control flow": `#include <stdio.h>
int pn; int pp;
int calls;
int g[3] = {7, 8, 9};
int *pick = 1 ? &g[2] : 0;
int *none = 0 ? g : 0;
int *other = 0 ? 0 : g + 1;
int count(int x) { calls++; return x; }
int last;
int note(int v) { last = last * 10 + v; return v; }
char *name(int n) {
	switch (n) {
	case 0: return "zero";
	case 1: return "one";
	case 2: case 3: return "a few";
	default: return n < 0 ? "negative" : "many";
	}
}
void say(char *s) { while (*s) putchar(*s++); }
int classify(char c) {
	int kind = 0;
	switch (c) {
	case 'a': case 'e': case 'i': case 'o': case 'u':
		kind = 1;
		break;
	case ' ':
		kind = 2;
	case '.':
		kind += 10; /* falls through from ' ' */
		break;
	default:
		kind = 3;
	}
	return kind;
}
void copy(char *to, char *from, int n) { /* Duff's device */
	int rounds = (n + 3) / 4;
	switch (n % 4) {
	case 0: do { *to++ = *from++;
	case 3:      *to++ = *from++;
	case 2:      *to++ = *from++;
	case 1:      *to++ = *from++;
		} while (--rounds > 0);
	}
}
int collatz(int n) { int steps = 0; do { n = n % 2 ? 3 * n + 1 : n / 2; steps++; } while (n != 1); return steps; }
int fact(int n) { return n < 2 ? 1 : n * fact(n - 1); }
char buf[16];
int main() {
	for (int i = -1; i < 6; i++) { say(name(i)); putchar(' '); }
	putchar('\n');
	char *text = "a b.c";
	for (char *p = text; *p; p++) putchar('0' + classify(*p) % 10);
	putchar('\n');
	for (int n = 1; n <= 7; n++) {
		for (int k = 0; k < 16; k++) buf[k] = 0;
		copy(buf, "abcdefg", n);
		say(buf); putchar(' ');
	}
	putchar('\n');
	` + printNum("collatz(27)") + printNum("fact(10)") + `
	int odd = 0, sum = 0;
	for (int i = 0; i < 20; i++) {
		switch (i % 5) {
		case 0: continue;
		case 4: if (i > 10) break; sum += 100;
		}
		if (i == 17) break;
		odd += i % 2 ? 1 : 0;
		sum += i;
	}
	` + printNum("odd * 10000 + sum") + `
	int j = 0;
	do {
		j++;
		if (j == 2) continue;
		if (j == 5) break;
		putchar('0' + j);
	} while (j < 9);
	putchar('\n');
	calls = 0;
	int x = 3;
	int y = x > 2 ? count(10) : count(20);
	int z = x < 2 ? count(30) : x == 3 ? count(40) : count(50);
	` + printNum("y * 1000 + z * 10 + calls") + `
	int arr[3] = {1, 2, 3}; int i = 1;
	arr[i++] += x ? 5 : 6;
	` + printNum("arr[0] * 100 + arr[1] * 10 + arr[2] + i * 1000") + `
	int a = 0, b = 0;
	int c = (a = 4, b = a + 1, a * b);
	for (a = 0, b = 10; a < b; a++, b--) ;
	` + printNum("c * 100 + a * 10 + b") + `
	int *pa = x ? &arr[2] : 0;
	char ch = x ? 300 : 0;
	` + printNum("*pa * 1000 + ch") + `
	if (x ? calls : 0) putchar('Y'); else putchar('N');
	putchar(x > 5 ? 'B' : x > 2 ? 'M' : 'S');
	if (none == 0 && pick - g == 2) putchar('0' + *pick + *other - 10);
	putchar('\n');
	x < 2 ? note(1) : note(2);
	x > 2 ? note(3) : note(4);
	` + printNum("last") + `
	int w = x ? count(7) : 1;
	` + printNum("x * 100 + w * 10 + calls") + printNum("(calls = 100) + (x ? count(1) : 0)") + `
	return 0;
}`,

	"structs": `#include <stdio.h>
int pn; int pp;
struct point { int x; int y; };
struct rect { struct point min, max; char name[8]; };
struct node { int value; struct node *next; };
struct tagged { struct point p; int tag; };
struct point corners[3] = {{1, 2}, {3, 4}, 5, 6};
struct rect box = {{1, 1}, {4, 3}, "box"};
struct node nodes[4];
struct node *head;
struct { int a; char b; } anon = {7, 300};
struct point *second = &corners[1];
struct point *third = corners + 2;
int *maxy = &box.max.y;
char *bname = box.name;
char *bsecond = &box.name[1];
int depth(int n) { struct point p; p.x = n; p.y = n * 2; if (n > 0) depth(n - 1); return p.x * 10 + p.y; }
int area(struct rect *r) { return (r->max.x - r->min.x) * (r->max.y - r->min.y); }
void move(struct point *p, int dx, int dy) { p->x += dx; p->y += dy; }
void say(char *s) { while (*s) putchar(*s++); }
void push(int v, int i) { nodes[i].value = v; nodes[i].next = head; head = &nodes[i]; }
int sum(struct node *n) { return n ? n->value + sum(n->next) : 0; }
struct point *farthest(struct point *ps, int n) {
	struct point *best = ps;
	for (int i = 1; i < n; i++)
		if (ps[i].x + ps[i].y > best->x + best->y) best = ps + i;
	return best;
}
int main() {
	` + printNum("area(&box)") + `
	move(&box.max, 2, 1);
	` + printNum("area(&box)") + `
	say(box.name); putchar('\n');
	struct point p = corners[1];
	p.x++;
	` + printNum("p.x * 10 + p.y + corners[1].x * 100") + `
	struct point q = {9};
	` + printNum("q.x * 10 + q.y") + `
	corners[0] = q;
	` + printNum("corners[0].x * 10 + corners[0].y") + `
	struct point *fp = farthest(corners, 3);
	` + printNum("(fp - corners) * 100 + fp->x * 10 + fp->y") + `
	for (int i = 0; i < 4; i++) push(i * 10, i);
	` + printNum("sum(head)") + printNum("head->next->value") + printNum("anon.a * 1000 + anon.b") + `
	struct rect r2;
	r2 = box;
	r2.name[0] = 'B';
	say(r2.name); say(box.name); putchar('\n');
	int i = 1;
	corners[i++].y = 100;
	corners[i].x += corners[i - 1].y;
	` + printNum("corners[1].y * 1000 + corners[2].x") + `
	struct point a, b, c;
	a.x = 1; a.y = 2;
	c = b = a;
	` + printNum("c.x * 10 + b.y") + `
	struct point *walk = &corners[0];
	walk++;
	` + printNum("walk->y + (third - second) * 1000 + (&corners[2] - corners) * 10000") + `
	char *np = box.name;
	struct rect *rp = &box;
	rp->name[2] = 'X';
	say(rp->name); putchar(np[1]); putchar('\n');
	struct point pts[2] = {{5, 6}, {7}};
	struct node local = {42, &nodes[0]};
	struct rect named = {{0, 0}, {1, 1}, "named"};
	say(named.name); putchar('\n');
	` + printNum("pts[0].x * 1000 + pts[1].x * 100 + pts[1].y * 10 + local.next->value") + `
	(*rp).min.x = 3;
	rp->min = rp->max;
	` + printNum("box.min.x * 10 + box.min.y + second->y * 100") + printNum("depth(3) * 100 + (1 + walk)->x") + `
	struct point arr2[3], a2, c2;
	a2.x = 8; a2.y = 9;
	int j = 0;
	c2 = arr2[j++] = a2;
	` + printNum("j * 100 + c2.x * 10 + arr2[0].y") + printNum("*maxy * 1000 + bname[2] + *bsecond") + `
	struct point a3 = {1, 2};
	struct point list[2] = {a3, {3, 4}};
	struct rect r3 = {a3, list[1], "r3"};
	int k = 7;
	struct point l3[2] = {k, 2, 3};
	struct tagged tg[2] = {a3, 3, {{5, 6}, 7}};
	` + printNum("tg[0].p.x * 100000 + tg[0].p.y * 10000 + tg[0].tag * 1000 + tg[1].p.x * 100 + tg[1].p.y * 10 + tg[1].tag") + `
	say(r3.name);
	` + printNum("list[0].y * 100000 + r3.min.x * 10000 + r3.max.y * 1000 + l3[0].x * 100 + l3[1].x * 10 + l3[1].y") + `
	return 0;
}`,

	"arrays of arrays and unsigned": `#include <stdio.h>
int pn; int pp;
int grid[3][4] = {{1, 2, 3, 4}, {5, 6, 7, 8}, 9, 10};
char names[3][6] = {"one", "two", "three"};
int cube[2][2][2] = {1, 2, 3, 4, 5, 6, 7, 8};
unsigned int big = 4000000000u;
unsigned mask = 0xFFFFFFFF;
unsigned char bytes[4] = {255, 256, 300, -1};
int *lastRow = grid[2];
int sum(int m[][4], int rows) {
	int s = 0;
	for (int i = 0; i < rows; i++)
		for (int j = 0; j < 4; j++) s += m[i][j];
	return s;
}
struct triple { int a[3]; int b; } tri;
int first(int m[][4]) { return **m * 10 + *m[1]; }
void say(char *s) { while (*s) putchar(*s++); }
void printU(unsigned int u) { if (u >= 10) printU(u / 10); putchar('0' + u % 10); }
void line(unsigned int u) { printU(u); putchar('\n'); }
int main() {
	` + printNum("sum(grid, 3) * 1000 + grid[1][2] * 100 + grid[2][1]") + `
	int *r = grid[2];
	r[3] = 42;
	` + printNum("grid[2][3] * 10 + lastRow[0]") + `
	for (int i = 0; i < 3; i++) { say(names[i]); putchar(' '); }
	names[1][0] = 'T';
	say(names[1]); putchar('\n');
	int i = 2, j = 3;
	grid[i][j] += 1;
	grid[i - 1][j - 1] *= 2;
	` + printNum("grid[2][3] * 100 + grid[1][2]") + printNum("cube[1][0][1] * 10 + cube[0][1][1] + sum(grid + 1, 1) * 1000") + `
	line(big); line(mask); line(big / 3); line(big % 7);
	line(mask >> 1); line(mask >> 4); line(big >> 31); line(mask / big); line(4000000000u % 3000000000u);
	unsigned v = 0x80000000;
	for (int k = 0; k < 32; k += 5) line(v >> k);
	unsigned a = 3000000000u, b = 1;
	int neg = -1;
	if (a > b) putchar('Y'); else putchar('N');
	if (neg < b) putchar('Y'); else putchar('N');
	if (neg < 1) putchar('Y'); else putchar('N');
	putchar('\n');
	int x = -8;
	unsigned y = 2;
	line(x / y);
	` + printNum("x >> 1") + `
	` + printNum("bytes[0] * 1000000 + bytes[1] * 10000 + bytes[2] * 10 + bytes[3] % 10") + `
	unsigned char uc = 250;
	uc += 10;
	unsigned char z = 255;
	int old = z++;
	` + printNum("old * 1000 + z * 100 + uc") + `
	unsigned char m = 200;
	unsigned five = 5;
	` + printNum("m") + `
	line(five >> 1); line(mask >> 1);
	int *p0 = *grid;
	int *rowEnd = &grid[0][4];
	int *triEnd = &tri.a[3];
	if (p0 == grid[0] && rowEnd == &grid[1][0] && triEnd == &tri.b && **grid == 1) putchar('E');
	` + printNum("first(grid) * 10 + first(grid + 1)") + `
	unsigned w = 7;
	w -= 10;
	line(w); line(w * 2); line(-w); line(~w);
	return 0;
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

// Programs that read: they compile to Very Sorted! (only that dialect can
// read), and print the same as the C program on the same input, in English,
// German and back through C. The inputs have no CR and no Ctrl-Z, which the
// interpreter reads as Win32's text mode does.
var programsWithInput = map[string]struct{ src, stdin string }{
	// Reading makes these Very Sorted!, which indexes computed values
	// directly (#27): stores that change their own index, used as values.
	"aliased stores": {`#include <stdio.h>
int a[4];
int show(int v) { putchar('0' + v); putchar(':'); for (int k = 0; k < 4; k++) putchar('0' + a[k]); putchar(' '); return v; }
int main() {
	int b, *p = a;
	getchar();
	a[0] = 0; a[1] = 7; show(a[a[0]] = 1);
	a[0] = 0; show(a[a[0]]++);
	a[0] = 0; show(++a[a[0]]);
	a[0] = 0; show(a[a[0]] += 2);
	a[0] = 0; b = a[a[0]] = 3; show(b);
	a[0] = 1; a[1] = 0; show(*(p + *(p + a[1])) = 2);
	a[0] = 0; show(p[p[0]]--);
	putchar(10);
}`, "x"},

	"wc": {`#include <stdio.h>
int pn; int pp;
int main() {
	int lines = 0, words = 0, chars = 0, inword = 0, c;
	while ((c = getchar()) != -1) {
		chars++;
		if (c == '\n') lines++;
		if (c == ' ' || c == '\n' || c == '\t') inword = 0;
		else if (!inword) { inword = 1; words++; }
	}
	` + printNum("lines") + printNum("words") + printNum("chars") + `
	return 0;
}`, "This code is very cool.\nDieses Programm ist ganz hervorragend.\n\nhello\tworld\n"},
	"two reads in one expression": {`#include <stdio.h>
int main() {
	/* the order of the two reads is unspecified in C, an equality does not care */
	putchar('0' + (getchar() == getchar()));
	putchar('0' + (getchar() == getchar()));
	putchar('\n');
	return 0;
}`, "ABCC"},
	"rot13": {`#include <stdio.h>
int rot(int c) {
	if (c >= 'a' && c <= 'z') return (c - 'a' + 13) % 26 + 'a';
	if (c >= 'A' && c <= 'Z') return (c - 'A' + 13) % 26 + 'A';
	return c;
}
int main() { int c; while ((c = getchar()) != -1) putchar(rot(c)); return 0; }`, "Ordinata non errant!\nGur fbegrq qb abg ree.\n"},
	"reverse by recursion": {`#include <stdio.h>
void rev(void) { int c = getchar(); if (c == -1 || c == '\n') return; rev(); putchar(c); }
int main() { rev(); putchar('\n'); rev(); putchar('\n'); return 0; }`, "stressed\nlevel\n"},
	"sum of numbers": {`#include <stdio.h>
int pn; int pp;
int main() {
	int sum = 0, n = 0, c, any = 0;
	getchar(); /* a header character, thrown away */
	while ((c = getchar()) != -1) {
		if (c >= '0' && c <= '9') { n = n * 10 + c - '0'; any = 1; }
		else if (any) { sum += n; n = 0; any = 0; }
	}
	` + printNum("sum + n") + printNum("getchar() + getchar()") + `
	return 0;
}`, "#12 345, 6789 and 1000000\n"},
}

func TestDifferentialInput(t *testing.T) {
	for name, p := range programsWithInput {
		t.Run(name, func(t *testing.T) {
			want := runNativeIn(t, p.src, p.stdin)
			for _, lang := range []render.Lang{render.English, render.German} {
				text := toSorted(t, p.src, lang)
				if !strings.Contains(text, map[render.Lang]string{render.English: "This code is very cool.", render.German: "Dieses Programm ist ganz hervorragend."}[lang]) {
					t.Errorf("lang %d: no Very Sorted! marker:\n%s", lang, text)
				}
				if got := runSortedIn(t, text, p.stdin); got != want {
					t.Errorf("lang %d: Sorted! printed %q, C printed %q\n%s", lang, got, want, text)
				}
			}
			q, err := syntax.Parse([]byte(toSorted(t, p.src, render.English)))
			if err != nil {
				t.Fatal(err)
			}
			if got := runNativeIn(t, emit.Exact(q), p.stdin); got != want {
				t.Errorf("C -> Sorted! -> C printed %q, C printed %q", got, want)
			}
		})
	}
}

// A program that is Very Sorted! anyway (it reads, or has a NAND) indexes
// computed values directly (#27); one that is not keeps its pointer cells
// and stays the original's Sorted!.
func TestVeryIndexing(t *testing.T) {
	body := `int a[4]; int i = 2; a[i] = 'x'; a[i + 1] = a[i] + 1; int *p = &a[i]; *p += 1; putchar(*p); putchar(a[3]);`
	indexes := func(p *syntax.Program) bool {
		for _, e := range p.Code {
			for _, o := range e.Ops {
				if o.Type&syntax.Indirect != 0 && o.Type&0xFF != syntax.Number {
					return true
				}
			}
		}
		return false
	}
	plain, err := compileC(t, "int main() { "+body+" }")
	if err != nil {
		t.Fatal(err)
	}
	reading, err := compileC(t, "int main() { getchar(); "+body+" }")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Very || indexes(plain) {
		t.Errorf("a program that need not be very: very %v, indexes %v", plain.Very, indexes(plain))
	}
	// An operator whose value is never computed needs no NAND, so it does
	// not make the program very: a discarded expression, main's result.
	discarded, err := compileC(t, "int main() { int x = 6; x | 1; x & 3, x ^ x; "+body+" return x | 1; }")
	if err != nil {
		t.Fatal(err)
	}
	if discarded.Very || indexes(discarded) || discarded.Tables[syntax.Nands].Count != 0 {
		t.Errorf("discarded operators: very %v, indexes %v, %d nands", discarded.Very, indexes(discarded), discarded.Tables[syntax.Nands].Count)
	}
	if !reading.Very || !indexes(reading) {
		t.Errorf("a program that reads: very %v, indexes %v", reading.Very, indexes(reading))
	}
	if na, nb := plain.Tables[syntax.Assigns].Count, reading.Tables[syntax.Assigns].Count; nb >= na {
		t.Errorf("%d assignments in Very Sorted!, %d without: no pointer cells saved", nb, na)
	}
}

// A store whose value side changes what its location depends on keeps the
// location in a write pointer, evaluated first, in Very Sorted! as in the
// original's (C leaves the order unspecified; the compiler picks one, and
// the same one either way).
func TestVeryIndexingOrder(t *testing.T) {
	src := `int a[8]; int i = 1; int *p;
int next() { i++; return 7; }
int main() {
	p = a;
	a[i] = next(); a[i] += next(); *(p + i) = (i = 5); p[i] = (i = 2) + 40; a[i + 1] = i++;
	for (int k = 0; k < 8; k++) putchar('0' + a[k]);
	putchar(10);
}`
	var outs []string
	for _, very := range []bool{false, true} {
		prog, err := cc.Parse(src) // the compiler rewrites the tree
		if err != nil {
			t.Fatal(err)
		}
		_, p, err := compileProgram(prog, very)
		if err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := interp.Run(p, strings.NewReader(""), &out, 10000000); err != nil {
			t.Fatal(err)
		}
		outs = append(outs, out.String())
	}
	if outs[0] != outs[1] {
		t.Errorf("the original's Sorted! printed %q, Very Sorted! %q", outs[0], outs[1])
	}
}

// Every program also compiles as Very Sorted! (#27), where computed indexes
// are operands ("the cell indexed by the first sum") instead of pointer
// cells, and must still print what C prints, also through --to-c.
func TestDifferentialVery(t *testing.T) {
	for name, src := range programs {
		t.Run(name, func(t *testing.T) {
			want := runNative(t, src)
			text := toSortedAs(t, src, render.English, true)
			if !strings.Contains(text, "This code is very cool.") {
				t.Fatalf("no Very Sorted! marker:\n%s", text)
			}
			if got := runSorted(t, text); got != want {
				t.Errorf("Sorted! printed %q, C printed %q\n%s", got, want, text)
			}
			p, err := syntax.Parse([]byte(text))
			if err != nil {
				t.Fatal(err)
			}
			if got := runNative(t, emit.Exact(p)); got != want {
				t.Errorf("C -> Sorted! -> C printed %q, C printed %q", got, want)
			}
		})
	}
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
			// and back to C (M5): the exact translation prints the same
			p, err := syntax.Parse([]byte(toSorted(t, src, render.English)))
			if err != nil {
				t.Fatal(err)
			}
			if got := runNative(t, emit.Exact(p)); got != want {
				t.Errorf("C -> Sorted! -> C printed %q, C printed %q", got, want)
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

// Bitwise operators that arithmetic can do need no runtime function (so no
// loop, no jump): constants fold, ~x is -1 - x, shifts by a constant are a
// product or a floored ratio, and x & (2^k - 1) is a remainder. The rest call
// the runtime library.
// Bitwise operators are arithmetic where they can be, a runtime loop for a
// shift by a variable count, and NANDs otherwise, which make the program
// Very Sorted! (#26): two for &, three for |, four for ^.
func TestBitwiseArithmetic(t *testing.T) {
	for _, tt := range []struct {
		src         string
		jumps, very bool
		nands       int
	}{
		{`int main() { putchar((1 << 4) | 3 ^ ~0x10 & 0xFF); }`, false, false, 0},
		{`int main() { int x = 7; putchar(~x + (x << 3) + (x >> 2) + (x & 15) + (63 & x)); }`, false, false, 0},
		{`int main() { int x = 7; putchar(1 << x); }`, true, false, 0},
		{`int main() { int x = 7; putchar(x & 6); }`, false, true, 2},
		{`int main() { int x = 7; putchar(x | 1); }`, false, true, 3},
		{`int main() { int x = 7, y = 3; putchar(x ^ y); }`, false, true, 4},
		{`int main() { int x = 7, y = 3; putchar((x + 1) ^ (y - 2)); }`, false, true, 4},
	} {
		p, err := compileC(t, tt.src)
		if err != nil {
			t.Fatal(err)
		}
		if jumps := p.Tables[syntax.Jumps].Count > 0; jumps != tt.jumps || p.Very != tt.very || p.Tables[syntax.Nands].Count != tt.nands {
			t.Errorf("%s: jumps %v, very %v, %d nands; want %v, %v, %d", tt.src, jumps, p.Very, p.Tables[syntax.Nands].Count, tt.jumps, tt.very, tt.nands)
		}
		for _, e := range p.Entries(syntax.Nands) {
			if e.Flags != syntax.LogicalNand {
				t.Errorf("%s: a logical operation that is not a NAND: %v", tt.src, e)
			}
			// NANDs read their operands more than once: an operand is a cell
			// or another NAND, never an expression evaluated again each time.
			for _, o := range e.Ops {
				if o.Type != syntax.Number && o.Type != syntax.Nand {
					t.Errorf("%s: a NAND reads %v", tt.src, o)
				}
			}
		}
	}
	p, err := compileC(t, `int main() { putchar((1 << 4) | 3); }`)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(p.Data) != "[19]" {
		t.Errorf("numbers %v, want the folded [19]", p.Data)
	}
}

// The stack grows into the free memory after the variables; recursing past
// its end reads or writes a cell Sorted! does not have, a run-time error.
func TestStackOverflow(t *testing.T) {
	p, err := compileC(t, `int f(int n) { if (n == 0) return 0; return f(n - 1) + 1; }
int main() { putchar(f(100000)); putchar(f(3)); }`)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = interp.Run(p, strings.NewReader(""), &out, 10000000)
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Errorf("got %v, want a cell out of range", err)
	}
}

// A recursive call saves the caller's frame (return address, parameters,
// locals) and the temporaries of its own expression, not those of earlier
// statements: each statement starts a new expression.
func TestRecursiveSave(t *testing.T) {
	for src, want := range map[string]int{
		// statement: frame ra, n, r; the argument n - 1
		`int g[4]; int f(int n) { int r = 0; g[n % 4] = n; if (g[n % 4] > 0) r = f(n - 1); return r + 1; }`: 4,
		// return: frame ra, n; the argument
		`int g[4]; int f(int n) { if (n <= 0) return 0; g[n % 4] = n; return f(n - 1) + 1; }`: 3,
		// test: frame ra, n; the argument
		`int g[4]; int f(int n) { if (n <= 0) return 0; g[n % 4] = n; if (f(n - 1) > 5) return 1; return 2; }`: 3,
		// for's increment: frame ra, n, k; the argument and k, settled before the call
		`int g[4]; int f(int n) { if (n <= 0) return 1; for (int k = 0; g[k % 4] < 0 || k < 1; k = k + f(n - 1)) ; return 0; }`: 5,
	} {
		prog, err := cc.Parse(src + ` int main() { putchar(f(3)); }`)
		if err != nil {
			t.Fatal(err)
		}
		c, _, err := compileProgram(prog, false)
		if err != nil {
			t.Fatal(err)
		}
		pushes := 0
		for _, e := range c.assigns.entries {
			if e.b == (val{vInd, c.sp.i}) {
				pushes++
			}
		}
		if pushes != want {
			t.Errorf("%s: %d cells saved, want %d", src, pushes, want)
		}
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
		{`int main() { int a = 1 << 32; }`, "1:27: shift count 32 is out of range (0 to 31)"},
		{`int main() { int a[2]; int *p = &a[3]; }`, "1:36: index 3 is out of range for 'a' (2 elements)"},
		{`struct p { int x; }; int f(int n) { struct p s; int *q = &s.x; if (n) f(n - 1); return *q; } int main() { f(1); }`, "1:58: taking the address of 's' in the recursive function 'f' is not supported yet (make it a global)"},
		{`struct p { int x; }; int f(struct p *q, int n) { int *r = &q->x; if (n) f(q, n - 1); return *r; } int main() { struct p s; f(&s, 1); }`, ""},
		{`struct p { int x; }; int f(int n) { struct p s; int *q = &(*&s).x; if (n) f(n - 1); return *q; } int main() { f(1); }`, "1:58: taking the address of 's' in the recursive function 'f' is not supported yet (make it a global)"},
		{`struct p { int x; }; int main() { struct p a, b; a = (b, a); }`, "1:56: this struct value is not supported yet (only variables, elements, members and *p)"},
		{`int g[2][3]; int main() { g[1][3] = 0; }`, "1:32: index 3 is out of range (3 elements)"},
		{`int g[2][3]; int main() { g[2][0] = 0; }`, "1:29: index 2 is out of range for 'g' (2 elements)"},
		{`int main() { int a[2]; a[2] = 1; }`, "1:26: index 2 is out of range for 'a' (2 elements)"},
		{`int f(int n) { int x = n; int *p = &x; if (n) f(n - 1); return *p; } int main() { f(2); }`, "1:36: taking the address of 'x' in the recursive function 'f' is not supported yet (make it a global)"},
		{`int f(int n) { int a[2]; int *p = a; if (n) return f(n - 1); return *p; } int main() { f(2); }`, "1:35: taking the address of 'a' in the recursive function 'f' is not supported yet (make it a global)"},
		{`int g; int f(int n) { int a[2]; int *p = &g; a[n] = *p; *&a[n] += 1; if (n) return f(n - 1); return *a; } int main() { f(1); }`, ""},
		{`int main() { int a, b = a >> -1; }`, "1:30: shift count -1 is out of range (0 to 31)"},
		{`int f(int x) { return x << 40; } int main() { }`, "1:28: shift count 40 is out of range (0 to 31)"},
		{`int main() { int a = -999999999; putchar(a); }`, ""},
		{`int main() { int a = putchar(65); }`, "1:22: using the result of putchar is not supported yet"},
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
	c := &compiler{pool: []int32{5}, addrs: []int{3}, exprs: map[valKind]*table{}, out: -1, in: -1}
	for _, k := range exprKinds {
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
		c, _, err := compileProgram(prog, false)
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
	// A call that needs no stack costs no stack constants: the array, the
	// result and its copy, and 0 fill the memory exactly.
	call := fmt.Sprintf("int a[%d]; int f() { return 0; } int main() { a[0] = f(); }", memory-3)
	if _, err := compileC(t, call); err != nil {
		t.Errorf("call at the limit: %v", err)
	}
	tooBig := fmt.Sprintf("int a[%d]; int main() { putchar(65); }", memory-1)
	_, err := compileC(t, tooBig)
	if !errors.As(err, &e) || err.Error() != fmt.Sprintf("1:1: the program needs %d memory cells; Sorted! has %d", memory+1, memory) {
		t.Errorf("one cell too many: %v", err)
	}
}
