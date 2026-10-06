// Package compile lowers a C program (internal/cc) into Sorted! tables
// (syntax.Program), which internal/render then writes as Sorted! text.
//
// Sorted! is three-address code in disguise, and the lowering is mostly a
// change of vocabulary:
//
//   - Constants become the declared numbers. Each value is declared once
//     ("thou shalt not have the same cardinal more than once"), so the pool
//     is deduplicated; a negative constant is 0 - n.
//   - Variables live in the cells after the pool, which start at zero like C
//     globals.
//   - + - * / become sums, ordered differences, products and ratios; equal
//     expressions share one entry. a % b is a - (a/b)*b, which matches C's
//     truncating division.
//   - == and < become conditions, whose value is 1 or 0; != is 1 - (a==b) and
//     a <= b is 1 - (b<a).
//   - if, while and for become labels and conditional jumps; break and
//     continue jump to the end of the loop or to its next round. A jump is
//     taken when its condition is true, so each test jumps on the negated
//     condition, and negating a condition is a condition again ("the first
//     condition is equal to zero").
//   - && and || short-circuit: in a test they become a chain of jumps, and as
//     a value they set a temporary cell to 0 or 1 by jumping, because a
//     product of conditions would evaluate both sides. !x is x == 0.
//   - Arrays are runs of cells. An element with a constant index is a cell
//     like any other. Otherwise it is reached through a pointer cell that an
//     assignment fills just before the statement that uses it: "the cell
//     indexed by" a cell reads Data[p-1] but writes Data[p] (the original's
//     off-by-one, which also gives itoa.s its NUL), so a read pointer holds
//     the element's cell number plus one and a write pointer the cell number
//     itself. Indexing a computed value directly ("the cell indexed by the
//     first sum") is undefined behaviour in the original, so it is avoided.
//     Cell numbers depend on how many numbers are declared, and the
//     addresses are declared numbers themselves; see program.
//   - A store into a char wraps the value to -128..127, as C does, by
//     arithmetic: ((v + 128) % 256 + 256) % 256 - 128.
//   - putchar(e) assigns e to an output cell and runs the program's only
//     output, which writes that cell as a character.
//   - return jumps past the last statement.
//
// Assignments inside expressions are hoisted into statements before the
// statement that contains them. putchar is a statement: its result cannot be
// used.
package compile

import (
	"fmt"

	"github.com/gersonkurz/sorted/internal/cc"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// Error reports a program the compiler cannot lower.
type Error struct {
	Pos cc.Pos
	Msg string
}

func (e *Error) Error() string { return e.Pos.String() + ": " + e.Msg }

// largest is the largest number a Sorted! declaration can spell in one go.
const largest = 999999999

// memory is the number of cells a Sorted! program has (MAX_DATA_PER_PROGRAM
// in the original's SortedSyntax.h).
const memory = 193719

// valKind says what a value refers to.
type valKind int

const (
	vConst valKind = iota // a declared number, by pool index
	vVar                  // a variable cell, by variable index
	vSum
	vDiff
	vProd
	vRatio
	vCond
	vInd  // the cell a pointer cell (variable i) indexes
	vAddr // an address: a declared number, the cell of variable i (see program)
)

// val is an operand before cells are numbered.
type val struct {
	kind valKind
	i    int
}

// entry is a two-operand table entry (expression, condition or assignment).
type entry struct {
	a, b  val
	flags int32
}

// table collects entries, sharing equal ones.
type table struct {
	entries []entry
	index   map[entry]int
}

func (t *table) add(e entry) int {
	if t.index == nil {
		t.index = map[entry]int{}
	}
	if i, ok := t.index[e]; ok {
		return i
	}
	t.entries = append(t.entries, e)
	t.index[e] = len(t.entries) - 1
	return len(t.entries) - 1
}

type stmtKind int

const (
	sAssign stmtKind = iota
	sWrite
	sJump
	sLabel
)

type stmt struct {
	kind stmtKind
	i    int
}

type jump struct {
	label int
	cond  int // -1: unconditional
}

// loop holds the labels of the innermost loop: where break and continue go.
// They are made when first needed (-1 until then).
type loop struct {
	brk, cont int
}

// compiler holds the tables being built.
type compiler struct {
	pool      []int32       // declared numbers
	poolIndex map[int32]int // value → pool index
	vars      map[*cc.Obj]int
	nvars     int
	out       int // the output cell's variable index, -1 until needed

	addrs     []int            // address numbers, as variable-cell offsets (see program)
	fillers   int              // numbers program declared only to keep the pool size
	addrIndex map[int]int      // offset → index in addrs
	reads     map[*cc.Node]val // array elements whose read pointer is already set

	exprs   map[valKind]*table // sums, differences, products, ratios, conditions
	assigns table
	jumps   []jump
	jumpIdx map[jump]int
	labels  int
	stmts   []stmt
	loops   []*loop
}

// Compile lowers a parsed C program into Sorted! tables.
func Compile(prog *cc.Program) (*syntax.Program, error) {
	_, p, err := compileProgram(prog)
	return p, err
}

// compileProgram is Compile, also returning the compiler for inspection.
func compileProgram(prog *cc.Program) (c *compiler, p *syntax.Program, err error) {
	c = &compiler{
		poolIndex: map[int32]int{},
		vars:      map[*cc.Obj]int{},
		out:       -1,
		exprs:     map[valKind]*table{},
		jumpIdx:   map[jump]int{},
		addrIndex: map[int]int{},
		reads:     map[*cc.Node]val{},
	}
	for _, k := range []valKind{vSum, vDiff, vProd, vRatio, vCond} {
		c.exprs[k] = &table{}
	}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			p, err = nil, e
		}
	}()
	for _, g := range prog.Globals {
		base := c.variable(g)
		for i, v := range g.Init {
			if g.Char {
				v = int32(int8(v))
			}
			if v != 0 {
				c.assign(val{vVar, base.i + i}, c.number(v, g.Pos))
			}
		}
	}
	exit := -1
	for i, s := range prog.Main.Body {
		if s.Kind == cc.NdReturn && i == len(prog.Main.Body)-1 {
			c.effects(s.Lhs) // a final return just ends the program
			continue
		}
		c.stmt(s, &exit)
	}
	if exit >= 0 {
		c.place(exit)
	}
	if len(c.stmts) == 0 {
		c.place(c.newLabel()) // a program needs at least one statement
	}
	return c, c.program(), nil
}

