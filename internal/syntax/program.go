// Package syntax turns Sorted! source text into the program tables the
// interpreter runs. It ports ReadSortedSourcecode (Sorted.cpp) and
// SortedSyntax.cpp from legacy/sorted.win32 line by line, including the
// order in which alternatives consume input and the table state that failed
// alternatives leave behind, because both are observable (in what parses and
// in the /D dump).
package syntax

// OperandType says what an operand refers to (SLID_TYPE_* in SortedSyntax.h).
type OperandType int32

// Operand types. Number and Cell are the same type: "the third number" and
// "the third cell" both mean data cell 2.
const (
	Number OperandType = iota
	Sum
	Diff
	Prod
	Ratio
	Nand
	Assign
	Write
	Read
	Condition
	Jump
	Label

	Cell = Number

	// Indirect is or-ed into the type for "the cell indexed by ...".
	Indirect OperandType = 0xF00000
)

// Operand is a reference such as "the second sum" (SLID). Index is zero-based.
type Operand struct {
	Type  OperandType
	Index int32
}

// Slide is one table entry: up to two operands and a flag (SLIDE).
type Slide struct {
	Ops   [2]Operand
	Flags int32
}

// Category names a table of the program (SLIDE_INFO_* in SortedSyntax.h).
type Category int

// Categories, in the order of SortedSyntax.h, which the /D dump follows.
const (
	Sums Category = iota
	Diffs
	Prods
	Ratios
	Nands
	Assigns
	Writes
	Reads
	Conditions
	Statements
	Jumps
	NumCategories
)

// Flags of writes and reads (SLIDE_FORMAT_*), conditions (SLIDE_COMPARE_*)
// and jumps (SLIDE_*_JUMP).
const (
	FormatCharacter       int32 = 0
	FormatEnglishCardinal int32 = 1
	FormatEnglishOrdinal  int32 = 2
	FormatGermanCardinal  int32 = 3
	FormatGermanOrdinal   int32 = 4
	// Very Very Sorted! (#30) also writes Italian numbers.
	FormatItalianCardinal int32 = 5
	FormatItalianOrdinal  int32 = 6
	// ... and Vaudois French numbers (#29).
	FormatVaudoisCardinal int32 = 7
	FormatVaudoisOrdinal  int32 = 8
	// ... and Brazilian Portuguese numbers (#31).
	FormatBrazilianCardinal int32 = 9
	FormatBrazilianOrdinal  int32 = 10

	CompareEqual int32 = 0
	CompareLess  int32 = 1

	UnconditionalJump int32 = 0
	ConditionalJump   int32 = 1

	// Logical operations: the original's computes ~a & ~b, a NOR, as its
	// sentence says ("of not a and not b"); Very Sorted! adds the NAND (#26).
	LogicalNor  int32 = 0
	LogicalNand int32 = 1
)

// Table locates a category's entries in Code (SLIDE_INFO).
type Table struct {
	Count, Index int
}

// Program is the parse result (CODEINFO).
type Program struct {
	// Tables locates each category in Code. Categories are laid out in the
	// order their sentences appear; Index is the running TypeCount when the
	// sentence started, so gaps are possible (see TypeCount).
	Tables [NumCategories]Table
	// TypeCount counts table entries (nTypeCount, the dump's ELEMENTS). It
	// also counts entries that a "single" alternative accepted and then
	// withdrew because no period followed, so it can exceed the real total.
	TypeCount int
	// LabelsCount is the number of labels the program declares.
	LabelsCount int
	// Code holds every table entry. Entries past a table's Count are scratch
	// left by failed alternatives.
	Code []Slide
	// Data holds the declared numbers, in order (Data[:nNumbersUsed]).
	Data []int32
	// Verys is the program's dialect, counted in verys (#39): 0 for the
	// original's Sorted!, 1 for Very Sorted! ("This code is very cool."),
	// 2 for Very Very Sorted! ("This code is very very cool."), which also
	// speaks Italian, French and Portuguese (see Parse). Each dialect has everything the one
	// before it has.
	Verys int
}

// Entries returns the entries of category c. An empty table may start past
// the end of Code, because TypeCount can run ahead of the entries written.
func (p *Program) Entries(c Category) []Slide {
	t := p.Tables[c]
	if t.Count == 0 {
		return nil
	}
	return p.Code[t.Index : t.Index+t.Count]
}

// slot returns the entry after the last one of category c, growing Code as
// needed (&Code[Index+Count]). The pointer stays valid until Code grows
// again, which happens only in the next slot call.
func (p *Program) slot(c Category) *Slide {
	i := p.Tables[c].Index + p.Tables[c].Count
	for len(p.Code) <= i {
		p.Code = append(p.Code, Slide{})
	}
	return &p.Code[i]
}
