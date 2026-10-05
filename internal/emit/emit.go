// Package emit renders a parsed program the two ways the original can write
// it to a file: the table dump of /D (DumpCodeInfo) and the C translation of
// /C (GenerateSourceInC), ported from legacy/sorted.win32/Sorted.cpp byte for
// byte, apart from line ends: the original writes in text mode, so Windows
// files have CRLF where these have LF.
//
// The C translation keeps the original's oddities: every output becomes
// putchar whatever its format, and an indirect target is 1-based (N(N(k)))
// although the interpreter writes 0-based, so the generated program does not
// always behave like the interpreted one.
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

// operand renders an operand like SlideType: N(3), S(1), N(N(5)); without
// the closing parenthesis for statements, which append ");".
func operand(o syntax.Operand, closing bool) string {
	letters := []string{"N", "S", "D", "P", "R", "L", "A", "W", "r", "C", "J", "L"}
	var s string
	if o.Type&syntax.Indirect != 0 {
		s = fmt.Sprintf("N(%s(%d))", letters[o.Type&0xFF], o.Index+1)
	} else {
		s = fmt.Sprintf("%s(%d)", letters[o.Type], o.Index+1)
	}
	if !closing {
		s = s[:len(s)-1]
	}
	return s
}

// C renders the /C translation into C.
func C(p *syntax.Program) string {
	var b strings.Builder
	b.WriteString("#include \"stdio.h\"\n")
	b.WriteString("#include \"stdlib.h\"\n")
	b.WriteString("#include \"string.h\"\n\n")
	fmt.Fprintf(&b, "long _[%d]", dataCells)
	if len(p.Data) == 0 {
		b.WriteString(";\n")
	} else {
		b.WriteString(" = { " + joinData(p.Data) + "};\n\n")
	}
	b.WriteString("#define N(x) _[x-1]\n")
	b.WriteString("#define G(x) long x\n")
	b.WriteString("#define B(x,y) case x: return y;\n")
	b.WriteString("#define d(x) G(x)(G(n))\n")
	b.WriteString("#define E(x) d(x){switch(n){\n")
	b.WriteString("#define L(n) label##n:\n")
	b.WriteString("#define UJ(n) goto label##n\n")
	b.WriteString("#define CJ(n,c) if(C(c)) UJ(n)\n\n")
	b.WriteString("#define H(x) putchar(x)\n")
	b.WriteString("#define i(x) x = getchar()\n")
	b.WriteString("#define F } return 0; }\n")
	b.WriteString("\n")

	// One function per category up to the conditions. Note "L" for logical
	// operations and "R" for both ratios and reads, as in the original.
	names := []string{"S", "D", "P", "R", "L", "A", "W", "R", "C"}
	formats := []string{"%s+%s", "%s-%s", "%s*%s", "%s/%s", "~(%s) & ~(%s)", "%s=%s", "H(%s)", "i(%s)"}
	for c := syntax.Sums; c <= syntax.Conditions; c++ {
		if p.Tables[c].Count > 0 {
			fmt.Fprintf(&b, "d(%s);", names[c])
		}
	}
	b.WriteString("\n")

	// A newline after every fourth item, counted across the functions and
	// main together.
	q := 0
	item := func(s string) {
		b.WriteString(s)
		if q++; q%4 == 0 {
			b.WriteString("\n")
		}
	}
	for c := syntax.Sums; c <= syntax.Conditions; c++ {
		t := p.Tables[c]
		if t.Count == 0 {
			continue
		}
		fmt.Fprintf(&b, "E(%s)", names[c])
		for j := 0; j < t.Count; j++ {
			s := code(p, t.Index+j)
			a, z := operand(s.Ops[0], true), operand(s.Ops[1], true)
			switch c {
			case syntax.Conditions:
				cmp := "<"
				if s.Flags == syntax.CompareEqual {
					cmp = "=="
				}
				item(fmt.Sprintf("B(%d,%s%s%s) ", j+1, a, cmp, z))
			case syntax.Assigns:
				item(fmt.Sprintf("B(%d,"+formats[c]+")", j+1, z, a))
			case syntax.Writes, syntax.Reads:
				item(fmt.Sprintf("B(%d,"+formats[c]+")", j+1, a))
			default:
				item(fmt.Sprintf("B(%d,"+formats[c]+")", j+1, a, z))
			}
		}
		b.WriteString("F ")
	}

	b.WriteString("int main(int,char*[]) {")
	jumps := p.Tables[syntax.Jumps].Index
	stmts := p.Tables[syntax.Statements]
	for j := 0; j < stmts.Count; j++ {
		op := code(p, stmts.Index+j).Ops[0]
		if op.Type == syntax.Jump {
			jump := code(p, jumps+int(op.Index))
			if jump.Flags == syntax.UnconditionalJump {
				item(fmt.Sprintf("UJ(%d);", jump.Ops[0].Index+1))
			} else {
				item(fmt.Sprintf("CJ(%d,%d);", jump.Ops[0].Index+1, jump.Ops[1].Index+1))
			}
			continue
		}
		item(operand(op, false) + ");")
	}
	b.WriteString("{ F\n")
	return b.String()
}
