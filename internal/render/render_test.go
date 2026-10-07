package render

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gersonkurz/sorted/internal/interp"
	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

var samples = []string{"hello", "hallo", "fibo", "itoa"}

func readFile(t *testing.T, path ...string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, path...)...))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func parse(t *testing.T, src []byte) *syntax.Program {
	t.Helper()
	p, err := syntax.Parse(src)
	if err != nil {
		t.Fatalf("%v in:\n%s", err, src)
	}
	return p
}

// Each sample, rendered in each language, parses back into the same tables
// (including their layout in Code, so even references past a table behave
// alike), and prints what Sorted.exe printed (testdata/golden/<sample>.out).
func TestSamplesRoundTrip(t *testing.T) {
	for _, name := range samples {
		for _, lang := range []Lang{English, German} {
			t.Run(fmt.Sprintf("%s/%d", name, lang), func(t *testing.T) {
				p := parse(t, readFile(t, "legacy", "sorted.win32", name+".s"))
				text, err := Render(p, lang)
				if err != nil {
					t.Fatal(err)
				}
				q := parse(t, []byte(text))
				if p.TypeCount != q.TypeCount || p.Tables != q.Tables || !Equal(p, q) {
					t.Errorf("tables differ:\n%s", text)
				}
				var out bytes.Buffer
				if err := interp.Run(q, strings.NewReader(""), &out, 1000000); err != nil {
					t.Fatal(err)
				}
				want := strings.TrimSuffix(strings.ReplaceAll(string(readFile(t, "testdata", "golden", name+".out")), "\r\n", "\n"), "\n")
				if got := strings.TrimSuffix(out.String(), "\n"); got != want {
					t.Errorf("output %q, want %q", got, want)
				}
			})
		}
	}
}

// Every reference the renderer can write reads back as the same operand:
// all types, both languages, every grammatical case, indices 1 to 2000
// (some German "-n" endings do not parse, "zwanzigsten" for one, so this
// covers both forms).
func TestReferencesRoundTrip(t *testing.T) {
	r := &renderer{}
	for _, lang := range []Lang{English, German} {
		for typ := range nouns {
			for _, indirect := range []syntax.OperandType{0, syntax.Indirect} {
				for _, c := range []gcase{nominative, accusative, dative} {
					for i := int32(0); i < 2000; i++ {
						op := syntax.Operand{Type: typ | indirect, Index: i}
						s, err := r.ref(lang, op, c)
						if err != nil {
							t.Fatal(err)
						}
						got, ok := parseRef(s + " x")
						if !ok || got != op {
							t.Fatalf("%q reads back as %v, %v; want %v", s, got, ok, op)
						}
					}
				}
			}
		}
	}
}

// parseRef parses a reference through a minimal program: the reference is
// the only statement.
func parseRef(s string) (syntax.Operand, bool) {
	src := strings.Replace(skeleton, "STATEMENT", strings.TrimSuffix(s, " x"), 1)
	if strings.Contains(s, "input") || strings.Contains(s, "Eingabe") || strings.Contains(s, "logical") || strings.Contains(s, "logische") { // Very Sorted! only
		src = strings.Replace(src, "Cool.", "This code is very cool.", 1)
	}
	p, err := syntax.Parse([]byte(src))
	if err != nil {
		return syntax.Operand{}, false
	}
	return p.Entries(syntax.Statements)[0].Ops[0], true
}

const skeleton = `This code does not use any numbers.
This code does never go anywhere.
This code cannot write.
This code cannot read.
This code does not use any sums.
This code does not use any conditions.
This code does not use any labels.
This code does not use any ordered differences.
This code does not use any assignments.
This code does not use any products.
This code implements STATEMENT.
This code does not use any ratios.
This code does not use any logical operations.
Cool.`

// Numbers from zero to 999999999 are declarable in both languages.
func TestNumbersRoundTrip(t *testing.T) {
	values := []int32{0, 1, 2, 15, 99, 100, 101, 999, 1000, 1001, 65536, 999999, 1000000, 41281927, 123456789, 999999999}
	for _, lang := range []Lang{English, German} {
		p := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
		p.Data = values // the data does not affect the layout
		if _, err := Render(p, lang); err != nil {
			t.Errorf("lang %d: %v", lang, err)
		}
	}
}

// One program that uses every sentence, with lists, in both languages. The
// German version has to fall back to English for ratios, the list of ordered
// differences and logical operations.
const everything = `This code uses the numbers seven, eight, nine, and ten.
This code always goes to the first label, and sometimes goes to the second label if the first condition is true.
This code writes the first sum as a german cardinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the third number and the cell indexed by the first number.
This code uses the condition that the first number is equal to the second number, and the condition that the first number is less than the first ratio.
This code uses two labels.
This code uses the ordered differences between the first number and the second number, and between the second number and the first number.
This code assigns the first sum to the third number, and the first product to the cell indexed by the second number.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the second jump, the first output, and the second label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operation of not the first number and not the second number.
Cool.`

