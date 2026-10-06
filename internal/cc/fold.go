package cc

import "math"

// Fold evaluates an integer constant expression (constants with the
// arithmetic, bitwise, comparison and logical operators) as 32-bit C does. Division by zero, the one overflowing
// division and shift counts outside 0..31 do not fold.
func Fold(n *Node) (int32, bool) {
	switch n.Kind {
	case NdNum:
		return n.Val, true
	case NdNeg, NdBitNot:
		a, ok := Fold(n.Lhs)
		if n.Kind == NdNeg {
			return -a, ok
		}
		return ^a, ok
	case NdNot:
		a, ok := Fold(n.Lhs)
		return b2i(a == 0), ok
	case NdLogAnd, NdLogOr:
		a, ok := Fold(n.Lhs)
		if ok && (a == 0) == (n.Kind == NdLogAnd) { // decided by the left side
			return b2i(a != 0), true
		}
		b, okB := Fold(n.Rhs)
		return b2i(b != 0), ok && okB
	case NdAdd, NdSub, NdMul, NdDiv, NdMod, NdBitAnd, NdBitOr, NdBitXor, NdShl, NdShr, NdEq, NdNe, NdLt, NdLe:
	default:
		return 0, false
	}
	a, okA := Fold(n.Lhs)
	b, okB := Fold(n.Rhs)
	if !okA || !okB {
		return 0, false
	}
	switch n.Kind {
	case NdAdd:
		return a + b, true
	case NdSub:
		return a - b, true
	case NdMul:
		return a * b, true
	case NdDiv, NdMod:
		if b == 0 || a == math.MinInt32 && b == -1 {
			return 0, false
		}
		if n.Kind == NdDiv {
			return a / b, true
		}
		return a % b, true
	case NdBitAnd:
		return a & b, true
	case NdBitOr:
		return a | b, true
	case NdBitXor:
		return a ^ b, true
	case NdEq:
		return b2i(a == b), true
	case NdNe:
		return b2i(a != b), true
	case NdLt:
		return b2i(a < b), true
	case NdLe:
		return b2i(a <= b), true
	}
	if b < 0 || b > 31 {
		return 0, false
	}
	if n.Kind == NdShl {
		return int32(uint32(a) << b), true
	}
	return a >> b, true
}

func b2i(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
