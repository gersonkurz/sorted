package compile

import (
	"math"

	"github.com/gersonkurz/sorted/internal/cc"
)

// runtimeSource holds the functions that bitwise operators call when they
// cannot be lowered to plain arithmetic. They are written in the C subset
// itself, so they compile like any user function, and only those a program
// reaches end up in its Sorted! text. Sorted! has no bit operations at all
// (its logical operations can be declared but never referenced), so they
// take numbers apart digit by binary digit. Arithmetic wraps at 32 bits in
// Sorted!, which these rely on; they never run natively. Parse wants a main.
const runtimeSource = `
int bits(int a, int b, int m, int s) {
	int r = 0, bit = 1, i = 0;
	int sa = a < 0, sb = b < 0;
	if (sa) a = a + 2147483647 + 1;
	if (sb) b = b + 2147483647 + 1;
	while (i < 31) {
		int x = a % 2, y = b % 2;
		r = r + (m * x * y + s * (x + y)) * bit;
		a = a / 2;
		b = b / 2;
		bit = bit + bit;
		i++;
	}
	if (m * sa * sb + s * (sa + sb)) r = r - 2147483647 - 1;
	return r;
}
int shl(int x, int n) {
	while (n > 0) { x = x + x; n--; }
	return x;
}
int shr(int x, int n) {
	while (n > 0) { int q = x / 2; if (q + q > x) q--; x = q; n--; }
	return x;
}
int main() { return 0; }
`

// bitsCoefficients are bits' m and s for &, | and ^: the bit of the result
// is m*x*y + s*(x+y) for the operand bits x and y.
var bitsCoefficients = map[cc.NodeKind][2]int32{
	cc.NdBitAnd: {1, 0},
	cc.NdBitOr:  {-1, 1},
	cc.NdBitXor: {-2, 1},
}

// runtime returns the runtime function name, parsing the library on first
// use.
func (c *compiler) runtime(name string) *cc.Function {
	if c.rt == nil {
		prog, err := cc.Parse(runtimeSource)
		if err != nil {
			panic("compile: runtime library: " + err.Error())
		}
		c.rt = prog.Funcs
	}
	return c.rt[name]
}

// bitwise rewrites, in place, the bitwise operators that need a loop into
// calls of the runtime functions, before functions() looks for calls. What
// arithmetic can do stays for value: constants (folded), ~x (-1 - x),
// shifts by a constant (a product or a floored ratio) and x & (2^k - 1)
// (a non-negative remainder).
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
	if _, ok := cc.Fold(n); ok {
		return
	}
	call := func(name string, args ...*cc.Node) {
		*n = cc.Node{Kind: cc.NdFuncall, Pos: n.Pos, Func: name, Fn: c.runtime(name), Args: args, Ty: n.Ty}
	}
	num := func(v int32) *cc.Node { return &cc.Node{Kind: cc.NdNum, Pos: n.Pos, Val: v} }
	switch n.Kind {
	case cc.NdBitAnd, cc.NdBitOr, cc.NdBitXor:
		if _, ok := mask(n.Rhs); ok && n.Kind == cc.NdBitAnd {
			return
		}
		if _, ok := mask(n.Lhs); ok && n.Kind == cc.NdBitAnd {
			return
		}
		k := bitsCoefficients[n.Kind]
		call("bits", n.Lhs, n.Rhs, num(k[0]), num(k[1]))
	case cc.NdShl, cc.NdShr:
		if k, ok := cc.Fold(n.Rhs); ok {
			if k < 0 || k > 31 {
				fail(n.Rhs.Pos, "shift count %d is out of range (0 to 31)", k)
			}
			return
		}
		name := map[cc.NodeKind]string{cc.NdShl: "shl", cc.NdShr: "shr"}[n.Kind]
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

// bitValue lowers what bitwise left to value (see there).
func (c *compiler) bitValue(n *cc.Node) val {
	if k, ok := cc.Fold(n); ok {
		return c.number(k, n.Pos)
	}
	switch n.Kind {
	case cc.NdBitNot: // ~x == -1 - x
		return c.expr(vDiff, c.number(-1, n.Pos), c.value(n.Lhs), 0)
	case cc.NdBitAnd: // x & (2^k - 1) == (x % 2^k + 2^k) % 2^k
		x, m := n.Lhs, n.Rhs
		k, ok := mask(m)
		if !ok {
			x, m = m, x
			k, _ = mask(m)
		}
		p := c.number(1<<k, n.Pos)
		return c.mod(c.expr(vSum, c.mod(c.value(x), p), p, 0), p)
	case cc.NdShl: // x << k == x * 2^k (and 2^31 is -2^31 in 32 bits)
		k, _ := cc.Fold(n.Rhs)
		return c.expr(vProd, c.value(n.Lhs), c.number(int32(uint32(1)<<k), n.Pos), 0)
	}
	// x >> k: the ratio truncates towards zero, the shift floors, so one less
	// when the ratio times 2^k came out above x. x >> 31 is -(x < 0).
	k, _ := cc.Fold(n.Rhs)
	x := c.value(n.Lhs)
	if k == 0 {
		return x
	}
	if k == 31 {
		return c.expr(vDiff, c.number(0, n.Pos), c.lt(x, c.number(0, n.Pos)), 0)
	}
	p := c.number(1<<k, n.Pos)
	q := c.expr(vRatio, x, p, 0)
	return c.expr(vDiff, q, c.lt(x, c.expr(vProd, q, p, 0)), 0)
}
