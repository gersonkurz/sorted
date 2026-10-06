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
	ops := map[NodeKind]string{NdAdd: "+", NdSub: "-", NdMul: "*", NdDiv: "/", NdMod: "%", NdEq: "==", NdNe: "!=", NdLt: "<", NdLe: "<=", NdAssign: "=", NdLogAnd: "&&", NdLogOr: "||"}
	switch n.Kind {
	case NdNum:
		return fmt.Sprint(n.Val)
	case NdVar:
		if n.Var.IsGlobal {
			return "@" + n.Var.Name
		}
		return n.Var.Name
	case NdNeg:
		return "(neg " + sexpr(n.Lhs) + ")"
	case NdNot:
		return "(! " + sexpr(n.Lhs) + ")"
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
		return "(" + n.Func + " " + sexpr(n.Args[0]) + ")"
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
	want := "2:int@1:1 0:x@1:5 1:=@1:7 3:0x1F@1:9 1:+@1:14 3:017@1:16 1:<=@3:13 3:9@3:16 1:;@3:17 4:@3:18"
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
		{"int main() { int i; for (i = 0; i < 3; i = i + 1) putchar(i); }", "{ {} (for (= i 0) (< i 3) (= i (+ i 1)) (putchar i))}"},
		{"int main() { for (int i = 0; ; ) { break; continue; } for (;;) ; }", "{ (for { (= i 0)} nil nil { break continue}) (for {} nil nil {})}"},
		{"int main() { int a; a += 2; a -= 3; a *= 4; a /= 5; a %= 6; }", "{ {} (= a (+ a 2)) (= a (- a 3)) (= a (* a 4)) (= a (/ a 5)) (= a (% a 6))}"},
		{"int main() { int a; a = ++a + a-- - --a + a++; }", "{ {} (= a (+ (- (+ (= a (+ a 1)) (+ (= a (- a 1)) 1)) (= a (- a 1))) (- (= a (+ a 1)) 1)))}"},
		{"int main() { int a; a = !a || a && !(a == 1); }", "{ {} (= a (|| (! a) (&& a (! (== a 1)))))}"},
		{"int main() { putchar('A'); putchar('\\n'); putchar('\\x41'); putchar('\\101'); putchar('\\0'); putchar('\\''); putchar('\\q'); putchar('\\xff'); }",
			"{ (putchar 65) (putchar 10) (putchar 65) (putchar 65) (putchar 0) (putchar 39) (putchar 113) (putchar -1)}"},
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
		got = append(got, fmt.Sprintf("%s=%d", g.Name, g.Init))
	}
	if strings.Join(got, " ") != "a=0 b=7 c=-2" {
		t.Errorf("globals %v", got)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct{ src, want string }{
		{"int main() { do ; while (0); }", "1:14: not supported in Sorted! (yet): 'do'"},
		{"int main() { int a; a = a ? 1 : 2; }", "1:27: not supported in Sorted! (yet): ?:"},
		{"int main() { int a; a &= 1; }", "1:23: not supported in Sorted! (yet): &="},
		{"int main() { int a; a = 1 & 2; }", "1:27: not supported in Sorted! (yet): &"},
		{"int main() { int a = ~1; }", "1:22: not supported in Sorted! (yet): unary '~'"},
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
		{"int main() { char c; }", "1:14: not supported in Sorted! (yet): the type or specifier 'char' (int is the only type)"},
		{"int *p; int main() {}", "1:5: not supported in Sorted! (yet): pointers"},
		{"int a[3]; int main() {}", "1:6: not supported in Sorted! (yet): arrays"},
		{"int f() { } int main() {}", "1:5: not supported in Sorted! (yet): functions other than main"},
		{"int main(int argc) {}", "1:10: not supported in Sorted! (yet): parameters of main"},
		{"int main() { printf(1); }", "1:14: not supported in Sorted! (yet): calling 'printf' (putchar is the only function)"},
		{"int main() { putchar(\"a\"); }", "1:22: not supported in Sorted! (yet): string literals"},
		{"#include <stdlib.h>\nint main() {}", "1:1: not supported in Sorted! (yet): the preprocessor (except #include <stdio.h>)"},
		{"#define N 3\nint main() {}", "1:1: not supported in Sorted! (yet): the preprocessor (except #include <stdio.h>)"},
		{"int main() { int x = 65; #include <stdio.h>\nputchar(x); }", "1:26: stray '#' (a preprocessing line must start with it)"},
		{"int main() { int a = sizeof(a); }", "1:22: not supported in Sorted! (yet): 'sizeof'"},
		{"int main() { putchar(1, 2); }", "1:23: putchar takes one argument"},
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
		{"int main() { inline int a; }", "1:14: not supported in Sorted! (yet): the type or specifier 'inline' (int is the only type)"},
		{"int main() { _Bool b; }", "1:14: not supported in Sorted! (yet): the type or specifier '_Bool' (int is the only type)"},
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
		{"int main() { putchar(); }", "1:22: putchar takes one argument"},
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
