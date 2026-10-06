package cc

import "strconv"

// TypeKind says what a type is.
type TypeKind int

// Type kinds, as in chibicc's TypeKind, slimmed to the subset.
const (
	TyInt   TypeKind = iota // int
	TyChar                  // char
	TyPtr                   // pointer to Base
	TyArray                 // array of Len Base
)

// Type is the type of a variable or an expression (chibicc's Type). Every
// value occupies one Sorted! cell, ints, chars and pointers alike, so a type
// says how a value behaves (a store into a char wraps, a pointer can be
// dereferenced), never how big it is.
type Type struct {
	Kind TypeKind
	Base *Type // pointer and array: the type pointed to, or of the elements
	Len  int   // array: the number of elements
}

var (
	tyInt  = &Type{Kind: TyInt}
	tyChar = &Type{Kind: TyChar}
)

func pointerTo(t *Type) *Type      { return &Type{Kind: TyPtr, Base: t} }
func arrayOf(t *Type, n int) *Type { return &Type{Kind: TyArray, Base: t, Len: n} }

// IsInteger reports whether t is int or char.
func (t *Type) IsInteger() bool { return t.Kind == TyInt || t.Kind == TyChar }

// sameType reports whether a and b are the same type: for pointers, to the
// same type. Arrays are never compared (they decay, and there are no
// pointers to arrays).
func sameType(a, b *Type) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Kind == TyPtr {
		return sameType(a.Base, b.Base)
	}
	return true
}

// String spells t as C would, for messages: "int", "char *", "int *[3]".
func (t *Type) String() string {
	switch t.Kind {
	case TyChar:
		return "char"
	case TyPtr:
		s := t.Base.String()
		if t.Base.Kind != TyPtr {
			s += " "
		}
		return s + "*"
	case TyArray:
		return t.Base.String() + "[" + strconv.Itoa(t.Len) + "]"
	}
	return "int"
}
