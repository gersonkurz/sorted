package interp

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gersonkurz/sorted/internal/syntax"
)

const steps = 1000000 // generous; the samples need a few hundred

func readFile(t *testing.T, path ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, path...)...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// normalise applies the golden-test rule: CRLF is LF and a final newline is
// ignored (Gerson: "that final newline is negligible").
func normalise(s string) string {
	return strings.TrimSuffix(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// The four samples print what Sorted.exe printed for them
// (testdata/golden/<sample>.out, captured 2026-10-04).
func TestSamplesMatchCapturedOutput(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) {
			p, err := syntax.Parse(readFile(t, "legacy", "sorted.win32", name+".s"))
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err := Run(p, strings.NewReader(""), &out, steps); err != nil {
				t.Fatal(err)
			}
			if got, want := normalise(out.String()), normalise(string(readFile(t, "testdata", "golden", name+".out"))); got != want {
				t.Errorf("output %q, want %q", got, want)
			}
		})
	}
}

// --- small programs; expectations follow from reading Sorted.cpp ---

// sentence keys in program order, with their "none" forms.
var order = []struct{ key, none string }{
	{"numbers", "This code does not use any numbers."},
	{"jumps", "This code does never go anywhere."},
	{"writes", "This code cannot write."},
	{"reads", "This code cannot read."},
	{"sums", "This code does not use any sums."},
	{"conditions", "This code does not use any conditions."},
	{"labels", "This code does not use any labels."},
	{"diffs", "This code does not use any ordered differences."},
	{"assigns", "This code does not use any assignments."},
	{"prods", "This code does not use any products."},
	{"impl", ""},
	{"ratios", "This code does not use any ratios."},
	{"nands", "This code does not use any logical operations."},
	{"cool", "Cool."},
}

type sentences map[string]string

func program(t *testing.T, s sentences) *syntax.Program {
	t.Helper()
	var src []string
	for _, o := range order {
		if v, ok := s[o.key]; ok {
			src = append(src, v)
		} else {
			src = append(src, o.none)
		}
	}
	p, err := syntax.Parse([]byte(strings.Join(src, "\n")))
	if err != nil {
		t.Fatalf("%v in:\n%s", err, strings.Join(src, "\n"))
	}
	return p
}

func run(t *testing.T, p *syntax.Program, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(p, strings.NewReader(input), &out, steps)
	return out.String(), err
}

func runtimeError(t *testing.T, err error, what string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.What != what {
		t.Errorf("err %v, want runtime error %q", err, what)
	}
}

func TestArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		s        sentences
		want     string
		wantFail string
	}{
		{"sum", sentences{
			"numbers": "This code uses the numbers seven, and five.",
			"writes":  "This code writes the first sum as a english cardinal.",
			"sums":    "This code uses the sum of the first number and the second number.",
			"impl":    "This code implements the first output.",
		}, "twelve\n", ""},
		{"ordered difference", sentences{
			"numbers": "This code uses the numbers seven, and five.",
			"writes":  "This code writes the first ordered difference as a english cardinal.",
			"diffs":   "This code uses the ordered difference between the first number and the second number.",
			"impl":    "This code implements the first output.",
		}, "two\n", ""},
		{"product", sentences{
			"numbers": "This code uses the numbers seven, and five.",
			"writes":  "This code writes the first product as a german cardinal.",
			"prods":   "This code uses the product of the first number and the second number.",
			"impl":    "This code implements the first output.",
		}, "fuenfunddreissig\n", ""},
		{"product wraps at 32 bits, and zero prints as an empty line", sentences{
			"numbers": "This code uses the number sixtyfive thousand fivehundredthirtysix.",
			"writes":  "This code writes the first product as a english cardinal.",
			"prods":   "This code uses the product of the first number and the first number.",
			"impl":    "This code implements the first output.",
		}, "\n", ""},
		// (0 - 7) / 2 == 0 - 3: division truncates toward zero.
		{"ratio truncates toward zero", sentences{
			"numbers":    "This code uses the numbers zero, seven, two, and three.",
			"writes":     "This code writes the first condition as a english cardinal.",
			"conditions": "This code uses the condition that the first ratio is equal to the second ordered difference.",
			"diffs":      "This code uses the ordered differences between the first number and the second number, and between the first number and the fourth number.",
			"impl":       "This code implements the first output.",
			"ratios":     "This code uses the ratio of the first ordered difference to the third number.",
		}, "one\n", ""},
		{"less than", sentences{
			"numbers":    "This code uses the numbers seven, and five.",
			"writes":     "This code writes the first condition as a english cardinal.",
			"conditions": "This code uses the condition that the second number is less than the first number.",
			"impl":       "This code implements the first output.",
		}, "one\n", ""},
		{"a character is the low byte", sentences{
			"numbers": "This code uses the number threehundredtwentyone.",
			"writes":  "This code writes the first number as a character.",
			"impl":    "This code implements the first output.",
		}, "A", ""}, // 321 & 0xFF == 65
		{"german ordinal", sentences{
			"numbers": "Dieses Programm benutzt die Zahl drei.",
			"writes":  "Dieses Programm schreibt die erste Zahl als eine deutsche Ordinalzahl.",
			"impl":    "This code implements the first output.",
		}, "dritte\n", ""},
		{"division by zero", sentences{
			"numbers": "This code uses the numbers seven, and zero.",
			"writes":  "This code writes the first ratio as a english cardinal.",
			"impl":    "This code implements the first output.",
			"ratios":  "This code uses the ratio of the first number to the second number.",
		}, "", "division by zero"},
		{"a negative cardinal stops the program, keeping earlier output", sentences{
			"numbers": "This code uses the numbers seventy, sixtyfive, and eighty.",
			"writes":  "This code writes the first ordered difference as a english cardinal.",
			"diffs":   "This code uses the ordered difference between the first number and the second number.",
			"assigns": "This code assigns the third number to the second number.",
			"impl":    "This code implements the first output, the first assignment, and the first output.",
		}, "five\n", "cannot write a negative number as a cardinal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := run(t, program(t, tt.s), "")
			if tt.wantFail != "" {
				runtimeError(t, err, tt.wantFail)
			} else if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("output %q, want %q", got, tt.want)
			}
		})
	}
}

// hand lays out tables in category order, as the parser does when nothing
// is withdrawn, followed by the statements. The parser cannot produce
// references to inputs or logical operations, so their interpreter paths are
// only reachable this way.
func hand(data []int32, tables map[syntax.Category][]syntax.Slide, statements ...syntax.Operand) *syntax.Program {
	p := &syntax.Program{Data: data}
	for c := syntax.Sums; c < syntax.NumCategories; c++ {
		if c == syntax.Statements {
			continue
		}
		p.Tables[c] = syntax.Table{Count: len(tables[c]), Index: len(p.Code)}
		p.Code = append(p.Code, tables[c]...)
	}
	p.Tables[syntax.Statements] = syntax.Table{Count: len(statements), Index: len(p.Code)}
	for _, op := range statements {
		p.Code = append(p.Code, syntax.Slide{Ops: [2]syntax.Operand{op}})
	}
	return p
}

func num(i int32) syntax.Operand { return syntax.Operand{Type: syntax.Number, Index: i} }

func runHand(t *testing.T, p *syntax.Program, input string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(p, strings.NewReader(input), &out, steps)
	return out.String(), err
}

