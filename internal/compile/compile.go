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
//   - switch is a chain of "go to case k if the value is k" jumps, then one
//     to default or past the end; case labels are ordinary labels, so
//     fall-through (and Duff's device) works. do/while jumps back on its
//     test. As a value, c ? a : b sets a temporary on each branch, so only
//     one side runs.
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
//     first sum") is undefined behaviour in the original, so it is avoided,
//     except in a program that is Very Sorted! anyway (it reads, or has a
//     NAND), where it is defined with the same off-by-one and replaces the
//     pointer cells (see indexed).
//     Cell numbers depend on how many numbers are declared, and the
//     addresses are declared numbers themselves; see program.
//   - Every scalar is one cell, and a struct or an array the cells of its
//     members or elements, so a pointer is a cell number and pointer
//     arithmetic counts cells (an index or a step times the size of what is
//     pointed to): &x is an address constant, *p = v writes through p itself
//     (a write pointer), and *p reads through p + 1. A member is an offset
//     from its struct; assigning a struct copies it cell by cell.
//     A function on a cycle of calls cannot take the address of its own
//     locals, which a recursive call would save and restore under the
//     pointer's feet.
//   - Functions exist once. A call stores the arguments in the parameter
//     cells and its number in a return-address cell and jumps; the return
//     jumps through a chain of conditional jumps back to the call site.
//     Functions on a cycle of calls save their frame on a stack in the free
//     memory around each call that can come back to them (see function).
//   - Bitwise operators are arithmetic where they can be: constants fold,
//     ~x is -1 - x, shifts by a constant are products or floored ratios, x &
//     (2^k - 1) is a remainder, and shifts by a variable count call runtime
//     functions written in the C subset (see runtime.go). The other & | ^
//     are Very Sorted! NANDs (two, three and four of them), which make the
//     program Very Sorted!: the original's logical operation is a NOR that
//     cannot be referenced. Numbers beyond what a declaration
//     spells (999999999) are 1000000 * q + r.
//   - A store into a char wraps the value to -128..127, as C does, by
//     arithmetic: ((v + 128) % 256 + 256) % 256 - 128; an unsigned char
//     wraps to 0..255.
//   - unsigned int is the same 32 bits, so + - * and the bit operations
//     are unchanged. A comparison flips both sign bits (adds -2^31) and
//     compares signed; division, remainder and right shifts by a variable
//     count call udiv, umod and ushr (runtime.go); a right shift by a
//     constant keeps the low bits of the arithmetic shift.
//   - putchar(e) assigns e to an output cell and runs the program's only
//     output, which writes that cell as a character. getchar() runs the
//     program's only input, which reads into an input cell, and copies it;
//     a program that reads is Very Sorted! (the original's cannot name an
//     input in a statement).
//   - return in main jumps past the last statement.
//
// Assignments inside expressions are hoisted into statements before the
// statement that contains them. putchar is a statement: its result cannot be
// used.
package compile

import (
	"cmp"
	"fmt"
	"maps"
	"math"
	"slices"

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
	vNand // a Very Sorted! NAND
	vInd  // the cell a pointer cell (variable i) indexes
	vAddr // an address: a declared number, the cell of variable i (see program)
	vIdx  // Very Sorted!: the cell the value c.idxs[i] indexes (see indexed)
)

// exprKinds are the kinds that are table entries (c.exprs).
var exprKinds = []valKind{vSum, vDiff, vProd, vRatio, vCond, vNand}

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
	sRead
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
	sw        bool // a switch: break applies, continue goes to the loop around it
}

// compiler holds the tables being built.
type compiler struct {
	pool      []int32       // declared numbers
	poolIndex map[int32]int // value → pool index
	vars      map[*cc.Obj]int
	nvars     int
	out       int   // the output cell's variable index, -1 until needed
	in        int   // the input cell's variable index, -1 until a getchar needs it
	very      bool  // the program is Very Sorted! from the start (see compileProgram)
	verys     int   // its dialect at least, counted in verys
	idxs      []val // the values vIdx operands index

	addrs     []int            // address numbers, as variable-cell offsets (see program)
	fillers   int              // numbers program declared only to keep the pool size
	addrIndex map[int]int      // offset → index in addrs
	reads     map[*cc.Node]val // array elements whose read pointer is already set
	cases     map[*cc.Node]int // the labels of the switches' case and default

	exprs   map[valKind]*table // sums, differences, products, ratios, conditions
	assigns table
	jumps   []jump
	jumpIdx map[jump]int
	labels  int
	stmts   []stmt
	loops   []*loop

	ret   *returnTo                  // where return goes in the code being lowered
	exit  int                        // label after everything, where main's return goes; -1 until needed
	funcs map[*cc.Function]*function // the functions main reaches
	order []*function                // the same, in the order their code is laid out
	rt    map[string]*cc.Function    // the runtime library, parsed on first use (see runtime)

	// recursion (see recursiveCall)
	cur       *function // the function being lowered, nil in main
	mark      int       // first cell allocated for the expression being lowered
	sp, rp    val       // stack pointer, and a read pointer for pops
	stack     bool      // a function recurses, so there is a stack
	stackAddr int       // index in addrs of the stack's first cell
}

// returnTo says what a return does: store the value in rv (when the
// function has one) and jump to label.
type returnTo struct {
	label int // -1 until needed (main only)
	rv    val
	hasRV bool
	ty    *cc.Type // the result type, which a char result wraps to
}

