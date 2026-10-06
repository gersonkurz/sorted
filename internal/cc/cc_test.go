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
	case NdAddr:
		return "(& " + sexpr(n.Lhs) + ")"
	case NdDeref:
		return "(* " + sexpr(n.Lhs) + ")"
	case NdMember:
		return "(. " + sexpr(n.Lhs) + " " + n.Member.Name + ")"
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
	case NdDo:
		return "(do " + sexpr(n.Then) + " " + sexpr(n.Cond) + ")"
	case NdSwitch:
		return "(switch " + sexpr(n.Cond) + " " + sexpr(n.Then) + ")"
	case NdCase:
		return "(case " + fmt.Sprint(n.Val) + " " + sexpr(n.Then) + ")"
	case NdCond:
		return "(?: " + sexpr(n.Cond) + " " + sexpr(n.Then) + " " + sexpr(n.Els) + ")"
	case NdComma:
		return "(, " + sexpr(n.Lhs) + " " + sexpr(n.Rhs) + ")"
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
		{"int main() { break; }", "1:14: 'break' outside a loop or switch"},
		{"int main() { switch (1) { continue; } }", "1:27: 'continue' outside a loop"},
		{"int main() { case 1: ; }", "1:14: 'case' outside a switch"},
		{"int main() { default: ; }", "1:14: 'default' outside a switch"},
		{"int main() { switch (1) { case 1: case 2: case 1: ; } }", "1:43: duplicate case value 1"},
		{"int main() { switch (1) { default: default: ; } }", "1:36: a second 'default' in one switch"},
		{"int main() { int x; switch (1) { case x: ; } }", "1:39: not supported in Sorted! (yet): case values other than integer constants"},
		{"int main() { int *p; switch (p) { } }", "1:30: the value of a switch must be an integer, not 'int *'"},
		{"int main() { int *p; int x; x = 1 ? p : x; }", "1:35: the branches of '?:' have different types ('int *' and 'int')"},
		{"int main() { do ; while (1) }", "1:29: expected ';'"},
		{"int main() { if (1) continue; }", "1:21: 'continue' outside a loop"},
		{"int main() { int a; a++++; }", "1:24: the left side of '++' cannot be assigned to"},
		{"int main() { int a; ++(+a); }", "1:21: the left side of '++' cannot be assigned to"},
		{"int main() { 3 += 1; }", "1:16: the left side of '+=' cannot be assigned to"},
		{"int main() { putchar('ab'); }", "1:22: not supported in Sorted! (yet): multi-character constants"},
		{"int main() { putchar(''); }", "1:22: empty char literal"},
		{"int main() { putchar('a); }", "1:22: unclosed char literal"},
		{"int main() { putchar('\\x'); }", "1:22: invalid hex escape sequence"},
		{"int main() { putchar('\\x100'); }", "1:22: hex escape sequence out of range"},
		{"int main() { putchar('\\777'); }", "1:22: octal escape sequence out of range"},
		{"int main() { putchar('\xc3\xa4'); }", "1:22: not supported in Sorted! (yet): non-ASCII characters"},
		{"int main() { long c; }", "1:14: not supported in Sorted! (yet): the type or specifier 'long' (int, char, unsigned and struct are the only types)"},
		{"void *p; int main() {}", "1:6: not supported in Sorted! (yet): void pointers"},
		{"int a[2][]; int main() {}", "1:10: only the first length of an array may be left out"},
		{"int a[n]; int main() {}", "1:7: undefined variable 'n'"},
		{"int n; int a[n]; int main() {}", "1:14: not supported in Sorted! (yet): array lengths other than integer constants"},
		{"int a[0]; int main() {}", "1:7: the length of an array must be positive"},
		{"int a[]; int main() {}", "1:6: an array without a length needs an initializer"},
		{"int main() { int a[]; }", "1:19: an array without a length needs an initializer"},
		{"int a[2] = {1, 2, 3}; int main() {}", "1:19: too many initializers for the array"},
		{"int a[2] = {}; int main() {}", "1:12: empty initializer"},
		{"char s[2] = \"abc\"; int main() {}", "1:13: the string is longer than the array"},
		{"int s[] = \"abc\"; int main() {}", "1:11: a string literal can only initialize a char array"},
		{"int main() { int a[3]; putchar(a); }", "1:32: putchar's argument must be an integer, not 'int *'"},
		{"int main() { int x; x[0] = 1; }", "1:22: subscripted value is not an array or a pointer"},
		{"int main() { int a[1]; a++; }", "1:24: an array cannot be assigned to, only its elements"},
		{"int main() { int a[1]; a--; }", "1:24: an array cannot be assigned to, only its elements"},
		{"int main() { int a[1]; ++a; }", "1:26: an array cannot be assigned to, only its elements"},
		{"int main() { int a[1]; int b[1]; a = b; }", "1:34: an array cannot be assigned to, only its elements"},
		{"int main() { int a[3]; 3[a] = 1; }", "1:25: subscripted value is not an array or a pointer"},
		{"char main() {}", "1:6: main must return int"},
		{"int main() { putchar(\"abc); }", "1:22: unclosed string literal"},
		{"int f(int a, int a) { } int main() {}", "1:18: redefinition of parameter 'a'"},
		{"int f(int x) { int x = 65; return x; } int main() {}", "1:20: redefinition of 'x'"},
		{"int main() { int a; a = &a; }", "1:23: the assignment needs 'int', not 'int *'"},
		{"int f(int) { } int main() {}", "1:5: parameter 1 of 'f' has no name"},
		{"int f(void x) { } int main() {}", "1:7: a parameter cannot be void"},
		{"int f(int a[2][]) { } int main() {}", "1:16: only the first length of an array may be left out"},
		{"int f(int (*p)) { } int main() {}", "1:11: not supported in Sorted! (yet): declarators in parentheses (pointers to arrays, function pointers)"},
		{"int main() { int x; *x = 1; }", "1:21: indirection needs a pointer, not 'int'"},
		{"int main() { int *p; int *q; p + q; }", "1:32: invalid operands to '+' ('int *' and 'int *')"},
		{"int main() { int *p; 1 - p; }", "1:24: invalid operands to '-' ('int' and 'int *')"},
		{"int main() { int *p; char *q; p - q; }", "1:33: invalid operands to '-' ('int *' and 'char *')"},
		{"int main() { int *p; p * 2; }", "1:24: invalid operands to '*' ('int *' and 'int')"},
		{"int main() { int *p; p < 1; }", "1:24: invalid operands to '<' ('int *' and 'int')"},
		{"int main() { int *p; p == 1; }", "1:24: invalid operands to '==' ('int *' and 'int')"},
		{"int main() { int *p; -p; }", "1:23: the operand of unary '-' must be an integer, not 'int *'"},
		{"int main() { int x; int *p = &x; int *q = +p; }", "1:44: the operand of unary '+' must be an integer, not 'int *'"},
		{"int main() { int a[2]; +a; }", "1:25: the operand of unary '+' must be an integer, not 'int *'"},
		{"int main() { int *p; ~p; }", "1:23: the operand of unary '~' must be an integer, not 'int *'"},
		{"int main() { int *p; int a[2]; a[p]; }", "1:34: an array index must be an integer, not 'int *'"},
		{"int main() { int *p; char *q; p = q; }", "1:33: the assignment needs 'int *', not 'char *'"},
		{"int main() { int *p = 5; }", "1:21: the assignment needs 'int *', not 'int'"},
		{"int f(int *p) { return 0; } int main() { int x; f(x); }", "1:51: argument 1 of 'f' needs 'int *', not 'int'"},
		{"int *f(void) { int x; return x; } int main() { }", "1:30: the return value needs 'int *', not 'int'"},
		{"int main() { int *p; return p; }", "1:29: the return value needs 'int', not 'int *'"},
		{"int main() { &1; }", "1:14: cannot take the address of a value, only of a variable or an element"},
		{"int main() { int a; &+a; }", "1:21: cannot take the address of a value, only of a variable or an element"},
		{"int main() { int a[2]; &a; }", "1:24: not supported in Sorted! (yet): pointers to arrays (&array; the array itself is the address of its first element)"},
		{"int main() { int a; a[0]; }", "1:22: subscripted value is not an array or a pointer"},
		{"int g; int *p = &g + &g; int main() {}", "1:20: invalid operands to '+' ('int *' and 'int *')"},
		{"int main() { int x; } int *p = 3; ", "1:32: the initializer needs 'int *', not 'int'"},
		{"int g[2]; int x = g; int main() {}", "1:19: the initializer needs 'int', not 'int *'"},
		{"int f(int *p); int f(char *p) { return 0; } int main() {}", "1:20: conflicting declarations of 'f'"},
		{"int f(int *p); int *f(int *p) { return 0; } int main() {}", "1:21: conflicting declarations of 'f'"},
		{"int g[2]; int x; int *p = &g[x]; int main() {}", "1:27: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int g[2]; int x; int *p = g + x; int main() {}", "1:27: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int g[2]; int x; int *p = x + g; int main() {}", "1:27: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int *g; int *p = g; int main() {}", "1:18: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int g; char *p = 1 ? 0 : &g; int main() {}", "1:18: the initializer needs 'char *', not 'int *'"},
		{"int g; int main() { switch (1) { case 1 ? 0 : &g: ; } }", "1:39: a constant expression must be an integer, not 'int *'"},
		{"int g; int a[1 ? 3 : &g]; int main() {}", "1:16: the branches of '?:' have different types ('int' and 'int *')"},
		{"int g; int x = 1 ? 0 : &g; int main() {}", "1:16: the initializer needs 'int', not 'int *'"},
		{"int g; int *p = (1 ? 0 : &g) + 1; int main() {}", "1:17: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"struct p { int x; }; int main() { struct p a; a.y; }", "1:49: 'struct p' has no member 'y'"},
		{"int main() { int a; a.x; }", "1:22: '.' needs a struct, not 'int'"},
		{"struct p { int x; }; int main() { struct p a; a->x; }", "1:48: '->' needs a pointer to a struct, not 'struct p'"},
		{"struct p; int main() { struct p a; }", "1:33: 'struct p' is incomplete: its members are not known here"},
		{"struct p; int main() { struct p *a; a->x; }", "1:38: 'struct p' is incomplete: its members are not known here"},
		{"struct p { int x; }; struct p { int y; }; int main() {}", "1:29: redefinition of 'struct p'"},
		{"struct p { int x; int x; }; int main() {}", "1:23: duplicate member 'x'"},
		{"struct p { int x[]; }; int main() {}", "1:17: a member array needs a length"},
		{"struct p { struct p q; }; int main() {}", "1:21: 'struct p' is incomplete: its members are not known here"},
		{"struct p { }; int main() {}", "1:1: a struct needs at least one member"},
		{"struct { int x; } a, b; struct { int x; } c; int main() { a = c; }", "1:61: the assignment needs 'struct (anonymous)', not 'struct (anonymous)'"},
		{"struct p { int x; }; int f(struct p a) { return 0; } int main() {}", "1:38: not supported in Sorted! (yet): struct parameters (pass a pointer)"},
		{"struct p { int x; }; struct p f(void) { } int main() {}", "1:31: not supported in Sorted! (yet): functions returning structs (return a pointer)"},
		{"struct p { int x; }; int main() { struct p a; if (a) ; }", "1:51: a condition must be an integer or a pointer, not 'struct p'"},
		{"struct p { int x; }; int main() { struct p a; !a; }", "1:48: the operand of '!' must be an integer or a pointer, not 'struct p'"},
		{"struct p { int x; }; int main() { struct p a; a + 1; }", "1:49: invalid operands to '+' ('struct p' and 'int')"},
		{"struct p { int x; }; int main() { struct p a, b; a = 1 ? a : b; }", "1:56: not supported in Sorted! (yet): '?:' choosing between structs"},
		{"struct p { int x; }; struct p g = {1, 2}; int main() {}", "1:39: too many initializers for the struct"},
		{"struct p { int x; }; struct p a; struct p g[1] = {a}; int main() {}", "1:51: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"struct p { int x; }; struct q { int y; }; int main() { struct q b; struct p a[1] = {b}; }", "1:82: the assignment needs 'int', not 'struct q'"},
		{"struct p { int x; }; struct p g = 1; int main() {}", "1:35: an array or a struct is initialized with a list in braces"},
		{"struct p { int x[2]; }; struct p g = {\"ab\"}; int main() {}", "1:39: a string literal can only initialize a char array"},
		{"int g[2] = {1 2}; int main() {}", "1:15: expected '}'"},
		{"int main() { struct 3 a; }", "1:21: expected a struct tag or '{'"},
		{"struct p { int x;", "1:18: expected '}'"},
		{"struct p { int x; }; int main() { struct p a; a.3; }", "1:49: expected a member name"},
		{"int x = 3000000000; int main() {}", "1:9: integer literal out of range: 3000000000"},
		{"int x = 0x100000000; int main() {}", "1:9: integer literal out of range: 0x100000000"},
		{"int x = 4294967296u; int main() {}", "1:9: integer literal out of range: 4294967296"},
		{"unsigned signed x; int main() {}", "1:10: 'signed' after 'unsigned'"},
		{"x; int main() {}", "1:1: expected 'int'"},
		{"signed unsigned x; int main() {}", "1:8: 'unsigned' after 'signed'"},
		{"unsigned long x; int main() {}", "1:10: not supported in Sorted! (yet): the type or specifier 'long' (int, char, unsigned and struct are the only types)"},
		{"int main() { int *p; unsigned *q = p; }", "1:34: the assignment needs 'unsigned int *', not 'int *'"},
		{"int f(int m[][4]); int g[2][3]; int main() { f(g); }", "1:48: argument 1 of 'f' needs 'int (*)[4]', not 'int (*)[3]'"},
		{"int main() { int m[2][2]; m[0] = 0; }", "1:28: an array cannot be assigned to, only its elements"},
		{"int (*p)[3]; int main() {}", "1:5: not supported in Sorted! (yet): declarators in parentheses (pointers to arrays, function pointers)"},
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
		{"int main() { putchar(\"a\"); }", "1:22: putchar's argument must be an integer, not 'char *'"},
		{"#include <stdlib.h>\nint main() {}", "1:1: not supported in Sorted! (yet): #include other than <stdio.h>"},
		{"#if 1\nint main() {}", "1:1: not supported in Sorted! (yet): #if (the preprocessor knows #define, #undef and #include <stdio.h>)"},
		{"#include \"x.h\"\nint main() {}", "1:1: not supported in Sorted! (yet): #include other than <stdio.h>"},
		{"#define\nint main() {}", "1:2: #define needs a macro name (an identifier)"},
		{"#define int long\nint main() {}", "1:2: #define needs a macro name (an identifier)"},
		{"#define S(x) #x\nint main() {}", "1:14: not supported in Sorted! (yet): '#' in macros"},
		{"#define C(a, b) a ## b\nint main() {}", "1:19: not supported in Sorted! (yet): '##' in macros"},
		{"#define V(...) 1\nint main() {}", "1:11: not supported in Sorted! (yet): variadic macros"},
		{"#define F(a, a) 1\nint main() {}", "1:14: duplicate parameter 'a' in macro 'F'"},
		{"#define F(a b) 1\nint main() {}", "1:9: expected ')' after the parameters of macro 'F'"},
		{"#define F(1) 1\nint main() {}", "1:11: expected a parameter name in macro 'F'"},
		{"#undef\nint main() {}", "1:2: #undef needs exactly one macro name"},
		{"#define F(a, b) a\nint main() { putchar(F(1)); }", "2:22: macro 'F' takes 2 argument(s), not 1"},
		{"#define F() 1\nint main() { putchar(F(2)); }", "2:22: macro 'F' takes 0 argument(s), not 1"},
		{"#define F(a) a\nint main() { putchar(F(1", "2:22: unterminated call of macro 'F'"},
		{"#define F(a) a\nint main() { putchar(F(1,\n#define X\n)); }", "3:1: not supported in Sorted! (yet): directives inside the arguments of a macro"},
		{"#define N 3\n#undef N\nint main() { putchar(N); }", "3:22: undefined variable 'N'"},
		{"int main() { int x = 65; #include <stdio.h>\nputchar(x); }", "1:26: stray '#' (a preprocessing line must start with it)"},
		{"int main() { int a = sizeof(a); }", "1:22: not supported in Sorted! (yet): 'sizeof'"},
		{"int main() { putchar(1, 2); }", "1:26: putchar takes 1 argument(s), not 2"},
		{"int main() { x = 1; }", "1:14: undefined variable 'x'"},
		{"int main() { int a; int a; }", "1:25: redefinition of 'a'"},
		{"int main() { 1 = 2; }", "1:16: the left side of '=' cannot be assigned to"},
		{"int main() { for (int i = 0; ; ) ; i = 1; }", "1:36: undefined variable 'i'"},
		{"int a; int a; int main() {}", "1:12: redefinition of 'a'"},
		{"int a;", "1:7: no main function"},
		{"int main() { return 0 }", "1:23: expected ';'"},
		{"int main() { int a = 99999999999; }", "1:22: integer literal out of range: 99999999999"},
		{"int main() { int a = 09; }", "1:22: invalid integer literal: 09"},
		{"int main() { /* open", "1:14: unclosed block comment"},
		{"int main() { $ }", "1:14: invalid token"},
		{"int main() {", "1:13: expected '}'"},
		{"int main() { int a; +a = 1; }", "1:24: the left side of '=' cannot be assigned to"},
		{"int main() { int a; (+a) = 1; }", "1:26: the left side of '=' cannot be assigned to"},
		{"int main; int main() {}", "1:15: redefinition of 'main' as a function"},
		{"int main() {} int main;", "1:19: redefinition of 'main' as a variable"},
		{"int main() { int putchar = 0; putchar(65); }", "1:31: 'putchar' is a variable, not a function"},
		{"int putchar; int main() { putchar(65); }", "1:27: 'putchar' is a variable, not a function"},
		{"int main() { int inline = 1; }", "1:18: expected a variable name"},
		{"int main() { inline int a; }", "1:14: not supported in Sorted! (yet): the type or specifier 'inline' (int, char, unsigned and struct are the only types)"},
		{"int main() { _Bool b; }", "1:14: not supported in Sorted! (yet): the type or specifier '_Bool' (int, char, unsigned and struct are the only types)"},
		{"int main() { int a; a = restrict; }", "1:25: not supported in Sorted! (yet): 'restrict'"},
		{"int main() { _Static_assert(1, 2); }", "1:14: not supported in Sorted! (yet): '_Static_assert'"},
		{"int main();", "1:11: expected the body of main"},
		{"int main() {} int main() {}", "1:19: redefinition of main"},
		{"int g = 1 / 0; int main() {}", "1:9: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int g = 1 int main() {}", "1:11: not supported in Sorted! (yet): global initializers other than constants and addresses"},
		{"int g int main() {}", "1:7: expected ','"},
		{"int h; int g = h; int main() {}", "1:16: not supported in Sorted! (yet): global initializers other than constants and addresses"},
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
func TestPreprocess(t *testing.T) {
	tests := []struct{ src, want string }{
		{"#define N 3\nint main() { putchar(N); }", "{ (putchar 3)}"},
		{"#define A B\n#define B 7\nint main() { putchar(A); }", "{ (putchar 7)}"},
		{"int v;\n#define v v + 1\nint main() { putchar(v); }", "{ (putchar (+ @v 1))}"},
		{"#define SQ(x) ((x) * (x))\nint main() { putchar(SQ(1 + 2)); }", "{ (putchar (* (+ 1 2) (+ 1 2)))}"},
		{"#define MAX(a, b) ((a) > (b) ? (a) : (b))\n#define ADD(a, b) a + b\nint main() { putchar(ADD((1, 2), MAX(3, 4))); }", "{ (putchar (+ (, 1 2) (?: (< 4 3) 3 4)))}"},
		{"#define ADD(a, b) (a + b)\nint main() { putchar(ADD(ADD(1, 2), 3)); }", "{ (putchar (+ (+ 1 2) 3))}"},
		{"#define F(a) a * 2\n#define G F\nint main() { putchar(G(3)); }", "{ (putchar (* 3 2))}"},
		{"#define F(a) a\nint main() { int F = 1; putchar(F); }", "{ { (= F 1)} (putchar F)}"},
		{"#define F (a) + a\nint main() { int a = 4; putchar(F); }", "{ { (= a 4)} (putchar (+ a a))}"},
		{"#define E()\nint main() { putchar(1 E()); }", "{ (putchar 1)}"},
		{"int f(int a) { return a; }\n#define f(x) x(1)\nint main() { putchar(f(f)); }", "{ (putchar (f 1))}"},
		{"#define N 3\n#undef N\n#define N 4\nint main() { putchar(N); }", "{ (putchar 4)}"},
		{"#\n# /* comment */\nint main() { }", "{}"},
		{"#define L 1 /* spans\n lines */ + 2\nint main() { putchar(L); }", "{ (putchar (+ 1 2))}"},
		{"#define N /*\n      */ (1)\nint main() { putchar(N); }", "{ (putchar 1)}"},
		{"#define N/**/(1)\nint main() { putchar(N); }", "{ (putchar 1)}"},
		{"#define IGNORE(x) 65\n#define PAIR(a, b) a + b\nint main() { putchar(IGNORE(PAIR(1))); }", "{ (putchar 65)}"},
		{"#define TWICE(x) x + x\n#define ONE 1\nint main() { putchar(TWICE(ONE)); }", "{ (putchar (+ 1 1))}"},
		{"int A = 65, F = 66;\n#define A F\n#define F(x) A\nint main() { putchar(A(0)); }", "{ (putchar @F)}"},
		{"int A = 65, F = 66;\n#define F(x) A\n#define A F(0)\nint main() { putchar(A); }", "{ (putchar @A)}"},
		{"#define N 10\nint a[N * 2 + 1]; int g = N << 2 | 1, h = -N, k = 'a' + (N > 3); int main() { putchar(g); }", "{ (putchar @g)}"},
	}
	for _, tt := range tests {
		p, err := Parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.src, err)
			continue
		}
		if got := sexpr(p.Main); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.src, got, tt.want)
		}
	}
	p, err := Parse("#define N 10\nint a[N * 2 + 1]; int g = N << 2 | 1, h = -N, k = 'a' + (N > 3); char s[N] = {N, N + 1}; int main() { }")
	if err != nil {
		t.Fatal(err)
	}
	g := p.Globals
	if g[0].Len != 21 || g[1].Init[0] != 41 || g[2].Init[0] != -10 || g[3].Init[0] != 98 || g[4].Len != 10 || g[4].Init[1] != 11 {
		t.Errorf("globals %+v %+v %+v %+v %+v", g[0], g[1], g[2], g[3], g[4])
	}
}