func TestEverythingRoundTrip(t *testing.T) {
	p := parse(t, []byte(everything))
	for _, lang := range []Lang{English, German} {
		text, err := Render(p, lang)
		if err != nil {
			t.Fatalf("lang %d: %v", lang, err)
		}
		q := parse(t, []byte(text))
		if p.TypeCount != q.TypeCount || p.Tables != q.Tables || !Equal(p, q) {
			t.Errorf("lang %d: tables differ:\n%s", lang, text)
		}
		if lang == German {
			for _, english := range []string{"This code uses the ordered differences", "This code uses the ratios", "This code uses the logical operation"} {
				if !strings.Contains(text, english) {
					t.Errorf("German text lacks the English fallback %q:\n%s", english, text)
				}
			}
		}
	}
}

// The layout: short sentences on one line, long lists as verses.
func TestLayout(t *testing.T) {
	p := parse(t, readFile(t, "legacy", "sorted.win32", "fibo.s"))
	text, err := Render(p, English)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"This code uses the numbers twenty, seventyone, three, two, and one.\n",
		"This code writes the eleventh number as a english cardinal.\n",
		"This code implements\n\tthe third assignment,\n\tthe first label,\n",
		"\tand the second label.\nThis code does not use any ratios.\n",
		"Cool.\n",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	de, err := Render(parse(t, readFile(t, "legacy", "sorted.win32", "hallo.s")), German)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\tnull,\n", "aus der ersten Zahl und der zweiten Zahl", "den ersten Sprungbefehl", "Hervorragend.\n"} {
		if !strings.Contains(de, want) {
			t.Errorf("missing %q in:\n%s", want, de)
		}
	}
	// "gleich" takes the dative, "kleiner als" the nominative.
	itoa, err := Render(parse(t, readFile(t, "legacy", "sorted.win32", "itoa.s")), German)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ist gleich der achten Zahl", "ist kleiner als die erste Zahl"} {
		if !strings.Contains(itoa, want) {
			t.Errorf("missing %q in:\n%s", want, itoa)
		}
	}
}

func TestUnrenderable(t *testing.T) {
	base := func() *syntax.Program {
		return parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first output", 1)))
	}
	tests := []struct {
		name   string
		change func(*syntax.Program)
	}{
		{"no statements", func(p *syntax.Program) { p.Tables[syntax.Statements].Count = 0 }},
		{"negative number", func(p *syntax.Program) { p.Data = []int32{-1} }},
		{"number too large", func(p *syntax.Program) { p.Data = []int32{1000000000} }},
		{"unknown output format", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{Flags: 7})
			p.Tables[syntax.Writes] = syntax.Table{Count: 1, Index: len(p.Code) - 1}
		}},
		{"two outputs", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{}, syntax.Slide{})
			p.Tables[syntax.Writes] = syntax.Table{Count: 2, Index: len(p.Code) - 2}
		}},
		{"reference to an input", func(p *syntax.Program) {
			p.Code[p.Tables[syntax.Statements].Index].Ops[0] = syntax.Operand{Type: syntax.Read}
		}},
		{"negative label count", func(p *syntax.Program) { p.LabelsCount = -1 }},
		{"two inputs", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{}, syntax.Slide{})
			p.Tables[syntax.Reads] = syntax.Table{Count: 2, Index: len(p.Code) - 2}
		}},
		{"input not as a character", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{Flags: syntax.FormatEnglishCardinal})
			p.Tables[syntax.Reads] = syntax.Table{Count: 1, Index: len(p.Code) - 1}
		}},
		{"jump to a non-label", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{})
			p.Tables[syntax.Jumps] = syntax.Table{Count: 1, Index: len(p.Code) - 1}
		}},
		{"conditional jump without a condition", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{Ops: [2]syntax.Operand{{Type: syntax.Label}}, Flags: syntax.ConditionalJump})
			p.Tables[syntax.Jumps] = syntax.Table{Count: 1, Index: len(p.Code) - 1}
		}},
		{"assignment into a sum", func(p *syntax.Program) {
			p.Code = append(p.Code, syntax.Slide{Ops: [2]syntax.Operand{{}, {Type: syntax.Sum}}})
			p.Tables[syntax.Assigns] = syntax.Table{Count: 1, Index: len(p.Code) - 1}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := base()
			tt.change(p)
			var e *Error
			if _, err := Render(p, English); !errors.As(err, &e) {
				t.Errorf("err %v, want *Error", err)
			}
		})
	}
}

