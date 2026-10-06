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
// scopes; putchar is the only function a program can call, and main the only
// function it can define. With int as the only type, there is no type pass
// yet.
//
// #include <stdio.h> is the one preprocessor line it accepts (and ignores), so
// that a program for Sorted! is also a C program that declares putchar.
//
// The subset: int and char variables and one-dimensional arrays of them,
// global (with constant initializers) and local (initializers are
// assignments, with {...} lists and string literals for arrays), integer
// and character constants, a[i], + - * / % and unary -, =, the compound
// assignments += -= *= /= %=, ++ and -- (prefix and postfix), == != < <= >
// >=, && || !, if/else, while, for, break, continue, return, blocks, and
// putchar(expr). As in chibicc, x op= e is x = x op e, ++x is x = x + 1 and
// x++ is (x = x + 1) - 1; in the subset x is always a plain variable, so
// evaluating it twice is harmless.
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
	NdEq                       // ==
	NdNe                       // !=
	NdLt                       // <
	NdLe                       // <=
	NdAssign                   // =
	NdReturn                   // "return"
	NdIf                       // "if"
	NdWhile                    // "while"
	NdFor                      // "for"
	NdBreak                    // "break"
	NdContinue                 // "continue"
	NdLogAnd                   // &&
	NdLogOr                    // ||
	NdNot                      // !
	NdIndex                    // a[i]
	NdBlock                    // { ... }
	NdFuncall                  // putchar(...)
	NdExprStmt                 // expression statement
	NdVar                      // variable
	NdNum                      // integer
)

// Obj is a variable: an int or char, or a one-dimensional array of them.
type Obj struct {
	Name     string
	IsGlobal bool
	Char     bool    // char (stores wrap to -128..127) rather than int
	Len      int     // number of elements of an array; 0 for a scalar
	Init     []int32 // initial values of a global (shorter than Len: the rest is 0)
	Pos      Pos     // where it is declared
}

// Node is an AST node.
type Node struct {
	Kind NodeKind
	Pos  Pos

	Lhs, Rhs *Node // operands, assignment target and value

	Cond, Then, Els *Node // if, while, for
	Init, Inc       *Node // for

	Body []*Node // block

	Func string  // called function (putchar)
	Args []*Node // call arguments

	Var *Obj  // NdVar, NdIndex (the array; Lhs is the index)
	Val int32 // NdNum

	// Rvalue marks a variable that is not assignable: the operand of a
	// unary + (C's +a is a value, not an lvalue).
	Rvalue bool

	// WrapChar marks the value of a postfix ++ or -- on a char: the old value
	// is rebuilt as new - 1 (or + 1), which must wrap to char again, since the
	// new value wrapped (127++ stores -128, and -128 - 1 must give 127).
	WrapChar bool
}

// Program is a parsed translation unit.
type Program struct {
	Globals []*Obj
	Main    *Node // the body of main, a block
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
