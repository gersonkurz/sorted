// Package render writes a program's tables as Sorted! source: the inverse of
// the parser, in English or German, laid out to be sung.
//
// What it writes is exactly what the parser accepts. Some things have no
// form in one language, or none at all:
//
//   - German has no ratios, no logical operations and no lists of ordered
//     differences, so those sentences fall back to English. Sorted! allows
//     mixing languages per sentence.
//   - Output formats are written the only way the parser accepts them. Its
//     alternatives share an unrestored cursor, so each failed alternative eats
//     a word: "as a english english ordinal", "as a german german ordinal",
//     "als ein ein englischer Kardinal", "als ein ein ein englische
//     Ordinalzahl", "als ein ein ein ein deutscher Kardinal".
//   - More than one output or input, references to inputs or logical
//     operations, negative numbers and numbers from 1000000000 on cannot be
//     written at all; Render reports an *Error.
//
// Render checks its own work: it parses the text it produced and fails unless
// the result is the same program in every observable respect, table layout
// and scratch slots included (references past a table read them). A program
// whose layout the natural sentence forms cannot reproduce is refused.
package render

import (
	"fmt"
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// Lang selects the language of the generated program.
type Lang int

// The languages Sorted! speaks.
const (
	English Lang = iota
	German
)

// Error reports a program that cannot be written as Sorted! source.
type Error struct {
	What string
}

func (e *Error) Error() string { return "cannot render: " + e.What }

func fail(format string, args ...any) error { return &Error{fmt.Sprintf(format, args...)} }

// Render writes p as Sorted! source in lang.
func Render(p *syntax.Program, lang Lang) (string, error) {
	r := &renderer{p: p, lang: lang}
	sentences := []func() (sentence, error){
		r.numbers, r.jumps, r.outputs, r.inputs, r.sums, r.conditions, r.labels,
		r.diffs, r.assigns, r.prods, r.implementation, r.ratios, r.nands, r.cool,
	}
	var b strings.Builder
	for _, f := range sentences {
		s, err := f()
		if err != nil {
			return "", err
		}
		b.WriteString(s.text())
		b.WriteString("\n")
	}
	text := b.String()
	if err := check(p, text); err != nil {
		return "", err
	}
	return text, nil
}

// check parses text and compares the result with p.
func check(p *syntax.Program, text string) error {
	q, err := syntax.Parse(syntax.Filter([]byte(text)))
	if err != nil {
		return fail("the generated text does not parse (%v)", err)
	}
	if !Equal(p, q) {
		return fail("the program's table layout cannot be reproduced in Sorted! text")
	}
	return nil
}

// Equal reports whether two programs are the same in every respect the
// interpreter can observe: declared numbers, label count, element count,
// table positions, and every slot of the static Code array (zero past what
// was written), since references past a table read neighbouring slots.
func Equal(p, q *syntax.Program) bool {
	if p.LabelsCount != q.LabelsCount || p.TypeCount != q.TypeCount || p.Tables != q.Tables ||
		fmt.Sprint(p.Data) != fmt.Sprint(q.Data) {
		return false
	}
	for i := 0; i < len(p.Code) || i < len(q.Code); i++ {
		if slot(p, i) != slot(q, i) {
			return false
		}
	}
	return true
}

func slot(p *syntax.Program, i int) syntax.Slide {
	if i < len(p.Code) {
		return p.Code[i]
	}
	return syntax.Slide{}
}

// sentence is a Sorted! sentence: a head, then one item, or a list whose last
// item follows ", and"/", und", then a period.
type sentence struct {
	head  string
	items []string
	conj  string
}

// width is the line length up to which a sentence stays on one line.
const width = 78

// text lays the sentence out: on one line if it fits, otherwise as a verse
// with one item per indented line.
func (s sentence) text() string {
	n := len(s.items)
	var line string
	switch n {
	case 0:
		return s.head + "."
	case 1:
		line = s.head + " " + s.items[0] + "."
	default:
		line = s.head + " " + strings.Join(s.items[:n-1], ", ") + ", " + s.conj + " " + s.items[n-1] + "."
	}
	if len(line) <= width {
		return line
	}
	var b strings.Builder
	b.WriteString(s.head)
	for i, item := range s.items {
		b.WriteString("\n\t")
		switch {
		case n == 1:
			b.WriteString(item + ".")
		case i < n-1:
			b.WriteString(item + ",")
		default:
			b.WriteString(s.conj + " " + item + ".")
		}
	}
	return b.String()
}

type renderer struct {
	p    *syntax.Program
	lang Lang
}

// say picks the English or the German text.
func (r *renderer) say(en, de string) string {
	if r.lang == German {
		return de
	}
	return en
}

// list makes a sentence in language l.
func list(l Lang, head string, items []string) sentence {
	conj := "and"
	if l == German {
		conj = "und"
	}
	return sentence{head: head, items: items, conj: conj}
}

func (r *renderer) entries(c syntax.Category) []syntax.Slide { return r.p.Entries(c) }

// --- numbers ---

func (r *renderer) numbers() (sentence, error) {
	data := r.p.Data
	if len(data) == 0 {
		return list(r.lang, r.say("This code does not use any numbers", "Dieses Programm benutzt keine Zahlen"), nil), nil
	}
	items := make([]string, len(data))
	for i, d := range data {
		w, err := cardinal(r.lang, d)
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	if len(items) == 1 {
		return list(r.lang, r.say("This code uses the number", "Dieses Programm benutzt die Zahl"), items), nil
	}
	return list(r.lang, r.say("This code uses the numbers", "Dieses Programm benutzt die Zahlen"), items), nil
}

// cardinal writes a declarable number: zero, or 1 to 999999999.
func cardinal(l Lang, n int32) (string, error) {
	switch {
	case n == 0 && l == German:
		return "null", nil
	case n == 0:
		return "zero", nil
	case n < 0:
		return "", fail("negative numbers cannot be declared (%d)", n)
	case n >= 1000000000:
		return "", fail("numbers from 1000000000 on cannot be declared (%d)", n)
	case l == German:
		return numbers.GermanCardinal(n)
	}
	return numbers.EnglishCardinal(n)
}

// --- references ---

// grammatical case of a German reference
type gcase int

const (
	nominative gcase = iota
	accusative
	dative
)

type noun struct {
	en, de string
	gender byte // 'f', 'm' or 'n'
}

var nouns = map[syntax.OperandType]noun{
	syntax.Number:    {"number", "Zahl", 'f'},
	syntax.Sum:       {"sum", "Summe", 'f'},
	syntax.Diff:      {"ordered difference", "geordnete Differenz", 'f'},
	syntax.Prod:      {"product", "Produkt", 'n'},
	syntax.Ratio:     {"ratio", "Verhaeltnis", 'n'},
	syntax.Assign:    {"assignment", "Zuweisung", 'f'},
	syntax.Jump:      {"jump", "Sprungbefehl", 'm'},
	syntax.Label:     {"label", "Sprungziel", 'n'},
	syntax.Condition: {"condition", "Bedingung", 'f'},
	syntax.Write:     {"output", "Ausgabe", 'f'},
}

// ref writes a reference such as "the third number" or "der dritten Zahl".
func (r *renderer) ref(l Lang, op syntax.Operand, c gcase) (string, error) {
	if op.Type&syntax.Indirect != 0 {
		inner, err := r.ref(l, syntax.Operand{Type: op.Type &^ syntax.Indirect, Index: op.Index}, accusative)
		if err != nil {
			return "", err
		}
		if l == German {
			return "diejenige Zelle die indiziert wird durch " + inner, nil
		}
		return "the cell indexed by " + inner, nil
	}
	nn, ok := nouns[op.Type]
	if !ok {
		return "", fail("there is no way to refer to an operand of type %d", op.Type)
	}
	n := op.Index + 1
	if n < 1 {
		return "", fail("reference index %d out of range", op.Index)
	}
	if l == English {
		// "eighth" does not read back: the cardinal "eight" matches and the
		// "h" is left over. Like itoa.s ("the eight number"), fall back to the
		// cardinal, which the ordinal parser accepts too.
		ord, _ := numbers.EnglishOrdinal(n)
		if !ordinalParses(ord, n, nn.en) {
			if ord, _ = numbers.EnglishCardinal(n); !ordinalParses(ord, n, nn.en) {
				return "", fail("no English ordinal for %d reads back", n)
			}
		}
		return "the " + ord + " " + nn.en, nil
	}
	ord, _ := numbers.GermanOrdinal(n)
	article := map[byte][3]string{
		'f': {"die", "die", "der"},
		'n': {"das", "das", "das"}, // "dem" is not a keyword
		'm': {"der", "den", "den"}, // "dem" is not a keyword
	}[nn.gender][c]
	// Inflect where the grammar wants it ("der ersten Zahl") and the parser
	// reads it back; from "zwanzigsten" on it does not.
	inflect := c == dative && nn.gender == 'f' || c != nominative && nn.gender == 'm'
	if inflect && ordinalParses(ord+"n", n, nn.de) {
		ord += "n"
	} else if !ordinalParses(ord, n, nn.de) {
		return "", fail("no German ordinal for %d reads back", n)
	}
	return article + " " + ord + " " + nn.de, nil
}

// ordinalParses reports whether "<ord> <noun>" reads back as n with the
// cursor on the noun, trying English first and German second, as the parser
// does.
func ordinalParses(ord string, n int32, noun string) bool {
	s := ord + " " + noun
	v, next := numbers.ParseEnglishOrdinal(s, 0)
	if v == 0 {
		v, next = numbers.ParseGermanOrdinal(s, next)
	}
	return v == n && s[next:] == noun
}

// pair writes "<a> <word> <b>" for two operands.
func (r *renderer) pair(l Lang, s syntax.Slide, word string, c gcase) (string, error) {
	a, err := r.ref(l, s.Ops[0], c)
	if err != nil {
		return "", err
	}
	b, err := r.ref(l, s.Ops[1], c)
	if err != nil {
		return "", err
	}
	return a + " " + word + " " + b, nil
}

// --- jumps ---

func (r *renderer) jumps() (sentence, error) {
	js := r.entries(syntax.Jumps)
	if len(js) == 0 {
		return list(r.lang, r.say("This code does never go anywhere", "Dieses Programm geht nirgendwo hin"), nil), nil
	}
	items := make([]string, len(js))
	for i, j := range js {
		if j.Ops[0].Type != syntax.Label {
			return sentence{}, fail("jump %d does not go to a label", i+1)
		}
		label, err := r.ref(r.lang, j.Ops[0], accusative)
		if err != nil {
			return sentence{}, err
		}
		if j.Flags == syntax.UnconditionalJump {
			items[i] = r.say("always goes to ", "springt immer an ") + label
			continue
		}
		if j.Ops[1].Type != syntax.Condition {
			return sentence{}, fail("jump %d does not depend on a condition", i+1)
		}
		cond, err := r.ref(r.lang, j.Ops[1], nominative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.say("sometimes goes to "+label+" if "+cond+" is true", "springt manchmal an "+label+" wenn "+cond+" wahr ist")
	}
	return list(r.lang, r.say("This code", "Dieses Programm"), items), nil
}

// --- output and input ---

func (r *renderer) outputs() (sentence, error) {
	ws := r.entries(syntax.Writes)
	switch len(ws) {
	case 0:
		return list(r.lang, r.say("This code cannot write", "Dieses Programm kann nicht schreiben"), nil), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one output")
	}
	w := ws[0]
	l := r.lang
	formats := map[int32][2]string{
		syntax.FormatCharacter:       {"as a character", "als ein Zeichen"},
		syntax.FormatEnglishCardinal: {"as a english cardinal", "als ein ein englischer Kardinal"},
		syntax.FormatEnglishOrdinal:  {"as a english english ordinal", "als ein ein ein englische Ordinalzahl"},
		syntax.FormatGermanCardinal:  {"as a german cardinal", "als ein ein ein ein deutscher Kardinal"},
		syntax.FormatGermanOrdinal:   {"as a german german ordinal", "als eine deutsche Ordinalzahl"},
	}
	phrases, ok := formats[w.Flags]
	if !ok {
		return sentence{}, fail("output format %d does not exist", w.Flags)
	}
	format := phrases[l]
	what, err := r.ref(l, w.Ops[0], accusative)
	if err != nil {
		return sentence{}, err
	}
	return list(l, r.say("This code writes", "Dieses Programm schreibt"), []string{what + " " + format}), nil
}

func (r *renderer) inputs() (sentence, error) {
	rs := r.entries(syntax.Reads)
	switch len(rs) {
	case 0:
		return list(r.lang, r.say("This code cannot read", "Dieses Programm kann nicht lesen"), nil), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one input")
	}
	if rs[0].Flags != syntax.FormatCharacter {
		return sentence{}, fail("input format %d does not exist", rs[0].Flags)
	}
	what, err := r.ref(r.lang, rs[0].Ops[0], accusative)
	if err != nil {
		return sentence{}, err
	}
	return list(r.lang, r.say("This code reads", "Dieses Programm liest"), []string{what + r.say(" as a character", " als ein Zeichen")}), nil
}

// --- expressions ---

// binary writes the sentence of an expression table. single and plural are
// the heads for one entry and for a list; prep starts each entry ("of",
// "between"), word joins its operands ("and", "to").
func (r *renderer) binary(l Lang, c syntax.Category, none, single, plural, prep, word string) (sentence, error) {
	es := r.entries(c)
	if len(es) == 0 {
		return list(l, none, nil), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		s, err := r.pair(l, e, word, dative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = prep + " " + s
	}
	if len(items) == 1 {
		return list(l, single, items), nil
	}
	return list(l, plural, items), nil
}

func (r *renderer) sums() (sentence, error) {
	if r.lang == German {
		return r.binary(German, syntax.Sums, "Dieses Programm benutzt keine Summen", "Dieses Programm benutzt die Summe", "Dieses Programm benutzt die Summen", "aus", "und")
	}
	return r.binary(English, syntax.Sums, "This code does not use any sums", "This code uses the sum", "This code uses the sums", "of", "and")
}

func (r *renderer) diffs() (sentence, error) {
	if r.lang == German && len(r.entries(syntax.Diffs)) < 2 {
		return r.binary(German, syntax.Diffs, "Dieses Programm benutzt keine geordneten Differenzen", "Dieses Programm benutzt die geordnete Differenz", "", "zwischen", "und")
	}
	return r.binary(English, syntax.Diffs, "This code does not use any ordered differences", "This code uses the ordered difference", "This code uses the ordered differences", "between", "and")
}

func (r *renderer) prods() (sentence, error) {
	if r.lang == German {
		return r.binary(German, syntax.Prods, "Dieses Programm benutzt keine Produkte", "Dieses Programm benutzt das Produkt", "Dieses Programm benutzt die Produkte", "von", "und")
	}
	return r.binary(English, syntax.Prods, "This code does not use any products", "This code uses the product", "This code uses the products", "of", "and")
}

func (r *renderer) ratios() (sentence, error) {
	if r.lang == German && len(r.entries(syntax.Ratios)) == 0 {
		return list(German, "Dieses Programm benutzt keine Verhaeltnisse", nil), nil
	}
	return r.binary(English, syntax.Ratios, "This code does not use any ratios", "This code uses the ratio", "This code uses the ratios", "of", "to")
}

func (r *renderer) nands() (sentence, error) {
	es := r.entries(syntax.Nands)
	if len(es) == 0 {
		return list(r.lang, r.say("This code does not use any logical operations", "Dieses Programm ist unlogisch"), nil), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		a, err := r.ref(English, e.Ops[0], nominative)
		if err != nil {
			return sentence{}, err
		}
		b, err := r.ref(English, e.Ops[1], nominative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = "of not " + a + " and not " + b
	}
	if len(items) == 1 {
		return list(English, "This code uses the logical operation", items), nil
	}
	return list(English, "This code uses the logical operations", items), nil
}

// --- conditions, labels, assignments, implementation ---

func (r *renderer) conditions() (sentence, error) {
	cs := r.entries(syntax.Conditions)
	if len(cs) == 0 {
		return list(r.lang, r.say("This code does not use any conditions", "Dieses Programm benutzt keine Bedingungen"), nil), nil
	}
	items := make([]string, len(cs))
	for i, c := range cs {
		a, err := r.ref(r.lang, c.Ops[0], nominative)
		if err != nil {
			return sentence{}, err
		}
		b, err := r.ref(r.lang, c.Ops[1], dative)
		if err != nil {
			return sentence{}, err
		}
		cmp := r.say("is equal to", "ist gleich")
		if c.Flags != syntax.CompareEqual {
			cmp = r.say("is less than", "ist kleiner als")
		}
		items[i] = r.say("the condition that ", "die Bedingung dass ") + a + " " + cmp + " " + b
	}
	return list(r.lang, r.say("This code uses", "Dieses Programm benutzt"), items), nil
}

func (r *renderer) labels() (sentence, error) {
	n := r.p.LabelsCount
	switch {
	case n == 0:
		return list(r.lang, r.say("This code does not use any labels", "Dieses Programm benutzt keine Sprungziele"), nil), nil
	case n < 0:
		return sentence{}, fail("negative label count %d", n)
	case n >= 1000000000:
		return sentence{}, fail("label count %d cannot be written", n)
	}
	w, _ := cardinal(r.lang, int32(n))
	var item string
	switch {
	case n == 1:
		item = w + r.say(" label", " Sprungziel")
	default:
		item = w + r.say(" labels", " Sprungziele")
	}
	return list(r.lang, r.say("This code uses", "Dieses Programm benutzt"), []string{item}), nil
}

func (r *renderer) assigns() (sentence, error) {
	as := r.entries(syntax.Assigns)
	if len(as) == 0 {
		return list(r.lang, r.say("This code does not use any assignments", "Dieses Programm benutzt keine Zuweisungen"), nil), nil
	}
	items := make([]string, len(as))
	for i, a := range as {
		if a.Ops[1].Type&0xFF != syntax.Number {
			return sentence{}, fail("assignment %d does not store into a cell", i+1)
		}
		from, err := r.ref(r.lang, a.Ops[0], accusative)
		if err != nil {
			return sentence{}, err
		}
		to, err := r.ref(r.lang, a.Ops[1], accusative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = from + r.say(" to ", " an ") + to
	}
	return list(r.lang, r.say("This code assigns", "Dieses Programm weisst zu"), items), nil
}

func (r *renderer) implementation() (sentence, error) {
	ss := r.entries(syntax.Statements)
	if len(ss) == 0 {
		return sentence{}, fail("a program needs at least one statement")
	}
	items := make([]string, len(ss))
	for i, s := range ss {
		w, err := r.ref(r.lang, s.Ops[0], accusative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	return list(r.lang, r.say("This code implements", "Dieses Programm implementiert"), items), nil
}

func (r *renderer) cool() (sentence, error) {
	return sentence{head: r.say("Cool", "Hervorragend")}, nil
}