// Ordinals that do not read back must be noticed, not assumed: German "-n"
// endings parse only sometimes ("ersten", "einhundertersten" but not
// "zwanzigsten"), and English "eighth" leaves its "h" behind.
func TestOrdinalLimits(t *testing.T) {
	if ordinalParses("eighth", 8, "number") || !ordinalParses("eight", 8, "number") {
		t.Error("\"eighth\" should not read back, \"eight\" should")
	}
	r := &renderer{}
	if s, err := r.ref(English, syntax.Operand{Type: syntax.Number, Index: 27}, nominative); err != nil || s != "the twentyeight number" {
		t.Errorf("28th reference: %q, %v", s, err)
	}
	ord, _ := numbers.GermanOrdinal(19)
	if !ordinalParses(ord+"n", 19, "Zahl") {
		t.Errorf("%sn should parse", ord)
	}
	ord, _ = numbers.GermanOrdinal(20)
	if ordinalParses(ord+"n", 20, "Zahl") {
		t.Errorf("%sn should not parse", ord)
	}
	ord, _ = numbers.GermanOrdinal(101)
	if !ordinalParses(ord+"n", 101, "Zahl") {
		t.Errorf("%sn should parse", ord)
	}
}

// A Very Sorted! program renders with its marker and its input
// references, in both languages, and reads back as the same program.
func TestVery(t *testing.T) {
	src := strings.Replace(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first input, and the first output", 1),
		"This code cannot read.", "This code reads the first number as a character.", 1), "Cool.", "This code is very cool.", 1)
	p, err := syntax.Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[Lang][]string{English: {"the first input", "This code is very cool."}, German: {"die erste Eingabe", "Dieses Programm ist ganz hervorragend."}} {
		text, err := Render(p, lang)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range want {
			if !strings.Contains(text, w) {
				t.Errorf("lang %d: no %q in\n%s", lang, w, text)
			}
		}
	}
	// Not very, the same tables cannot be written: the original has no
	// input references.
	p.Very = false
	if _, err := Render(p, English); err == nil {
		t.Error("an input reference rendered outside Very Sorted!")
	}
}

// Very Sorted! German is written in UTF-8 (#28), and reads back; the
// original's German keeps its ASCII spelling.
func TestVeryGermanSpelling(t *testing.T) {
	src := strings.Replace(skeleton, "STATEMENT", "the first number", 1)
	src = strings.Replace(src, "This code does not use any numbers.", "This code uses the numbers five, twelve, and thirtyfive.", 1)
	for _, tt := range []struct {
		end  string
		want []string
	}{
		{"Cool.", []string{"fuenf", "zwoelf", "fuenfunddreissig", "Verhaeltnisse", "Hervorragend."}},
		{"This code is very cool.", []string{"fünf", "zwölf", "fünfunddreißig", "Verhältnisse", "Dieses Programm ist ganz hervorragend."}},
	} {
		p, err := syntax.Parse([]byte(strings.Replace(src, "Cool.", tt.end, 1)))
		if err != nil {
			t.Fatal(err)
		}
		text, err := Render(p, German)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range tt.want {
			if !strings.Contains(text, w) {
				t.Errorf("%s: no %q in\n%s", tt.end, w, text)
			}
		}
	}
}

