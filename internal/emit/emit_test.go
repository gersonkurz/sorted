package emit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gersonkurz/sorted/internal/syntax"
)

func readFile(t *testing.T, path ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, path...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func parseSample(t *testing.T, name string) *syntax.Program {
	t.Helper()
	p, err := syntax.Parse(syntax.Filter([]byte(readFile(t, "legacy", "sorted.win32", name+".s"))))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// golden reads a capture from Sorted.exe. The original writes its files in
// text mode, so only CRLF is normalised; everything else must match exactly.
func golden(t *testing.T, name string) string {
	t.Helper()
	return strings.ReplaceAll(readFile(t, "testdata", "golden", name), "\r\n", "\n")
}

// The /C output of all four samples, byte for byte, against what Sorted.exe
// generated for them (testdata/golden/<sample>.c, captured 2026-10-04).
func TestCMatchesCaptures(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) {
			if got, want := C(parseSample(t, name)), golden(t, name+".c"); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// The /D dump of all four samples, byte for byte (testdata/golden/<sample>.dump,
// captured 2026-10-04 and 2026-10-05). ELEMENTS exceeding the entries
// (hallo: 26 for 23) pins the TypeCount that withdrawn single entries leave
// behind.
func TestDumpMatchesCaptures(t *testing.T) {
	for _, name := range []string{"hello", "hallo", "fibo", "itoa"} {
		t.Run(name, func(t *testing.T) {
			if got, want := Dump(parseSample(t, name)), golden(t, name+".dump"); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// minimal declares no numbers, so the C data line has no initialiser.
const minimal = `This code does not use any numbers.
This code does never go anywhere.
This code writes the first number as a character.
This code cannot read.
This code does not use any sums.
This code does not use any conditions.
This code does not use any labels.
This code does not use any ordered differences.
This code does not use any assignments.
This code does not use any products.
This code implements the first output, the first output, and the first output.
This code does not use any ratios.
This code does not use any logical operations.
Cool.`

func parseMinimal(t *testing.T) *syntax.Program {
	t.Helper()
	p, err := syntax.Parse(syntax.Filter([]byte(minimal)))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Without numbers the original writes "long _[193719];" and a single newline
// (no blank line before the macros). Follows from reading GenerateSourceInC.
func TestCWithoutNumbers(t *testing.T) {
	got := C(parseMinimal(t))
	if !strings.Contains(got, "#include \"string.h\"\n\nlong _[193719];\n#define N(x) _[x-1]\n") {
		t.Errorf("data line:\n%s", got)
	}
	// The newline after every fourth item counts the B(...) entry too.
	if !strings.HasSuffix(got, "d(W);\nE(W)B(1,H(N(1)))F int main(int,char*[]) {W(1);W(1);W(1);\n{ F\n") {
		t.Errorf("body:\n%s", got)
	}
}

// The withdrawn single statement leaves TypeCount ahead of the entries, so
// the empty tables after it start past the end of Code; the dump must still
// render every category.
func TestDumpEmptyTablesPastCode(t *testing.T) {
	got := Dump(parseMinimal(t))
	want := "ELEMENTS=5\nLABELS=0\n0 NUMBERS: \n0 SUMS: \n0 DIFFS: \n0 PRODS: \n0 RATIOS: \n0 NANDS: \n0 ASSIGNS: \n" +
		"1 WRITES: { (0,CELL),0}\n0 READS: \n0 CONDITIONS: \n" +
		"3 STATEMENTS: { (0,WRITE),0}, { (0,WRITE),0}, { (0,WRITE),0}\n0 JUMPS: \n"
	if got != want {
		t.Errorf("got:\n%q\nwant:\n%q", got, want)
	}
}

// A jump statement past the jump table reads the next Code slot, here a zero
// slot: an unconditional jump to label 1. Follows from GenerateSourceInC
// indexing psJ[] without a bounds check into the static Code array.
func TestCJumpPastTable(t *testing.T) {
	p := parseMinimal(t)
	stmts := p.Tables[syntax.Statements]
	p.Code[stmts.Index].Ops[0] = syntax.Operand{Type: syntax.Jump, Index: 7}
	if got := C(p); !strings.Contains(got, "{UJ(1);W(1);W(1);\n{ F\n") {
		t.Errorf("main:\n%s", got)
	}
}

// Read operands render as "r(k)". The parser never produces a reference to
// an input, so this needs a hand-built statement.
func TestCReadOperandLetter(t *testing.T) {
	p := parseMinimal(t)
	p.Code[p.Tables[syntax.Statements].Index].Ops[0] = syntax.Operand{Type: syntax.Read, Index: 0}
	if got := C(p); !strings.Contains(got, "{r(1);W(1);W(1);\n{ F\n") {
		t.Errorf("main:\n%s", got)
	}
}