func fail(pos cc.Pos, format string, args ...any) {
	panic(&Error{pos, fmt.Sprintf(format, args...)})
}

// --- values ---

// number returns a constant value, declaring it if necessary. Negative
// values are 0 - n.
func (c *compiler) number(v int32, pos cc.Pos) val {
	if v < 0 { // never -2147483648: C has no such literal, only -2147483647 - 1
		return c.expr(vDiff, c.number(0, pos), c.number(-v, pos), 0)
	}
	if v > largest {
		fail(pos, "constants above %d are not supported yet (%d)", largest, v)
	}
	i, ok := c.poolIndex[v]
	if !ok {
		i = len(c.pool)
		c.pool = append(c.pool, v)
		c.poolIndex[v] = i
	}
	return val{vConst, i}
}

// variable returns the cell of a scalar, or the first cell of an array.
func (c *compiler) variable(o *cc.Obj) val {
	i, ok := c.vars[o]
	if !ok {
		i = c.cells(max(1, o.Len), o.Pos)
		c.vars[o] = i
	}
	return val{vVar, i}
}

// cells reserves n variable cells and returns the first. It fails as soon as
// the variables alone exceed Sorted!'s memory, so the count never grows past
// it (and cannot overflow an int on 32-bit platforms); program checks the
// total with the declared numbers.
func (c *compiler) cells(n int, pos cc.Pos) int {
	if int64(c.nvars)+int64(n) > memory {
		fail(pos, "the program needs more than %d memory cells, which is all Sorted! has", memory)
	}
	c.nvars += n
	return c.nvars - n
}

// address returns the declared number that is the cell number of variable
// cell v plus extra (see program).
func (c *compiler) address(v val, extra int) val {
	off := v.i + extra
	i, ok := c.addrIndex[off]
	if !ok {
		i = len(c.addrs)
		c.addrs = append(c.addrs, off)
		c.addrIndex[off] = i
	}
	return val{vAddr, i}
}

// constValue reports the value of a constant: a declared number, or 0 - n.
func (c *compiler) constValue(v val) (int32, bool) {
	switch v.kind {
	case vConst:
		return c.pool[v.i], true
	case vDiff:
		e := c.exprs[vDiff].entries[v.i]
		a, okA := c.constValue(e.a)
		b, okB := c.constValue(e.b)
		return a - b, okA && okB
	}
	return 0, false
}

