package emit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// runC compiles C source with the host compiler and runs it with stdin. It
// skips the test when there is no C compiler.
func runC(t *testing.T, src, stdin string) (stdout, stderr string, code int) {
	t.Helper()
	compiler, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler (cc) on PATH")
	}
	dir := t.TempDir()
	file, bin := filepath.Join(dir, "prog.c"), filepath.Join(dir, "prog")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(compiler, "-std=c17", "-w", "-o", bin, file).CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s\n%s", err, out, src)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin)
	cmd.Stdin = strings.NewReader(stdin)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err = cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return o.String(), e.String(), code
}

// sameAsInterpreter checks that Exact(p) prints what the interpreter prints,
// and fails where it fails, with the same message.
func sameAsInterpreter(t *testing.T, p *syntax.Program, stdin string) {
	t.Helper()
	var want bytes.Buffer
	err := interp.Run(p, strings.NewReader(stdin), &want, 1000000)
	if errors.Is(err, interp.ErrStepLimit) {
		t.Fatal("the test program does not end")
	}
	wantErr, wantCode := "", 0
	if err != nil {
		wantErr, wantCode = err.Error()+"\n", 1
	}
	src := Exact(p)
	got, gotErr, code := runC(t, src, stdin)
	if got != want.String() || gotErr != wantErr || code != wantCode {
		t.Errorf("C printed %q, stderr %q, exit %d\ninterpreter printed %q, error %q\n%s", got, gotErr, code, want.String(), wantErr, src)
	}
}

func TestExactSamples(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) { sameAsInterpreter(t, parseSample(t, name), "") })
	}
}

// The number words in C against internal/numbers, for every format.
// TestExactWords compares the C number words with internal/numbers, in the
// 2000 spelling and in Very Sorted!'s (German in UTF-8, #28).
func TestExactWords(t *testing.T) {
	for _, very := range []bool{false, true} {
		t.Run(fmt.Sprintf("very=%v", very), func(t *testing.T) { exactWordsTest(t, very) })
	}
}