// Fold evaluates constant expressions in 32 bits (overflow wraps), and
// refuses division by zero, INT_MIN / -1 and shift counts outside 0..31.
func TestFold(t *testing.T) {
	for expr, want := range map[string]int32{
		"7 + 3 * 2 - 10 / 3 % 2": 12, "-7 / 2": -3, "-7 % 2": -1, "2147483647 + 1": -2147483648,
		"6 & 3": 2, "6 | 3": 7, "6 ^ 3": 5, "~5": -6, "1 << 31": -2147483648, "-16 >> 2": -4,
		"3 == 3": 1, "3 != 3": 0, "2 < 3": 1, "3 <= 2": 0, "3 > 2": 1, "2 >= 3": 0,
		"!0": 1, "!7": 0, "2 && 3": 1, "2 && 0": 0, "0 || 0": 0, "0 || 5": 1,
		"0 && 1 / 0": 0, "1 || 1 / 0": 1, "(-2147483647 - 1) / 1": -2147483648,
		"0xFFFFFFFF / 2": 2147483647, "0xFFFFFFFF % 10": 5, "0x80000000 >> 31": 1, "-1 < 1u": 0, "1u <= 0xFFFFFFFF": 1,
		"(0u - 1) / 2": 2147483647, "-1 >> 31": -1,
	} {
		p, err := Parse("int g = " + expr + "; int main() {}")
		if err != nil {
			t.Errorf("%s: %v", expr, err)
			continue
		}
		if got := p.Globals[0].Init[0]; got != want {
			t.Errorf("%s = %d, want %d", expr, got, want)
		}
	}
	for _, expr := range []string{"1 / 0", "1 % 0", "(-2147483647 - 1) / -1", "(-2147483647 - 1) % -1", "1 << 32", "1 >> -1", "1 && 1 / 0", "0 || 1 / 0", "!(1 / 0)", "~(1 / 0)", "-(1 / 0)"} {
		if _, err := Parse("int g = " + expr + "; int main() {}"); err == nil || !strings.Contains(err.Error(), "global initializers other than constants and addresses") {
			t.Errorf("%s: %v", expr, err)
		}
	}
}

