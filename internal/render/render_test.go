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
//
// In Italian, each becomes a Very Very Sorted! program (#30) with the same
// tables.
func TestSamplesRoundTrip(t *testing.T) {
	for _, name := range samples {
		for _, lang := range []Lang{English, German, Italian, French, Portuguese, Japanese, Mandarin, Pinyin} {
			t.Run(fmt.Sprintf("%s/%d", name, lang), func(t *testing.T) {
				p := parse(t, readFile(t, "legacy", "sorted.win32", name+".s"))
				text, err := Render(p, lang)
				if err != nil {
					t.Fatal(err)
				}
				q := parse(t, []byte(text))
				p.Verys = lang.Verys()
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
//
// Italian (#30): every preposition its article fuses with.
func TestReferencesRoundTrip(t *testing.T) {
	r := &renderer{}
	for _, lang := range []Lang{English, German, Italian, French, Portuguese, Japanese, Mandarin, Pinyin} {
		cases := []gcase{nominative, accusative, dative}
		switch lang {
		case Italian:
			cases = []gcase{nominative, itDi, itA, itDa}
		case French:
			cases = []gcase{nominative, frA, frDe}
		case Portuguese:
			cases = []gcase{nominative, ptDe, ptA, ptPor}
		case Japanese, Mandarin, Pinyin:
			cases = []gcase{nominative}
		}
		for typ := range nouns {
			for _, indirect := range []syntax.OperandType{0, syntax.Indirect} {
				for _, c := range cases {
					for i := int32(0); i < 2000; i++ {
						if lang >= Italian && i >= 300 && i%97 != 0 { // numbers tests its ordinals
							continue
						}
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
// the only statement, in the oldest dialect that reads it (inputs and logical
// operations need Very Sorted!, Italian Very Very Sorted!).
func parseRef(s string) (syntax.Operand, bool) {
	src := strings.Replace(skeleton, "STATEMENT", strings.TrimSuffix(s, " x"), 1)
	for n := 0; n <= syntax.Newest; n++ {
		if p, err := syntax.Parse([]byte(strings.Replace(src, "Cool.", syntax.Marker(n), 1))); err == nil {
			return p.Entries(syntax.Statements)[0].Ops[0], true
		}
	}
	return syntax.Operand{}, false
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
	for _, lang := range []Lang{English, German, Italian, French, Portuguese, Japanese, Mandarin, Pinyin} {
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
	p.Verys = 0
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
	p.Verys = 0
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

// Very Sorted! writes German ratios and lists of ordered differences (#46);
// in the original's Sorted! those sentences fall back to English.
func TestVeryGerman(t *testing.T) {
	src := strings.Replace(skeleton, "STATEMENT", "the first ratio", 1)
	src = strings.Replace(src, "This code does not use any numbers.", "This code uses the numbers twelve, and ten.", 1)
	src = strings.Replace(src, "This code does not use any ordered differences.", "This code uses the ordered differences between the first number and the second number, and between the second number and the first number.", 1)
	src = strings.Replace(src, "This code does not use any ratios.", "This code uses the ratios of the first number to the second number, and of the second number to the first ratio.", 1)
	for _, tt := range []struct {
		end  string
		want []string
	}{
		{"This code is very cool.", []string{
			"Dieses Programm benutzt die geordneten Differenzen zwischen der ersten Zahl und der zweiten Zahl, und zwischen der zweiten Zahl und der ersten Zahl.",
			// No "dem": the parser does not know it ("von das erste Produkt").
			"Dieses Programm benutzt die Verhältnisse von der ersten Zahl zu der zweiten Zahl, und von der zweiten Zahl zu das erste Verhältnis.",
		}},
		{"Cool.", []string{
			"This code uses the ordered differences between the first number and the second number, and between the second number and the first number.",
			"This code uses the ratios of the first number to the second number, and of the second number to the first ratio.",
		}},
	} {
		p, err := syntax.Parse([]byte(strings.Replace(src, "Cool.", tt.end, 1)))
		if err != nil {
			t.Fatal(err)
		}
		text, err := Render(p, German)
		if err != nil {
			t.Fatalf("%s: %v", tt.end, err)
		}
		for _, w := range tt.want {
			if !strings.Contains(strings.Join(strings.Fields(text), " "), w) {
				t.Errorf("%s: no %q in\n%s", tt.end, w, text)
			}
		}
	}
}

// Very Sorted! writes stores into the cell a value indexes (#27).
func TestVeryIndexedStore(t *testing.T) {
	src := strings.Replace(skeleton, "STATEMENT", "the first assignment", 1)
	src = strings.Replace(src, "This code does not use any numbers.", "This code uses the number one.", 1)
	src = strings.Replace(src, "This code does not use any sums.", "This code uses the sum of the first number and the first number.", 1)
	src = strings.Replace(src, "This code does not use any assignments.", "This code assigns the first number to the cell indexed by the first sum.", 1)
	p, err := syntax.Parse([]byte(strings.Replace(src, "Cool.", "This code is very cool.", 1)))
	if err != nil {
		t.Fatal(err)
	}
	for lang, want := range map[Lang]string{English: "the first number to the cell indexed by the first sum", German: "die erste Zahl an diejenige Zelle die indiziert wird durch die erste Summe"} {
		text, err := Render(p, lang)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(strings.Join(strings.Fields(text), " "), want) {
			t.Errorf("lang %d: no %q in\n%s", lang, want, text)
		}
	}
	p.Verys = 0
	if _, err := Render(p, English); err == nil || !strings.Contains(err.Error(), "does not store into a cell") {
		t.Errorf("an indexed store outside Very Sorted!: %v", err)
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
	q.Verys = 1
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
//
// Very Very Sorted! (#30) has them in Italian too, and the Italian numbers in
// all three languages; an older program in Italian becomes one.
func TestOutputFormats(t *testing.T) {
	want := map[int32]string{
		syntax.FormatCharacter:         "\x15",
		syntax.FormatEnglishCardinal:   "twentyone\n",
		syntax.FormatEnglishOrdinal:    "twentyfirst\n",
		syntax.FormatGermanCardinal:    "einundzwanzig\n",
		syntax.FormatGermanOrdinal:     "einundzwanzigste\n",
		syntax.FormatItalianCardinal:   "ventuno\n",
		syntax.FormatItalianOrdinal:    "ventunesimo\n",
		syntax.FormatVaudoisCardinal:   "vingt-et-un\n",
		syntax.FormatVaudoisOrdinal:    "vingt-et-unième\n",
		syntax.FormatBrazilianCardinal: "vinte e um\n",
		syntax.FormatBrazilianOrdinal:  "vigésimo primeiro\n",
		syntax.FormatJapaneseCardinal:  "nijūichi\n",
		syntax.FormatJapaneseOrdinal:   "dai-nijūichi\n",
		syntax.FormatChineseCardinal:   "二十一\n",
		syntax.FormatChineseOrdinal:    "第二十一\n",
	}
	for format, output := range want {
		for _, lang := range []Lang{English, German, Italian, French, Portuguese, Japanese, Mandarin, Pinyin} {
			p := parse(t, []byte(strings.Replace(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1),
				"This code does not use any numbers.", "This code uses the number twentyone.", 1),
				"This code cannot write.", "This code writes the first number as a character.", 1)))
			p.Code[p.Tables[syntax.Writes].Index].Flags = format // flags do not affect the layout
			if format >= syntax.FormatItalianCardinal {
				if _, err := Render(p, lang); lang < Italian && err == nil {
					t.Errorf("format %d, lang %d: rendered outside Very Very Sorted!", format, lang)
				}
				p.Verys = 2
			} else if lang >= Italian && (format == syntax.FormatGermanCardinal || format == syntax.FormatGermanOrdinal) {
				if _, err := Render(p, lang); err == nil || !strings.Contains(err.Error(), "UTF-8") {
					t.Errorf("format %d in Italian: %v", format, err)
				}
				p.Verys = 2
			}
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

// Italian (#30): the articles fuse with their prepositions and elide before
// a vowel, simple lists have no comma before "e" and lists of pairs do, and
// a program in Italian is Very Very Sorted!.
func TestItalian(t *testing.T) {
	src := `This code uses the numbers seven, eight, nine, and ten.
This code always goes to the first label, and sometimes goes to the second label if the eleventh condition is true.
This code writes the first sum as an italian ordinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the eight number and the cell indexed by the first number.
This code uses the condition that the first number is equal to the eight number, and the condition that the first number is less than the first ratio.
This code uses one label.
This code uses the ordered difference between the eleventh number and the second number.
This code assigns the first sum to the third number, the first input to the eight number, and the first product to the cell indexed by the first sum.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the first input, the first output, and the eleventh label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operations of not the first number and not the eight number, and of not both the first logical operation and the second number.
This code is very very cool.`
	p := parse(t, []byte(src))
	text, err := Render(p, Italian)
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"Questo programma usa i numeri sette, otto, nove e dieci.",
		"va sempre alla prima etichetta e va talvolta alla seconda etichetta se l'undicesima condizione è vera.",
		"Questo programma scrive la prima somma come ordinale italiano.",
		"Questo programma legge il primo numero come carattere.",
		"le somme del primo numero e del secondo numero, e dell'ottavo numero e della cella indicizzata dal primo numero.",
		"la condizione che il primo numero sia uguale all'ottavo numero e la condizione che il primo numero sia minore del primo rapporto.",
		"Questo programma usa un'etichetta.",
		"Questo programma usa la differenza ordinata tra l'undicesimo numero e il secondo numero.",
		"Questo programma assegna la prima somma al terzo numero, il primo ingresso all'ottavo numero e il primo prodotto alla cella indicizzata dalla prima somma.",
		"i prodotti del primo numero e del secondo numero, e della prima somma e della prima differenza ordinata.",
		"implementa il primo assegnamento, la prima etichetta, il primo salto, il secondo assegnamento, il primo ingresso, la prima uscita e l'undicesima etichetta.",
		"i rapporti tra il primo numero e il secondo numero, e tra il secondo numero e il primo numero.",
		"le operazioni logiche né il primo numero né l'ottavo numero, e non entrambi la prima operazione logica e il secondo numero.",
		"Questo programma è molto molto figo.",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); !Equal(p, q) {
		t.Errorf("tables differ:\n%s", text)
	}
	// The other languages say Very Very Sorted! too.
	for lang, want := range map[Lang][]string{
		English: {"as an italian ordinal", "This code is very very cool."},
		German:  {"als eine italienische Ordinalzahl", "Dieses Programm ist ganz ganz hervorragend."},
	} {
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
	// None of anything, in Italian.
	none := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	text, err = Render(none, Italian)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"non usa numeri.", "non va mai da nessuna parte.", "non può scrivere.", "non può leggere.", "non usa somme.",
		"non usa condizioni.", "non usa etichette.", "non usa differenze ordinate.", "non usa assegnamenti.", "non usa prodotti.",
		"implementa il primo numero.", "non usa rapporti.", "è illogico.", "è molto molto figo."} {
		if !strings.Contains(text, "Questo programma "+want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
}

// A program of the original's Sorted! becomes Very Very Sorted! in Italian,
// unless that changes what it does: German numbers would print in UTF-8,
// and a statement past its table might store or read elsewhere. A Very
// Sorted! one always can.
func TestItalianDialect(t *testing.T) {
	base := func() *syntax.Program {
		return parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1), "This code cannot write.", "This code writes the first number as a character.", 1)))
	}
	if _, err := Render(base(), Italian); err != nil {
		t.Fatal(err)
	}
	for name, change := range map[string]func(*syntax.Program){
		"German numbers": func(p *syntax.Program) { p.Code[p.Tables[syntax.Writes].Index].Flags = syntax.FormatGermanOrdinal },
		"assignment past its table": func(p *syntax.Program) {
			p.Code[p.Tables[syntax.Statements].Index].Ops[0] = syntax.Operand{Type: syntax.Assign}
		},
		"jump past its table": func(p *syntax.Program) {
			p.Code[p.Tables[syntax.Statements].Index].Ops[0] = syntax.Operand{Type: syntax.Jump, Index: 3}
		},
	} {
		p := base()
		change(p)
		var e *Error
		if _, err := Render(p, Italian); !errors.As(err, &e) {
			t.Errorf("%s: %v", name, err)
		}
		if _, _, err := Compose(p, Italian); !errors.As(err, &e) {
			t.Errorf("%s: Compose: %v", name, err)
		}
		p.Verys = 1
		if _, err := Render(p, English); err != nil && name == "German numbers" {
			t.Errorf("%s in Very Sorted!: %v", name, err)
		}
		if _, err := Render(p, Italian); err != nil && name == "German numbers" {
			t.Errorf("%s from Very Sorted!: %v", name, err)
		}
	}
}

// French (#29): articles fused with à and de, the comparison agreeing with
// its subject, Vaud's numbers, "Y'a pas le feu au lac." for no jumps, and
// the fillers by their rule; a program in French is Very Very Sorted!.
func TestFrench(t *testing.T) {
	src := `This code uses the numbers seventyone, eighty, ninetyone, and onehundred.
This code always goes to the first label, and sometimes goes to the second label if the eleventh condition is true.
This code writes the first sum as a vaudois ordinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the eight number and the cell indexed by the first number.
This code uses the condition that the first sum is equal to the eight number, and the condition that the first number is less than the first ratio.
This code uses twentyone labels.
This code uses the ordered difference between the eleventh number and the second number.
This code assigns the first sum to the third number, the first input to the eight number, and the first product to the cell indexed by the first sum.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the first input, the first output, and the eleventh label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operations of not the first number and not the eight number, and of not both the first logical operation and the second number.
This code is very very cool.`
	p := parse(t, []byte(src))
	text, err := Render(p, French)
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"Ce programme utilise les nombres septante-et-un, huitante, nonante-et-un et cent.",
		"va toujours à la première étiquette et va parfois à la deuxième étiquette si la onzième condition est vraie.",
		"Ce programme, eh, écrit la première somme comme ordinal vaudois.",
		"Ce programme lit le premier nombre comme caractère.",
		"les sommes du premier nombre et du deuxième nombre, et du huitième nombre et de la cellule indexée par le premier nombre.",
		"Ce programme, hein, utilise la condition que la première somme soit égale au huitième nombre et la condition que le premier nombre soit inférieur au premier rapport.",
		"Ce programme utilise vingt-et-une étiquettes.",
		"Ce programme utilise la différence ordonnée entre le onzième nombre et le deuxième nombre.",
		"Ce programme, quoi, affecte la première somme au troisième nombre, la première entrée au huitième nombre et le premier produit à la cellule indexée par la première somme.",
		"les produits du premier nombre et du deuxième nombre, et de la première somme et de la première différence ordonnée.",
		"implémente la première affectation, la première étiquette, le premier saut, la deuxième affectation, la première entrée, la première sortie et la onzième étiquette, voilà.",
		"Ce programme, voilà, utilise les rapports du premier nombre au deuxième nombre et du deuxième nombre au premier nombre.",
		"les opérations logiques ni le premier nombre ni le huitième nombre, et pas à la fois la première opération logique et le deuxième nombre.",
		"Ce programme est très très chouette.",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); !Equal(p, q) {
		t.Errorf("tables differ:\n%s", text)
	}
	for lang, want := range map[Lang]string{English: "as a vaudois ordinal", German: "als eine waadtländische Ordinalzahl", Italian: "come ordinale vodese"} {
		if text, err := Render(p, lang); err != nil || !strings.Contains(text, want) {
			t.Errorf("lang %d: %v, no %q in\n%s", lang, err, want, text)
		}
	}
	none := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	text, err = Render(none, French)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Ce programme n'utilise aucun nombre.", "Y'a pas le feu au lac.", "Ce programme, eh, ne peut pas écrire.",
		"Ce programme ne peut pas lire.", "Ce programme n'utilise aucune somme.", "Ce programme, hein, n'utilise aucune condition.",
		"Ce programme n'utilise aucune étiquette.", "Ce programme n'utilise aucune différence ordonnée.",
		"Ce programme, quoi, n'utilise aucune affectation.", "Ce programme n'utilise aucun produit.",
		"Ce programme implémente le premier nombre, voilà.", "Ce programme, voilà, n'utilise aucun rapport.",
		"Ce programme est illogique.", "Ce programme est très très chouette."} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); q.Verys != 2 {
		t.Errorf("verys %d", q.Verys)
	}
	// German numbers would change in Very Very Sorted!, as in Italian.
	g := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1), "This code cannot write.", "This code writes the first number as a german german ordinal.", 1)))
	if _, err := Render(g, French); err == nil {
		t.Error("German numbers rendered in French")
	}
}

// Portuguese (#31): articles fused with de, a and por, ordinals agreeing
// with their nouns, Brazil's numbers, with a comma before the "e" of a list
// only where two numbers would otherwise read back as one; a program in
// Portuguese is Very Very Sorted!.
func TestPortuguese(t *testing.T) {
	src := `This code uses the numbers twentythree, twenty, one, and onehundred.
This code always goes to the first label, and sometimes goes to the second label if the eleventh condition is true.
This code writes the first sum as a brazilian ordinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the eight number and the cell indexed by the first sum.
This code uses the condition that the first sum is equal to the eight number, and the condition that the first number is less than the first ratio.
This code uses twentyone labels.
This code uses the ordered difference between the eleventh number and the second number.
This code assigns the first sum to the third number, the first input to the eight number, and the first product to the cell indexed by the first sum.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the first input, the first output, and the eleventh label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operations of not the first number and not the eight number, and of not both the first logical operation and the second number.
This code is very very cool.`
	p := parse(t, []byte(src))
	text, err := Render(p, Portuguese)
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"Este programa usa os números vinte e três, vinte, um e cem.",
		"Este programa sempre vai para o primeiro rótulo e às vezes vai para o segundo rótulo se a décima primeira condição for verdadeira.",
		"Este programa escreve a primeira soma como ordinal brasileiro.",
		"Este programa lê o primeiro número como caractere.",
		"as somas do primeiro número e do segundo número, e do oitavo número e da célula indexada pela primeira soma.",
		"Este programa usa a condição de que a primeira soma seja igual ao oitavo número e a condição de que o primeiro número seja menor que a primeira razão.",
		"Este programa usa vinte e um rótulos.",
		"Este programa usa a diferença ordenada entre o décimo primeiro número e o segundo número.",
		"Este programa atribui a primeira soma ao terceiro número, a primeira entrada ao oitavo número e o primeiro produto à célula indexada pela primeira soma.",
		"os produtos do primeiro número e do segundo número, e da primeira soma e da primeira diferença ordenada.",
		"implementa a primeira atribuição, o primeiro rótulo, o primeiro salto, a segunda atribuição, a primeira entrada, a primeira saída e o décimo primeiro rótulo.",
		"Este programa usa as razões entre o primeiro número e o segundo número, e entre o segundo número e o primeiro número.",
		"as operações lógicas nem o primeiro número nem o oitavo número, e não ambos a primeira operação lógica e o segundo número.",
		"Este programa é muito muito legal.",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); !Equal(p, q) {
		t.Errorf("tables differ:\n%s", text)
	}
	for lang, want := range map[Lang]string{English: "as a brazilian ordinal", German: "als eine brasilianische Ordinalzahl", Italian: "come ordinale brasiliano", French: "comme ordinal brésilien"} {
		if text, err := Render(p, lang); err != nil || !strings.Contains(text, want) {
			t.Errorf("lang %d: %v, no %q in\n%s", lang, err, want, text)
		}
	}
	// Lists of numbers that would read back as one get a comma; a long
	// list keeps it in its verse.
	for _, tt := range []struct {
		data []int32
		want string
	}{
		{[]int32{20, 1}, "os números vinte, e um."},
		{[]int32{2, 3}, "os números dois e três."},
		{[]int32{100, 1}, "os números cem e um."},
		{[]int32{1000, 100}, "os números mil, e cem."},
		{[]int32{200, 30}, "os números duzentos, e trinta."},
		{[]int32{1, 21, 2}, "os números um, vinte e um e dois."},
		{[]int32{999999999, 888888888, 777777777, 1}, "setecentos e setenta e sete e um."},
		{[]int32{999999999, 888888888, 777777000, 1}, "setecentos e setenta e sete mil, e um."},
		{[]int32{999999999, 888888888, 777777777, 2000000}, "e setenta e sete mil setecentos e setenta e sete e dois milhões."},
		{[]int32{999999999, 888888888, 777777000, 2000}, "setecentos e setenta e sete mil, e dois mil."},
	} {
		q := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
		q.Data = tt.data
		text, err := Render(q, Portuguese)
		if err != nil {
			t.Errorf("%v: %v", tt.data, err)
			continue
		}
		if flat := strings.Join(strings.Fields(text), " "); !strings.Contains(flat, tt.want) {
			t.Errorf("%v: no %q in\n%s", tt.data, tt.want, text)
		}
	}
	none := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	text, err = Render(none, Portuguese)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Este programa não usa nenhum número.", "Este programa não vai a lugar nenhum.", "Este programa não pode escrever.",
		"Este programa não pode ler.", "Este programa não usa nenhuma soma.", "Este programa não usa nenhuma condição.",
		"Este programa não usa nenhum rótulo.", "Este programa não usa nenhuma diferença ordenada.",
		"Este programa não faz nenhuma atribuição.", "Este programa não usa nenhum produto.",
		"Este programa implementa o primeiro número.", "Este programa não usa nenhuma razão.",
		"Este programa é ilógico.", "Este programa é muito muito legal."} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); q.Verys != 2 {
		t.Errorf("verys %d", q.Verys)
	}
	one := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first number", 1), "This code does not use any labels.", "This code uses one label.", 1)))
	if text, err := Render(one, Portuguese); err != nil || !strings.Contains(text, "Este programa usa um rótulo.") {
		t.Errorf("one label: %v\n%s", err, text)
	}
	// German numbers would change in Very Very Sorted!, as in Italian.
	g := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1), "This code cannot write.", "This code writes the first number as a german german ordinal.", 1)))
	if _, err := Render(g, Portuguese); err == nil {
		t.Error("German numbers rendered in Portuguese")
	}
}

