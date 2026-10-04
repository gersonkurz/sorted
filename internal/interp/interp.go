// Package interp runs a parsed Sorted! program. It ports SortedInterpreter
// from legacy/sorted.win32/Sorted.cpp line by line, quirks included:
//
//   - A read stores into the read entry's second operand, which the parser
//     never fills, so input always lands in the first cell (or wherever
//     scratch left by a failed parse alternative points).
//   - Indirect reads use Data[v-1], indirect writes Data[v].
//   - "Logical operations" compute ~a & ~b.
//   - A jump continues after its label; a declared label that is never placed
//     is 0, so jumping to it skips statement 0.
//   - A reference past the end of its table ("the second output" when there
//     is one) reads the next slot of the static Code array: another table's
//     entry, scratch, or zeros.
//
// Undefined behaviour of the C code is not emulated (Gerson's ruling, see
// CLAUDE.md). The port defines it instead:
//
//   - A runtime *Error: a cell outside the 193719 cells, a code slot outside
//     the 128192 slots, division by zero, formatting a negative cardinal,
//     and recursion deeper than 65536 (a sum that contains itself).
//   - What the code evidently intends: an indirect sum, difference, product,
//     ratio, logical operation or condition reads the cell its value indexes
//     (the original indexes its table directory with the unmasked type).
//   - Nothing: a number or an indirect operand used as a statement.
//   - Zero: a label beyond the declared count that is never placed (the
//     original reads uninitialised memory or past its label array).
//   - Two's complement wrap-around for +, - and *, and for MinInt32 / -1.
package interp

