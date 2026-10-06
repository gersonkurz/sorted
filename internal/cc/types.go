package cc

import "strconv"

// TypeKind says what a type is.
type TypeKind int

// Type kinds, as in chibicc's TypeKind, slimmed to the subset.
const (
	TyInt    TypeKind = iota // int
	TyChar                   // char
	TyPtr                    // pointer to Base
	TyArray                  // array of Len Base
	TyStruct                 // struct with Members
)

// Type is the type of a variable or an expression (chibicc's Type). Every
// scalar occupies one Sorted! cell, ints, chars and pointers alike; an array
// or a struct occupies the cells of its elements or members, in order (see
// Size).
type Type struct {
	Kind     TypeKind
	Unsigned bool  // int and char: unsigned int, unsigned char
	Base     *Type // pointer and array: the type pointed to, or of the elements
	Len      int   // array: the number of elements

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
	tyInt   = &Type{Kind: TyInt}
	tyChar  = &Type{Kind: TyChar}
	tyUInt  = &Type{Kind: TyInt, Unsigned: true}
	tyUChar = &Type{Kind: TyChar, Unsigned: true}
)

// IsUnsignedInt reports whether t is unsigned int, the type that makes
// arithmetic and comparisons unsigned (an unsigned char becomes an int).
func (t *Type) IsUnsignedInt() bool { return t != nil && t.Kind == TyInt && t.Unsigned }

// promoted is the type an integer operand has in arithmetic: int, or
// unsigned int (C's integer promotions).
func promoted(t *Type) *Type {
	if t.IsUnsignedInt() {
		return tyUInt
	}
	return tyInt
}

// arith is the type of arithmetic on two integers: unsigned int if either
// is, otherwise int (C's usual arithmetic conversions, for these types).
func arith(a, b *Type) *Type {
	if a.IsUnsignedInt() || b.IsUnsignedInt() {
		return tyUInt
	}
	return tyInt
}

func pointerTo(t *Type) *Type      { return &Type{Kind: TyPtr, Base: t} }
func arrayOf(t *Type, n int) *Type { return &Type{Kind: TyArray, Base: t, Len: n} }

// IsInteger reports whether t is int or char.
func (t *Type) IsInteger() bool { return t.Kind == TyInt || t.Kind == TyChar }

// sameType reports whether a and b are the same type: the same signedness,
// for pointers to the same type, for arrays (pointed to by array
// parameters) of the same length, and for structs the same declaration.
func sameType(a, b *Type) bool {
	if a.Kind != b.Kind || a.Unsigned != b.Unsigned {
		return false
	}
	switch a.Kind {
	case TyPtr:
		return sameType(a.Base, b.Base)
	case TyArray:
		return a.Len == b.Len && sameType(a.Base, b.Base)
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
	u := ""
	if t.Unsigned {
		u = "unsigned "
	}
	switch t.Kind {
	case TyChar:
		return u + "char"
	case TyPtr:
		if t.Base.Kind == TyArray {
			return t.Base.Base.String() + " (*)[" + strconv.Itoa(t.Base.Len) + "]"
		}
		s := t.Base.String()
		if t.Base.Kind != TyPtr {
			s += " "
		}
		return s + "*"
	case TyArray: // int[3][4]: the outer length first
		dims := ""
		for ; t.Kind == TyArray; t = t.Base {
			dims += "[" + strconv.Itoa(t.Len) + "]"
		}
		return t.String() + dims
	case TyStruct:
		if t.Tag == "" {
			return "struct (anonymous)"
		}
		return "struct " + t.Tag
	}
	return u + "int"
}