func TestPointers(t *testing.T) {
	tests := []struct{ src, want string }{
		{"int main() { int x; int *p = &x; *p = 3; }", "{ {} { (= p (& x))} (= (* p) 3)}"},
		{"int main() { int a[3]; int *p = a; p[1] = *a; }", "{ {} { (= p (& a))} (= (* (+ p 1)) (* (& a)))}"},
		{"int main() { int **pp; int *p; pp = &p; **pp = 1; }", "{ {} {} (= pp (& p)) (= (* (* pp)) 1)}"},
		{"int main() { char *s = \"hi\"; s++; }", "{ { (= s (& @.str0))} (- (= s (+ s 1)) 1)}"},
		{"int main() { int a[2]; int *p = &a[1]; p = p - 1; }", "{ {} { (= p (& a[1]))} (= p (- p 1))}"},
		{"int *f(int *a, char b[]) { return a; } int main() { int x; f(&x, \"z\"); }", "{ {} (f (& x) (& @.str0))}"},
	}
	for _, tt := range tests {
		p, err := Parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.src, err)
			continue
		}
		if got := sexpr(p.Main); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.src, got, tt.want)
		}
	}
	// The types, as the type pass gives them.
	p, err := Parse(`char *names[] = {"a", "bc"}; int g[4]; int *gp = &g[2], *ge = g + 4; char *s = "a";
int *f(char **x) { return 0; } int main() { }`)
	if err != nil {
		t.Fatal(err)
	}
	// String literals come first, created while their user is parsed, and
	// equal ones are shared: s points at the same "a" as names[0].
	for i, want := range []string{"char[2]", "char[3]", "char *[2]", "int[4]", "int *", "int *", "char *"} {
		if got := p.Globals[i].Ty.String(); got != want {
			t.Errorf("global %d (%s): %s, want %s", i, p.Globals[i].Name, got, want)
		}
	}
	f := p.Funcs["f"]
	if f.Ret.String() != "int *" || f.Params[0].Ty.String() != "char **" {
		t.Errorf("f: %s (%s)", f.Ret, f.Params[0].Ty)
	}
	// Global addresses in every form addrConst knows.
	q, err := Parse(`int g[6]; int *a = 2 + g, *b = g + 4 - 1, *c = &g[3] - 2, *d = 0, *e = g; int main() { }`)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int32{2, 3, 1, 0, 0} {
		v := q.Globals[i+1]
		if v.Init[0] != want || (i == 3) != (v.InitRef == nil) || i != 3 && v.InitRef[0] != q.Globals[0] {
			t.Errorf("%s: %v %v, want offset %d", v.Name, v.Init, v.InitRef, want)
		}
	}
	// Global addresses: the variable and the offset.
	names, g, gp, ge, str := p.Globals[2], p.Globals[3], p.Globals[4], p.Globals[5], p.Globals[6]
	if names.InitRef[0] != p.Globals[0] || names.InitRef[1] != p.Globals[1] || gp.InitRef[0] != g || gp.Init[0] != 2 ||
		ge.InitRef[0] != g || ge.Init[0] != 4 || str.InitRef[0] != p.Globals[0] {
		t.Errorf("init refs: %+v %+v %+v %+v", names, gp, ge, str)
	}
}