func exactWordsTest(t *testing.T, very bool) {
	values := []int32{-5, -1, 0, 1000000000, 2000000000, 2147483647}
	for v := int32(1); v <= 120; v++ {
		values = append(values, v)
	}
	values = append(values, 199, 200, 201, 219, 999, 1000, 1001, 1015, 1100, 2002, 12345, 19999, 20000, 99999, 100000, 101101, 999999,
		1000000, 1000001, 1001000, 7000013, 12012012, 999999999, 1000000001, 1234567890)
	var b strings.Builder
	b.WriteString(exactHeader + "static I _[193719];\n" + exactRuntime + words(very) + "int main(void) {\n")
	var want strings.Builder
	for _, v := range values {
		for f := int32(1); f <= 4; f++ {
			if v < 0 && (f == 1 || f == 3) {
				continue // a runtime error, see TestExactQuirks
			}
			fmt.Fprintf(&b, "\tw_(%d, %d);\n", f, v)
			var s string
			switch f {
			case 1:
				s, _ = numbers.EnglishCardinal(v)
			case 2:
				s, _ = numbers.EnglishOrdinal(v)
			case 3:
				s, _ = numbers.GermanCardinal(v)
			case 4:
				s, _ = numbers.GermanOrdinal(v)
			}
			if very && f >= 3 {
				s = numbers.VerySpelling(s)
			}
			want.WriteString(s + "\n")
		}
	}
	b.WriteString("\treturn 0;\n}\n")
	got, stderr, code := runC(t, b.String(), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want.String(), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("%d lines, want %d", len(gotLines), len(wantLines))
	}
	for i := range gotLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d: %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

// TestExactItalianWords compares the C Italian number words with
// internal/numbers (#30).
func TestExactItalianWords(t *testing.T) {
	values := []int32{math.MinInt32, -1000000, -23, -1, 0, 1000000000, 2000000000, math.MaxInt32}
	for v := int32(1); v <= 1200; v++ {
		values = append(values, v)
	}
	for v := int64(1201); v <= math.MaxInt32; v = v*7/5 + 3 {
		values = append(values, int32(v), int32(v/1000*1000), int32(v/1000000*1000000))
	}
	var b, want strings.Builder
	b.WriteString(exactHeader + "static I _[193719];\n" + exactRuntime + words(true) + exactItalian + "int main(void) {\n")
	for _, v := range values {
		fmt.Fprintf(&b, "\twi_(5, %d);\n\twi_(6, %d);\n", v, v)
		want.WriteString(numbers.ItalianCardinal(v) + "\n" + numbers.ItalianOrdinal(v, false) + "\n")
	}
	b.WriteString("\treturn 0;\n}\n")
	got, stderr, code := runC(t, b.String(), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want.String(), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("%d lines, want %d", len(gotLines), len(wantLines))
	}
	for i := range gotLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d: %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

// TestExactVaudoisWords compares the C Vaudois French number words with
// internal/numbers (#29).
func TestExactVaudoisWords(t *testing.T) {
	values := []int32{math.MinInt32, -1000000, -91, -1, 0, 1000000000, 2000000000, math.MaxInt32,
		200000000, -200000000, 300000000, 200000003, 200001000, 900000000}
	for v := int32(1); v <= 1200; v++ {
		values = append(values, v)
	}
	for v := int64(1201); v <= math.MaxInt32; v = v*7/5 + 3 {
		values = append(values, int32(v), int32(v/1000*1000), int32(v/1000000*1000000))
	}
	var b, want strings.Builder
	b.WriteString(exactHeader + "static I _[193719];\n" + exactRuntime + exactVaudois + "int main(void) {\n")
	for _, v := range values {
		fmt.Fprintf(&b, "\twf_(7, %d);\n\twf_(8, %d);\n", v, v)
		want.WriteString(numbers.VaudoisCardinal(v) + "\n" + numbers.VaudoisOrdinal(v, false) + "\n")
	}
	b.WriteString("\treturn 0;\n}\n")
	got, stderr, code := runC(t, b.String(), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want.String(), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("%d lines, want %d", len(gotLines), len(wantLines))
	}
	for i := range gotLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d: %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

// TestExactBrazilianWords compares the C Brazilian Portuguese number words
// with internal/numbers (#31).
func TestExactBrazilianWords(t *testing.T) {
	values := []int32{math.MinInt32, -1000000, -91, -1, 0, 1000000000, 2000000000, math.MaxInt32,
		1001, 1100, 1101, 1200000, 1234000, 2001234, 1000100, 200000, 21000}
	for v := int32(1); v <= 2100; v++ {
		values = append(values, v)
	}
	for v := int64(2101); v <= math.MaxInt32; v = v*7/5 + 3 {
		values = append(values, int32(v), int32(v/1000*1000), int32(v/1000000*1000000), int32(v/1000000*1000000+v%1000))
	}
	var b, want strings.Builder
	b.WriteString(exactHeader + "static I _[193719];\n" + exactRuntime + exactBrazilian + "int main(void) {\n")
	for _, v := range values {
		fmt.Fprintf(&b, "\twp_(9, %d);\n\twp_(10, %d);\n", v, v)
		want.WriteString(numbers.BrazilianCardinal(v) + "\n" + numbers.BrazilianOrdinal(v, false) + "\n")
	}
	b.WriteString("\treturn 0;\n}\n")
	got, stderr, code := runC(t, b.String(), "")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want.String(), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("%d lines, want %d", len(gotLines), len(wantLines))
	}
	for i := range gotLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line %d: %q, want %q", i, gotLines[i], wantLines[i])
		}
	}
}

// builder lays out tables by hand: each category gets a block of slots.
type builder struct{ p syntax.Program }

func (b *builder) table(c syntax.Category, slides ...syntax.Slide) {
	b.p.Tables[c] = syntax.Table{Count: len(slides), Index: len(b.p.Code)}
	b.p.Code = append(b.p.Code, slides...)
}

