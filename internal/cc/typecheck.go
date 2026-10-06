package cc

import "strconv"

// The type pass, chibicc's add_type, with the checks C's constraints ask
// for. The subset is stricter than a compiler that only warns: mixing
// pointers and integers, or pointers to different types, is an error
// (except for the null pointer constant 0).

// operator names for messages
var opNames = map[NodeKind]string{
	NdAdd: "+", NdSub: "-", NdMul: "*", NdDiv: "/", NdMod: "%", NdBitAnd: "&", NdBitOr: "|",
	NdBitXor: "^", NdShl: "<<", NdShr: ">>", NdEq: "==", NdNe: "!=", NdLt: "<", NdLe: "<=",
}

// typed sets the type of n and everything below it, checking the operands,
// and returns it. It is called on every expression the parser finishes, and
// earlier wherever the parser needs a type to decide; a node keeps the type
// it got first.
func (ps *parser) typed(n *Node) *Type {
	if n == nil {
		return nil
	}
	if n.Ty != nil {
		return n.Ty
	}
	ps.typed(n.Lhs)
	ps.typed(n.Rhs)
	if n.Kind == NdCond {
		ps.typed(n.Cond)
		ps.typed(n.Then)
		ps.typed(n.Els)
	}
	for _, a := range n.Args {
		ps.typed(a)
	}
	n.Ty = ps.typeOf(n)
	return n.Ty
}

func (ps *parser) typeOf(n *Node) *Type {
	switch n.Kind {
	case NdNum, NdNot, NdLogAnd, NdLogOr:
		return tyInt
	case NdVar:
		return n.Var.Ty
	case NdIndex:
		ps.integer(n.Lhs, "an array index")
		return n.Var.Ty.Base
	case NdAddr:
		if decayed(n) {
			return pointerTo(n.Lhs.Var.Ty.Base)
		}
		return pointerTo(n.Lhs.Ty)
	case NdDeref:
		if n.Lhs.Ty.Kind != TyPtr {
			ps.fail(n.Pos, "indirection needs a pointer, not '%s'", n.Lhs.Ty)
		}
		return n.Lhs.Ty.Base
	case NdNeg, NdBitNot:
		ps.integer(n.Lhs, "the operand of unary '"+map[NodeKind]string{NdNeg: "-", NdBitNot: "~"}[n.Kind]+"'")
		return tyInt
	case NdAdd:
		a, b := n.Lhs.Ty, n.Rhs.Ty
		switch {
		case a.IsInteger() && b.IsInteger():
			return tyInt
		case a.Kind == TyPtr && b.IsInteger():
			return a
		case a.IsInteger() && b.Kind == TyPtr:
			return b
		}
	case NdSub:
		a, b := n.Lhs.Ty, n.Rhs.Ty
		switch {
		case a.IsInteger() && b.IsInteger():
			return tyInt
		case a.Kind == TyPtr && b.IsInteger():
			return a
		case a.Kind == TyPtr && b.Kind == TyPtr && sameType(a, b):
			return tyInt // the number of elements between them
		}
	case NdMul, NdDiv, NdMod, NdBitAnd, NdBitOr, NdBitXor, NdShl, NdShr:
		if n.Lhs.Ty.IsInteger() && n.Rhs.Ty.IsInteger() {
			return tyInt
		}
	case NdEq, NdNe, NdLt, NdLe:
		a, b := n.Lhs.Ty, n.Rhs.Ty
		equality := n.Kind == NdEq || n.Kind == NdNe
		switch {
		case a.IsInteger() && b.IsInteger(),
			a.Kind == TyPtr && b.Kind == TyPtr && sameType(a, b),
			equality && a.Kind == TyPtr && isNull(n.Rhs),
			equality && b.Kind == TyPtr && isNull(n.Lhs):
			return tyInt
		}
	case NdAssign:
		ps.assignable(n.Lhs.Ty, n.Rhs, n.Pos, "the assignment")
		return n.Lhs.Ty
	case NdComma:
		return n.Rhs.Ty
	case NdCond:
		a, b := n.Then.Ty, n.Els.Ty
		switch {
		case a.IsInteger() && b.IsInteger():
			return tyInt
		case a.Kind == TyPtr && b.Kind == TyPtr && sameType(a, b),
			a.Kind == TyPtr && isNull(n.Els):
			return a
		case b.Kind == TyPtr && isNull(n.Then):
			return b
		}
		ps.fail(n.Pos, "the branches of '?:' have different types ('%s' and '%s')", a, b)
	case NdFuncall:
		if n.Fn == nil { // putchar
			ps.integer(n.Args[0], "putchar's argument")
			return tyInt
		}
		for i, a := range n.Args {
			ps.assignable(n.Fn.Params[i].Ty, a, a.Pos, "argument "+strconv.Itoa(i+1)+" of '"+n.Func+"'")
		}
		if n.Fn.Ret == nil {
			return tyInt // void: using the value is a compile error later
		}
		return n.Fn.Ret
	default:
		ps.fail(n.Pos, "cannot type this expression")
	}
	ps.fail(n.Pos, "invalid operands to '%s' ('%s' and '%s')", opNames[n.Kind], n.Lhs.Ty, n.Rhs.Ty)
	return nil
}

// integer checks that n is an int or a char.
func (ps *parser) integer(n *Node, what string) {
	if !ps.typed(n).IsInteger() {
		ps.fail(n.Pos, "%s must be an integer, not '%s'", what, n.Ty)
	}
}

// isNull reports whether n is the null pointer constant: an integer
// constant expression that is 0.
func isNull(n *Node) bool {
	v, ok := Fold(n)
	return ok && v == 0 && n.Ty.IsInteger()
}

// assignable checks that n can be stored where a t goes (an assignment, an
// argument, a return value, an initializer): integers into integers (a char
// wraps), a pointer into the same pointer type, and 0 into any pointer.
func (ps *parser) assignable(t *Type, n *Node, pos Pos, what string) {
	v := ps.typed(n)
	switch {
	case t.IsInteger() && v.IsInteger(),
		t.Kind == TyPtr && v.Kind == TyPtr && sameType(t, v),
		t.Kind == TyPtr && isNull(n):
		return
	}
	ps.fail(pos, "%s needs '%s', not '%s'", what, t, v)
}