// Very Sorted! writes its logical operations, NAND and NOR, in either
// language, and references to them (#26); the original's NOR stays English.
func TestVeryNand(t *testing.T) {
	src := strings.Replace(skeleton, "STATEMENT", "the first logical operation", 1)
	src = strings.Replace(src, "This code does not use any numbers.", "This code uses the numbers twelve, and ten.", 1)
	src = strings.Replace(src, "This code does not use any logical operations.", "This code uses the logical operations of not both the first number and the second number, and of not the second logical operation and not the first number.", 1)
	p, err := syntax.Parse([]byte(strings.Replace(src, "Cool.", "This code is very cool.", 1)))
	if err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[Lang][]string{
		English: {"This code uses the logical operations", "of not both the first number and the second number", "and of not the second logical operation and not the first number", "This code implements the first logical operation."},
		German:  {"Dieses Programm benutzt die logischen Verknüpfungen", "von nicht beiden, der ersten Zahl und der zweiten Zahl", "und von nicht der zweiten logischen Verknüpfung und nicht der ersten Zahl", "Dieses Programm implementiert die erste logische Verknüpfung."},
	} {
		text, err := Render(p, lang)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range want {
			if !strings.Contains(strings.Join(strings.Fields(text), " "), w) {
				t.Errorf("lang %d: no %q in\n%s", lang, w, text)
			}
		}
	}
	// Not very, a NAND cannot be written, nor a reference.
	p.Very = false
	if _, err := Render(p, English); err == nil || !strings.Contains(err.Error(), "only Very Sorted! NAND") {
		t.Errorf("a NAND rendered outside Very Sorted!: %v", err)
	}
	// The original's NOR in German falls back to English.
	q, err := syntax.Parse([]byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first number", 1), "This code does not use any logical operations.", "This code uses the logical operation of not the first number and not the first number.", 1)))
	if err != nil {
		t.Fatal(err)
	}
	if text, err := Render(q, German); err != nil || !strings.Contains(strings.Join(strings.Fields(text), " "), "This code uses the logical operation of not the first number and not the first number.") {
		t.Errorf("the original's NOR in German: %v\n%s", err, text)
	}
}

// The dialect is part of a program: the same tables in Very Sorted! are a
// different program (inputs are read elsewhere).
func TestEqualDialect(t *testing.T) {
	p, err := syntax.Parse([]byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	if err != nil {
		t.Fatal(err)
	}
	q := *p
	q.Very = true
	if Equal(p, &q) || SameEntries(p, &q) {
		t.Error("a Very Sorted! program equals its original's twin")
	}
	if !Equal(p, p) || !SameEntries(p, p) {
		t.Error("a program differs from itself")
	}
}

func TestEqual(t *testing.T) {
	p := parse(t, []byte(everything))
	for name, change := range map[string]func(*syntax.Program){
		"labels": func(q *syntax.Program) { q.LabelsCount++ },
		"data":   func(q *syntax.Program) { q.Data = append([]int32{}, q.Data[1:]...) },
		"entry":  func(q *syntax.Program) { q.Code[q.Tables[syntax.Sums].Index].Flags++ },
	} {
		q := parse(t, []byte(everything))
		change(q)
		if Equal(p, q) {
			t.Errorf("%s: programs compare equal", name)
		}
	}
	if !Equal(p, parse(t, []byte(everything))) {
		t.Error("a program differs from itself")
	}
}

// Lists of logical operations take the plural head.
func TestLogicalOperationList(t *testing.T) {
	src := strings.Replace(everything, "This code uses the logical operation of not the first number and not the second number.",
		"This code uses the logical operations of not the first number and not the second number, and of not the second number and not the first number.", 1)
	p := parse(t, []byte(src))
	text, err := Render(p, German)
	if err != nil || !strings.Contains(text, "This code uses the logical operations") {
		t.Errorf("%v:\n%s", err, text)
	}
}

// Every output format can be written in both languages, in the only
// phrasings the parser accepts ("as a english english ordinal", "als ein ein
// ein ein deutscher Kardinal", ...), and prints the same.
func TestOutputFormats(t *testing.T) {
	want := map[int32]string{
		syntax.FormatCharacter:       "\x15",
		syntax.FormatEnglishCardinal: "twentyone\n",
		syntax.FormatEnglishOrdinal:  "twentyfirst\n",
		syntax.FormatGermanCardinal:  "einundzwanzig\n",
		syntax.FormatGermanOrdinal:   "einundzwanzigste\n",
	}
	for format, output := range want {
		for _, lang := range []Lang{English, German} {
			p := parse(t, []byte(strings.Replace(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1),
				"This code does not use any numbers.", "This code uses the number twentyone.", 1),
				"This code cannot write.", "This code writes the first number as a character.", 1)))
			p.Code[p.Tables[syntax.Writes].Index].Flags = format // flags do not affect the layout
			text, err := Render(p, lang)
			if err != nil {
				t.Fatalf("format %d, lang %d: %v", format, lang, err)
			}
			var out bytes.Buffer
			if err := interp.Run(parse(t, []byte(text)), strings.NewReader(""), &out, 100); err != nil || out.String() != output {
				t.Errorf("format %d, lang %d: %q, %v; want %q\n%s", format, lang, out.String(), err, output, text)
			}
		}
	}
}

// A table layout that no Sorted! text produces is refused, not silently
// changed: moving the empty sums table one slot on makes "the first sum" read
// a zero slot instead of the statement slot, which changes what runs.
func TestLayoutIsPreserved(t *testing.T) {
	src := strings.Replace(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1),
		"This code does not use any numbers.", "This code uses the number one.", 1),
		"This code cannot write.", "This code writes the first sum as a english cardinal.", 1)
	p := parse(t, []byte(src))
	run := func(p *syntax.Program) string {
		var out bytes.Buffer
		if err := interp.Run(p, strings.NewReader(""), &out, 100); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	before := run(p)
	p.Tables[syntax.Sums].Index++
	if after := run(p); after == before {
		t.Fatalf("moving the table should change the output (%q)", after)
	}
	var e *Error
	if _, err := Render(p, English); !errors.As(err, &e) {
		t.Errorf("err %v, want a refusal", err)
	}
}
