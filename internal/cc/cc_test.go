package cc

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// sexpr prints a node as an S-expression, e.g. (= x (+ x 1)).
func sexpr(n *Node) string {
	if n == nil {
		return "nil"
	}
	ops := map[NodeKind]string{NdAdd: "+", NdSub: "-", NdMul: "*", NdDiv: "/", NdMod: "%", NdEq: "==", NdNe: "!=", NdLt: "<", NdLe: "<=", NdAssign: "=", NdLogAnd: "&&", NdLogOr: "||", NdBitAnd: "&", NdBitOr: "|", NdBitXor: "^", NdShl: "<<", NdShr: ">>"}
	switch n.Kind {
	case NdNum:
		return fmt.Sprint(n.Val)
	case NdVar:
		if n.Var.IsGlobal {
			return "@" + n.Var.Name
		}
		return n.Var.Name
	case NdIndex:
		name := n.Var.Name
		if n.Var.IsGlobal {
			name = "@" + name
		}
		return name + "[" + sexpr(n.Lhs) + "]"
	case NdNeg:
		return "(neg " + sexpr(n.Lhs) + ")"
	case NdNot:
		return "(! " + sexpr(n.Lhs) + ")"
	case NdBitNot:
		return "(~ " + sexpr(n.Lhs) + ")"
	case NdFor:
		return "(for " + sexpr(n.Init) + " " + sexpr(n.Cond) + " " + sexpr(n.Inc) + " " + sexpr(n.Then) + ")"
	case NdBreak:
		return "break"
	case NdContinue:
		return "continue"
	case NdReturn:
		return "(return " + sexpr(n.Lhs) + ")"
	case NdIf:
		return "(if " + sexpr(n.Cond) + " " + sexpr(n.Then) + " " + sexpr(n.Els) + ")"
	case NdWhile:
		return "(while " + sexpr(n.Cond) + " " + sexpr(n.Then) + ")"
	case NdExprStmt:
		return sexpr(n.Lhs)
	case NdFuncall:
		parts := []string{n.Func}
		for _, a := range n.Args {
			parts = append(parts, sexpr(a))
		}
		return "(" + strings.Join(parts, " ") + ")"
	case NdBlock:
		parts := []string{"{"}
		for _, b := range n.Body {
			parts = append(parts, sexpr(b))
		}
		return strings.Join(parts, " ") + "}"
	}
	return "(" + ops[n.Kind] + " " + sexpr(n.Lhs) + " " + sexpr(n.Rhs) + ")"
}

func TestTokenize(t *testing.T) {
	toks, err := Tokenize("int x = 0x1F + 017 // comment\n /* block\n comment */ <= 9;")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tk := range toks {
		got = append(got, fmt.Sprintf("%d:%s@%v", tk.Kind, tk.Text, tk.Pos))
	}
	want := "2:int@1:1 0:x@1:5 1:=@1:7 3:0x1F@1:9 1:+@1:14 3:017@1:16 1:<=@3:13 3:9@3:16 1:;@3:17 5:@3:18"
	if strings.Join(got, " ") != want {
		t.Errorf("tokens\n got %s\nwant %s", strings.Join(got, " "), want)
	}
	if toks[3].Val != 31 || toks[5].Val != 15 {
		t.Errorf("values %d %d", toks[3].Val, toks[5].Val)
	}
}