// mod is a % b with C's sign rule.
func (c *compiler) mod(a, b val) val {
	return c.expr(vDiff, a, c.expr(vProd, c.expr(vRatio, a, b, 0), b, 0), 0)
}

// wrapChar converts v to char: -128..127, as a store into a char does.
func (c *compiler) wrapChar(v val, pos cc.Pos) val {
	if k, ok := c.constValue(v); ok {
		if k == int32(int8(k)) {
			return v
		}
		return c.number(int32(int8(k)), pos)
	}
	n128, n256 := c.number(128, pos), c.number(256, pos)
	m := c.mod(c.expr(vSum, c.mod(c.expr(vSum, v, n128, 0), n256), n256, 0), n256)
	return c.expr(vDiff, m, n128, 0)
}

// element returns array element a[i] for reading: a cell for a constant
// index, otherwise the cell a read pointer indexes.
func (c *compiler) element(n *cc.Node) val {
	if v, ok := c.reads[n]; ok {
		return v
	}
	base := c.variable(n.Var)
	if k, ok := c.constIndex(n); ok {
		return val{vVar, base.i + k}
	}
	p := c.temporary()
	c.assign(p, c.expr(vSum, c.value(n.Lhs), c.address(base, 1), 0))
	return val{vInd, p.i}
}

// constIndex reports a constant index of a[i], checking its bounds.
func (c *compiler) constIndex(n *cc.Node) (int, bool) {
	if n.Lhs.Kind != cc.NdNum && !(n.Lhs.Kind == cc.NdNeg && n.Lhs.Lhs.Kind == cc.NdNum) {
		return 0, false
	}
	k := int(n.Lhs.Val)
	if n.Lhs.Kind == cc.NdNeg {
		k = -int(n.Lhs.Lhs.Val)
	}
	if k < 0 || k >= n.Var.Len {
		fail(n.Lhs.Pos, "index %d is out of range for '%s' (%d elements)", k, n.Var.Name, n.Var.Len)
	}
	return k, true
}

func (c *compiler) expr(k valKind, a, b val, flags int32) val {
	return val{k, c.exprs[k].add(entry{a, b, flags})}
}

func (c *compiler) eq(a, b val) val { return c.expr(vCond, a, b, syntax.CompareEqual) }
func (c *compiler) lt(a, b val) val { return c.expr(vCond, a, b, syntax.CompareLess) }

// not is 1 when v is 0 and 0 otherwise.
func (c *compiler) not(v val, pos cc.Pos) val { return c.eq(v, c.number(0, pos)) }

// value lowers an expression, hoisting the assignments it contains.
func (c *compiler) value(n *cc.Node) val {
	switch n.Kind {
	case cc.NdNum:
		return c.number(n.Val, n.Pos)
	case cc.NdVar:
		return c.variable(n.Var)
	case cc.NdIndex:
		return c.element(n)
	case cc.NdNeg:
		if n.Lhs.Kind == cc.NdNum {
			return c.number(-n.Lhs.Val, n.Pos)
		}
		return c.expr(vDiff, c.number(0, n.Pos), c.value(n.Lhs), 0)
	case cc.NdAssign:
		return c.assignment(n, true)
	case cc.NdFuncall:
		fail(n.Pos, "using the result of putchar is not supported yet")
	case cc.NdNot:
		return c.not(c.value(n.Lhs), n.Pos)
	case cc.NdLogAnd, cc.NdLogOr:
		// t = 0; if (n) t = 1;  so that the right side runs only when needed
		t := c.temporary()
		c.assign(t, c.number(0, n.Pos))
		skip := c.newLabel()
		c.jumpIf(n, skip, false)
		c.assign(t, c.number(1, n.Pos))
		c.place(skip)
		return t
	}
	if n.WrapChar {
		w := *n
		w.WrapChar = false
		return c.wrapChar(c.value(&w), n.Pos)
	}
	a, b := c.value(n.Lhs), c.value(n.Rhs)
	switch n.Kind {
	case cc.NdAdd:
		return c.expr(vSum, a, b, 0)
	case cc.NdSub:
		return c.expr(vDiff, a, b, 0)
	case cc.NdMul:
		return c.expr(vProd, a, b, 0)
	case cc.NdDiv:
		return c.expr(vRatio, a, b, 0)
	case cc.NdMod:
		return c.mod(a, b)
	case cc.NdEq:
		return c.eq(a, b)
	case cc.NdNe:
		return c.expr(vDiff, c.number(1, n.Pos), c.eq(a, b), 0)
	case cc.NdLt:
		return c.lt(a, b)
	case cc.NdLe:
		return c.expr(vDiff, c.number(1, n.Pos), c.lt(b, a), 0)
	}
	fail(n.Pos, "cannot lower this expression")
	return val{}
}

