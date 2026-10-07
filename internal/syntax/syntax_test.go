package syntax

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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

func parseSample(t *testing.T, name string) *Program {
	t.Helper()
	p, err := Parse(readFile(t, "legacy", "sorted.win32", name+".s"))
	if err != nil {
		t.Fatalf("%s.s: %v", name, err)
	}
	return p
}

// All four samples parse. Their tables are checked byte for byte against
// the captured /C and /D output in internal/emit.
func TestSamplesParse(t *testing.T) {
	for _, name := range samples {
		parseSample(t, name)
	}
}

// --- behaviour that follows from reading SortedSyntax.cpp ---

// minimal is a program using every sentence's "none" form, apart from the
// one output and one statement a program needs to do anything.
const minimal = `This code uses the number seven.
This code does never go anywhere.
This code writes the first number as a character.
This code cannot read.
This code does not use any sums.
This code does not use any conditions.
This code does not use any labels.
This code does not use any ordered differences.
This code does not use any assignments.
This code does not use any products.
This code implements the first output.
This code does not use any ratios.
This code does not use any logical operations.
Cool.`

func parse(src string) (*Program, error) { return Parse([]byte(src)) }

// Very Sorted! (#25): a program that ends very coolly may implement its
// input. Everything the original accepts or rejects stays exactly as it was.
func TestVerySorted(t *testing.T) {
	reading := strings.Replace(minimal, "This code cannot read.", "This code reads the first number as a character.", 1)
	reading = strings.Replace(reading, "implements the first output.", "implements the first input, and the first output.", 1)
	for _, end := range []string{"This code is very cool.", "Very cool.", "Dieses Programm ist ganz hervorragend.", "Ganz hervorragend."} {
		p, err := parse(strings.Replace(reading, "Cool.", end, 1))
		if err != nil {
			t.Errorf("%s: %v", end, err)
			continue
		}
		if !p.Very || p.Entries(Statements)[0].Ops[0] != (Operand{Read, 0}) {
			t.Errorf("%s: very %v, statements %v", end, p.Very, p.Entries(Statements))
		}
	}
	// German reference, in a German program.
	p, err := parse(strings.Replace(strings.Replace(reading, "the first input", "die erste Eingabe", 1), "Cool.", "Ganz hervorragend.", 1))
	if err != nil || p.Entries(Statements)[0].Ops[0] != (Operand{Read, 0}) {
		t.Errorf("die erste Eingabe: %v", err)
	}
	// A very program need not read.
	if p, err := parse(strings.Replace(minimal, "Cool.", "This code is very cool.", 1)); err != nil || !p.Very {
		t.Errorf("very without input: %v", err)
	}
	// The original's programs are not very, and an input reference needs the
	// very ending: without it the original's error stands.
	if p, err := parse(minimal); err != nil || p.Very {
		t.Errorf("minimal: very %v, %v", p != nil && p.Very, err)
	}
	if _, err := parse(reading); err == nil || err.Error() != "ERROR, missing or invalid declaration of implementation" {
		t.Errorf("input reference in the original's Sorted!: %v", err)
	}
	// Broken elsewhere, the very program reports the original's error.
	broken := strings.Replace(strings.Replace(reading, "Cool.", "This code is very cool.", 1), "This code does not use any sums.", "This code is broken.", 1)
	if _, err := parse(broken); err == nil || err.Error() != "ERROR, missing or invalid sum declaration" {
		t.Errorf("broken very program: %v", err)
	}
	if _, err := parse(strings.Replace(minimal, "Cool.", "This code is very very cool.", 1)); err == nil || err.Error() != "ERROR, missing or invalid coolness" {
		t.Errorf("a dialect not spoken yet: %v", err)
	}
}