// function is a function main reaches. Its code exists once. A call stores
// the arguments in the parameter cells, the call site's number in ra, and
// jumps to entry; a return jumps to the dispatch, a chain of jumps that
// goes back to the call site whose number ra holds. A function called from
// one place only needs neither ra nor the chain: it returns straight there.
//
// A function on a cycle of calls (recursion, direct or through others) can
// be entered again while it runs, so before each call that may come back to
// it, it pushes its frame onto a stack and pops it afterwards.
type function struct {
	decl     *cc.Function
	entry    int   // label of the first statement
	dispatch int   // label returns jump to: the dispatch chain, or the one return label
	ra, rv   val   // return-address cell (several call sites) and result cell
	nsites   int   // call sites in reachable code
	sites    []int // return label of each call site, by number

	callees []*function        // functions it calls
	reach   map[*function]bool // functions its calls lead to, directly or not
	frame   []int              // on a cycle: ra, parameters and locals, which a call saves
}

// Compile lowers a parsed C program into Sorted! tables, in the dialect
// with verys verys or a newer one: 0 for whatever the program needs, 2 for
// a program to be written in Italian (render.Lang.Verys), which is Very
// Very Sorted!.
func Compile(prog *cc.Program, verys int) (*syntax.Program, error) {
	_, p, err := compileProgram(prog, verys)
	return p, err
}

// compileProgram is Compile, also returning the compiler for inspection.
// A program that reads or has a NAND is Very Sorted!, and one that is very
// anyway also indexes computed values directly (see indexed); one that is
// not keeps to what Sorted.exe runs (Gerson, 2026-10-07). Whether it is
// shows in what the lowering emits, so such a program is compiled a second
// time, as Very Sorted! from the start. With verys set, the program is Very
// Sorted! (or Very Very Sorted!) even when it need not be (the tests run
// every program that way too).
func compileProgram(prog *cc.Program, verys int) (*compiler, *syntax.Program, error) {
	c, p, err := compileAs(prog, verys)
	if err == nil && !c.very && p.Verys > 0 {
		return compileAs(prog, p.Verys)
	}
	return c, p, err
}