// temporary returns a fresh cell for an intermediate value.
func (c *compiler) temporary() val {
	return val{vVar, c.cells(1, cc.Pos{Line: 1, Col: 1})}
}

// cond returns a condition that is true exactly when n is true (sense) or
// exactly when it is false (!sense), using the cheapest form for
// comparisons.
func (c *compiler) cond(n *cc.Node, sense bool) val {
	switch {
	case n.Kind == cc.NdEq && sense, n.Kind == cc.NdNe && !sense:
		return c.eq(c.value(n.Lhs), c.value(n.Rhs))
	case n.Kind == cc.NdLt && sense:
		return c.lt(c.value(n.Lhs), c.value(n.Rhs))
	case n.Kind == cc.NdLe && !sense:
		a, b := c.value(n.Lhs), c.value(n.Rhs)
		return c.lt(b, a)
	case sense:
		return c.not(c.not(c.value(n), n.Pos), n.Pos)
	}
	return c.not(c.value(n), n.Pos)
}

// jumpIf jumps to label when n is true (sense) or false (!sense). && and ||
// become chains of jumps, so their right side is only evaluated when it
// matters, and ! flips the sense.
func (c *compiler) jumpIf(n *cc.Node, label int, sense bool) {
	switch n.Kind {
	case cc.NdNot:
		c.jumpIf(n.Lhs, label, !sense)
	case cc.NdLogAnd, cc.NdLogOr:
		// "a && b is false" and "a || b is true" each hold as soon as the left
		// side says so; the other two need both sides.
		decidesAlone := n.Kind == cc.NdLogAnd != sense
		if decidesAlone {
			c.jumpIf(n.Lhs, label, sense)
			c.jumpIf(n.Rhs, label, sense)
			return
		}
		skip := c.newLabel()
		c.jumpIf(n.Lhs, skip, !sense)
		c.jumpIf(n.Rhs, label, sense)
		c.place(skip)
	default:
		c.jumpTo(label, c.cond(n, sense).i)
	}
}

// --- statements ---

func (c *compiler) emit(k stmtKind, i int) { c.stmts = append(c.stmts, stmt{k, i}) }

func (c *compiler) newLabel() int { c.labels++; return c.labels - 1 }

func (c *compiler) place(label int) { c.emit(sLabel, label) }

func (c *compiler) jumpTo(label, cond int) {
	j := jump{label, cond}
	i, ok := c.jumpIdx[j]
	if !ok {
		i = len(c.jumps)
		c.jumps = append(c.jumps, j)
		c.jumpIdx[j] = i
	}
	c.emit(sJump, i)
}

// jumpIfFalse jumps to label when n is false.
func (c *compiler) jumpIfFalse(n *cc.Node, label int) { c.jumpIf(n, label, false) }

func (c *compiler) assign(target, source val) {
	c.emit(sAssign, c.assigns.add(entry{source, target, 0}))
}

// assignment lowers "x = e" and, when want is set, returns x's new value.
// For an array element with a computed index, the index is evaluated once:
// the write pointer is set first, and the value side (x op= e and x++ read
// the same element) reads through a read pointer derived from it.
func (c *compiler) assignment(n *cc.Node, want bool) val {
	lhs := n.Lhs
	if lhs.Kind == cc.NdVar || func() bool { _, ok := c.constIndex(lhs); return ok }() {
		target := c.value(lhs)
		v := c.value(n.Rhs)
		if lhs.Var.Char {
			v = c.wrapChar(v, n.Pos)
		}
		c.assign(target, v)
		return target
	}
	base := c.variable(lhs.Var)
	w := c.temporary()
	c.assign(w, c.expr(vSum, c.value(lhs.Lhs), c.address(base, 0), 0))
	var read val
	readPointer := func() val {
		if read.kind != vInd {
			r := c.temporary()
			c.assign(r, c.expr(vSum, w, c.number(1, n.Pos), 0))
			read = val{vInd, r.i}
		}
		return read
	}
	if uses(n.Rhs, lhs) {
		c.reads[lhs] = readPointer()
	}
	v := c.value(n.Rhs)
	delete(c.reads, lhs)
	if lhs.Var.Char {
		v = c.wrapChar(v, n.Pos)
	}
	c.assign(val{vInd, w.i}, v)
	if !want {
		return val{}
	}
	return readPointer()
}