// Input goes to the read entry's second operand, which the parser never
// fills: the first cell, whichever cell the read names.
func TestReadStoresIntoSecondOperand(t *testing.T) {
	reads := []syntax.Slide{{Ops: [2]syntax.Operand{num(1)}}} // "reads the second number"
	writeFirst := []syntax.Slide{{Ops: [2]syntax.Operand{num(0)}}}
	writeSecond := []syntax.Slide{{Ops: [2]syntax.Operand{num(1)}}}
	read := syntax.Operand{Type: syntax.Read}
	write := syntax.Operand{Type: syntax.Write}
	got, err := runHand(t, hand([]int32{'A', 'B'}, map[syntax.Category][]syntax.Slide{syntax.Reads: reads, syntax.Writes: writeFirst}, read, write), "Z")
	if err != nil || got != "Z" {
		t.Errorf("first cell after reading: %q, %v; want \"Z\"", got, err)
	}
	got, err = runHand(t, hand([]int32{'A', 'B'}, map[syntax.Category][]syntax.Slide{syntax.Reads: reads, syntax.Writes: writeSecond}, read, write), "Z")
	if err != nil || got != "B" {
		t.Errorf("second cell after reading: %q, %v; want unchanged \"B\"", got, err)
	}
	// At the end of input the read stores -1, a character write prints 0xFF.
	got, err = runHand(t, hand([]int32{'A'}, map[syntax.Category][]syntax.Slide{syntax.Reads: reads, syntax.Writes: writeFirst}, read, write), "")
	if err != nil || got != "\xff" {
		t.Errorf("end of input: %q, %v; want \"\\xff\"", got, err)
	}
}

// In Very Sorted!, a read stores into the cell it declares.
func TestVeryReadStoresIntoDeclaredCell(t *testing.T) {
	reads := []syntax.Slide{{Ops: [2]syntax.Operand{num(1)}}} // "reads the second number"
	writeSecond := []syntax.Slide{{Ops: [2]syntax.Operand{num(1)}}}
	writeFirst := []syntax.Slide{{Ops: [2]syntax.Operand{num(0)}}}
	read := syntax.Operand{Type: syntax.Read}
	write := syntax.Operand{Type: syntax.Write}
	for _, tt := range []struct {
		writes []syntax.Slide
		want   string
	}{{writeSecond, "Z"}, {writeFirst, "A"}} {
		p := hand([]int32{'A', 'B'}, map[syntax.Category][]syntax.Slide{syntax.Reads: reads, syntax.Writes: tt.writes}, read, write)
		p.Verys = 1
		if got, err := runHand(t, p, "Z"); err != nil || got != tt.want {
			t.Errorf("got %q, %v; want %q", got, err, tt.want)
		}
	}
}

// Very Sorted! stores into the cell a sum indexes, Data[v], and reads
// Data[v-1] through it, as through a cell (#27). The original's store into
// it is a run-time error (a parse never yields one).
func TestVeryIndexedStore(t *testing.T) {
	sums := []syntax.Slide{{Ops: [2]syntax.Operand{num(0), num(0)}}} // 1 + 1 = 2
	idx := syntax.Operand{Type: syntax.Sum | syntax.Indirect}
	assigns := []syntax.Slide{{Ops: [2]syntax.Operand{num(3), idx}}} // 'Z' to Data[2]
	writes := []syntax.Slide{{Ops: [2]syntax.Operand{num(2)}}, {Ops: [2]syntax.Operand{idx}}}
	a, w := syntax.Operand{Type: syntax.Assign}, syntax.Operand{Type: syntax.Write}
	w1 := syntax.Operand{Type: syntax.Write, Index: 1}
	for _, tt := range []struct {
		verys int
		want  string
		err   bool
	}{{1, "ZB", false}, {2, "ZB", false}, {0, "", true}} {
		p := hand([]int32{1, 'B', 'C', 'Z'}, map[syntax.Category][]syntax.Slide{syntax.Sums: sums, syntax.Assigns: assigns, syntax.Writes: writes}, a, w, w1)
		p.Verys = tt.verys
		got, err := runHand(t, p, "")
		if got != tt.want || (err != nil) != tt.err {
			t.Errorf("verys %d: %q, %v; want %q", tt.verys, got, err, tt.want)
		}
	}
}

