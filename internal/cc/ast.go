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
// Subset 1: int globals (with constant initializers) and locals, integer
// constants, + - * / % and unary -, =, == != < <= > >=, if/else, while,
// return, blocks, and putchar(expr).
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
	NdBlock                    // { ... }
	NdFuncall                  // putchar(...)
	NdExprStmt                 // expression statement
	NdVar                      // variable
	NdNum                      // integer
)

// Obj is a variable.
type Obj struct {
	Name     string
	IsGlobal bool
	Init     int32 // initial value of a global
	Pos      Pos   // where it is declared
}

// Node is an AST node.
type Node struct {
	Kind NodeKind
	Pos  Pos

	Lhs, Rhs *Node // operands, assignment target and value

	Cond, Then, Els *Node // if, while

	Body []*Node // block

	Func string  // called function (putchar)
	Args []*Node // call arguments

	Var *Obj  // NdVar
	Val int32 // NdNum

	// Rvalue marks a variable that is not assignable: the operand of a
	// unary + (C's +a is a value, not an lvalue).
	Rvalue bool
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