func TestParse(t *testing.T) {
	tests := []struct{ src, want string }{
		{"int main() { return 0; }", "{ (return 0)}"},
		{"#include <stdio.h>\n  #  include   <stdio.h>  \nint main() { putchar(1); }", "{ (putchar 1)}"},
		{"/* a */ #include <stdio.h>\n// b\n\t/* c\n d */ #include <stdio.h>\nint main() { }", "{}"},
		{"int main(void) { }", "{}"},
		{"int main() { int x = 1, y; y = x * 2 + 3 % 2; putchar(y); }",
			"{ { (= x 1)} (= y (+ (* x 2) (% 3 2))) (putchar y)}"},
		{"int main() { int a = 1; a = -a - -2; }", "{ { (= a 1)} (= a (- (neg a) (neg 2)))}"},
		{"int main() { int a; a = 1 < 2 == 3 > 4; }", "{ {} (= a (== (< 1 2) (< 4 3)))}"},
		{"int main() { int a; a = 1 <= 2 != 3 >= 4; }", "{ {} (= a (!= (<= 1 2) (<= 4 3)))}"},
		{"int main() { int a; int b; a = b = 7; }", "{ {} {} (= a (= b 7))}"},
		{"int main() { int a; if (a) a = 1; else if (a == 2) a = 3; }", "{ {} (if a (= a 1) (if (== a 2) (= a 3) nil))}"},
		{"int main() { int a; while (a < 10) { a = a + 1; } }", "{ {} (while (< a 10) { (= a (+ a 1))})}"},
		{"int main() { ; {} }", "{ {} {}}"},
		{"int g = 5; int h = -3, k; int main() { g = h + k; }", "{ (= @g (+ @h @k))}"},
		{"int main() { int x = 1; { int x = 2; putchar(x); } putchar(x); }", "{ { (= x 1)} { { (= x 2)} (putchar x)} (putchar x)}"},
		{"int g; int main() { int g = 1; putchar(g); }", "{ { (= g 1)} (putchar g)}"},
		{"int main() { int a; a = +(7 / (2 - 1)); }", "{ {} (= a (/ 7 (- 2 1)))}"},
		{"int main() { int a; (a) = 1; }", "{ {} (= a 1)}"},
		{"int main() { int a[3] = {7}; a[a[0] - 6] += 2; putchar(a[1]++); }",
			"{ { (= a[0] 7) (= a[1] 0) (= a[2] 0)} (= a[(- a[0] 6)] (+ a[(- a[0] 6)] 2)) (putchar (- (= a[1] (+ a[1] 1)) 1))}"},
		{"int main() { char s[] = \"a\\n\"; char c = 'z'; for (int i = 0; s[i]; i++) putchar(s[i]); }",
			"{ { (= s[0] 97) (= s[1] 10) (= s[2] 0)} { (= c 122)} (for { (= i 0)} s[i] (- (= i (+ i 1)) 1) (putchar s[i]))}"},
		{"int g[2]; int main() { g[1] = g[0]; }", "{ (= @g[1] @g[0])}"},
		{"int main() { int i; for (i = 0; i < 3; i = i + 1) putchar(i); }", "{ {} (for (= i 0) (< i 3) (= i (+ i 1)) (putchar i))}"},
		{"int main() { for (int i = 0; ; ) { break; continue; } for (;;) ; }", "{ (for { (= i 0)} nil nil { break continue}) (for {} nil nil {})}"},
		{"int main() { int a; a += 2; a -= 3; a *= 4; a /= 5; a %= 6; }", "{ {} (= a (+ a 2)) (= a (- a 3)) (= a (* a 4)) (= a (/ a 5)) (= a (% a 6))}"},
		{"int main() { int a; a = ++a + a-- - --a + a++; }", "{ {} (= a (+ (- (+ (= a (+ a 1)) (+ (= a (- a 1)) 1)) (= a (- a 1))) (- (= a (+ a 1)) 1)))}"},
		{"int main() { int a; a = !a || a && !(a == 1); }", "{ {} (= a (|| (! a) (&& a (! (== a 1)))))}"},
		{"int main() { putchar('A'); putchar('\\n'); putchar('\\x41'); putchar('\\101'); putchar('\\0'); putchar('\\''); putchar('\\q'); putchar('\\xff'); }",
			"{ (putchar 65) (putchar 10) (putchar 65) (putchar 65) (putchar 0) (putchar 39) (putchar 113) (putchar -1)}"},
		{"int main() { int a; a = 1 | 2 ^ 3 & 4 == 5 || 6 && 7 | 8; }", "{ {} (= a (|| (| 1 (^ 2 (& 3 (== 4 5)))) (&& 6 (| 7 8))))}"},
		{"int main() { int a; a = 1 << 2 + 3 < 4 >> 5 - ~6 * 7; }", "{ {} (= a (< (<< 1 (+ 2 3)) (>> 4 (- 5 (* (~ 6) 7)))))}"},
		{"int main() { int a; a = 1 << 2 >> 3 & ~~4; }", "{ {} (= a (& (>> (<< 1 2) 3) (~ (~ 4))))}"},
		{"int main() { int a; a &= 1; a |= 2; a ^= 3; a <<= 4; a >>= 5; }", "{ {} (= a (& a 1)) (= a (| a 2)) (= a (^ a 3)) (= a (<< a 4)) (= a (>> a 5))}"},
		{"int main() { int bool = 1, true = 2; putchar(bool + true); }", "{ { (= bool 1) (= true 2)} (putchar (+ bool true))}"},
	}
	for _, tt := range tests {
		p, err := Parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.src, err)
			continue
		}
		if got := sexpr(p.Main); got != tt.want {
			t.Errorf("%s\n got %s\nwant %s", tt.src, got, tt.want)
		}
	}
}