// Very Sorted! prints German numbers in UTF-8 (#28); the original's spelling
// stays the original's, and English is English.
func TestVeryGermanSpelling(t *testing.T) {
	writes := []syntax.Slide{
		{Ops: [2]syntax.Operand{num(0)}, Flags: syntax.FormatGermanCardinal},
		{Ops: [2]syntax.Operand{num(1)}, Flags: syntax.FormatGermanOrdinal},
		{Ops: [2]syntax.Operand{num(0)}, Flags: syntax.FormatEnglishCardinal},
	}
	w := func(i int) syntax.Operand { return syntax.Operand{Type: syntax.Write, Index: int32(i)} }
	for _, tt := range []struct {
		verys int
		want  string
	}{
		{0, "fuenfunddreissig\nzwoelfte\nthirtyfive\n"},
		{1, "fünfunddreißig\nzwölfte\nthirtyfive\n"},
		{2, "fünfunddreißig\nzwölfte\nthirtyfive\n"},
	} {
		p := hand([]int32{35, 12}, map[syntax.Category][]syntax.Slide{syntax.Writes: writes}, w(0), w(1), w(2))
		p.Verys = tt.verys
		if got, err := runHand(t, p, ""); err != nil || got != tt.want {
			t.Errorf("verys %d: %q, %v; want %q", tt.verys, got, err, tt.want)
		}
	}
}

// Very Very Sorted! writes Italian numbers (#30), any number at all.
func TestItalianOutput(t *testing.T) {
	writes := []syntax.Slide{
		{Ops: [2]syntax.Operand{num(0)}, Flags: syntax.FormatItalianCardinal},
		{Ops: [2]syntax.Operand{num(0)}, Flags: syntax.FormatItalianOrdinal},
		{Ops: [2]syntax.Operand{num(1)}, Flags: syntax.FormatItalianCardinal},
		{Ops: [2]syntax.Operand{num(1)}, Flags: syntax.FormatItalianOrdinal},
		{Ops: [2]syntax.Operand{num(2)}, Flags: syntax.FormatItalianCardinal},
		{Ops: [2]syntax.Operand{num(2)}, Flags: syntax.FormatItalianOrdinal},
	}
	w := func(i int) syntax.Operand { return syntax.Operand{Type: syntax.Write, Index: int32(i)} }
	p := hand([]int32{23, 0, -7}, map[syntax.Category][]syntax.Slide{syntax.Writes: writes}, w(0), w(1), w(2), w(3), w(4), w(5))
	p.Verys = 2
	want := "ventitré\nventitreesimo\nzero\nzeresimo\nmeno sette\nmeno settimo\n"
	if got, err := runHand(t, p, ""); err != nil || got != want {
		t.Errorf("%q, %v; want %q", got, err, want)
	}
}

// "Logical operations" compute ~a & ~b (NOR); Very Sorted!'s NAND (#26)
// computes ~(a & b).
func TestLogicalOperation(t *testing.T) {
	for _, tt := range []struct {
		flags int32
		want  string
	}{{syntax.LogicalNor, "\xf1"}, {syntax.LogicalNand, "\xf7"}} { // ~12 & ~10 == -15, ~(12 & 10) == -9
		nands := []syntax.Slide{{Ops: [2]syntax.Operand{num(0), num(1)}, Flags: tt.flags}}
		writes := []syntax.Slide{{Ops: [2]syntax.Operand{{Type: syntax.Nand}}}}
		got, err := runHand(t, hand([]int32{12, 10}, map[syntax.Category][]syntax.Slide{syntax.Nands: nands, syntax.Writes: writes}, syntax.Operand{Type: syntax.Write}), "")
		if err != nil || got != tt.want {
			t.Errorf("flags %d: %q, %v; want %q", tt.flags, got, err, tt.want)
		}
	}
}

// Cells 2, 'A', 'B': an indirect write via the first cell stores into
// Data[2] (the third cell), an indirect read via it reads Data[1].
func TestIndirectAsymmetry(t *testing.T) {
	write := sentences{
		"numbers": "This code uses the numbers two, sixtyfive, and sixtysix.",
		"writes":  "This code writes the third number as a character.",
		"assigns": "This code assigns the first number to the cell indexed by the first number.",
		"impl":    "This code implements the first assignment, and the first output.",
	}
	if got, err := run(t, program(t, write), ""); err != nil || got != "\x02" {
		t.Errorf("indirect write: %q, %v; want the third cell overwritten with 2", got, err)
	}
	read := sentences{
		"numbers": "This code uses the numbers two, sixtyfive, and sixtysix.",
		"writes":  "This code writes the cell indexed by the first number as a character.",
		"impl":    "This code implements the first output.",
	}
	if got, err := run(t, program(t, read), ""); err != nil || got != "A" {
		t.Errorf("indirect read: %q, %v; want the second cell", got, err)
	}
}