func TestControlFlow(t *testing.T) {
	tests := []struct{ src, want string }{
		{"int main() { int i; do i++; while (i < 3); }", "{ {} (do (- (= i (+ i 1)) 1) (< i 3))}"},
		{"int main() { int x; switch (x) { case 1: x = 2; break; case 3 + 1: default: x = 0; } }",
			"{ {} (switch x { (case 1 (= x 2)) break (case 4 (case 0 (= x 0)))})}"},
		{"int main() { int x; x = x ? 1 : x < 0 ? -1 : 0; }", "{ {} (= x (?: x 1 (?: (< x 0) (neg 1) 0)))}"},
		{"int main() { int i, j; for (i = 0, j = 9; i < j; i++, j--) ; }", "{ {} (for (, (= i 0) (= j 9)) (< i j) (, (- (= i (+ i 1)) 1) (+ (= j (- j 1)) 1)) {})}"},
		{"int main() { int *p; int *q; q = p ? p : 0; }", "{ {} {} (= q (?: p p 0))}"},
		{"int g[1 ? 3 : 4]; int main() { }", "{}"},
	}
	for _, tt := range tests {
		p, err := Parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.src, err)
			continue
		}
		if got := sexpr(p.Main); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.src, got, tt.want)
		}
	}
	// A switch records its cases and default, which also stay in its body.
	p, err := Parse("int main() { switch (1) { case 5: default: case 7: ; } }")
	if err != nil {
		t.Fatal(err)
	}
	sw := p.Main.Body[0]
	if len(sw.Cases) != 2 || sw.Cases[0].Val != 5 || sw.Cases[1].Val != 7 || sw.Default == nil {
		t.Errorf("switch: %+v", sw)
	}
}