// compileAs compiles prog, as Very Sorted! (or newer) from the start when
// verys is set. The first compilation's rewrites of the tree (see bitwise)
// are done again harmlessly: what they rewrote is no longer an operator
// they rewrite.
func compileAs(prog *cc.Program, verys int) (c *compiler, p *syntax.Program, err error) {
	c = &compiler{
		poolIndex: map[int32]int{},
		vars:      map[*cc.Obj]int{},
		out:       -1,
		in:        -1,
		exprs:     map[valKind]*table{},
		jumpIdx:   map[jump]int{},
		addrIndex: map[int]int{},
		reads:     map[*cc.Node]val{},
		cases:     map[*cc.Node]int{},
		funcs:     map[*cc.Function]*function{},
		exit:      -1,
		very:      verys > 0,
		verys:     verys,
	}
	for _, k := range exprKinds {
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
			if g.InitRef != nil && g.InitRef[i] != nil { // an address
				c.assign(val{vVar, base.i + i}, c.address(c.variable(g.InitRef[i]), int(v)))
				continue
			}
			if v != 0 { // chars came wrapped from the parser
				c.assign(val{vVar, base.i + i}, c.number(v, g.Pos))
			}
		}
	}
	c.bitwise(prog.Main)
	for _, f := range slices.SortedFunc(maps.Values(prog.Funcs), func(a, b *cc.Function) int {
		return cmp.Or(cmp.Compare(a.Pos.Line, b.Pos.Line), cmp.Compare(a.Pos.Col, b.Pos.Col))
	}) {
		c.bitwise(f.Body)
	}
	c.functions(prog)
	if c.stack {
		c.assign(c.sp, val{vAddr, c.stackAddr})
	}

	// main first; a final return just ends the program, unless functions
	// follow, which main must jump over.
	c.ret = &returnTo{label: -1}
	for i, s := range prog.Main.Body {
		if s.Kind == cc.NdReturn && i == len(prog.Main.Body)-1 && len(c.order) == 0 {
			c.effects(s.Lhs)
			continue
		}
		c.stmt(s)
	}
	if len(c.order) > 0 {
		c.jumpTo(c.exitLabel(), -1)
	}
	for _, f := range c.order {
		c.functionBody(f)
	}
	for _, f := range c.order {
		c.dispatch(f)
	}
	if c.exit >= 0 {
		c.place(c.exit)
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
// values are 0 - n (and -2^31 is -2147483647 - 1); values beyond what a
// declaration can spell are 1000000 * q + r.
func (c *compiler) number(v int32, pos cc.Pos) val {
	switch {
	case v == math.MinInt32:
		return c.expr(vDiff, c.number(v+1, pos), c.number(1, pos), 0)
	case v < 0:
		return c.expr(vDiff, c.number(0, pos), c.number(-v, pos), 0)
	case v > largest:
		e := c.expr(vProd, c.number(1000000, pos), c.number(v/1000000, pos), 0)
		if v%1000000 == 0 {
			return e
		}
		return c.expr(vSum, e, c.number(v%1000000, pos), 0)
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
		i = c.cells(o.Ty.Size(), o.Pos)
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

// constValue reports the value of a constant: a declared number, or one
// that number builds from them (0 - n, 1000000 * q + r).
func (c *compiler) constValue(v val) (int32, bool) {
	switch v.kind {
	case vConst:
		return c.pool[v.i], true
	case vDiff, vSum, vProd:
		e := c.exprs[v.kind].entries[v.i]
		a, okA := c.constValue(e.a)
		b, okB := c.constValue(e.b)
		switch v.kind {
		case vSum:
			return a + b, okA && okB
		case vProd:
			return a * b, okA && okB
		}
		return a - b, okA && okB
	}
	return 0, false
}

// mod is a % b with C's sign rule.
func (c *compiler) mod(a, b val) val {
	return c.expr(vDiff, a, c.expr(vProd, c.expr(vRatio, a, b, 0), b, 0), 0)
}

// wrapTo converts v as a store into a t does: a char wraps to -128..127,
// an unsigned char to 0..255; anything else keeps its 32 bits.
func (c *compiler) wrapTo(t *cc.Type, v val, pos cc.Pos) val {
	switch {
	case t == nil || t.Kind != cc.TyChar:
		return v
	case !t.Unsigned:
		return c.wrapChar(v, pos)
	}
	if k, ok := c.constValue(v); ok {
		return c.number(int32(uint8(k)), pos)
	}
	p := c.number(256, pos)
	return c.mod(c.expr(vSum, c.mod(v, p), p, 0), p)
}

// less is a < b for the comparison n, unsigned when an operand is unsigned
// int: then both have their sign bit flipped (-2^31 added), which orders
// them as unsigned.
func (c *compiler) less(n *cc.Node, a, b val) val {
	if n.Lhs.Ty.IsUnsignedInt() || n.Rhs.Ty.IsUnsignedInt() {
		flip := c.number(math.MinInt32, n.Pos)
		a, b = c.expr(vSum, a, flip, 0), c.expr(vSum, b, flip, 0)
	}
	return c.lt(a, b)
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

// indexed returns the cell v indexes: a read takes cell v - 1, a write cell
// v (the indirect off-by-one). A Very Sorted! program indexes v itself; the
// original can only index a cell, so v is stored in a pointer cell first.
// So is a v that is itself an indexed cell (**pp): an operand indexes once.
func (c *compiler) indexed(v val) val {
	if v.kind == vVar {
		return val{vInd, v.i}
	}
	if c.very && v.kind != vInd && v.kind != vIdx {
		c.idxs = append(c.idxs, v)
		return val{vIdx, len(c.idxs) - 1}
	}
	p := c.temporary()
	c.assign(p, v)
	return val{vInd, p.i}
}

// element returns array element a[i] for reading: a cell for a constant
// index, otherwise the cell a read pointer indexes.
func (c *compiler) element(n *cc.Node) val {
	if v, ok := c.reads[n]; ok {
		return v
	}
	// Only scalars are read, and a scalar element is one cell; elements of
	// arrays of structs are reached through direct and location.
	base := c.variable(n.Var)
	if k, ok := c.constIndex(n); ok {
		return val{vVar, base.i + k}
	}
	return c.indexed(c.expr(vSum, c.value(n.Lhs), c.address(base, 1), 0))
}

// isPtr reports whether n is a pointer. Nodes the compiler makes itself (the
// constant arguments of runtime calls) have no type: they are integers.
func isPtr(n *cc.Node) bool { return n.Ty != nil && n.Ty.Kind == cc.TyPtr }

// scaled is v times size: an index or a pointer step counted in cells.
func (c *compiler) scaled(v val, size int, pos cc.Pos) val {
	if size == 1 {
		return v
	}
	return c.expr(vProd, v, c.number(int32(size), pos), 0)
}

// plus is v + k, for a constant k.
func (c *compiler) plus(v val, k int, pos cc.Pos) val {
	if k == 0 {
		return v
	}
	return c.expr(vSum, v, c.number(int32(k), pos), 0)
}

// memberValue reads s.m: the cell itself when it is known while compiling,
// otherwise through a read pointer (its address plus one).
func (c *compiler) memberValue(n *cc.Node) val {
	if v, ok := c.reads[n]; ok {
		return v
	}
	if cell, ok := c.direct(n); ok {
		return cell
	}
	return c.indexed(c.plus(c.location(n), 1, n.Pos))
}

// rootVar returns the variable an lvalue is part of (a.b[2].c is part of
// a), or nil when it is reached through a pointer.
func rootVar(x *cc.Node) *cc.Obj {
	switch x.Kind {
	case cc.NdVar, cc.NdIndex:
		return x.Var
	case cc.NdMember:
		return rootVar(x.Lhs)
	case cc.NdDeref:
		if x.Lhs.Kind == cc.NdAddr {
			return rootVar(x.Lhs.Lhs)
		}
	}
	return nil
}

// addressOf lowers &x: the cell number of a variable or an element (an
// address constant), or of what a pointer points to. An array used as a
// value is the address of its first element.
func (c *compiler) addressOf(n *cc.Node) val {
	x := n.Lhs
	if v := rootVar(x); v != nil && !v.IsGlobal && c.cur != nil && c.cur.reach[c.cur] {
		// A recursive call saves and restores the caller's locals, so a write
		// through a pointer to one of them would be undone (see save).
		fail(n.Pos, "taking the address of '%s' in the recursive function '%s' is not supported yet (make it a global)", v.Name, c.cur.decl.Name)
	}
	if cell, ok := c.directAt(x, true); ok { // &a[len] is the end
		return c.address(cell, 0)
	}
	return c.location(x) // &*p is p
}

// deref lowers *p for reading: the cell itself when p is the address of a
// variable or an element, otherwise the cell a read pointer indexes (p + 1,
// the indirect read's off-by-one).
func (c *compiler) deref(n *cc.Node) val {
	if v, ok := c.reads[n]; ok {
		return v
	}
	if n.Lhs.Kind == cc.NdAddr && n.Lhs.Lhs.Kind != cc.NdDeref {
		return c.value(n.Lhs.Lhs) // *&x, and *a for an array a
	}
	if cell, ok := c.direct(n); ok { // a[k] on a member or a row
		return cell
	}
	return c.indexed(c.expr(vSum, c.value(n.Lhs), c.number(1, n.Pos), 0))
}

// constIndex reports a constant index of a[i], checking its bounds.
func (c *compiler) constIndex(n *cc.Node) (int, bool) { return c.constOffset(n, n.Var.Len-1) }

// constOffset reports a constant index of a[i] up to last. An element must
// exist; an address (&a[i]) may also be one past the end, as in C.
func (c *compiler) constOffset(n *cc.Node, last int) (int, bool) {
	if n.Lhs.Kind != cc.NdNum && !(n.Lhs.Kind == cc.NdNeg && n.Lhs.Lhs.Kind == cc.NdNum) {
		return 0, false
	}
	k := int(n.Lhs.Val)
	if n.Lhs.Kind == cc.NdNeg {
		k = -int(n.Lhs.Lhs.Val)
	}
	if k < 0 || k > last {
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
	case cc.NdAddr:
		return c.addressOf(n)
	case cc.NdMember:
		return c.memberValue(n)
	case cc.NdDeref:
		return c.deref(n)
	case cc.NdNeg:
		if n.Lhs.Kind == cc.NdNum {
			return c.number(-n.Lhs.Val, n.Pos)
		}
		return c.expr(vDiff, c.number(0, n.Pos), c.value(n.Lhs), 0)
	case cc.NdAssign:
		return c.assignment(n, true)
	case cc.NdFuncall:
		if n.Func == "getchar" {
			return c.getchar(n.Pos)
		}
		if n.Fn == nil {
			fail(n.Pos, "using the result of putchar is not supported yet")
		}
		if n.Fn.Void {
			fail(n.Pos, "'%s' returns nothing (void)", n.Fn.Name)
		}
		return c.call(n)
	case cc.NdNot:
		return c.not(c.value(n.Lhs), n.Pos)
	case cc.NdBitNot, cc.NdBitAnd, cc.NdBitOr, cc.NdBitXor, cc.NdShl, cc.NdShr:
		return c.bitValue(n)
	case cc.NdCond:
		// t = a; or t = b;  so that only one side runs
		t := c.temporary()
		els, end := c.newLabel(), c.newLabel()
		c.jumpIf(n.Cond, els, false)
		c.assign(t, c.value(n.Then))
		c.jumpTo(end, -1)
		c.place(els)
		c.assign(t, c.value(n.Els))
		c.place(end)
		return t
	case cc.NdComma:
		c.effects(n.Lhs)
		return c.value(n.Rhs)
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
	if n.Wrap != nil {
		w := *n
		w.Wrap = nil
		return c.wrapTo(n.Wrap, c.value(&w), n.Pos)
	}
	a, b := c.operands(n.Lhs, n.Rhs)
	if n.Kind == cc.NdAdd || n.Kind == cc.NdSub {
		// pointer arithmetic counts elements, which may span several cells
		switch {
		case isPtr(n.Lhs) && isPtr(n.Rhs):
			if size := n.Lhs.Ty.Base.Size(); size > 1 {
				return c.expr(vRatio, c.expr(vDiff, a, b, 0), c.number(int32(size), n.Pos), 0)
			}
		case isPtr(n.Lhs):
			b = c.scaled(b, n.Lhs.Ty.Base.Size(), n.Pos)
		case isPtr(n.Rhs):
			a = c.scaled(a, n.Rhs.Ty.Base.Size(), n.Pos)
		}
	}
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
		return c.less(n, a, b)
	case cc.NdLe:
		return c.expr(vDiff, c.number(1, n.Pos), c.less(n, b, a), 0)
	}
	fail(n.Pos, "cannot lower this expression")
	return val{}
}

// operands lowers the two operands of a binary operator, left first. Values
// are lazy, so when the right side calls a function, which may change the
// cells the left value reads (and does in "(g = 65) + reset()"), the left
// value is settled into a temporary before the call.
func (c *compiler) operands(l, r *cc.Node) (val, val) {
	a := c.value(l)
	if hasCall(r) {
		a = c.settle(a)
	}
	return a, c.value(r)
}

// settle returns v's current value in a temporary, or v if it is constant.
func (c *compiler) settle(v val) val {
	if _, ok := c.constValue(v); ok || v.kind == vAddr {
		return v
	}
	t := c.temporary()
	c.assign(t, v)
	return t
}

// hasCall reports whether n calls a user function.
func hasCall(n *cc.Node) bool {
	if n == nil {
		return false
	}
	if n.Kind == cc.NdFuncall && n.Fn != nil {
		return true
	}
	if hasCall(n.Lhs) || hasCall(n.Rhs) || hasCall(n.Cond) || hasCall(n.Then) || hasCall(n.Els) {
		return true
	}
	for _, a := range n.Args {
		if hasCall(a) {
			return true
		}
	}
	return false
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
		return c.eq(c.operands(n.Lhs, n.Rhs))
	case n.Kind == cc.NdLt && sense:
		a, b := c.operands(n.Lhs, n.Rhs)
		return c.less(n, a, b)
	case n.Kind == cc.NdLe && !sense:
		a, b := c.operands(n.Lhs, n.Rhs)
		return c.less(n, b, a)
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
// jumpIfFalse lowers the test of an if or a loop, a full expression.
func (c *compiler) jumpIfFalse(n *cc.Node, label int) {
	c.mark = c.nvars
	c.jumpIf(n, label, false)
}

func (c *compiler) assign(target, source val) {
	c.emit(sAssign, c.assigns.add(entry{source, target, 0}))
}

// assignment lowers "x = e" and, when want is set, returns x's new value.
// For an element with a computed index, or *p, the location is evaluated
// once: the write pointer is set first, and the value side (x op= e and x++
// read the same cell) reads through a read pointer derived from it. A store
// into a char wraps.
func (c *compiler) assignment(n *cc.Node, want bool) val {
	lhs := n.Lhs
	if lhs.Ty.Kind == cc.TyStruct {
		c.copyStruct(n)
		return val{}
	}
	if target, ok := c.direct(lhs); ok {
		v := c.value(n.Rhs)
		v = c.wrapTo(lhs.Ty, v, n.Pos)
		c.assign(target, v)
		return target
	}
	// The location, in a write pointer unless nothing can change what it
	// depends on before it is used (Very Sorted! indexes it directly): the
	// value side, or, when the result is wanted, the store itself
	// (a[a[0]] = 1 reads back a[0]'s old index).
	loc := c.location(lhs)
	if !c.very || want || sideEffects(n.Rhs) {
		w := c.temporary()
		c.assign(w, loc)
		loc = w
	}
	var read val
	readPointer := func() val {
		if read == (val{}) {
			read = c.indexed(c.plus(loc, 1, n.Pos))
		}
		return read
	}
	if uses(n.Rhs, lhs) {
		c.reads[lhs] = readPointer()
	}
	v := c.value(n.Rhs)
	delete(c.reads, lhs)
	v = c.wrapTo(lhs.Ty, v, n.Pos)
	c.assign(c.indexed(loc), v)
	if !want {
		return val{}
	}
	return readPointer()
}

// sideEffects reports whether evaluating n assigns or calls (and so may
// change a cell an expression reads).
func sideEffects(n *cc.Node) bool {
	if n == nil {
		return false
	}
	if n.Kind == cc.NdAssign || n.Kind == cc.NdFuncall {
		return true
	}
	for _, m := range []*cc.Node{n.Lhs, n.Rhs, n.Cond, n.Then, n.Els} {
		if sideEffects(m) {
			return true
		}
	}
	for _, m := range n.Args {
		if sideEffects(m) {
			return true
		}
	}
	return false
}

// direct returns the cell an lvalue is, when it is known while compiling: a
// variable, an element with a constant index, *&x.
func (c *compiler) direct(x *cc.Node) (val, bool) { return c.directAt(x, false) }

// directAt is direct; with end set, the outermost index may also be one
// past the last element, which an address may name (&a[len]) but an access
// may not.
func (c *compiler) directAt(x *cc.Node, end bool) (val, bool) {
	last := func(n int) int {
		if end {
			return n
		}
		return n - 1
	}
	switch x.Kind {
	case cc.NdVar:
		return c.variable(x.Var), true
	case cc.NdIndex:
		if k, ok := c.constOffset(x, last(x.Var.Len)); ok {
			return val{vVar, c.variable(x.Var).i + k*x.Var.Ty.Base.Size()}, true
		}
	case cc.NdMember:
		if s, ok := c.direct(x.Lhs); ok {
			return val{vVar, s.i + x.Member.Offset}, true
		}
	case cc.NdDeref:
		if x.Lhs.Kind == cc.NdAddr {
			return c.direct(x.Lhs.Lhs)
		}
		// a[k] on an array that is not a variable (a member, a row of an
		// array of arrays) is *(&a + k): with a constant k, a cell
		if add := x.Lhs; add.Kind == cc.NdAdd && add.Lhs.Kind == cc.NdAddr {
			k, ok := cc.Fold(add.Rhs)
			base, okB := c.direct(add.Lhs.Lhs)
			if ok && okB {
				if arr := add.Lhs.Lhs.Ty; arr != nil && arr.Kind == cc.TyArray && (k < 0 || int(k) > last(arr.Len)) {
					fail(add.Rhs.Pos, "index %d is out of range (%d elements)", k, arr.Len)
				}
				return val{vVar, base.i + int(k)*x.Ty.Size()}, true
			}
		}
	}
	return val{}, false
}

// location returns the cell number of an lvalue, which is what a write
// pointer holds: an address constant when direct knows the cell, otherwise
// computed (a[i] with a computed index, *p, members of those).
func (c *compiler) location(x *cc.Node) val {
	if cell, ok := c.direct(x); ok {
		return c.address(cell, 0)
	}
	switch x.Kind {
	case cc.NdIndex:
		return c.expr(vSum, c.scaled(c.value(x.Lhs), x.Var.Ty.Base.Size(), x.Pos), c.address(c.variable(x.Var), 0), 0)
	case cc.NdMember:
		return c.plus(c.location(x.Lhs), x.Member.Offset, x.Pos)
	case cc.NdDeref:
		if x.Lhs.Kind == cc.NdAddr { // *&a[i]
			return c.location(x.Lhs.Lhs)
		}
		return c.value(x.Lhs)
	}
	fail(x.Pos, "this struct value is not supported yet (only variables, elements, members and *p)")
	return val{}
}

// copyStruct lowers s = t for structs, cell by cell, and returns where s
// is. In a = b = c, the inner copy's destination is the outer one's source,
// at the location already evaluated (b[i++] advances i once).
func (c *compiler) copyStruct(n *cc.Node) structAt {
	var from structAt
	if n.Rhs.Kind == cc.NdAssign {
		from = c.copyStruct(n.Rhs)
	}
	dst := c.structAt(n.Lhs)
	if n.Rhs.Kind != cc.NdAssign {
		from = c.structAt(n.Rhs)
	}
	for i := range n.Lhs.Ty.Size() {
		c.assign(c.cellOf(dst, i, false), c.cellOf(from, i, true))
	}
	return dst
}

// structAt is where a struct is: a known cell (direct), or a cell holding
// its computed location (a write pointer to its first cell).
type structAt struct {
	direct bool
	cell   int // the first cell, or the cell holding the location
	pos    cc.Pos
}

// structAt evaluates where struct lvalue x is, once.
func (c *compiler) structAt(x *cc.Node) structAt {
	if cell, ok := c.direct(x); ok {
		return structAt{true, cell.i, x.Pos}
	}
	w := c.temporary()
	c.assign(w, c.location(x))
	return structAt{false, w.i, x.Pos}
}

// cellOf returns cell i of a struct, for writing or for reading (a read
// pointer is one more than the cell).
func (c *compiler) cellOf(s structAt, i int, read bool) val {
	if s.direct {
		return val{vVar, s.cell + i}
	}
	if read {
		i++
	}
	return c.indexed(c.plus(val{vVar, s.cell}, i, s.pos))
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

// getchar reads a character (or -1 at the end of the input) through the
// program's only input, into the input cell, and returns a copy, since the
// next read overwrites the cell. Reading makes the program Very Sorted!: in
// the original's Sorted!, no statement can refer to an input.
func (c *compiler) getchar(pos cc.Pos) val {
	if c.in < 0 {
		c.in = c.cells(1, pos)
	}
	c.emit(sRead, 0)
	t := c.temporary()
	c.assign(t, val{vVar, c.in})
	return t
}

// effects lowers an expression for its side effects only.
func (c *compiler) effects(n *cc.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case cc.NdFuncall:
		if n.Fn != nil {
			c.call(n)
			return
		}
		if n.Func == "getchar" {
			c.getchar(n.Pos) // read, and forget
			return
		}
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
	case cc.NdCond:
		els, end := c.newLabel(), c.newLabel()
		c.jumpIf(n.Cond, els, false)
		c.effects(n.Then)
		c.jumpTo(end, -1)
		c.place(els)
		c.effects(n.Els)
		c.place(end)
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

func (c *compiler) stmt(n *cc.Node) {
	switch n.Kind {
	case cc.NdBlock:
		for _, s := range n.Body {
			c.stmt(s)
		}
	case cc.NdExprStmt:
		c.mark = c.nvars
		c.effects(n.Lhs)
	case cc.NdReturn:
		c.mark = c.nvars
		c.returning(n)
	case cc.NdIf:
		end := c.newLabel()
		if n.Els == nil {
			c.jumpIfFalse(n.Cond, end)
			c.stmt(n.Then)
		} else {
			els := c.newLabel()
			c.jumpIfFalse(n.Cond, els)
			c.stmt(n.Then)
			c.jumpTo(end, -1)
			c.place(els)
			c.stmt(n.Els)
		}
		c.place(end)
	case cc.NdWhile:
		top := c.newLabel()
		l := &loop{brk: -1, cont: top}
		c.place(top)
		c.jumpIfFalse(n.Cond, c.breakLabel(l))
		c.body(l, n.Then)
		c.jumpTo(top, -1)
		c.place(l.brk)
	case cc.NdFor:
		c.stmt(n.Init)
		top := c.newLabel()
		l := &loop{brk: -1, cont: -1}
		c.place(top)
		if n.Cond != nil {
			c.jumpIfFalse(n.Cond, c.breakLabel(l))
		}
		c.body(l, n.Then)
		if l.cont >= 0 {
			c.place(l.cont)
		}
		c.mark = c.nvars
		c.effects(n.Inc)
		c.jumpTo(top, -1)
		if l.brk >= 0 {
			c.place(l.brk)
		}
	case cc.NdDo:
		top := c.newLabel()
		l := &loop{brk: -1, cont: -1}
		c.place(top)
		c.body(l, n.Then)
		if l.cont >= 0 {
			c.place(l.cont)
		}
		c.mark = c.nvars
		c.jumpIf(n.Cond, top, true)
		if l.brk >= 0 {
			c.place(l.brk)
		}
	case cc.NdSwitch:
		// "go to case k if the value is k", then to default or past the end.
		// Nothing changes between the tests, so the value needs no temporary.
		c.mark = c.nvars
		v := c.value(n.Cond)
		l := &loop{brk: -1, cont: -1, sw: true}
		for _, k := range n.Cases {
			c.cases[k] = c.newLabel()
			c.jumpTo(c.cases[k], c.eq(v, c.number(k.Val, k.Pos)).i)
		}
		if n.Default != nil {
			c.cases[n.Default] = c.newLabel()
			c.jumpTo(c.cases[n.Default], -1)
		} else {
			c.jumpTo(c.breakLabel(l), -1)
		}
		c.body(l, n.Then)
		if l.brk >= 0 {
			c.place(l.brk)
		}
	case cc.NdCase:
		c.place(c.cases[n])
		c.stmt(n.Then)
	case cc.NdBreak:
		c.jumpTo(c.breakLabel(c.loops[len(c.loops)-1]), -1)
	case cc.NdContinue:
		l := c.loops[len(c.loops)-1]
		for i := len(c.loops) - 1; l.sw; i-- { // the loop around the switches
			l = c.loops[i-1]
		}
		if l.cont < 0 {
			l.cont = c.newLabel()
		}
		c.jumpTo(l.cont, -1)
	default:
		fail(n.Pos, "cannot lower this statement")
	}
}

// --- functions ---

// functions finds the functions main reaches, in the order of first call,
// counts their call sites, finds the ones on cycles of calls, and gives each
// its labels and cells.
func (c *compiler) functions(prog *cc.Program) {
	var visit func(from *function, body *cc.Node)
	visit = func(from *function, body *cc.Node) {
		walk(body, func(call *cc.Node) {
			fn := call.Fn
			if !fn.Defined {
				fail(call.Pos, "'%s' is declared but never defined", fn.Name)
			}
			f, seen := c.funcs[fn]
			if !seen {
				f = &function{decl: fn}
				c.funcs[fn] = f
				c.order = append(c.order, f)
				visit(f, fn.Body)
			}
			if from != nil && !slices.Contains(from.callees, f) {
				from.callees = append(from.callees, f)
			}
			f.nsites++
		})
	}
	visit(nil, prog.Main)
	for _, f := range c.order {
		f.reach = map[*function]bool{}
		var follow func(g *function)
		follow = func(g *function) {
			for _, h := range g.callees {
				if !f.reach[h] {
					f.reach[h] = true
					follow(h)
				}
			}
		}
		follow(f)
	}
	for _, f := range c.order {
		f.entry, f.dispatch = c.newLabel(), c.newLabel()
		if f.nsites > 1 {
			f.ra = c.temporary()
		}
		if !f.decl.Void {
			f.rv = c.temporary()
		}
		if !f.reach[f] {
			continue
		}
		if !c.stack {
			c.stack = true
			c.sp, c.rp = c.temporary(), c.temporary()
			c.stackAddr = len(c.addrs)
			c.addrs = append(c.addrs, 0) // the first free cell, fixed in program
		}
		if f.nsites > 1 {
			f.frame = append(f.frame, f.ra.i)
		}
		seen := map[*cc.Obj]bool{}
		add := func(o *cc.Obj) {
			if !o.IsGlobal && !seen[o] {
				seen[o] = true
				v := c.variable(o)
				for i := range o.Ty.Size() {
					f.frame = append(f.frame, v.i+i)
				}
			}
		}
		for _, p := range f.decl.Params {
			add(p)
		}
		objects(f.decl.Body, add)
	}
}

// objects calls add for every variable n uses.
func objects(n *cc.Node, add func(*cc.Obj)) {
	if n == nil {
		return
	}
	if n.Var != nil {
		add(n.Var)
	}
	for _, m := range []*cc.Node{n.Lhs, n.Rhs, n.Cond, n.Then, n.Els, n.Init, n.Inc} {
		objects(m, add)
	}
	for _, m := range n.Body {
		objects(m, add)
	}
	for _, m := range n.Args {
		objects(m, add)
	}
}

// walk calls visit for every call of a user function in n, in source order.
func walk(n *cc.Node, visit func(*cc.Node)) {
	if n == nil {
		return
	}
	for _, m := range []*cc.Node{n.Lhs, n.Rhs, n.Cond, n.Then, n.Els, n.Init, n.Inc} {
		walk(m, visit)
	}
	for _, m := range n.Body {
		walk(m, visit)
	}
	for _, m := range n.Args {
		walk(m, visit)
	}
	if n.Kind == cc.NdFuncall && n.Fn != nil {
		visit(n)
	}
}

// call lowers a call of a user function and returns its result (a copy, so
// that a later call cannot overwrite it before it is used). The arguments
// are evaluated first, all of them, since one may call the same function;
// each is settled when a later one calls a function (see operands).
func (c *compiler) call(n *cc.Node) val {
	f := c.funcs[n.Fn]
	recursive := c.cur != nil && f.reach[c.cur]
	args := make([]val, len(n.Args))
	for i, a := range n.Args {
		args[i] = c.value(a)
		if recursive || slices.ContainsFunc(n.Args[i+1:], hasCall) {
			args[i] = c.settle(args[i])
		}
	}
	var saved []int
	if recursive {
		saved = c.save()
	}
	for i, p := range n.Fn.Params {
		v := args[i]
		v = c.wrapTo(p.Ty, v, n.Pos)
		c.assign(c.variable(p), v)
	}
	ret := f.dispatch // the only call site returns straight here
	if f.nsites > 1 {
		ret = c.newLabel()
		c.assign(f.ra, c.number(int32(len(f.sites)), n.Pos))
	}
	f.sites = append(f.sites, ret)
	c.jumpTo(f.entry, -1)
	c.place(ret)
	var t val
	if !n.Fn.Void {
		t = c.temporary()
		c.assign(t, f.rv)
	}
	if recursive {
		c.restore(saved)
	}
	return t
}

// save pushes what the function being lowered still needs after a call
// that may come back to it, and returns those cells for restore: its frame
// and the temporaries of the expression so far (an operand settled before
// the call, a write pointer, the arguments). The stack grows from the first
// free cell after the variables; sp holds the cell number of its top,
// which is what "the cell indexed by" writes to.
func (c *compiler) save() []int {
	cells := append(slices.Clone(c.cur.frame), c.temps()...)
	if len(cells) == 0 {
		return nil // declares no 1, which costs a cell
	}
	one := c.number(1, cc.Pos{Line: 1, Col: 1})
	for _, i := range cells {
		c.assign(val{vInd, c.sp.i}, val{vVar, i})
		c.assign(c.sp, c.expr(vSum, c.sp, one, 0))
	}
	return cells
}

// temps lists the cells allocated for the expression being lowered so far,
// except a function's frame (allocated up front, see functions).
func (c *compiler) temps() []int {
	var t []int
	for i := c.mark; i < c.nvars; i++ {
		t = append(t, i)
	}
	return t
}

// restore pops the cells save pushed, in reverse. A read through a pointer
// reads the cell before the one it holds, so rp = sp reads the top.
func (c *compiler) restore(cells []int) {
	if len(cells) == 0 {
		return
	}
	one := c.number(1, cc.Pos{Line: 1, Col: 1})
	for _, i := range slices.Backward(cells) {
		c.assign(c.rp, c.sp)
		c.assign(c.sp, c.expr(vDiff, c.sp, one, 0))
		c.assign(val{vVar, i}, val{vInd, c.rp.i})
	}
}

// functionBody lays out a function's code: its entry label, its body, and a
// return for falling off the end.
func (c *compiler) functionBody(f *function) {
	c.cur = f
	c.ret = &returnTo{label: f.dispatch, rv: f.rv, hasRV: !f.decl.Void, ty: f.decl.Ret}
	c.place(f.entry)
	c.stmt(f.decl.Body)
	if b := f.decl.Body.Body; len(b) == 0 || b[len(b)-1].Kind != cc.NdReturn {
		c.jumpTo(f.dispatch, -1)
	}
}

// dispatch lays out the chain that sends a return back to its call site:
// "go to return label k if ra is k", the last one unconditionally.
func (c *compiler) dispatch(f *function) {
	if f.nsites < 2 {
		return
	}
	c.place(f.dispatch)
	last := len(f.sites) - 1
	for k, label := range f.sites[:last] {
		c.jumpTo(label, c.eq(f.ra, c.number(int32(k), f.decl.Pos)).i)
	}
	c.jumpTo(f.sites[last], -1)
}

// returning lowers return: the value, if the function has one, goes to its
// result cell; then jump to where returns go.
func (c *compiler) returning(n *cc.Node) {
	r := c.ret
	if r.hasRV && n.Lhs != nil {
		v := c.value(n.Lhs)
		v = c.wrapTo(r.ty, v, n.Pos)
		c.assign(r.rv, v)
	} else {
		c.effects(n.Lhs)
	}
	label := r.label
	if label < 0 {
		label = c.exitLabel()
	}
	c.jumpTo(label, -1)
}

// exitLabel returns the label after all code, where main's return goes.
func (c *compiler) exitLabel() int {
	if c.exit < 0 {
		c.exit = c.newLabel()
	}
	return c.exit
}

// body lowers a loop body with l as the innermost loop.
func (c *compiler) body(l *loop, n *cc.Node) {
	c.loops = append(c.loops, l)
	c.stmt(n)
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
	if c.stack {
		c.addrs[c.stackAddr] = c.nvars
	}
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
	var operand func(v val) syntax.Operand
	operand = func(v val) syntax.Operand {
		switch v.kind {
		case vConst:
			return syntax.Operand{Type: syntax.Number, Index: int32(v.i)}
		case vVar:
			return syntax.Operand{Type: syntax.Number, Index: int32(size + v.i)}
		case vInd:
			return syntax.Operand{Type: syntax.Number | syntax.Indirect, Index: int32(size + v.i)}
		case vAddr:
			return syntax.Operand{Type: syntax.Number, Index: int32(addrSlot[v.i])}
		case vIdx:
			o := operand(c.idxs[v.i])
			if o.Type&syntax.Indirect != 0 {
				panic("compile: an operand indexed twice")
			}
			o.Type |= syntax.Indirect
			return o
		}
		types := map[valKind]syntax.OperandType{vSum: syntax.Sum, vDiff: syntax.Diff, vProd: syntax.Prod, vRatio: syntax.Ratio, vCond: syntax.Condition, vNand: syntax.Nand}
		return syntax.Operand{Type: types[v.kind], Index: int32(v.i)}
	}
	put := func(cat syntax.Category, slides []syntax.Slide) {
		p.Tables[cat] = syntax.Table{Count: len(slides), Index: len(p.Code)}
		p.Code = append(p.Code, slides...)
	}
	for _, k := range []struct {
		kind valKind
		cat  syntax.Category
	}{{vSum, syntax.Sums}, {vDiff, syntax.Diffs}, {vProd, syntax.Prods}, {vRatio, syntax.Ratios}, {vCond, syntax.Conditions}, {vNand, syntax.Nands}} {
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
	// The original cannot refer to a logical operation or an input.
	p.Verys = c.verys
	if p.Verys == 0 && (len(c.exprs[vNand].entries) > 0 || c.in >= 0) {
		p.Verys = 1
	}
	if c.in >= 0 {
		put(syntax.Reads, []syntax.Slide{{Ops: [2]syntax.Operand{operand(val{vVar, c.in})}, Flags: syntax.FormatCharacter}})
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
		types := map[stmtKind]syntax.OperandType{sAssign: syntax.Assign, sWrite: syntax.Write, sRead: syntax.Read, sJump: syntax.Jump, sLabel: syntax.Label}
		stmts = append(stmts, syntax.Slide{Ops: [2]syntax.Operand{{Type: types[s.kind], Index: int32(s.i)}}})
	}
	put(syntax.Statements, stmts)
	p.TypeCount = len(p.Code)
	return p
}