// uses reports whether node x occurs in n (the same node, as the parser
// shares it between the target and value of x op= e and x++).
func uses(n, x *cc.Node) bool {
	if n == nil {
		return false
	}
	if n == x {
		return true
	}
	if uses(n.Lhs, x) || uses(n.Rhs, x) {
		return true
	}
	for _, a := range n.Args {
		if uses(a, x) {
			return true
		}
	}
	return false
}

// effects lowers an expression for its side effects only.
func (c *compiler) effects(n *cc.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case cc.NdFuncall:
		v := c.value(n.Args[0])
		if c.out < 0 {
			c.out = c.cells(1, n.Pos)
		}
		c.assign(val{vVar, c.out}, v)
		c.emit(sWrite, 0)
	case cc.NdAssign:
		c.assignment(n, false)
	case cc.NdIndex:
		c.effects(n.Lhs)
	case cc.NdLogAnd, cc.NdLogOr:
		// the right side runs only when the left does not decide
		skip := c.newLabel()
		c.jumpIf(n.Lhs, skip, n.Kind == cc.NdLogOr)
		c.effects(n.Rhs)
		c.place(skip)
	case cc.NdNum, cc.NdVar:
	case cc.NdNeg, cc.NdNot:
		c.effects(n.Lhs)
	default: // arithmetic and comparisons: only their operands' effects
		c.effects(n.Lhs)
		c.effects(n.Rhs)
	}
}

func (c *compiler) stmt(n *cc.Node, exit *int) {
	switch n.Kind {
	case cc.NdBlock:
		for _, s := range n.Body {
			c.stmt(s, exit)
		}
	case cc.NdExprStmt:
		c.effects(n.Lhs)
	case cc.NdReturn:
		c.effects(n.Lhs)
		if *exit < 0 {
			*exit = c.newLabel()
		}
		c.jumpTo(*exit, -1)
	case cc.NdIf:
		end := c.newLabel()
		if n.Els == nil {
			c.jumpIfFalse(n.Cond, end)
			c.stmt(n.Then, exit)
		} else {
			els := c.newLabel()
			c.jumpIfFalse(n.Cond, els)
			c.stmt(n.Then, exit)
			c.jumpTo(end, -1)
			c.place(els)
			c.stmt(n.Els, exit)
		}
		c.place(end)
	case cc.NdWhile:
		top := c.newLabel()
		l := &loop{brk: -1, cont: top}
		c.place(top)
		c.jumpIfFalse(n.Cond, c.breakLabel(l))
		c.body(l, n.Then, exit)
		c.jumpTo(top, -1)
		c.place(l.brk)
	case cc.NdFor:
		c.stmt(n.Init, exit)
		top := c.newLabel()
		l := &loop{brk: -1, cont: -1}
		c.place(top)
		if n.Cond != nil {
			c.jumpIfFalse(n.Cond, c.breakLabel(l))
		}
		c.body(l, n.Then, exit)
		if l.cont >= 0 {
			c.place(l.cont)
		}
		c.effects(n.Inc)
		c.jumpTo(top, -1)
		if l.brk >= 0 {
			c.place(l.brk)
		}
	case cc.NdBreak:
		c.jumpTo(c.breakLabel(c.loops[len(c.loops)-1]), -1)
	case cc.NdContinue:
		l := c.loops[len(c.loops)-1]
		if l.cont < 0 {
			l.cont = c.newLabel()
		}
		c.jumpTo(l.cont, -1)
	default:
		fail(n.Pos, "cannot lower this statement")
	}
}

// body lowers a loop body with l as the innermost loop.
func (c *compiler) body(l *loop, n *cc.Node, exit *int) {
	c.loops = append(c.loops, l)
	c.stmt(n, exit)
	c.loops = c.loops[:len(c.loops)-1]
}

// breakLabel returns the label after loop l, making it if needed.
func (c *compiler) breakLabel(l *loop) int {
	if l.brk < 0 {
		l.brk = c.newLabel()
	}
	return l.brk
}

// --- tables ---