func op(t syntax.OperandType, i int32) syntax.Operand { return syntax.Operand{Type: t, Index: i} }

func slide(a, b syntax.Operand, flags int32) syntax.Slide {
	return syntax.Slide{Ops: [2]syntax.Operand{a, b}, Flags: flags}
}

func stmt(t syntax.OperandType, i int32) syntax.Slide { return slide(op(t, i), syntax.Operand{}, 0) }

// Quirks and runtime errors, each a small program built from tables.
func TestExactQuirks(t *testing.T) {
	const (
		N  = syntax.Number
		S  = syntax.Sum
		D  = syntax.Diff
		P  = syntax.Prod
		R  = syntax.Ratio
		L  = syntax.Nand
		C  = syntax.Condition
		LB = syntax.Label
		IN = syntax.Indirect
	)
	write := func(f int32, o syntax.Operand) syntax.Slide { return slide(o, syntax.Operand{}, f) }
	assign := func(src, dst syntax.Operand) syntax.Slide { return slide(src, dst, 0) }
	cases := map[string]func(b *builder){
		"arithmetic": func(b *builder) {
			b.p.Data = []int32{2147483647, 1, -7, 2, -2147483648, -1, 65}
			b.table(syntax.Sums, slide(op(N, 0), op(N, 1), 0), slide(op(N, 6), op(R, 0), 0))
			b.table(syntax.Diffs, slide(op(N, 4), op(N, 1), 0))
			b.table(syntax.Prods, slide(op(N, 0), op(N, 3), 0))
			b.table(syntax.Ratios, slide(op(N, 2), op(N, 3), 0), slide(op(N, 4), op(N, 5), 0))
			b.table(syntax.Nands, slide(op(N, 3), op(N, 2), 0))
			b.table(syntax.Writes,
				write(syntax.FormatCharacter, op(S, 1)),             // 65 + -7/2 = 62 '>'
				write(syntax.FormatCharacter, op(S, 0)),             // wraps: low byte of INT_MIN
				write(syntax.FormatEnglishOrdinal, op(D, 0)),        // INT_MIN - 1 wraps to INT_MAX
				write(syntax.FormatGermanOrdinal, op(P, 0)),         // INT_MAX * 2 = -2: ERROR
				write(syntax.FormatEnglishOrdinal, op(R, 1)),        // INT_MIN / -1 = INT_MIN: ERROR
				write(syntax.FormatGermanCardinal, op(L, 0)),        // ~2 & ~-7 = 4
				write(syntax.FormatCharacter, op(syntax.Assign, 0)), // no value: 0
			)
			b.table(syntax.Conditions)
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2), stmt(syntax.Write, 3),
				stmt(syntax.Write, 4), stmt(syntax.Write, 5), stmt(syntax.Write, 6))
		},
		"indirection and conditions": func(b *builder) {
			// _[_[3]] = 70 writes cell 4 (0-based); the read of indirect 5
			// finds cell 4 (1-based).
			b.p.Data = []int32{70, 3, 4, 4, 0, 5}
			b.table(syntax.Conditions, slide(op(N, 2), op(N, 3), syntax.CompareEqual), slide(op(N, 1), op(N, 2), syntax.CompareLess),
				slide(op(N, 2), op(N, 1), syntax.CompareLess))
			b.table(syntax.Sums, slide(op(C, 0), op(C, 1), 0), slide(op(C, 2), op(N|IN, 5), 0))
			b.table(syntax.Assigns, assign(op(N, 0), op(N|IN, 3)))
			b.table(syntax.Writes, write(syntax.FormatEnglishCardinal, op(S, 0)), write(syntax.FormatEnglishCardinal, op(S, 1)))
			b.table(syntax.Statements, stmt(syntax.Assign, 0), stmt(syntax.Write, 0), stmt(syntax.Write, 1))
		},
		"past the end of a table": func(b *builder) {
			// The second sum is the first difference's slot, read as a sum;
			// the fourth number is past the declared three.
			b.p.Data = []int32{10, 20, 30}
			b.table(syntax.Sums, slide(op(N, 0), op(N, 1), 0))
			b.table(syntax.Diffs, slide(op(N, 2), op(N, 0), 0))
			b.table(syntax.Writes, write(syntax.FormatEnglishCardinal, op(S, 1)), write(syntax.FormatEnglishCardinal, op(N, 3)),
				write(syntax.FormatEnglishCardinal, op(S, 7))) // a slot past everything: zeros, 10 + 10
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2))
		},
		"labels and jumps": func(b *builder) {
			// Count down from 3. At 0, a jump to a label that is never placed
			// goes to 0, so the program continues at statement 1 and skips
			// statement 0, which would end it now: one more round prints
			// "0". Label 1 is placed twice; the last place counts, so the
			// write between the two never runs.
			b.p.Data = []int32{3, 1, 0, 48}
			b.table(syntax.Diffs, slide(op(N, 0), op(N, 1), 0))
			b.table(syntax.Sums, slide(op(N, 0), op(N, 3), 0))
			b.table(syntax.Conditions, slide(op(N, 2), op(N, 0), syntax.CompareLess), slide(op(N, 0), op(N, 2), syntax.CompareEqual),
				slide(op(N, 0), op(N, 2), syntax.CompareLess))
			b.table(syntax.Assigns, assign(op(D, 0), op(N, 0)))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(S, 0)))
			b.table(syntax.Jumps, slide(op(LB, 0), op(C, 0), syntax.ConditionalJump), slide(op(LB, 5), syntax.Operand{}, syntax.UnconditionalJump),
				slide(op(LB, 1), op(C, 1), syntax.ConditionalJump), slide(op(LB, 1), op(C, 2), syntax.ConditionalJump))
			b.table(syntax.Statements,
				stmt(syntax.Jump, 2), // 0: to the end when n == 0
				stmt(LB, 0),          // 1
				stmt(syntax.Write, 0),
				stmt(syntax.Assign, 0),                          // n--
				stmt(syntax.Jump, 0),                            // 4: back while 0 < n
				stmt(syntax.Jump, 3),                            // 5: to the end when n < 0
				stmt(syntax.Jump, 1),                            // 6: to the label never placed
				stmt(LB, 1), stmt(syntax.Write, 0), stmt(LB, 1)) // the write after the first place never runs
		},
		"a jump that is not to a label": func(b *builder) {
			// Statement 1 implements the second jump of one: statement 0's
			// slot read as a jump, taken while cell 2 is not 0, to the first
			// number. pc -4 + 1 runs the slots before the statements: the
			// conditions table, filled with statements here, writes "B" and
			// clears cell 2.
			b.p.Data = []int32{-4, 66, 1, 0}
			b.table(syntax.Assigns, assign(op(N, 3), op(N, 2)))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(N, 1)))
			b.table(syntax.Conditions, stmt(syntax.Write, 0), stmt(syntax.Assign, 0))
			b.table(syntax.Jumps, slide(op(LB, 0), syntax.Operand{}, syntax.UnconditionalJump))
			b.table(syntax.Statements, slide(op(N, 0), op(N, 2), syntax.ConditionalJump), stmt(syntax.Jump, 1))
		},
		"very input": func(b *builder) {
			// Very Sorted! reads into the declared cell (here the second);
			// the original would read into the first.
			b.p.Verys = 1
			b.p.Data = []int32{'A', 'B'}
			b.table(syntax.Reads, stmt(N, 1))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(N, 1)), write(syntax.FormatCharacter, op(N, 0)))
			b.table(syntax.Statements, stmt(syntax.Read, 0), stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Read, 0), stmt(syntax.Write, 0))
		},
		"very indexed store": func(b *builder) {
			// Very Sorted! stores into the cell a sum indexes (#27), Data[2],
			// and reads Data[1] through it.
			b.p.Verys = 1
			b.p.Data = []int32{1, 'B', 'C', 'Z'}
			b.table(syntax.Sums, slide(op(N, 0), op(N, 0), 0))
			b.table(syntax.Assigns, slide(op(N, 3), op(syntax.Sum|syntax.Indirect, 0), 0))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(N, 2)), write(syntax.FormatCharacter, op(syntax.Sum|syntax.Indirect, 0)))
			b.table(syntax.Statements, stmt(syntax.Assign, 0), stmt(syntax.Write, 0), stmt(syntax.Write, 1))
		},
		"nand and nor": func(b *builder) {
			// Very Sorted!'s NAND and the original's NOR (#26); the third
			// reference reads past the table into a condition's slot, whose
			// flags (CompareLess, 1) make it a NAND.
			b.p.Verys = 1
			b.p.Data = []int32{12, 10, 100}
			b.table(syntax.Nands, slide(op(N, 0), op(N, 1), syntax.LogicalNand), slide(op(N, 0), op(N, 1), syntax.LogicalNor))
			b.table(syntax.Conditions, slide(op(N, 2), op(N, 1), syntax.CompareLess))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(syntax.Nand, 0)), write(syntax.FormatCharacter, op(syntax.Nand, 1)), write(syntax.FormatCharacter, op(syntax.Nand, 2)))
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2))
		},
		"very german numbers": func(b *builder) {
			// Very Sorted! prints German numbers in UTF-8 (#28).
			b.p.Verys = 1
			b.p.Data = []int32{35, 12, 1005, 30}
			b.table(syntax.Writes, write(syntax.FormatGermanCardinal, op(N, 0)), write(syntax.FormatGermanOrdinal, op(N, 1)),
				write(syntax.FormatGermanCardinal, op(N, 2)), write(syntax.FormatGermanOrdinal, op(N, 3)), write(syntax.FormatEnglishCardinal, op(N, 2)))
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2), stmt(syntax.Write, 3), stmt(syntax.Write, 4))
		},
		"italian numbers": func(b *builder) {
			// Very Very Sorted! prints Italian numbers (#30), negative ones too.
			b.p.Verys = 2
			b.p.Data = []int32{23, 0, -1000000, 1000000}
			b.table(syntax.Writes, write(syntax.FormatItalianCardinal, op(N, 0)), write(syntax.FormatItalianOrdinal, op(N, 0)),
				write(syntax.FormatItalianOrdinal, op(N, 1)), write(syntax.FormatItalianCardinal, op(N, 2)), write(syntax.FormatItalianOrdinal, op(N, 3)))
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2), stmt(syntax.Write, 3), stmt(syntax.Write, 4))
		},
		"vaudois numbers": func(b *builder) {
			// Very Very Sorted! prints Vaudois French numbers (#29).
			b.p.Verys = 2
			b.p.Data = []int32{91, 0, -200, 1000000}
			b.table(syntax.Writes, write(syntax.FormatVaudoisCardinal, op(N, 0)), write(syntax.FormatVaudoisOrdinal, op(N, 0)),
				write(syntax.FormatVaudoisOrdinal, op(N, 1)), write(syntax.FormatVaudoisCardinal, op(N, 2)), write(syntax.FormatVaudoisOrdinal, op(N, 3)))
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2), stmt(syntax.Write, 3), stmt(syntax.Write, 4))
		},
		"brazilian numbers": func(b *builder) {
			// Very Very Sorted! prints Brazilian Portuguese numbers (#31).
			b.p.Verys = 2
			b.p.Data = []int32{23, 0, -1234567, 1000000}
			b.table(syntax.Writes, write(syntax.FormatBrazilianCardinal, op(N, 0)), write(syntax.FormatBrazilianOrdinal, op(N, 0)),
				write(syntax.FormatBrazilianOrdinal, op(N, 1)), write(syntax.FormatBrazilianCardinal, op(N, 2)), write(syntax.FormatBrazilianOrdinal, op(N, 3)))
			b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1), stmt(syntax.Write, 2), stmt(syntax.Write, 3), stmt(syntax.Write, 4))
		},
		"a jump below the code": func(b *builder) {
			b.p.Data = []int32{-100}
			b.table(syntax.Jumps, slide(op(N, 0), syntax.Operand{}, syntax.UnconditionalJump))
			b.table(syntax.Statements, stmt(syntax.Jump, 0))
		},
		"input": func(b *builder) {
			b.table(syntax.Reads, stmt(syntax.Read, 0))
			// ordinals, as the end of the input is -1 (ERROR, not a crash)
			b.table(syntax.Writes, write(syntax.FormatEnglishOrdinal, op(N, 0)))
			var s []syntax.Slide
			for range 9 { // past the Ctrl-Z, which ends the input for good
				s = append(s, stmt(syntax.Read, 0), stmt(syntax.Write, 0))
			}
			b.table(syntax.Statements, s...)
		},
		"division by zero":     func(b *builder) { b.p.Data = []int32{1, 0}; ratioWrite(b, op(N, 0), op(N, 1)) },
		"cell out of range":    func(b *builder) { b.p.Data = []int32{0}; ratioWrite(b, op(N|IN, 0), op(N, 0)) },
		"cell past the memory": func(b *builder) { ratioWrite(b, op(N, 193719), op(N, 0)) },
		"negative cardinal": func(b *builder) {
			b.p.Data = []int32{-1}
			b.table(syntax.Writes, write(syntax.FormatEnglishCardinal, op(N, 0)))
			b.table(syntax.Statements, stmt(syntax.Write, 0))
		},
		"a sum that contains itself": func(b *builder) {
			b.table(syntax.Sums, slide(op(N, 0), op(S, 0), 0))
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(S, 0)))
			b.table(syntax.Statements, stmt(syntax.Write, 0))
		},
		"store into a sum": func(b *builder) {
			b.table(syntax.Assigns, assign(op(N, 0), op(S, 0)))
			b.table(syntax.Statements, stmt(syntax.Assign, 0))
		},
		"store past the memory": func(b *builder) {
			b.p.Data = []int32{193719}
			b.table(syntax.Assigns, assign(op(N, 0), op(N|IN, 0)))
			b.table(syntax.Statements, stmt(syntax.Assign, 0))
		},
		"a statement past the code": func(b *builder) {
			b.table(syntax.Statements, stmt(syntax.Assign, 200000))
		},
		"statements past the last code slot": func(b *builder) {
			// The statements' last slot is the code's last; the next one does
			// not exist, which fails when the program counter gets there.
			b.p.Data = []int32{72}
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(N, 0)))
			b.p.Code = append(b.p.Code, make([]syntax.Slide, codeSlots-1)...)
			b.p.Code[codeSlots-1] = stmt(syntax.Write, 0)
			b.p.Tables[syntax.Statements] = syntax.Table{Count: 3, Index: codeSlots - 2}
		},
		"code slot out of range": func(b *builder) {
			b.table(syntax.Writes, write(syntax.FormatCharacter, op(S, 200000)))
			b.table(syntax.Statements, stmt(syntax.Write, 0))
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			b := &builder{}
			build(b)
			sameAsInterpreter(t, &b.p, "ab\r\nc\rd\x1Aef")
		})
	}
}

// ratioWrite writes a / b as a character, output first so that the error
// comes after it.
func ratioWrite(b *builder, a, z syntax.Operand) {
	b.table(syntax.Ratios, slide(a, z, 0))
	b.table(syntax.Writes, slide(op(syntax.Number, 0), syntax.Operand{}, syntax.FormatCharacter), slide(op(syntax.Ratio, 0), syntax.Operand{}, syntax.FormatCharacter))
	b.table(syntax.Statements, stmt(syntax.Write, 0), stmt(syntax.Write, 1))
}