func TestStructs(t *testing.T) {
	tests := []struct{ src, want string }{
		{"struct p { int x, y; }; int main() { struct p a; a.y = 1; }", "{ {} (= (. a y) 1)}"},
		{"struct p { int x; }; int main() { struct p *q; q->x = 2; }", "{ {} (= (. (* q) x) 2)}"},
		{"struct p { int x; }; int main() { struct p a[2]; a[1].x = 3; }", "{ {} (= (. a[1] x) 3)}"},
		{"struct p { char n[2]; }; int main() { struct p a; a.n[1] = 4; }", "{ {} (= (* (+ (& (. a n)) 1)) 4)}"},
		{"struct p { int x; int y; }; int main() { struct p a = {5}; }", "{ { (= (. a x) 5) (= (. a y) 0)}}"},
		{"struct s { char n[2]; }; int main() { struct s a = {\"z\"}; }", "{ { (= (* (+ (& (. a n)) 0)) 122) (= (* (+ (& (. a n)) 1)) 0)}}"},
		{"int main() { struct q { int v; struct q *next; } a, b; a.next = &b; }", "{ {} (= (. a next) (& b))}"},
		{"struct p { int x; }; int main() { { struct p { char c; } a; a.c = 1; } struct p b; b.x = 2; }", "{ { {} (= (. a c) 1)} {} (= (. b x) 2)}"},
	}
	for _, tt := range tests {
		p, err := Parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.src, err)
			continue
		}
		if got := sexpr(p.Main); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.src, got, tt.want)
		}
	}
	// "struct t;" declares a new t in its scope, which hides an outer t and
	// is completed there.
	sh, err := Parse("struct t { int x; }; int main() { struct t; struct t *p; struct t { char y; } s; p = &s; p->y = 1; }")
	if err != nil {
		t.Fatal(err)
	}
	if sexpr(sh.Main) != "{ {} {} {} (= p (& s)) (= (. (* p) y) 1)}" {
		t.Errorf("shadowing: %s", sexpr(sh.Main))
	}
	// A forward declaration is completed by its body; a scalar may have braces.
	q, err := Parse("struct p; struct p *first; struct p { int x; }; struct p g = {1}; int s = {5}; int main() { first->x = 2; }")
	if err != nil {
		t.Fatal(err)
	}
	if q.Globals[0].Ty.Base != q.Globals[1].Ty || q.Globals[2].Init[0] != 5 {
		t.Errorf("forward declaration: %v %v", q.Globals[0].Ty.Base, q.Globals[1].Ty)
	}
	// Layout in cells, and flattened global initializers.
	p, err := Parse(`struct pt { int x; char y; };
struct box { struct pt a, b; int tag[3]; char *name; };
struct box g = {{1, 2}, 3, 4, {5, 6}, "n"};
struct pt list[] = {1, 2, 3, 4, {5}};
int main() { }`)
	if err != nil {
		t.Fatal(err)
	}
	g, list := p.Globals[1], p.Globals[2]
	box := g.Ty
	if box.Size() != 8 || box.Members[1].Offset != 2 || box.Members[2].Offset != 4 || box.Members[3].Offset != 7 || box.String() != "struct box" {
		t.Errorf("box: size %d, members %+v %+v %+v", box.Size(), box.Members[1], box.Members[2], box.Members[3])
	}
	if fmt.Sprint(g.Init) != "[1 2 3 4 5 6 0 0]" || g.InitRef[7] != p.Globals[0] {
		t.Errorf("g init %v refs %v", g.Init, g.InitRef)
	}
	if list.Len != 3 || list.Ty.Size() != 6 || fmt.Sprint(list.Init) != "[1 2 3 4 5]" {
		t.Errorf("list: len %d, size %d, init %v", list.Len, list.Ty.Size(), list.Init)
	}
}

