// Package emit renders a parsed program into files: the table dump of the
// original's /D (DumpCodeInfo, ported from legacy/sorted.win32/Sorted.cpp
// byte for byte, apart from line ends: the original writes in text mode, so
// Windows files have CRLF where these have LF), and a translation into C
// that behaves like the interpreter (Exact).
//
// The original's own C translation (/C, GenerateSourceInC) is not ported
// (Gerson's decision, 2026-10-06): it does not do what the program does.
// It writes every output with putchar whatever its format, so fibo.s prints
// raw bytes instead of English cardinals, and it writes indirect cells
// 1-based where the interpreter writes them 0-based, so itoa.s prints its
// NUL after the digits instead of before. Its captures stay in
// testdata/golden as history.
package emit

import (
	"fmt"
	"strings"

	"github.com/gersonkurz/sorted/internal/syntax"
)

const (
	dataCells = 193719 // MAX_DATA_PER_PROGRAM: the size of _[] in the C code
	codeSlots = 128192 // MAX_INSTRUCTIONS_PER_PROGRAM
)

// code returns slot i of the static Code array: zeros past what the parser
// wrote (and, port-defined, past the array).
func code(p *syntax.Program, i int) syntax.Slide {
	if i >= 0 && i < len(p.Code) && i < codeSlots {
		return p.Code[i]
	}
	return syntax.Slide{}
}

// Dump renders the /D table dump.
func Dump(p *syntax.Program) string {
	categories := []string{"SUMS", "DIFFS", "PRODS", "RATIOS", "NANDS", "ASSIGNS", "WRITES", "READS", "CONDITIONS", "STATEMENTS", "JUMPS"}
	usage := []int{2, 2, 2, 2, 2, 2, 1, 1, 2, 1, 2}
	types := []string{"CELL", "SUM", "DIFF", "PROD", "RATIO", "NAND", "ASSIGN", "WRITE", "READ", "COMPARE", "JUMP", "LABEL"}
	op := func(o syntax.Operand) string {
		if o.Type&syntax.Indirect != 0 {
			return fmt.Sprintf("*(%d,%s)", o.Index, types[o.Type&0xFF])
		}
		return fmt.Sprintf("(%d,%s)", o.Index, types[o.Type])
	}

	var b strings.Builder
	fmt.Fprintf(&b, "ELEMENTS=%d\n", p.TypeCount)
	fmt.Fprintf(&b, "LABELS=%d\n", p.LabelsCount)
	fmt.Fprintf(&b, "%d NUMBERS: ", len(p.Data))
	b.WriteString(joinData(p.Data))
	for c := syntax.Sums; c < syntax.NumCategories; c++ {
		t := p.Tables[c]
		fmt.Fprintf(&b, "\n%d %s: ", t.Count, categories[c])
		for j := 0; j < t.Count; j++ {
			if j > 0 {
				b.WriteString(", ")
			}
			s := code(p, t.Index+j)
			b.WriteString("{ " + op(s.Ops[0]))
			if usage[c] == 2 {
				b.WriteString("," + op(s.Ops[1]))
			}
			fmt.Fprintf(&b, ",%d}", s.Flags)
		}
	}
	b.WriteString("\n")
	return b.String()
}

func joinData(data []int32) string {
	parts := make([]string, len(data))
	for i, d := range data {
		parts[i] = fmt.Sprint(d)
	}
	return strings.Join(parts, ",")
}