func TestJumps(t *testing.T) {
	// The first cell is zero, so the jump cannot depend on the zeroed second
	// operand that an unconditional jump leaves unused.
	t.Run("jump skips to after the label", func(t *testing.T) {
		got, err := run(t, program(t, sentences{
			"numbers": "This code uses the numbers zero, and sixtyfive.",
			"jumps":   "This code always goes to the first label.",
			"writes":  "This code writes the second number as a character.",
			"labels":  "This code uses one label.",
			"impl":    "This code implements the first jump, the first output, the first label, and the first output.",
		}), "")
		if err != nil || got != "A" {
			t.Errorf("%q, %v; want one \"A\"", got, err)
		}
	})
	// A declared label that is never placed is 0: the jump continues at
	// statement 1, so statement 0 runs once and the program loops forever.
	t.Run("unplaced label skips statement zero", func(t *testing.T) {
		var out bytes.Buffer
		err := Run(program(t, sentences{
			"numbers": "This code uses the number sixtyfive.",
			"jumps":   "This code always goes to the first label.",
			"writes":  "This code writes the first number as a character.",
			"labels":  "This code uses one label.",
			"impl":    "This code implements the first output, the first jump, and the first jump.",
		}), strings.NewReader(""), &out, 100)
		if !errors.Is(err, ErrStepLimit) || out.String() != "A" {
			t.Errorf("%q, %v; want one \"A\" and the step limit", out.String(), err)
		}
	})
}

// References past the end of a table read the next slots of the Code array.
func TestReferencesPastTheTable(t *testing.T) {
	// The read entry follows the only output: "the second output" writes the
	// read's operand, the first number, as a character (the read's flags).
	got, err := run(t, program(t, sentences{
		"numbers": "This code uses the numbers sixtyfive, and sixtysix.",
		"writes":  "This code writes the second number as a english cardinal.",
		"reads":   "This code reads the first number as a character.",
		"impl":    "This code implements the second output.",
	}), "")
	if err != nil || got != "A" {
		t.Errorf("second output: %q, %v; want \"A\"", got, err)
	}
	// Beyond the parsed code the slots are zero: write cell 0 as a character.
	got, err = run(t, program(t, sentences{
		"numbers": "This code uses the number sixtysix.",
		"impl":    "This code implements the ninth output.",
	}), "")
	if err != nil || got != "B" {
		t.Errorf("ninth output: %q, %v; want \"B\"", got, err)
	}
}

func TestStatementsWithoutEffect(t *testing.T) {
	got, err := run(t, program(t, sentences{
		"numbers": "This code uses the number sixtyfive.",
		"writes":  "This code writes the first number as a character.",
		"impl":    "This code implements the first number, the cell indexed by the first number, the first sum, and the first output.",
	}), "")
	if err != nil || got != "A" {
		t.Errorf("%q, %v; want \"A\"", got, err)
	}
}

// An indirect expression reads the cell its value indexes (port-defined).
func TestIndirectExpression(t *testing.T) {
	got, err := run(t, program(t, sentences{
		"numbers": "This code uses the numbers one, sixtyfive, and sixtysix.",
		"writes":  "This code writes the cell indexed by the first sum as a character.",
		"sums":    "This code uses the sum of the first number and the first number.",
		"impl":    "This code implements the first output.",
	}), "")
	if err != nil || got != "A" {
		t.Errorf("%q, %v; want \"A\" (cell 1 + 1 - 1)", got, err)
	}
}

