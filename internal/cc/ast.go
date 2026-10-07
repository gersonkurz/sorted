// Package cc is the C front end of the C-to-Sorted! compiler: a tokenizer
// and a recursive-descent parser for the subset of C that Sorted! can
// express.
//
// It is a Go port of the tokenizer and parser of chibicc, Rui Ueyama's small
// C compiler (https://github.com/rui314/chibicc, MIT license, see
// LICENSE.chibicc), taken from the state of its history where the language
// is still small (commit a4d3223, "Add global variables"). The structure and
// the names follow chibicc: one function per grammar symbol, each taking the
// token position and returning a node.
//
// Differences from chibicc: errors are returned instead of ending the
// process; constructs outside the subset are rejected with a message saying
// so; % is added, as are comments, hexadecimal and octal literals and block
// scopes; putchar and getchar are the only library functions. The type pass (typed, after
// chibicc's add_type) runs as expressions are parsed and checks what C's
// constraints require, stricter than a compiler that only warns: pointers
// and integers do not mix, except for the null pointer constant 0.
//
// A small preprocessor (Preprocess) runs #define and #undef and accepts
// #include <stdio.h>, so that a program for Sorted! is also a C program that
// declares putchar.
//
// The subset: int and char variables, signed and unsigned, structs
// (members of any of these types, tags in scopes, a struct pointing to its
// own kind), pointers to any of them (to any depth), and arrays of those,
// also of arrays (int m[3][4]; a parameter int m[][4]), global
// (with constant expressions and addresses of globals as initializers) and
// local (initializers are assignments, with nested {...} lists, brace
// elision, and string literals for char arrays), integer
// and character constants, string literals (char arrays of their own), a[i],
// unary & and *, . and ->, pointer arithmetic, + - * / % and unary -, =
// (also of whole structs), the compound
// assignments += -= *= /= %= &= |= ^= <<= >>=, ++ and -- (prefix and
// postfix), == != < <= > >=, && || !, & | ^ ~ << >>, ?:, the comma
// operator, if/else, while, do/while, for, switch (case, default,
// fall-through), break, continue, return, blocks, and
// putchar(expr), getchar(), and functions with int, char and pointer
// parameters (an
// array parameter is a pointer) returning int, char, a pointer or void
// (prototypes included), called by name. As in chibicc, x op= e is
// x = x op e, ++x is x = x + 1 and x++ is (x = x + 1) - 1; the compiler
// evaluates the location of x once.
package cc

import "fmt"

// NodeKind says what an AST node is.
type NodeKind int

// Node kinds, as in chibicc's NodeKind.
const (
	NdAdd      NodeKind = iota // +
	NdSub                      // -
	NdMul                      // *
	NdDiv                      // /
	NdMod                      // %
	NdNeg                      // unary -
	NdBitAnd                   // &
	NdBitOr                    // |
	NdBitXor                   // ^
	NdBitNot                   // ~
	NdShl                      // <<
	NdShr                      // >>
	NdEq                       // ==
	NdNe                       // !=
	NdLt                       // <
	NdLe                       // <=
	NdAssign                   // =
	NdReturn                   // "return"
	NdIf                       // "if"
	NdWhile                    // "while"
	NdFor                      // "for"
	NdDo                       // "do" … "while"
	NdSwitch                   // "switch"
	NdCase                     // "case" and "default" labels
	NdCond                     // ?:
	NdComma                    // the comma operator
	NdBreak                    // "break"
	NdContinue                 // "continue"
	NdLogAnd                   // &&
	NdLogOr                    // ||
	NdNot                      // !
	NdIndex                    // a[i] for an array variable a
	NdAddr                     // unary &, and an array used as a value (its first element's address)
	NdDeref                    // unary *, and p[i] for anything but an array variable (*(p + i))
	NdMember                   // s.m, and p->m as (*p).m
	NdBlock                    // { ... }
	NdFuncall                  // a call: a function, putchar(...) or getchar()
	NdExprStmt                 // expression statement
	NdVar                      // variable
	NdNum                      // integer
)

// Obj is a variable: an int, a char, a pointer, or a one-dimensional array
// of them.
type Obj struct {
	Name     string
	IsGlobal bool
	Ty       *Type
	Char     bool    // a char, or an array of chars (stores wrap to -128..127)
	Len      int     // number of elements of an array; 0 for a scalar
	Init     []int32 // initial values of a global (shorter than Len: the rest is 0)
	// InitRef, when not nil, holds for each initial value the variable whose
	// address it is (Init is then the offset into it), or nil for a number:
	// char *s = "hi", int *p = &a[2].
	InitRef []*Obj
	Pos     Pos // where it is declared
}

// Node is an AST node.
type Node struct {
	Kind NodeKind
	Pos  Pos

	Lhs, Rhs *Node // operands, assignment target and value

	Cond, Then, Els *Node // if, while, for, do, switch (Cond, Then), case (Then), ?:
	Init, Inc       *Node // for

	Cases   []*Node // switch: its case labels (also inside Then), in order
	Default *Node   // switch: its default label, if any

	Body []*Node // block

	Func string    // called function
	Fn   *Function // the function called, nil for putchar and getchar
	Args []*Node   // call arguments

	Var    *Obj    // NdVar, NdIndex (the array; Lhs is the index)
	Member *Member // NdMember (Lhs is the struct)
	Val    int32   // NdNum

	Ty *Type // the expression's type, set by the parser's type pass

	// Rvalue marks a variable that is not assignable: the operand of a
	// unary + (C's +a is a value, not an lvalue).
	Rvalue bool

	// Wrap marks the value of a postfix ++ or -- on a char with the char's
	// type: the old value is rebuilt as new - 1 (or + 1), which must wrap
	// again, since the new value wrapped (127++ stores -128, and -128 - 1 must
	// give 127; for an unsigned char 255++ stores 0).
	Wrap *Type
}

// Function is a function other than main.
type Function struct {
	Name    string
	Params  []*Obj
	Ret     *Type // the result type, nil for void
	Void    bool  // returns nothing
	Char    bool  // returns char (the result wraps to -128..127)
	Body    *Node
	Defined bool // false for a prototype not (yet) followed by a definition
	Pos     Pos
}

// Program is a parsed translation unit.
type Program struct {
	Globals []*Obj
	Main    *Node                // the body of main, a block
	Funcs   map[string]*Function // the other functions, by name
}

// Pos is a position in the source, 1-based.
type Pos struct {
	Line, Col int
}

func (p Pos) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// Error is a syntax error, or a construct outside the subset.
type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Msg }

func errorAt(p Pos, format string, args ...any) error {
	return &Error{p, fmt.Sprintf(format, args...)}
}