func TestUnsignedAndArraysOfArrays(t *testing.T) {
	p, err := Parse(`unsigned a = 0xFFFFFFFF; unsigned int b = 4000000000u; unsigned char c = 300;
signed char d = -1; signed e = 017u; int m[3][4]; char n[][3] = {"ab", "c"};
int f(int x[][4], unsigned char *y) { return 0; }
int main() { }`)
	if err != nil {
		t.Fatal(err)
	}
	g := p.Globals
	for i, want := range []string{"unsigned int", "unsigned int", "unsigned char", "char", "int", "int[3][4]", "char[2][3]"} {
		if got := g[i].Ty.String(); got != want {
			t.Errorf("global %s: %s, want %s", g[i].Name, got, want)
		}
	}
	if g[0].Init[0] != -1 || g[1].Init[0] != int32(-294967296) || g[2].Init[0] != 44 || g[4].Init[0] != 15 || g[6].Len != 2 || fmt.Sprint(g[6].Init) != "[97 98 0 99 0]" {
		t.Errorf("init: %v %v %v %v len %d %v", g[0].Init, g[1].Init, g[2].Init, g[4].Init, g[6].Len, g[6].Init)
	}
	f := p.Funcs["f"]
	if f.Params[0].Ty.String() != "int (*)[4]" || f.Params[1].Ty.String() != "unsigned char *" {
		t.Errorf("params %s, %s", f.Params[0].Ty, f.Params[1].Ty)
	}
	// The usual arithmetic conversions.
	q, err := Parse(`unsigned u; int i; unsigned char uc; int r[2][2];
int main() { u + i; i - uc; -uc; ~u; uc << 1; u >> i; i ? u : i; r[1]; r[1][1]; u < i; }`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range q.Main.Body {
		got = append(got, s.Lhs.Ty.String())
	}
	if want := "unsigned int, int, int, unsigned int, int, unsigned int, unsigned int, int *, int, int"; strings.Join(got, ", ") != want {
		t.Errorf("types: %s\nwant   %s", strings.Join(got, ", "), want)
	}
}

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
