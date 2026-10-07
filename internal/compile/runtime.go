package compile

import (
	"maps"
	"math"
	"slices"

	"github.com/gersonkurz/sorted/internal/cc"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// runtimeSource holds the functions that shifts by a variable count, and
// unsigned division, remainder and right shift, call: they cannot be lowered
// to plain arithmetic. The unsigned ones take and give unsigned values as
// their 32-bit patterns in an int, and compare them with the sign bit
// flipped (adding -2^31), which orders them as unsigned. They are written in the C subset
// itself, so they compile like any user function, and only those a program
// reaches end up in its Sorted! text. They take numbers apart by halving
// and doubling (& | ^ are NANDs, see bitValue). Arithmetic wraps at 32 bits in
// Sorted!, which these rely on; they never run natively. Parse wants a main.
const runtimeSource = `
int shl(int x, int n) {
	while (n > 0) { x = x + x; n--; }
	return x;
}
int shr(int x, int n) {
	while (n > 0) { int q = x / 2; if (q + q > x) q--; x = q; n--; }
	return x;
}
int ushr(int x, int n) {
	if (n <= 0) return x;
	if (x < 0) { x = (x >> 1) + -2147483647 - 1; n--; }
	return x >> n;
}
int udiv(int a, int b) {
	int flip = -2147483647 - 1;
	if (b < 0) return a + flip < b + flip ? 0 : 1;
	if (a >= 0) return a / b;
	int q = ushr(a, 1) / b * 2;
	if (!(a - q * b + flip < b + flip)) q++;
	return q;
}
int umod(int a, int b) { return a - udiv(a, b) * b; }
int main() { return 0; }
`

// runtime returns the runtime function name, parsing the library on first
// use.
func (c *compiler) runtime(name string) *cc.Function {
	if c.rt == nil {
		prog, err := cc.Parse(runtimeSource)
		if err != nil {
			panic("compile: runtime library: " + err.Error())
		}
		c.rt = prog.Funcs
		// The library's own operators get the same treatment (ushr shifts by
		// a variable count, which calls shr).
		for _, name := range slices.Sorted(maps.Keys(c.rt)) {
			c.bitwise(c.rt[name].Body)
		}
	}
	return c.rt[name]
}

// bitwise rewrites, in place, the operators that need a loop into calls of
// the runtime functions, before functions() looks for calls: shifts by a
// variable count, and unsigned division and remainder. The rest stays for
// value: constants (folded), ~x (-1 - x), shifts by a constant (a product or
// a floored ratio), x & (2^k - 1) (a non-negative remainder), and the other
// & | ^, which are Very Sorted! NANDs (see bitValue).
func (c *compiler) bitwise(n *cc.Node) {
	if n == nil {
		return
	}
	for _, m := range []*cc.Node{n.Lhs, n.Rhs, n.Cond, n.Then, n.Els, n.Init, n.Inc} {
		c.bitwise(m)
	}
	for _, m := range n.Body {
		c.bitwise(m)
	}
	for _, m := range n.Args {
		c.bitwise(m)
	}
	if v, ok := cc.Fold(n); ok {
		if n.Ty.IsUnsignedInt() && (n.Kind == cc.NdDiv || n.Kind == cc.NdMod || n.Kind == cc.NdShr) {
			// value would lower the operands' arithmetic as signed
			*n = cc.Node{Kind: cc.NdNum, Pos: n.Pos, Val: v, Ty: n.Ty}
		}
		return
	}
	call := func(name string, args ...*cc.Node) {
		*n = cc.Node{Kind: cc.NdFuncall, Pos: n.Pos, Func: name, Fn: c.runtime(name), Args: args, Ty: n.Ty}
	}
	unsigned := n.Ty.IsUnsignedInt()
	switch n.Kind {
	case cc.NdDiv, cc.NdMod:
		if unsigned {
			call(map[cc.NodeKind]string{cc.NdDiv: "udiv", cc.NdMod: "umod"}[n.Kind], n.Lhs, n.Rhs)
		}
	case cc.NdShl, cc.NdShr:
		if k, ok := cc.Fold(n.Rhs); ok {
			if k < 0 || k > 31 {
				fail(n.Rhs.Pos, "shift count %d is out of range (0 to 31)", k)
			}
			if k == 1 && unsigned && n.Kind == cc.NdShr { // the mask would be 2^31 - 1
				call("ushr", n.Lhs, n.Rhs)
			}
			return
		}
		name := map[cc.NodeKind]string{cc.NdShl: "shl", cc.NdShr: "shr"}[n.Kind]
		if unsigned && n.Kind == cc.NdShr {
			name = "ushr"
		}
		call(name, n.Lhs, n.Rhs)
	}
}

// mask reports k when n is the constant 2^k - 1 (k at most 30), for which
// x & n is the non-negative remainder of x by 2^k.
func mask(n *cc.Node) (int, bool) {
	v, ok := cc.Fold(n)
	if !ok || v < 0 || v == math.MaxInt32 || v&(v+1) != 0 {
		return 0, false
	}
	k := 0
	for v > 0 {
		v >>= 1
		k++
	}
	return k, true
}

// shiftCount is the constant count of a shift that bitwise left to value;
// any other shift became a runtime call there, so a variable count here is
// a compiler bug, not something to lower as 0.
func (c *compiler) shiftCount(n *cc.Node) int32 {
	k, ok := cc.Fold(n.Rhs)
	if !ok {
		fail(n.Pos, "internal error: a shift by a variable count was not rewritten")
	}
	return k
}

// nandOp reports whether n is an & | ^ that bitValue builds from NANDs: not
// a constant, and not x & (2^k - 1).
func nandOp(n *cc.Node) bool {
	switch n.Kind {
	case cc.NdBitAnd, cc.NdBitOr, cc.NdBitXor:
	default:
		return false
	}
	if _, ok := cc.Fold(n); ok {
		return false
	}
	if n.Kind == cc.NdBitAnd {
		if _, ok := mask(n.Rhs); ok {
			return false
		}
		if _, ok := mask(n.Lhs); ok {
			return false
		}
	}
	return true
}

// cheap returns v if reading it is cheap (a number or a cell), and
// otherwise v settled into a temporary.
func (c *compiler) cheap(v val) val {
	switch v.kind {
	case vConst, vVar, vInd, vAddr:
		return v
	}
	return c.settle(v)
}

// bitValue lowers what bitwise left to value (see there).
func (c *compiler) bitValue(n *cc.Node) val {
	if k, ok := cc.Fold(n); ok {
		return c.number(k, n.Pos)
	}
	switch n.Kind {
	case cc.NdBitNot: // ~x == -1 - x
		return c.expr(vDiff, c.number(-1, n.Pos), c.value(n.Lhs), 0)
	case cc.NdBitAnd, cc.NdBitOr, cc.NdBitXor:
		if !nandOp(n) { // x & (2^k - 1) == (x % 2^k + 2^k) % 2^k
			x, m := n.Lhs, n.Rhs
			k, ok := mask(m)
			if !ok {
				x, m = m, x
				k, _ = mask(m)
			}
			p := c.number(1<<k, n.Pos)
			return c.mod(c.expr(vSum, c.mod(c.value(x), p), p, 0), p)
		}
		// NANDs, which make the program Very Sorted! (#26). Each operand is
		// read more than once, so an expression is settled first.
		a, b := c.operands(n.Lhs, n.Rhs)
		a, b = c.cheap(a), c.cheap(b)
		nand := func(x, y val) val { return c.expr(vNand, x, y, syntax.LogicalNand) }
		switch n.Kind {
		case cc.NdBitAnd: // ~(~(a & b))
			ab := nand(a, b)
			return nand(ab, ab)
		case cc.NdBitOr: // ~(~a & ~b)
			return nand(nand(a, a), nand(b, b))
		}
		ab := nand(a, b) // a ^ b == (a | b) & ~(a & b)
		return nand(nand(a, ab), nand(b, ab))
	case cc.NdShl: // x << k == x * 2^k (and 2^31 is -2^31 in 32 bits)
		k := c.shiftCount(n)
		return c.expr(vProd, c.value(n.Lhs), c.number(int32(uint32(1)<<k), n.Pos), 0)
	}
	// x >> k: the ratio truncates towards zero, the shift floors, so one less
	// when the ratio times 2^k came out above x. x >> 31 is -(x < 0). An
	// unsigned x shifts in zeros: the arithmetic shift's low 32 - k bits, a
	// remainder (k = 1 calls ushr, see bitwise).
	k := c.shiftCount(n)
	x := c.value(n.Lhs)
	if k == 0 {
		return x
	}
	var s val
	if k == 31 {
		s = c.expr(vDiff, c.number(0, n.Pos), c.lt(x, c.number(0, n.Pos)), 0)
	} else {
		p := c.number(1<<k, n.Pos)
		q := c.expr(vRatio, x, p, 0)
		s = c.expr(vDiff, q, c.lt(x, c.expr(vProd, q, p, 0)), 0)
	}
	if n.Ty.IsUnsignedInt() {
		p := c.number(1<<(32-k), n.Pos)
		s = c.mod(c.expr(vSum, c.mod(s, p), p, 0), p)
	}
	return s
}