// Shadowing: an inner declaration is a different variable.
func TestScopes(t *testing.T) {
	p, err := Parse("int main() { int x = 1; { int x = 2; putchar(x); } putchar(x); }")
	if err != nil {
		t.Fatal(err)
	}
	outer := p.Main.Body[0].Body[0].Lhs.Lhs.Var
	inner := p.Main.Body[1].Body[1].Lhs.Args[0].Var
	after := p.Main.Body[2].Lhs.Args[0].Var
	if outer == inner || outer != after {
		t.Errorf("outer %p, inner %p, after %p", outer, inner, after)
	}
}

func TestGlobals(t *testing.T) {
	p, err := Parse("int a; int b = 7, c = -2; int main() {}")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, g := range p.Globals {
		got = append(got, fmt.Sprintf("%s=%v", g.Name, g.Init))
	}
	if strings.Join(got, " ") != "a=[] b=[7] c=[-2]" {
		t.Errorf("globals %v", got)
	}
	p, err = Parse(`int a[3]; int b[] = {1, -2, 'x',}; char s[] = "hi" "!"; char t[5] = "ab"; char u[2] = "ab"; char c = 'q'; int main() {}`)
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	for _, g := range p.Globals {
		got = append(got, fmt.Sprintf("%s:%d:%v:%v", g.Name, g.Len, g.Char, g.Init))
	}
	want := "a:3:false:[] b:3:false:[1 -2 120] s:4:true:[104 105 33 0] t:5:true:[97 98 0] u:2:true:[97 98] c:0:true:[113]"
	if strings.Join(got, " ") != want {
		t.Errorf("globals\n got %s\nwant %s", strings.Join(got, " "), want)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct{ src, want string }{
		{"int main() { do ; while (0); }", "1:14: not supported in Sorted! (yet): 'do'"},
		{"int main() { int a; a = a ? 1 : 2; }", "1:27: not supported in Sorted! (yet): ?:"},
		{"int main() { break; }", "1:14: 'break' outside a loop"},
		{"int main() { if (1) continue; }", "1:21: 'continue' outside a loop"},
		{"int main() { int a; a++++; }", "1:24: the left side of '++' must be a variable"},
		{"int main() { int a; ++(+a); }", "1:21: the left side of '++' must be a variable"},
		{"int main() { 3 += 1; }", "1:16: the left side of '+=' must be a variable"},
		{"int main() { putchar('ab'); }", "1:22: not supported in Sorted! (yet): multi-character constants"},
		{"int main() { putchar(''); }", "1:22: empty char literal"},
		{"int main() { putchar('a); }", "1:22: unclosed char literal"},
		{"int main() { putchar('\\x'); }", "1:22: invalid hex escape sequence"},
		{"int main() { putchar('\\x100'); }", "1:22: hex escape sequence out of range"},
		{"int main() { putchar('\\777'); }", "1:22: octal escape sequence out of range"},
		{"int main() { putchar('\xc3\xa4'); }", "1:22: not supported in Sorted! (yet): non-ASCII characters"},
		{"int main() { long c; }", "1:14: not supported in Sorted! (yet): the type or specifier 'long' (int and char are the only types)"},
		{"int *p; int main() {}", "1:5: not supported in Sorted! (yet): pointers"},
		{"int a[2][3]; int main() {}", "1:9: not supported in Sorted! (yet): arrays of arrays"},
		{"int a[n]; int main() {}", "1:7: not supported in Sorted! (yet): array lengths other than integer constants"},
		{"int a[0]; int main() {}", "1:7: the length of an array must be positive"},
		{"int a[]; int main() {}", "1:6: an array without a length needs an initializer"},
		{"int main() { int a[]; }", "1:19: an array without a length needs an initializer"},
		{"int a[2] = {1, 2, 3}; int main() {}", "1:12: too many initializers for the array"},
		{"int a[2] = {}; int main() {}", "1:12: empty initializer"},
		{"char s[2] = \"abc\"; int main() {}", "1:13: the string is longer than the array"},
		{"int s[] = \"abc\"; int main() {}", "1:11: a string literal can only initialize a char array"},
		{"int main() { int a[3]; putchar(a); }", "1:32: not supported in Sorted! (yet): using an array without an index (no pointers)"},
		{"int main() { int x; x[0] = 1; }", "1:22: subscripted value is not an array"},
		{"int main() { int a[1]; a++; }", "1:24: not supported in Sorted! (yet): using an array without an index (no pointers)"},
		{"int main() { int a[1]; a--; }", "1:24: not supported in Sorted! (yet): using an array without an index (no pointers)"},
		{"int main() { int a[1]; ++a; }", "1:26: not supported in Sorted! (yet): using an array without an index (no pointers)"},
		{"int main() { int a[1]; int b[1]; a = b; }", "1:34: not supported in Sorted! (yet): using an array without an index (no pointers)"},
		{"int main() { int a[3]; 3[a] = 1; }", "1:25: subscripted value is not an array"},
		{"char main() {}", "1:6: main must return int"},
		{"int main() { putchar(\"abc); }", "1:22: unclosed string literal"},
		{"int f(int a, int a) { } int main() {}", "1:18: redefinition of parameter 'a'"},
		{"int f(int x) { int x = 65; return x; } int main() {}", "1:20: redefinition of 'x'"},
		{"int main() { int a; a = &a; }", "1:25: not supported in Sorted! (yet): unary '&'"},
		{"int main() { int a; a = 1 ? 2 : 3; }", "1:27: not supported in Sorted! (yet): ?:"},
		{"int f(int) { } int main() {}", "1:5: parameter 1 of 'f' has no name"},
		{"int f(void x) { } int main() {}", "1:7: a parameter cannot be void"},
		{"int f(int a[]) { } int main() {}", "1:12: not supported in Sorted! (yet): array parameters (no pointers)"},
		{"int f(int *p) { } int main() {}", "1:11: not supported in Sorted! (yet): pointers"},
		{"int f(int a); char f(int a) { } int main() {}", "1:20: conflicting declarations of 'f'"},
		{"int f(int a); int f(char a) { } int main() {}", "1:19: conflicting declarations of 'f'"},
		{"int f(int a); int f(int a, int b); int main() {}", "1:19: conflicting declarations of 'f'"},
		{"int f() { } int f() { } int main() {}", "1:17: redefinition of 'f'"},
		{"int f() int main() {}", "1:9: expected the body of 'f'"},
		{"int g; int g() { } int main() {}", "1:12: redefinition of 'g' as a function"},
		{"int g() { } int g; int main() {}", "1:17: redefinition of 'g' as a variable"},
		{"void v; int main() {}", "1:6: a variable cannot be void"},
		{"int a[2](); int main() {}", "1:5: not supported in Sorted! (yet): functions returning arrays"},
		{"void f() { return 1; } int main() {}", "1:19: a void function cannot return a value"},
		{"int f(int a) { } int main() { f(); }", "1:33: f takes 1 argument(s), not 0"},
		{"int main() { g(); } int g() { }", "1:14: not supported in Sorted! (yet): calling 'g' (putchar is the only library function, and other functions must be declared first)"},
		{"int main() { main(); }", "1:14: not supported in Sorted! (yet): calling main (recursion)"},
		{"int main(int argc) {}", "1:10: not supported in Sorted! (yet): parameters of main"},
		{"int main() { printf(1); }", "1:14: not supported in Sorted! (yet): calling 'printf' (putchar is the only library function, and other functions must be declared first)"},
		{"int main() { putchar(\"a\"); }", "1:22: not supported in Sorted! (yet): string literals outside char array initializers"},
		{"#include <stdlib.h>\nint main() {}", "1:1: not supported in Sorted! (yet): the preprocessor (except #include <stdio.h>)"},
		{"#define N 3\nint main() {}", "1:1: not supported in Sorted! (yet): the preprocessor (except #include <stdio.h>)"},
		{"int main() { int x = 65; #include <stdio.h>\nputchar(x); }", "1:26: stray '#' (a preprocessing line must start with it)"},
		{"int main() { int a = sizeof(a); }", "1:22: not supported in Sorted! (yet): 'sizeof'"},
		{"int main() { putchar(1, 2); }", "1:26: putchar takes 1 argument(s), not 2"},
		{"int main() { x = 1; }", "1:14: undefined variable 'x'"},
		{"int main() { int a; int a; }", "1:25: redefinition of 'a'"},
		{"int main() { 1 = 2; }", "1:16: the left side of '=' must be a variable"},
		{"int main() { for (int i = 0; ; ) ; i = 1; }", "1:36: undefined variable 'i'"},
		{"int a; int a; int main() {}", "1:12: redefinition of 'a'"},
		{"int a;", "1:7: no main function"},
		{"int main() { return 0 }", "1:23: expected ';'"},
		{"int main() { int a = 99999999999; }", "1:22: integer literal out of range: 99999999999"},
		{"int main() { int a = 09; }", "1:22: invalid integer literal: 09"},
		{"int main() { /* open", "1:14: unclosed block comment"},
		{"int main() { $ }", "1:14: invalid token"},
		{"int main() {", "1:13: expected '}'"},
		{"int main() { int a; +a = 1; }", "1:24: the left side of '=' must be a variable"},
		{"int main() { int a; (+a) = 1; }", "1:26: the left side of '=' must be a variable"},
		{"int main; int main() {}", "1:15: redefinition of 'main' as a function"},
		{"int main() {} int main;", "1:19: redefinition of 'main' as a variable"},
		{"int main() { int putchar = 0; putchar(65); }", "1:31: 'putchar' is a variable, not a function"},
		{"int putchar; int main() { putchar(65); }", "1:27: 'putchar' is a variable, not a function"},
		{"int main() { int inline = 1; }", "1:18: expected a variable name"},
		{"int main() { inline int a; }", "1:14: not supported in Sorted! (yet): the type or specifier 'inline' (int and char are the only types)"},
		{"int main() { _Bool b; }", "1:14: not supported in Sorted! (yet): the type or specifier '_Bool' (int and char are the only types)"},
		{"int main() { int a; a = restrict; }", "1:25: not supported in Sorted! (yet): 'restrict'"},
		{"int main() { _Static_assert(1, 2); }", "1:14: not supported in Sorted! (yet): '_Static_assert'"},
		{"int main() { int a; a = 1, 2; }", "1:26: not supported in Sorted! (yet): the comma operator"},
		{"int main();", "1:11: expected the body of main"},
		{"int main() {} int main() {}", "1:19: redefinition of main"},
		{"int g = 1 + 2; int main() {}", "1:11: not supported in Sorted! (yet): global initializers other than integer constants"},
		{"int g = 1 int main() {}", "1:11: not supported in Sorted! (yet): global initializers other than integer constants"},
		{"int g int main() {}", "1:7: expected ','"},
		{"int g = h; int main() {}", "1:9: not supported in Sorted! (yet): global initializers other than integer constants"},
		{"int 5; int main() {}", "1:5: expected a variable name"},
		{"int main() { putchar(); }", "1:22: putchar takes 1 argument(s), not 0"},
		{"int main() { int a; a = ); }", "1:25: expected an expression"},
		{"int main() { int a; a = while; }", "1:25: not supported in Sorted! (yet): 'while'"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.src)
		var e *Error
		if !errors.As(err, &e) || err.Error() != tt.want {
			t.Errorf("%s\n got %v\nwant %s", tt.src, err, tt.want)
		}
	}
}

// Every escape sequence, in char and string literals.
func TestEscapes(t *testing.T) {
	escapes := map[string]int32{
		`\a`: 7, `\b`: 8, `\t`: 9, `\n`: 10, `\v`: 11, `\f`: 12, `\r`: 13, `\e`: 27,
		`\'`: 39, `\"`: 34, `\?`: 63, `\\`: 92, `\0`: 0, `\7`: 7, `\101`: 65, `\1012`: -1,
		`\x41`: 65, `\xff`: -1, `\q`: 113,
	}
	for esc, want := range escapes {
		if esc == `\1012` {
			continue // two characters: covered by the string case below
		}
		toks, err := Tokenize("'" + esc + "'")
		if err != nil || toks[0].Kind != TkNum || toks[0].Val != want {
			t.Errorf("'%s': %v, %+v; want %d", esc, err, toks, want)
		}
		toks, err = Tokenize(`"` + esc + `"`)
		if err != nil || len(toks[0].Str) != 1 || int32(int8(toks[0].Str[0])) != want {
			t.Errorf(`"%s": %v, %+v; want %d`, esc, err, toks, want)
		}
	}
	// \101 is one octal escape, followed by '2'.
	toks, err := Tokenize(`"\1012"`)
	if err != nil || string(toks[0].Str) != "A2" {
		t.Errorf(`"\1012": %v, %q`, err, toks[0].Str)
	}
}

func TestFunctions(t *testing.T) {
	p, err := Parse(`int add(int a, int b);
char up(char c) { return c - 32; }
void say(void) { putchar(up('a')); }
int main() { say(); putchar(add(1, 2)); }
int add(int x, int y) { return x + y; }`)
	if err != nil {
		t.Fatal(err)
	}
	add, up, say := p.Funcs["add"], p.Funcs["up"], p.Funcs["say"]
	if !add.Defined || len(add.Params) != 2 || add.Params[0].Name != "x" || add.Void || add.Char {
		t.Errorf("add: %+v", add)
	}
	if !up.Char || len(up.Params) != 1 || !up.Params[0].Char || sexpr(up.Body) != "{ (return (- c 32))}" {
		t.Errorf("up: %+v, %s", up, sexpr(up.Body))
	}
	if !say.Void || len(say.Params) != 0 {
		t.Errorf("say: %+v", say)
	}
	// The call in main, parsed before the definition, points at the
	// completed declaration, whose parameters are the definition's.
	call := p.Main.Body[1].Lhs.Args[0]
	if call.Fn != add || sexpr(p.Main) != "{ (say) (putchar (add 1 2))}" {
		t.Errorf("main: %s, fn %p want %p", sexpr(p.Main), call.Fn, add)
	}
	if sexpr(add.Body) != "{ (return (+ x y))}" {
		t.Errorf("add body: %s", sexpr(add.Body))
	}
}
