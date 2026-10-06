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
	TyStruct                // struct with Members
)

// Type is the type of a variable or an expression (chibicc's Type). Every
// scalar occupies one Sorted! cell, ints, chars and pointers alike; an array
// or a struct occupies the cells of its elements or members, in order (see
// Size).
type Type struct {
	Kind TypeKind
	Base *Type // pointer and array: the type pointed to, or of the elements
	Len  int   // array: the number of elements

	// struct
	Tag      string    // "" for an anonymous struct
	Members  []*Member // in order
	Complete bool      // the members are known (struct tag; declares it first)
	size     int       // the cells of all members
}

// Member is a member of a struct: its offset counts cells from the start.
type Member struct {
	Name   string
	Ty     *Type
	Offset int
	Pos    Pos
}

// Size is the number of cells a value of type t occupies.
func (t *Type) Size() int {
	switch t.Kind {
	case TyArray:
		return t.Len * t.Base.Size()
	case TyStruct:
		return t.size
	}
	return 1
}

// member finds a struct's member by name.
func (t *Type) member(name string) *Member {
	for _, m := range t.Members {
		if m.Name == name {
			return m
		}
	}
	return nil
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
// same type; structs are the same declaration. Arrays are never compared
// (they decay, and there are no pointers to arrays).
func sameType(a, b *Type) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case TyPtr:
		return sameType(a.Base, b.Base)
	case TyStruct:
		return a == b
	}
	return true
}

// IsScalar reports whether t is an integer or a pointer, which a condition
// can test.
func (t *Type) IsScalar() bool { return t.IsInteger() || t.Kind == TyPtr }

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
	case TyStruct:
		if t.Tag == "" {
			return "struct (anonymous)"
		}
		return "struct " + t.Tag
	}
	return "int"
}