import (
	"bufio"
	"errors"
	"io"
	"math"

	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

const (
	dataCells = 193719 // MAX_DATA_PER_PROGRAM
	codeSlots = 128192 // MAX_INSTRUCTIONS_PER_PROGRAM
	maxDepth  = 1 << 16
)

// Error is a runtime failure. For the original these are undefined
// behaviour, mostly a crash; the port stops the program with this error.
type Error struct {
	What string
}

func (e *Error) Error() string { return "runtime error: " + e.What }

// ErrStepLimit reports that Run stopped after its step limit.
var ErrStepLimit = errors.New("step limit reached")

// Run executes p, reading "reads" input from in and writing output to out.
// With maxSteps > 0 it stops with ErrStepLimit after that many statements.
// Output written before an error is flushed.
func Run(p *syntax.Program, in io.Reader, out io.Writer, maxSteps int) error {
	m := &machine{
		p:      p,
		data:   make([]int32, dataCells),
		labels: map[int32]int32{},
		in:     &stdin{r: bufio.NewReader(in)},
		out:    bufio.NewWriter(out),
	}
	copy(m.data, p.Data)
	err := m.run(maxSteps)
	if ferr := m.out.Flush(); err == nil {
		err = ferr
	}
	return err
}

type machine struct {
	p      *syntax.Program
	data   []int32
	labels map[int32]int32
	in     *stdin
	out    *bufio.Writer
	depth  int
}

func fail(what string) error { return &Error{what} }

// code returns slot i of the static Code array: zeros past what the parser
// wrote.
func (m *machine) code(i int) (syntax.Slide, error) {
	switch {
	case i < 0 || i >= codeSlots:
		return syntax.Slide{}, fail("code slot out of range")
	case i < len(m.p.Code):
		return m.p.Code[i], nil
	}
	return syntax.Slide{}, nil
}

// entry returns entry index of category c, reading past the table's end
// like the original does.
func (m *machine) entry(c syntax.Category, index int32) (syntax.Slide, error) {
	return m.code(m.p.Tables[c].Index + int(index))
}

func (m *machine) cell(i int32) (int32, error) {
	if i < 0 || i >= dataCells {
		return 0, fail("cell out of range")
	}
	return m.data[i], nil
}

// value evaluates an operand (GetDataValue). Expressions are evaluated anew
// on every reference.
func (m *machine) value(op syntax.Operand) (int32, error) {
	if m.depth++; m.depth > maxDepth {
		return 0, fail("recursion too deep")
	}
	defer func() { m.depth-- }()

	var r int32
	var err error
	switch t := op.Type & 0xFF; t {
	case syntax.Number:
		r, err = m.cell(op.Index)
	case syntax.Sum, syntax.Diff, syntax.Prod, syntax.Ratio, syntax.Nand, syntax.Condition:
		r, err = m.expression(t, op.Index)
	case syntax.Label:
		r = m.labels[op.Index]
	}
	if err != nil {
		return 0, err
	}
	if op.Type&syntax.Indirect != 0 {
		return m.cell(r - 1)
	}
	return r, nil
}

// expression evaluates entry index of the table for operand type t: the sum,
// difference, product, ratio, logical operation or condition.
func (m *machine) expression(t syntax.OperandType, index int32) (int32, error) {
	s, err := m.entry(syntax.Category(t-1), index) // Type[ps->Type-1]
	if err != nil {
		return 0, err
	}
	a, err := m.value(s.Ops[0])
	if err != nil {
		return 0, err
	}
	b, err := m.value(s.Ops[1])
	if err != nil {
		return 0, err
	}
	switch t {
	case syntax.Sum:
		return a + b, nil
	case syntax.Diff:
		return a - b, nil
	case syntax.Prod:
		return a * b, nil
	case syntax.Ratio:
		if b == 0 {
			return 0, fail("division by zero")
		}
		if a == math.MinInt32 && b == -1 {
			return math.MinInt32, nil
		}
		return a / b, nil
	case syntax.Nand:
		return ^a & ^b, nil
	}
	// Condition
	if s.Flags == syntax.CompareEqual {
		return b2i(a == b), nil
	}
	return b2i(a < b), nil
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

// pointer returns the cell an assignment or read stores into
// (GetDataPointer): a number directly, an indirect number at Data[Data[i]].
func (m *machine) pointer(op syntax.Operand) (int32, error) {
	if op.Type&0xFF != syntax.Number {
		return 0, fail("store into something that is not a cell")
	}
	i := op.Index
	if op.Type&syntax.Indirect != 0 {
		v, err := m.cell(i)
		if err != nil {
			return 0, err
		}
		i = v
	}
	if _, err := m.cell(i); err != nil {
		return 0, err
	}
	return i, nil
}

func (m *machine) run(maxSteps int) error {
	stmts := m.p.Tables[syntax.Statements]
	for j := 0; j < stmts.Count; j++ {
		s, _ := m.code(stmts.Index + j)
		if s.Ops[0].Type == syntax.Label {
			m.labels[s.Ops[0].Index] = int32(j)
		}
	}
	steps := 0
	for pc := int32(0); pc < int32(stmts.Count); pc++ {
		if maxSteps > 0 {
			if steps == maxSteps {
				return ErrStepLimit
			}
			steps++
		}
		s, err := m.code(stmts.Index + int(pc))
		if err != nil {
			return err
		}
		op := s.Ops[0]
		switch op.Type {
		case syntax.Assign:
			a, err := m.entry(syntax.Assigns, op.Index)
			if err != nil {
				return err
			}
			i, err := m.pointer(a.Ops[1])
			if err != nil {
				return err
			}
			v, err := m.value(a.Ops[0])
			if err != nil {
				return err
			}
			m.data[i] = v
		case syntax.Read:
			r, err := m.entry(syntax.Reads, op.Index)
			if err != nil {
				return err
			}
			i, err := m.pointer(r.Ops[1]) // sic: the parser fills Ops[0]
			if err != nil {
				return err
			}
			if err := m.out.Flush(); err != nil {
				return err
			}
			m.data[i] = m.in.getchar()
		case syntax.Write:
			w, err := m.entry(syntax.Writes, op.Index)
			if err != nil {
				return err
			}
			v, err := m.value(w.Ops[0])
			if err != nil {
				return err
			}
			if err := m.write(w.Flags, v); err != nil {
				return err
			}
		case syntax.Jump:
			j, err := m.entry(syntax.Jumps, op.Index)
			if err != nil {
				return err
			}
			take := j.Flags == syntax.UnconditionalJump
			if !take {
				c, err := m.value(j.Ops[1])
				if err != nil {
					return err
				}
				take = c != 0
			}
			if take {
				if pc, err = m.value(j.Ops[0]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// write prints v in the given format: a character (putchar), or a cardinal or
// ordinal followed by a newline.
func (m *machine) write(format int32, v int32) error {
	var s string
	var err error
	switch format {
	case syntax.FormatEnglishCardinal:
		s, err = numbers.EnglishCardinal(v)
	case syntax.FormatEnglishOrdinal:
		s, err = numbers.EnglishOrdinal(v)
	case syntax.FormatGermanCardinal:
		s, err = numbers.GermanCardinal(v)
	case syntax.FormatGermanOrdinal:
		s, err = numbers.GermanOrdinal(v)
	default:
		return m.out.WriteByte(byte(v))
	}
	if err != nil {
		return fail("cannot write a negative number as a cardinal")
	}
	_, err = m.out.WriteString(s + "\n")
	return err
}

// stdin reads like getchar on the Win32 C runtime's text-mode stdin: CRLF
// arrives as LF, and Ctrl-Z ends the input for good. The end of input is -1.
type stdin struct {
	r   *bufio.Reader
	eof bool
}

func (s *stdin) getchar() int32 {
	if s.eof {
		return -1
	}
	b, err := s.r.ReadByte()
	if err != nil || b == 0x1A {
		s.eof = true
		return -1
	}
	if b == '\r' {
		if next, err := s.r.Peek(1); err == nil && next[0] == '\n' {
			s.r.ReadByte()
			return '\n'
		}
	}
	return int32(b)
}