// Japanese (#32): verb-final sentences, lists of things ending with their
// verb, actions chained, the comma of a list of pairs after "to", prefixed
// ordinals and one-word numbers; a program in Japanese is Very Very
// Sorted!.
func TestJapanese(t *testing.T) {
	src := `This code uses the numbers twentythree, tenthousand, sixhundred, and onehundred.
This code always goes to the first label, and sometimes goes to the second label if the eleventh condition is true.
This code writes the first sum as a japanese ordinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the eight number and the cell indexed by the first sum.
This code uses the condition that the first sum is equal to the eight number, and the condition that the first number is less than the first ratio.
This code uses twentyone labels.
This code uses the ordered difference between the eleventh number and the second number.
This code assigns the first sum to the third number, the first input to the eight number, and the first product to the cell indexed by the first sum.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the first input, the first output, and the eleventh label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operations of not the first number and not the eight number, and of not both the first logical operation and the second number.
This code is very very cool.`
	p := parse(t, []byte(src))
	text, err := Render(p, Japanese)
	if err != nil {
		t.Fatal(err)
	}
	flat := strings.Join(strings.Fields(text), " ")
	for _, want := range []string{
		"Kono puroguramu wa kazu nijūsan, ichiman, roppyaku to hyaku o tsukaimasu.",
		"Kono puroguramu wa itsumo dai-ichi no raberu ni tobi, dai-jūichi no jōken ga shin nara dai-ni no raberu ni tobimasu.",
		"Kono puroguramu wa dai-ichi no wa o nihongo no josū to shite kakimasu.",
		"Kono puroguramu wa dai-ichi no kazu o moji to shite yomimasu.",
		"dai-ichi no kazu to dai-ni no kazu no wa to, dai-hachi no kazu to dai-ichi no wa ga sasu seru no wa o tsukaimasu.",
		"dai-ichi no wa ga dai-hachi no kazu to hitoshii to iu jōken to, dai-ichi no kazu ga dai-ichi no hi yori chiisai to iu jōken o tsukaimasu.",
		"Kono puroguramu wa raberu o nijūikko tsukaimasu.",
		"Kono puroguramu wa dai-jūichi no kazu to dai-ni no kazu no sa o tsukaimasu.",
		"dai-ichi no wa o dai-san no kazu ni dainyū shi, dai-ichi no nyūryoku o dai-hachi no kazu ni dainyū shi, dai-ichi no seki o dai-ichi no wa ga sasu seru ni dainyū shimasu.",
		"dai-ichi no kazu to dai-ni no kazu no seki to, dai-ichi no wa to dai-ichi no sa no seki o tsukaimasu.",
		"dai-ichi no dainyū, dai-ichi no raberu, dai-ichi no janpu, dai-ni no dainyū, dai-ichi no nyūryoku, dai-ichi no shutsuryoku to dai-jūichi no raberu o jissō shimasu.",
		"dai-ichi no kazu to dai-ni no kazu no hi to, dai-ni no kazu to dai-ichi no kazu no hi o tsukaimasu.",
		"dai-ichi no kazu demo dai-hachi no kazu demo nai ronri enzan to, dai-ichi no ronri enzan to dai-ni no kazu no ryōhō de wa nai ronri enzan o tsukaimasu.",
		"Kono puroguramu wa totemo totemo kakkoii desu.",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); !Equal(p, q) {
		t.Errorf("tables differ:\n%s", text)
	}
	for lang, want := range map[Lang]string{English: "as a japanese ordinal", German: "als eine japanische Ordinalzahl", Italian: "come ordinale giapponese",
		French: "comme ordinal japonais", Portuguese: "como ordinal japonês"} {
		if text, err := Render(p, lang); err != nil || !strings.Contains(text, want) {
			t.Errorf("lang %d: %v, no %q in\n%s", lang, err, want, text)
		}
	}
	none := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	text, err = Render(none, Japanese)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Kono puroguramu wa kazu o tsukaimasen.", "Kono puroguramu wa doko ni mo ikimasen.", "Kono puroguramu wa kakemasen.",
		"Kono puroguramu wa yomemasen.", "Kono puroguramu wa wa o tsukaimasen.", "Kono puroguramu wa jōken o tsukaimasen.",
		"Kono puroguramu wa raberu o tsukaimasen.", "Kono puroguramu wa sa o tsukaimasen.",
		"Kono puroguramu wa dainyū shimasen.", "Kono puroguramu wa seki o tsukaimasen.",
		"Kono puroguramu wa dai-ichi no kazu o jissō shimasu.", "Kono puroguramu wa hi o tsukaimasen.",
		"Kono puroguramu wa hironriteki desu.", "Kono puroguramu wa totemo totemo kakkoii desu."} {
		if !strings.Contains(text, want) {
			t.Errorf("no %q in\n%s", want, text)
		}
	}
	if q := parse(t, []byte(text)); q.Verys != 2 {
		t.Errorf("verys %d", q.Verys)
	}
	one := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first number", 1), "This code does not use any labels.", "This code uses one label.", 1)))
	if text, err := Render(one, Japanese); err != nil || !strings.Contains(text, "Kono puroguramu wa raberu o ikko tsukaimasu.") {
		t.Errorf("one label: %v\n%s", err, text)
	}
	// A long list of pairs keeps its "to," in the verse.
	long := parse(t, []byte(strings.Replace(src, "This code uses the ordered difference between the eleventh number and the second number.",
		"This code uses the ordered differences between the eleventh number and the second number, and between the eleventh number and the twelveth number.", 1)))
	if text, err := Render(long, Japanese); err != nil || !strings.Contains(text, "\tdai-jūichi no kazu to dai-ni no kazu no sa to,\n\tdai-jūichi no kazu to dai-jūni no kazu no sa o tsukaimasu.") {
		t.Errorf("verse: %v\n%s", err, text)
	}
	// German numbers would change in Very Very Sorted!, as in Italian.
	g := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1), "This code cannot write.", "This code writes the first number as a german german ordinal.", 1)))
	if _, err := Render(g, Japanese); err == nil {
		t.Error("German numbers rendered in Japanese")
	}
}