// program lays the tables out in Code, numbers the cells (declared numbers
// first, then variables) and resolves all references.
//
// Addresses are declared numbers whose value is a cell number, and cell
// numbers start after the declared numbers. So the pool size P is fixed
// first: the constants plus one slot per address. An address whose value
// happens to be a constant already uses that constant's slot, and unused
// numbers fill the pool up to P again, which keeps every cell where the
// addresses say it is.
func (c *compiler) program() *syntax.Program {
	size := len(c.pool) + len(c.addrs)
	pool := append([]int32(nil), c.pool...)
	slots := map[int32]int{}
	for i, v := range pool {
		slots[v] = i
	}
	addrSlot := make([]int, len(c.addrs))
	for j, off := range c.addrs {
		v := int64(size) + int64(off)
		if v > largest {
			fail(cc.Pos{}, "the program needs more memory than Sorted! numbers can address")
		}
		if i, ok := slots[int32(v)]; ok {
			addrSlot[j] = i
			continue
		}
		slots[int32(v)] = len(pool)
		addrSlot[j] = len(pool)
		pool = append(pool, int32(v))
	}
	if cells := int64(size) + int64(c.nvars); cells > memory {
		fail(cc.Pos{Line: 1, Col: 1}, "the program needs %d memory cells; Sorted! has %d", cells, memory)
	}
	for filler := int32(0); len(pool) < size; filler++ {
		if _, used := slots[filler]; !used {
			slots[filler] = len(pool)
			pool = append(pool, filler)
			c.fillers++
		}
	}

	p := &syntax.Program{Data: pool, LabelsCount: c.labels}
	operand := func(v val) syntax.Operand {
		switch v.kind {
		case vConst:
			return syntax.Operand{Type: syntax.Number, Index: int32(v.i)}
		case vVar:
			return syntax.Operand{Type: syntax.Number, Index: int32(size + v.i)}
		case vInd:
			return syntax.Operand{Type: syntax.Number | syntax.Indirect, Index: int32(size + v.i)}
		case vAddr:
			return syntax.Operand{Type: syntax.Number, Index: int32(addrSlot[v.i])}
		}
		types := map[valKind]syntax.OperandType{vSum: syntax.Sum, vDiff: syntax.Diff, vProd: syntax.Prod, vRatio: syntax.Ratio, vCond: syntax.Condition}
		return syntax.Operand{Type: types[v.kind], Index: int32(v.i)}
	}
	put := func(cat syntax.Category, slides []syntax.Slide) {
		p.Tables[cat] = syntax.Table{Count: len(slides), Index: len(p.Code)}
		p.Code = append(p.Code, slides...)
	}
	for _, k := range []struct {
		kind valKind
		cat  syntax.Category
	}{{vSum, syntax.Sums}, {vDiff, syntax.Diffs}, {vProd, syntax.Prods}, {vRatio, syntax.Ratios}, {vCond, syntax.Conditions}} {
		var slides []syntax.Slide
		for _, e := range c.exprs[k.kind].entries {
			slides = append(slides, syntax.Slide{Ops: [2]syntax.Operand{operand(e.a), operand(e.b)}, Flags: e.flags})
		}
		put(k.cat, slides)
	}
	var assigns []syntax.Slide
	for _, e := range c.assigns.entries {
		assigns = append(assigns, syntax.Slide{Ops: [2]syntax.Operand{operand(e.a), operand(e.b)}})
	}
	put(syntax.Assigns, assigns)
	if c.out >= 0 {
		put(syntax.Writes, []syntax.Slide{{Ops: [2]syntax.Operand{operand(val{vVar, c.out})}, Flags: syntax.FormatCharacter}})
	} else {
		put(syntax.Writes, nil)
	}
	var jumps []syntax.Slide
	for _, j := range c.jumps {
		s := syntax.Slide{Ops: [2]syntax.Operand{{Type: syntax.Label, Index: int32(j.label)}}, Flags: syntax.UnconditionalJump}
		if j.cond >= 0 {
			s.Ops[1] = syntax.Operand{Type: syntax.Condition, Index: int32(j.cond)}
			s.Flags = syntax.ConditionalJump
		}
		jumps = append(jumps, s)
	}
	put(syntax.Jumps, jumps)
	var stmts []syntax.Slide
	for _, s := range c.stmts {
		types := map[stmtKind]syntax.OperandType{sAssign: syntax.Assign, sWrite: syntax.Write, sJump: syntax.Jump, sLabel: syntax.Label}
		stmts = append(stmts, syntax.Slide{Ops: [2]syntax.Operand{{Type: types[s.kind], Index: int32(s.i)}}})
	}
	put(syntax.Statements, stmts)
	p.TypeCount = len(p.Code)
	return p
}