func TestRuntimeErrors(t *testing.T) {
	t.Run("recursion", func(t *testing.T) {
		_, err := run(t, program(t, sentences{
			"numbers": "This code uses the number one.",
			"writes":  "This code writes the first sum as a character.",
			"sums":    "This code uses the sum of the first sum and the first number.",
			"impl":    "This code implements the first output.",
		}), "")
		runtimeError(t, err, "recursion too deep")
	})
	t.Run("cell out of range", func(t *testing.T) {
		_, err := run(t, program(t, sentences{
			"numbers": "This code uses the number zero.",
			"writes":  "This code writes the cell indexed by the first number as a character.",
			"impl":    "This code implements the first output.",
		}), "")
		runtimeError(t, err, "cell out of range")
	})
	t.Run("store into a non-cell", func(t *testing.T) {
		p := &syntax.Program{
			Code: []syntax.Slide{
				{Ops: [2]syntax.Operand{{Type: syntax.Number}, {Type: syntax.Sum}}},
				{Ops: [2]syntax.Operand{{Type: syntax.Assign}}},
			},
		}
		p.Tables[syntax.Assigns] = syntax.Table{Count: 1, Index: 0}
		p.Tables[syntax.Statements] = syntax.Table{Count: 1, Index: 1}
		runtimeError(t, Run(p, strings.NewReader(""), &bytes.Buffer{}, steps), "store into something that is not a cell")
	})
	t.Run("code slot out of range", func(t *testing.T) {
		p := &syntax.Program{Code: []syntax.Slide{{Ops: [2]syntax.Operand{{Type: syntax.Write, Index: codeSlots}}}}}
		p.Tables[syntax.Statements] = syntax.Table{Count: 1, Index: 0}
		runtimeError(t, Run(p, strings.NewReader(""), &bytes.Buffer{}, steps), "code slot out of range")
	})
}

func TestGetchar(t *testing.T) {
	in := &stdin{r: newReader("a\r\nb\rc\x1ad")}
	var got []int32
	for i := 0; i < 8; i++ {
		got = append(got, in.getchar())
	}
	want := []int32{'a', '\n', 'b', '\r', 'c', -1, -1, -1}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("getchar sequence %v, want %v", got, want)
		}
	}
}

func newReader(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

func TestHandBuiltEdges(t *testing.T) {
	write := syntax.Operand{Type: syntax.Write}
	t.Run("MinInt32 / -1 wraps", func(t *testing.T) {
		ratios := []syntax.Slide{{Ops: [2]syntax.Operand{num(0), num(1)}}}
		conds := []syntax.Slide{{Ops: [2]syntax.Operand{{Type: syntax.Ratio}, num(0)}, Flags: syntax.CompareEqual}}
		writes := []syntax.Slide{{Ops: [2]syntax.Operand{{Type: syntax.Condition}}, Flags: syntax.FormatEnglishCardinal}}
		got, err := runHand(t, hand([]int32{-2147483648, -1}, map[syntax.Category][]syntax.Slide{syntax.Ratios: ratios, syntax.Conditions: conds, syntax.Writes: writes}, write), "")
		if err != nil || got != "one\n" {
			t.Errorf("%q, %v; want \"one\\n\"", got, err)
		}
	})
	t.Run("english ordinal output", func(t *testing.T) {
		writes := []syntax.Slide{{Ops: [2]syntax.Operand{num(0)}, Flags: syntax.FormatEnglishOrdinal}}
		got, err := runHand(t, hand([]int32{12}, map[syntax.Category][]syntax.Slide{syntax.Writes: writes}, write), "")
		if err != nil || got != "twelveth\n" {
			t.Errorf("%q, %v; want \"twelveth\\n\"", got, err)
		}
	})
	t.Run("indirect write out of range", func(t *testing.T) {
		assigns := []syntax.Slide{{Ops: [2]syntax.Operand{num(0), {Type: syntax.Number | syntax.Indirect}}}}
		_, err := runHand(t, hand([]int32{-5}, map[syntax.Category][]syntax.Slide{syntax.Assigns: assigns}, syntax.Operand{Type: syntax.Assign}), "")
		runtimeError(t, err, "cell out of range")
		if err == nil || err.Error() != "runtime error: cell out of range" {
			t.Errorf("message %v", err)
		}
	})
}