// Very Sorted! names logical operations and has a real NAND (#26), "of not
// both X and Y", with German for it and for the original's NOR.
func TestVeryNand(t *testing.T) {
	very := func(nands, impl string) string {
		s := strings.Replace(minimal, "This code does not use any logical operations.", nands, 1)
		s = strings.Replace(s, "This code implements the first output.", impl, 1)
		return strings.Replace(s, "Cool.", "This code is very cool.", 1)
	}
	n0, n1 := Operand{Number, 0}, Operand{Number, 1}
	for _, tt := range []struct {
		name, nands string
		want        []Slide
	}{
		{"english nand", "This code uses the logical operation of not both the first number and the second number.",
			[]Slide{{Ops: [2]Operand{n0, n1}, Flags: LogicalNand}}},
		{"english nor", "This code uses the logical operation of not the first number and not the second number.",
			[]Slide{{Ops: [2]Operand{n0, n1}, Flags: LogicalNor}}},
		{"english list", "This code uses the logical operations of not both the first number and the second number, of not the second number and not the first number, and of not both the second number and the second number.",
			[]Slide{{Ops: [2]Operand{n0, n1}, Flags: LogicalNand}, {Ops: [2]Operand{n1, n0}, Flags: LogicalNor}, {Ops: [2]Operand{n1, n1}, Flags: LogicalNand}}},
		{"german nand", "Dieses Programm benutzt die logische Verknüpfung von nicht beiden, der ersten Zahl und der zweiten Zahl.",
			[]Slide{{Ops: [2]Operand{n0, n1}, Flags: LogicalNand}}},
		{"german list", "Dieses Programm benutzt die logischen Verknüpfungen von nicht beiden, der ersten Zahl und der zweiten Zahl, von nicht der zweiten Zahl und nicht der ersten Zahl, und von nicht beiden, der ersten logischen Verknüpfung und der ersten Zahl.",
			[]Slide{{Ops: [2]Operand{n0, n1}, Flags: LogicalNand}, {Ops: [2]Operand{n1, n0}, Flags: LogicalNor}, {Ops: [2]Operand{{Nand, 0}, n0}, Flags: LogicalNand}}},
	} {
		p, err := parse(very(tt.nands, "This code implements the first output."))
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if got := p.Entries(Nands); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
	// References, in both languages and cases.
	nand := "This code uses the logical operation of not both the first number and the second number."
	for _, ref := range []string{"the first logical operation", "die erste logische Verknüpfung", "der ersten logischen Verknuepfung"} {
		p, err := parse(very(nand, "This code implements "+ref+"."))
		if err != nil || p.Entries(Statements)[0].Ops[0] != (Operand{Nand, 0}) {
			t.Errorf("%s: %v", ref, err)
		}
	}
	// The original has neither the NAND nor a reference: its errors stand.
	if _, err := parse(strings.Replace(very(nand, "This code implements the first output."), "This code is very cool.", "Cool.", 1)); err == nil || err.Error() != "ERROR, missing or invalid declaration of logical operations" {
		t.Errorf("NAND in the original's Sorted!: %v", err)
	}
	nor := "This code uses the logical operation of not the first number and not the second number."
	if _, err := parse(strings.Replace(very(nor, "This code implements the first logical operation."), "This code is very cool.", "Cool.", 1)); err == nil || err.Error() != "ERROR, missing or invalid declaration of implementation" {
		t.Errorf("a logical operation reference in the original's Sorted!: %v", err)
	}
	// German declarations are very only, even with the original's English
	// operation after them.
	for _, decl := range []string{
		"Dieses Programm benutzt die logische Verknuepfung von nicht der ersten Zahl und nicht der zweiten Zahl.",
		"Dieses Programm benutzt die logische Verknuepfung of not the first number and not the second number.",
	} {
		if _, err := parse(strings.Replace(very(decl, "This code implements the first output."), "This code is very cool.", "Cool.", 1)); err == nil || err.Error() != "ERROR, missing or invalid declaration of logical operations" {
			t.Errorf("%s in the original's Sorted!: %v", decl, err)
		}
	}
}

// germanVery is a Very Sorted! program in German, written in UTF-8 (#28).
const germanVery = `Dieses Programm benutzt die Zahlen fünf, zwölf, dreißig, und fünfunddreißig.
Dieses Programm geht nirgendwo hin.
Dieses Programm schreibt die vierte Zahl als eine deutsche Ordinalzahl.
Dieses Programm kann nicht lesen.
Dieses Programm benutzt keine Summen.
Dieses Programm benutzt keine Bedingungen.
Dieses Programm benutzt keine Sprungziele.
Dieses Programm benutzt keine geordneten Differenzen.
Dieses Programm benutzt keine Zuweisungen.
Dieses Programm benutzt keine Produkte.
Dieses Programm implementiert die erste Ausgabe.
Dieses Programm benutzt keine Verhältnisse.
Dieses Programm ist unlogisch.
Dieses Programm ist ganz hervorragend.`

// Very Sorted! reads UTF-8 (#28): German is written with umlauts and ß, in
// any case and either Unicode form, and the ASCII spellings still work. The
// original's dialect reads bytes, as before.
func TestVeryUTF8(t *testing.T) {
	want := []int32{5, 12, 30, 35}
	for _, tt := range []struct{ name, src string }{
		{"umlauts and ß", germanVery},
		{"upper case", strings.ToUpper(germanVery)},
		{"capital sharp s", strings.Replace(germanVery, "dreißig,", "DREIẞIG,", 1)},
		{"decomposed", strings.ReplaceAll(strings.ReplaceAll(germanVery, "ü", "u\u0308"), "ö", "o\u0308")},
		{"ASCII spelling", strings.NewReplacer("ü", "ue", "ö", "oe", "ä", "ae", "ß", "ss").Replace(germanVery)},
		{"with a byte order mark and CRLF", "\ufeff" + strings.ReplaceAll(germanVery, "\n", "\r\n")},
	} {
		p, err := parse(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !p.Very || !slices.Equal(p.Data[:4], want) || p.Entries(Writes)[0].Flags != FormatGermanOrdinal {
			t.Errorf("%s: very %v, data %v", tt.name, p.Very, p.Data[:4])
		}
	}
	// The original reads "fünf" as "f nf": its error stands.
	if _, err := parse(strings.Replace(germanVery, "ganz hervorragend", "hervorragend", 1)); err == nil || err.Error() != "ERROR, missing or invalid number declaration" {
		t.Errorf("UTF-8 in the original's dialect: %v", err)
	}
	// Letters of any script are letters, so a stray one is a word that is
	// not Sorted!. In the original it was a space.
	stray := strings.Replace(germanVery, "Dieses Programm geht", "Dieses Programm é geht", 1)
	if _, err := parse(stray); err == nil {
		t.Error("a stray letter in Very Sorted! parses")
	}
	if p, err := parse(strings.Replace(minimal, "uses the number", "uses é the number", 1)); err != nil || p.Very {
		t.Errorf("a stray letter in the original's Sorted!: %v", err)
	}
}

func TestFilterVery(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"letters, period and comma survive", "Ab.,c", "ab.,c"},
		{"everything else becomes a space", "a1-b\tc\"d", "a  b c d"},
		{"umlauts are written out", "Zwölf Fünf Verhältnis", "zwoelf fuenf verhaeltnis"},
		{"ß folds to ss", "Dreißig DREIẞIG", "dreissig dreissig"},
		{"decomposed umlauts compose first", "fu\u0308nf", "fuenf"},
		{"other letters stay", "Été 日本 ελλά", "été 日本 ελλά"},
		{"combining marks stay", "q\u0301", "q\u0301"},
		{"not UTF-8 becomes spaces", "a\xffb", "a b"},
		{"Ctrl-Z and NUL are just characters", "ab\x1acd\x00ef", "ab cd ef"},
		{"CRLF", "a\r\nb", "a  b"},
		{"byte order mark", "\ufeffa", " a"},
		{"punctuation of other scripts", "a\u3002b\u00a0c", "a b c"},
	}
	for _, tt := range tests {
		if got := FilterVery([]byte(tt.in)); got != tt.want {
			t.Errorf("%s: FilterVery(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}

func TestMinimal(t *testing.T) {
	p, err := parse(minimal)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Data) != 1 || p.Data[0] != 7 || p.TypeCount != 2 {
		t.Errorf("data %v, TypeCount %d", p.Data, p.TypeCount)
	}
	if w := p.Entries(Writes); len(w) != 1 || w[0].Flags != FormatCharacter || w[0].Ops[0] != (Operand{Number, 0}) {
		t.Errorf("writes %v", w)
	}
}

// Each sentence that is missing or invalid is reported by its own name, in
// the fixed order of the original.
func TestErrorPerSentence(t *testing.T) {
	sentences := strings.Split(minimal, "\n")
	names := []string{"number declaration", "declaration of jumps", "declaration of output", "declaration of input",
		"sum declaration", "declaration of conditions", "label declaration", "declaration of ordered differences",
		"declaration of assignments", "declaration of products", "declaration of implementation",
		"declaration of ratios", "declaration of logical operations", "coolness"}
	for i := range sentences {
		broken := append([]string{}, sentences...)
		broken[i] = "This code is broken."
		_, err := parse(strings.Join(broken, "\n"))
		var e *Error
		if !errors.As(err, &e) || e.What != names[i] {
			t.Errorf("sentence %d broken: err %v, want %q", i, err, names[i])
			continue
		}
		if want := "ERROR, missing or invalid " + names[i]; err.Error() != want {
			t.Errorf("message %q, want %q", err.Error(), want)
		}
	}
}

// replace swaps one sentence of minimal, identified by its prefix.
func replace(prefix, with string) string {
	lines := strings.Split(minimal, "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, prefix) {
			lines[i] = with
		}
	}
	return strings.Join(lines, "\n")
}

func TestSentences(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string // "" for success
		check   func(*Program) error
	}{
		{"duplicate numbers are refused", replace("This code uses the number", "This code uses the numbers seven, and seven."), "number declaration", nil},
		{"a number list needs the final comma and", replace("This code uses the number", "This code uses the numbers seven and eight."), "number declaration", nil},
		{"german numbers", replace("This code uses the number", "Dieses Programm benutzt die Zahlen sieben, acht, und neun."), "", func(p *Program) error {
			return expect(fmt.Sprint(p.Data) == "[7 8 9]", "data %v", p.Data)
		}},
		{"no numbers", replace("This code uses the number", "This code does not use any numbers."), "", nil},
		{"case does not matter", strings.ToUpper(minimal), "", nil},
		{"text after cool is ignored", minimal + " And now for something completely different", "", nil},
		{"is cool", replace("Cool.", "This code is cool."), "", nil},
		{"hervorragend", replace("Cool.", "Hervorragend."), "", nil},
		{"ist hervorragend", replace("Cool.", "Dieses Programm ist hervorragend."), "", nil},
		{"mixed languages", replace("This code cannot read.", "Dieses Programm kann nicht lesen."), "", nil},

		// Output formats: the alternatives share an unrestored cursor.
		{"english cardinal", replace("This code writes", "This code writes the first number as a english cardinal."), "", flagsOf(FormatEnglishCardinal)},
		{"german cardinal", replace("This code writes", "This code writes the first number as a german cardinal."), "", flagsOf(FormatGermanCardinal)},
		{"english ordinal never parses", replace("This code writes", "This code writes the first number as a english ordinal."), "declaration of output", nil},
		{"german ordinal never parses", replace("This code writes", "This code writes the first number as a german ordinal."), "declaration of output", nil},
		{"ein Zeichen", replace("This code writes", "Dieses Programm schreibt die erste Zahl als ein Zeichen."), "", flagsOf(FormatCharacter)},
		{"ein deutscher Kardinal never parses", replace("This code writes", "Dieses Programm schreibt die erste Zahl als ein deutscher Kardinal."), "declaration of output", nil},
		// Each failed alternative eats a word, so doubling it gets through.
		{"english english ordinal", replace("This code writes", "This code writes the first number as a english english ordinal."), "", flagsOf(FormatEnglishOrdinal)},
		{"german german ordinal", replace("This code writes", "This code writes the first number as a german german ordinal."), "", flagsOf(FormatGermanOrdinal)},
		{"ein ein englischer Kardinal", replace("This code writes", "Dieses Programm schreibt die erste Zahl als ein ein englischer Kardinal."), "", flagsOf(FormatEnglishCardinal)},
		{"ein ein ein englische Ordinalzahl", replace("This code writes", "Dieses Programm schreibt die erste Zahl als ein ein ein englische Ordinalzahl."), "", flagsOf(FormatEnglishOrdinal)},
		{"ein ein ein ein deutscher Kardinal", replace("This code writes", "Dieses Programm schreibt die erste Zahl als ein ein ein ein deutscher Kardinal."), "", flagsOf(FormatGermanCardinal)},
		{"eine deutsche Ordinalzahl", replace("This code writes", "Dieses Programm schreibt die erste Zahl als eine deutsche Ordinalzahl."), "", flagsOf(FormatGermanOrdinal)},
		{"only one output per program", replace("This code writes", "This code writes the first number as a character, and the second number as a character."), "declaration of output", nil},
		{"no output", replace("This code writes", "This code cannot write."), "", nil},

		{"input", replace("This code cannot read.", "This code reads the second number as a character."), "", func(p *Program) error {
			r := p.Entries(Reads)
			return expect(len(r) == 1 && r[0].Ops[0] == Operand{Number, 1}, "reads %v", r)
		}},
		{"indirect operand", replace("This code writes", "This code writes the cell indexed by the first number as a character."), "", func(p *Program) error {
			w := p.Entries(Writes)
			return expect(w[0].Ops[0] == Operand{Number | Indirect, 0}, "op %v", w[0].Ops[0])
		}},
		{"a period directly after a single assignment", replace("This code does not use any assignments.", "This code assigns the first number to the second number."), "", func(p *Program) error {
			return expect(p.Tables[Assigns].Count == 1 && p.TypeCount == 3, "assigns %v, TypeCount %d", p.Tables[Assigns], p.TypeCount)
		}},
		{"two assignments leave one extra TypeCount", replace("This code does not use any assignments.", "This code assigns the first number to the second number, and the second number to the third number."), "", func(p *Program) error {
			return expect(p.Tables[Assigns].Count == 2 && p.TypeCount == 5, "assigns %v, TypeCount %d", p.Tables[Assigns], p.TypeCount)
		}},
		{"assignment target must be a number", replace("This code does not use any assignments.", "This code assigns the first number to the first sum."), "declaration of assignments", nil},
		{"indirect assignment target", replace("This code does not use any assignments.", "This code assigns the first number to the cell indexed by the second number."), "", nil},
		{"labels", replace("This code does not use any labels.", "This code uses two labels."), "", func(p *Program) error {
			return expect(p.LabelsCount == 2, "labels %d", p.LabelsCount)
		}},
		{"one label", replace("This code does not use any labels.", "This code uses one label."), "", nil},
		{"sprungziele", replace("This code does not use any labels.", "Dieses Programm benutzt drei Sprungziele."), "", func(p *Program) error {
			return expect(p.LabelsCount == 3, "labels %d", p.LabelsCount)
		}},
		{"jump", replace("This code does never go anywhere.", "This code always goes to the first label."), "", func(p *Program) error {
			j := p.Entries(Jumps)
			return expect(len(j) == 1 && j[0].Ops[0] == Operand{Label, 0} && j[0].Flags == UnconditionalJump, "jumps %v", j)
		}},
		{"jump target must be a direct label", replace("This code does never go anywhere.", "This code always goes to the cell indexed by the first label."), "declaration of jumps", nil},
		{"conditional jump needs a condition", replace("This code does never go anywhere.", "This code sometimes goes to the first label if the first sum is true."), "declaration of jumps", nil},
		{"less than", replace("This code does not use any conditions.", "This code uses the condition that the first number is less than the second number."), "", func(p *Program) error {
			c := p.Entries(Conditions)
			return expect(len(c) == 1 && c[0].Flags == CompareLess, "conditions %v", c)
		}},
		{"ratio", replace("This code does not use any ratios.", "This code uses the ratio of the first number to the second number."), "", func(p *Program) error {
			return expect(p.Tables[Ratios].Count == 1, "ratios %v", p.Tables[Ratios])
		}},
		{"logical operation", replace("This code does not use any logical operations.", "This code uses the logical operation of not the first number and not the second number."), "", func(p *Program) error {
			return expect(p.Tables[Nands].Count == 1, "nands %v", p.Tables[Nands])
		}},
		{"products", replace("This code does not use any products.", "This code uses the products of the first number and the second number, and of the third number and the first sum."), "", func(p *Program) error {
			return expect(p.Tables[Prods].Count == 2, "prods %v", p.Tables[Prods])
		}},
		{"a product list needs the final comma and", replace("This code does not use any products.", "This code uses the products of the first number and the second number."), "declaration of products", nil},
		{"german plural differenzen is no list", replace("This code does not use any ordered differences.", "Dieses Programm benutzt die geordnete Differenzen zwischen der ersten Zahl und der zweiten Zahl, und zwischen der zweiten Zahl und der ersten Zahl."), "declaration of ordered differences", nil},
		{"german ordered difference", replace("This code does not use any ordered differences.", "Dieses Programm benutzt die geordnete Differenz zwischen der ersten Zahl und der zweiten Zahl."), "", nil},
		{"german ordered differences cannot be a list", replace("This code does not use any ordered differences.", "Dieses Programm benutzt die geordnete Differenz zwischen der ersten Zahl und der zweiten Zahl, und zwischen der zweiten Zahl und der ersten Zahl."), "declaration of ordered differences", nil},
		{"any identifier is a statement", replace("This code implements the first output.", "This code implements the first sum, and the second output."), "", func(p *Program) error {
			s := p.Entries(Statements)
			return expect(len(s) == 2 && s[0].Ops[0] == Operand{Sum, 0} && s[1].Ops[0] == Operand{Write, 1}, "statements %v", s)
		}},
		{"ratio list", replace("This code does not use any ratios.", "This code uses the ratios of the first number to the second number, and of the second number to the first number."), "", func(p *Program) error {
			return expect(p.Tables[Ratios].Count == 2, "ratios %v", p.Tables[Ratios])
		}},
		{"logical operation list", replace("This code does not use any logical operations.", "This code uses the logical operations of not the first number and not the second number, and of not the second number and not the first number."), "", func(p *Program) error {
			return expect(p.Tables[Nands].Count == 2, "nands %v", p.Tables[Nands])
		}},
		{"german sum list", replace("This code does not use any sums.", "Dieses Programm benutzt die Summen aus der ersten Zahl und der zweiten Zahl, und von der zweiten Zahl und der ersten Zahl."), "", func(p *Program) error {
			return expect(p.Tables[Sums].Count == 2, "sums %v", p.Tables[Sums])
		}},
		{"list item failing after the final and", replace("This code does not use any sums.", "This code uses the sums of the first number and the second number, and of nothing."), "sum declaration", nil},
		{"duplicate inside a number list", replace("This code uses the number", "This code uses the numbers one, two, one, and three."), "number declaration", nil},
		{"duplicate after the final and", replace("This code uses the number", "This code uses the numbers one, and one."), "number declaration", nil},
		{"every reference type", replace("This code implements the first output.", "This code implements the first sum, the first ordered difference, the first product, the first ratio, the first cell, the first assignment, the first jump, the first label, the first condition, and the first output."), "", func(p *Program) error {
			var types []OperandType
			for _, s := range p.Entries(Statements) {
				types = append(types, s.Ops[0].Type)
			}
			return expect(fmt.Sprint(types) == fmt.Sprint([]OperandType{Sum, Diff, Prod, Ratio, Cell, Assign, Jump, Label, Condition, Write}), "types %v", types)
		}},
		{"german reference types", replace("This code implements the first output.", "Dieses Programm implementiert die erste Summe, die erste geordnete Differenz, das erste Produkt, das erste Verhaeltnis, die erste Zelle, die erste Zuweisung, den ersten Sprungbefehl, das erste Sprungziel, die erste Bedingung, und die erste Ausgabe."), "", func(p *Program) error {
			return expect(p.Tables[Statements].Count == 10, "statements %v", p.Tables[Statements])
		}},
		{"unknown reference type", replace("This code implements the first output.", "This code implements the first nothing."), "declaration of implementation", nil},
		{"german comparisons", replace("This code does not use any conditions.", "Dieses Programm benutzt die Bedingung dass die erste Zahl ist gleich der zweiten Zahl, und die Bedingung dass die erste Zahl ist kleiner als die zweite Zahl."), "", func(p *Program) error {
			c := p.Entries(Conditions)
			return expect(len(c) == 2 && c[0].Flags == CompareEqual && c[1].Flags == CompareLess, "conditions %v", c)
		}},
		{"unknown comparison", replace("This code does not use any conditions.", "This code uses the condition that the first number is greater than the second number."), "declaration of conditions", nil},
		{"input only as a character", replace("This code cannot read.", "This code reads the second number as a english cardinal."), "declaration of input", nil},
		{"only one input per program", replace("This code cannot read.", "This code reads the second number as a character, and the third number as a character."), "declaration of input", nil},
		{"german input", replace("This code cannot read.", "Dieses Programm liest die zweite Zahl als ein Zeichen."), "", nil},
		{"ends right after cool", strings.TrimSuffix(minimal, "\n"), "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parse(tt.src)
			if tt.wantErr != "" {
				var e *Error
				if !errors.As(err, &e) || e.What != tt.wantErr {
					t.Fatalf("err %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				if err := tt.check(p); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// A withdrawn single statement leaves TypeCount ahead of the entries written,
// so the empty tables after it start past the end of Code.
func TestEmptyTablesPastCode(t *testing.T) {
	p, err := parse(replace("This code implements the first output.", "This code implements the first output, the first output, and the first output."))
	if err != nil {
		t.Fatal(err)
	}
	if p.TypeCount != 5 || len(p.Code) != 4 || p.Tables[Ratios].Index != 5 {
		t.Fatalf("TypeCount %d, len(Code) %d, ratios %v", p.TypeCount, len(p.Code), p.Tables[Ratios])
	}
	for c := Sums; c < NumCategories; c++ {
		if got, want := len(p.Entries(c)), p.Tables[c].Count; got != want {
			t.Errorf("category %d: %d entries, want %d", c, got, want)
		}
	}
}

func expect(ok bool, format string, args ...any) error {
	if ok {
		return nil
	}
	return fmt.Errorf(format, args...)
}

func flagsOf(f int32) func(*Program) error {
	return func(p *Program) error {
		w := p.Entries(Writes)
		return expect(len(w) == 1 && w[0].Flags == f, "writes %v, want flags %d", w, f)
	}
}

func TestFilter(t *testing.T) {
	long := strings.Repeat("a", 1022)
	tests := []struct {
		name, in, want string
	}{
		{"letters, period and comma survive", "Ab.,c", "Ab.,c"},
		{"everything else becomes a space", "a1-b\tc\"d", "a  b c d"},
		{"non-ASCII bytes become spaces", "Zw\xc3\xb6lf", "Zw  lf"},
		{"CRLF is one line end", "a\r\nb", "a b"},
		{"a lone CR is a space", "a\rb", "a b"},
		{"Ctrl-Z ends the file", "ab\x1acd", "ab"},
		{"Ctrl-Z at a line start", "ab\n\x1acd", "ab "},
		{"NUL drops the rest of the line", "ab\x00cd\nef", "abef"},
		// fgets reads 1023 bytes: a NUL in the next chunk drops only that chunk's rest.
		{"NUL in the second chunk of a long line", long + "xa\x00yy\nz", long + "xaz"},
		{"NUL right after a full chunk", long + "x" + "\x00yy\nz", long + "xz"},
		{"long line without NUL", long + "xyz", long + "xyz"},
	}
	for _, tt := range tests {
		if got := Filter([]byte(tt.in)); got != tt.want {
			t.Errorf("%s: Filter(%q) = %q, want %q", tt.name, tt.in, got, tt.want)
		}
	}
}