// Mandarin (#33), in hanzi and in pinyin: no spaces and full-width
// punctuation in hanzi, 把 for actions, "，并" between jumps, 和 for "and"
// and the sum, 两 in counts; a program in Mandarin is Very Very Sorted!.
func TestMandarin(t *testing.T) {
	src := `This code uses the numbers twentythree, tenthousand, onehundredone, and twohundred.
This code always goes to the first label, and sometimes goes to the second label if the eleventh condition is true.
This code writes the first sum as a chinese ordinal.
This code reads the first number as a character.
This code uses the sums of the first number and the second number, and of the eight number and the cell indexed by the first sum.
This code uses the condition that the first sum is equal to the eight number, and the condition that the first number is less than the first ratio.
This code uses two labels.
This code uses the ordered difference between the eleventh number and the second number.
This code assigns the first sum to the third number, the first input to the eight number, and the first product to the cell indexed by the first sum.
This code uses the products of the first number and the second number, and of the first sum and the first ordered difference.
This code implements the first assignment, the first label, the first jump, the second assignment, the first input, the first output, and the eleventh label.
This code uses the ratios of the first number to the second number, and of the second number to the first number.
This code uses the logical operations of not the first number and not the eight number, and of not both the first logical operation and the second number.
This code is very very cool.`
	p := parse(t, []byte(src))
	for _, tt := range []struct {
		lang  Lang
		wants []string
	}{
		{Mandarin, []string{
			"这个程序使用数字二十三、一万、一百零一和二百。",
			"这个程序总是跳到第一个标签，并在第十一个条件为真时跳到第二个标签。",
			"这个程序把第一个和作为中文序数写出。",
			"这个程序把第一个数字作为字符读入。",
			"第一个数字和第二个数字的和和第八个数字和第一个和所指的单元的和。",
			"第一个和等于第八个数字的条件和第一个数字小于第一个比的条件。",
			"这个程序使用两个标签。",
			"这个程序使用第十一个数字和第二个数字的差。",
			"把第一个和赋给第三个数字，", "把第一个输入赋给第八个数字，", "把第一个积赋给第一个和所指的单元。",
			"第一个数字和第二个数字的积和第一个和和第一个差的积。",
			"第一个赋值、", "和第十一个标签。",
			"第一个数字和第二个数字的比和第二个数字和第一个数字的比。",
			"既非第一个数字也非第八个数字的逻辑运算和并非第一个逻辑运算和第二个数字都成立的逻辑运算。",
			"这个程序非常非常酷。",
		}},
		{Pinyin, []string{
			"Zhège chéngxù shǐyòng shùzì èrshísān, yīwàn, yībǎilíngyī hé èrbǎi.",
			"zǒngshì tiàodào dì-yī gè biāoqiān,", "bìng zài dì-shíyī gè tiáojiàn wéi zhēn shí tiàodào dì-èr gè biāoqiān.",
			"Zhège chéngxù bǎ dì-yī gè hé zuòwéi Zhōngwén xùshù xiěchū.",
			"Zhège chéngxù shǐyòng liǎng gè biāoqiān.",
			"bǎ dì-yī gè jī fù gěi dì-yī gè hé suǒ zhǐ de dānyuán.",
			"bìngfēi dì-yī gè luójí yùnsuàn hé dì-èr gè shùzì dōu chénglì de luójí yùnsuàn.",
			"Zhège chéngxù fēicháng fēicháng kù.",
		}},
	} {
		text, err := Render(p, tt.lang)
		if err != nil {
			t.Fatal(err)
		}
		flat := strings.Join(strings.Fields(text), " ")
		for _, want := range tt.wants {
			if !strings.Contains(flat, want) && !strings.Contains(strings.ReplaceAll(text, "\n\t", ""), want) {
				t.Errorf("lang %d: no %q in\n%s", tt.lang, want, text)
			}
		}
		if q := parse(t, []byte(text)); !Equal(p, q) {
			t.Errorf("lang %d: tables differ:\n%s", tt.lang, text)
		}
	}
	for lang, want := range map[Lang]string{English: "as a chinese ordinal", German: "als eine chinesische Ordinalzahl", Italian: "come ordinale cinese",
		French: "comme ordinal chinois", Portuguese: "como ordinal chinês", Japanese: "chūgokugo no josū to shite"} {
		if text, err := Render(p, lang); err != nil || !strings.Contains(text, want) {
			t.Errorf("lang %d: %v, no %q in\n%s", lang, err, want, text)
		}
	}
	none := parse(t, []byte(strings.Replace(skeleton, "STATEMENT", "the first number", 1)))
	for lang, wants := range map[Lang][]string{
		Mandarin: {"这个程序不使用数字。", "这个程序哪儿也不去。", "这个程序不能写。", "这个程序不能读。", "这个程序不使用和。", "这个程序不使用条件。",
			"这个程序不使用标签。", "这个程序不使用差。", "这个程序不赋值。", "这个程序不使用积。", "这个程序实现第一个数字。", "这个程序不使用比。",
			"这个程序不合逻辑。", "这个程序非常非常酷。"},
		Pinyin: {"Zhège chéngxù bù shǐyòng shùzì.", "Zhège chéngxù nǎr yě bú qù.", "Zhège chéngxù bú fùzhí.", "Zhège chéngxù bù hé luójí."},
	} {
		text, err := Render(none, lang)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range wants {
			if !strings.Contains(text, want) {
				t.Errorf("lang %d: no %q in\n%s", lang, want, text)
			}
		}
		if q := parse(t, []byte(text)); q.Verys != 2 {
			t.Errorf("verys %d", q.Verys)
		}
	}
	one := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first number", 1), "This code does not use any labels.", "This code uses one label.", 1)))
	if text, err := Render(one, Mandarin); err != nil || !strings.Contains(text, "这个程序使用一个标签。") {
		t.Errorf("one label: %v\n%s", err, text)
	}
	g := parse(t, []byte(strings.Replace(strings.Replace(skeleton, "STATEMENT", "the first output", 1), "This code cannot write.", "This code writes the first number as a german german ordinal.", 1)))
	if _, err := Render(g, Mandarin); err == nil {
		t.Error("German numbers rendered in Mandarin")
	}
}

// A hanzi line is measured in columns, two to a character.
func TestMandarinWidth(t *testing.T) {
	if zhWidth("一a") != 3 {
		t.Errorf("zhWidth = %d", zhWidth("一a"))
	}
}

// all is every language Sorted! speaks.
var all = []Lang{English, German, Italian, French, Portuguese, Japanese, Mandarin, Pinyin}

// Babel mode (#34): the samples written in every language at once, each
// sentence in one of them, read back into the same tables and run as
// before, whatever the mix.
func TestBabel(t *testing.T) {
	for _, name := range samples {
		for _, b := range []Babel{
			{Langs: all, Mix: Alternate}, {Langs: all, Mix: Random, Seed: 1}, {Langs: all, Mix: Random, Seed: 2},
			{Langs: all, Mix: Singable}, {Langs: []Lang{Japanese, Mandarin}, Mix: Alternate}, {Langs: []Lang{German, English}, Mix: Alternate},
		} {
			p := parse(t, readFile(t, "legacy", "sorted.win32", name+".s"))
			text, err := RenderBabel(p, b)
			if err != nil {
				t.Fatalf("%s %+v: %v", name, b, err)
			}
			q := parse(t, []byte(text))
			p.Verys = b.Verys()
			if !Equal(p, q) {
				t.Errorf("%s %+v: tables differ:\n%s", name, b, text)
			}
			var out bytes.Buffer
			if err := interp.Run(q, strings.NewReader(""), &out, 1000000); err != nil {
				t.Fatal(err)
			}
			want := strings.TrimSuffix(strings.ReplaceAll(string(readFile(t, "testdata", "golden", name+".out")), "\r\n", "\n"), "\n")
			if got := strings.TrimSuffix(out.String(), "\n"); got != want {
				t.Errorf("%s %+v: output %q, want %q", name, b, got, want)
			}
		}
	}
	p := parse(t, readFile(t, "legacy", "sorted.win32", "hello.s"))
	// Alternating takes the languages in turn, sentence by sentence.
	text, err := RenderBabel(p, Babel{Langs: []Lang{Japanese, Mandarin, Italian}, Mix: Alternate})
	if err != nil {
		t.Fatal(err)
	}
	heads := []string{"Kono puroguramu wa", "这个程序", "Questo programma"}
	n := 0
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.HasPrefix(line, "\t") {
			continue
		}
		if !strings.HasPrefix(line, heads[n%3]) {
			t.Errorf("sentence %d: %q, want %s", n+1, line, heads[n%3])
		}
		n++
	}
	if n != 14 {
		t.Errorf("%d sentences", n)
	}
	// English and German alone stay the original's Sorted!; one newer
	// language makes the program Very Very Sorted!.
	if q := parse(t, []byte(mustBabel(t, p, Babel{Langs: []Lang{English, German}, Mix: Random, Seed: 7}))); q.Verys != 0 {
		t.Errorf("English and German: verys %d", q.Verys)
	}
	if q := parse(t, []byte(mustBabel(t, p, Babel{Langs: []Lang{English, French}, Mix: Random, Seed: 7}))); q.Verys != 2 {
		t.Errorf("English and French: verys %d", q.Verys)
	}
	// The same seed gives the same mix, another seed another one.
	a, b, c := mustBabel(t, p, Babel{Langs: all, Seed: 42}), mustBabel(t, p, Babel{Langs: all, Seed: 42}), mustBabel(t, p, Babel{Langs: all, Seed: 43})
	if a != b || a == c {
		t.Error("seeds do not decide the mix")
	}
	// The most singable sentence has the fewest syllables as it is written,
	// French fillers included: with French and German, the "no ratios"
	// sentence ties before its filler, so German must win it.
	frde := Babel{Langs: []Lang{French, German}, Mix: Singable}
	text = mustBabel(t, p, frde)
	got := sentenceTexts(text)
	p, err = inDialect(p, frde)
	if err != nil {
		t.Fatal(err)
	}
	f := fillers{}
	for i := range 14 {
		fr, err := sentences(p, French)[i]()
		if err != nil {
			t.Fatal(err)
		}
		g := f
		if i%3 == 2 {
			fr.head = g.after(fr.head)
		}
		de, err := sentences(p, German)[i]()
		if err != nil {
			t.Fatal(err)
		}
		want := fr.text()
		if Syllables(de.text()) < Syllables(want) {
			want = de.text()
		} else {
			f = g
		}
		if want = verySpelling(want); got[i] != want {
			t.Errorf("sentence %d: %q, want %q", i+1, got[i], want)
		}
	}
	if !strings.HasPrefix(got[11], "Dieses Programm benutzt keine Verh") {
		t.Errorf("the no-ratios sentence: %q", got[11])
	}
}

// sentenceTexts splits a rendered program into its sentences: a line that
// does not start with a tab starts one.
func sentenceTexts(text string) []string {
	var ss []string
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		if strings.HasPrefix(line, "\t") && len(ss) > 0 {
			ss[len(ss)-1] += "\n" + line
			continue
		}
		ss = append(ss, line)
	}
	return ss
}

func mustBabel(t *testing.T, p *syntax.Program, b Babel) string {
	t.Helper()
	text, err := RenderBabel(p, b)
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestSyllables(t *testing.T) {
	for _, tt := range []struct {
		s string
		n int
	}{{"cool", 1}, {"chouette", 2}, {"ichi", 2}, {"shùzì", 2}, {"这个程序", 4}, {"fünf", 1}, {"", 0}, {"Zhège chéngxù.", 4}} {
		if got := Syllables(tt.s); got != tt.n {
			t.Errorf("Syllables(%q) = %d, want %d", tt.s, got, tt.n)
		}
	}
}
