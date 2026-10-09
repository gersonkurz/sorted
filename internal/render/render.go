// Package render writes a program's tables as Sorted! source: the inverse of
// the parser, in English, German, Italian or French, laid out to be sung.
//
// What it writes is exactly what the parser accepts. Some things have no
// form in one language, or none at all:
//
//   - German has no ratios and no lists of ordered differences, and the
//     original's German has no logical operations (Very Sorted!'s has), so
//     those sentences fall back to English. Sorted! allows mixing languages
//     per sentence.
//   - Output formats are written the only way the parser accepts them. Its
//     alternatives share an unrestored cursor, so each failed alternative eats
//     a word: "as a english english ordinal", "as a german german ordinal",
//     "als ein ein englischer Kardinal", "als ein ein ein englische
//     Ordinalzahl", "als ein ein ein ein deutscher Kardinal".
//   - More than one output or input, negative numbers and numbers from
//     1000000000 on cannot be written at all, nor, outside Very Sorted!,
//     references to inputs and logical operations, or a NAND; Render
//     reports an *Error.
//   - A Very Sorted! program (syntax.Program.Very) ends with "This code is
//     very cool." / "Dieses Programm ist ganz hervorragend.", its
//     statements may be inputs ("the first input", "die erste Eingabe"),
//     it may refer to logical operations and have NANDs ("of not both a and
//     b", "von nicht beiden, a und b"), and its German is written in UTF-8
//     ("fünf", "Verhältnisse").
//   - Italian (#30) is Very Very Sorted! ("Questo programma è molto molto
//     figo."), and has a form for everything. A program written in Italian
//     is Very Very Sorted!: an older program becomes one, unless that would
//     change what it does (veryCompatible). Very Very Sorted! also writes
//     Italian numbers ("as an italian cardinal", "come ordinale italiano").
//   - Vaudois French (#29) is Very Very Sorted! too ("Ce programme est très
//     très chouette."), as complete as Italian, with Vaud's numbers
//     ("septante-et-un", "comme cardinal vaudois") and its fillers, which
//     the renderer sprinkles in by a fixed rule (fillers).
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
	Italian
	French
)

// Verys is the dialect a program written in l needs, counted in verys:
// Italian (#30) and French (#29) are Very Very Sorted!.
func (l Lang) Verys() int {
	if l == Italian || l == French {
		return 2
	}
	return 0
}

// inDialect returns p as it is written in lang: p itself, or, for a language
// of a newer dialect than p's, a copy of p in that dialect. A program of the
// original's Sorted! may do something else as a Very one (veryCompatible);
// such a program is refused.
func inDialect(p *syntax.Program, lang Lang) (*syntax.Program, error) {
	if p.Verys >= lang.Verys() {
		return p, nil
	}
	if p.Verys == 0 {
		if err := veryCompatible(p); err != nil {
			return nil, err
		}
	}
	q := *p
	q.Verys = lang.Verys()
	return &q, nil
}

// veryCompatible reports why a program of the original's Sorted! might do
// something else as Very Sorted!, where German numbers print in UTF-8 and
// stores and reads go elsewhere in some cases (see interp): it writes German
// numbers, or a statement refers past its table to an assignment or a jump
// (which can then run any slot as a statement).
func veryCompatible(p *syntax.Program) error {
	for _, s := range p.Code {
		if s.Flags == syntax.FormatGermanCardinal || s.Flags == syntax.FormatGermanOrdinal {
			return fail("its German numbers would print in UTF-8 in Very Very Sorted!")
		}
	}
	for _, s := range p.Entries(syntax.Statements) {
		op := s.Ops[0]
		if op.Type == syntax.Read ||
			op.Type == syntax.Assign && int(op.Index) >= p.Tables[syntax.Assigns].Count ||
			op.Type == syntax.Jump && int(op.Index) >= p.Tables[syntax.Jumps].Count {
			return fail("a statement past its table could mean something else in Very Very Sorted!")
		}
	}
	return nil
}

// Error reports a program that cannot be written as Sorted! source.
type Error struct {
	What string
}

func (e *Error) Error() string { return "cannot render: " + e.What }

func fail(format string, args ...any) error { return &Error{fmt.Sprintf(format, args...)} }

// Render writes p as Sorted! source in lang.
func Render(p *syntax.Program, lang Lang) (string, error) {
	p, err := inDialect(p, lang)
	if err != nil {
		return "", err
	}
	text, q, err := write(p, lang)
	if err != nil {
		return "", err
	}
	if !Equal(p, q) {
		return "", fail("the program's table layout cannot be reproduced in Sorted! text")
	}
	return text, nil
}

// Compose writes p as Sorted! text like Render, for programs built by the
// compiler. Those never refer past the end of a table, so their table layout
// does not matter and need not be reproducible; Compose checks that the text
// parses into the same entries (SameEntries) and returns the program as the
// parser sees it, with the layout any Sorted! interpreter will use.
func Compose(p *syntax.Program, lang Lang) (string, *syntax.Program, error) {
	p, err := inDialect(p, lang)
	if err != nil {
		return "", nil, err
	}
	text, q, err := write(p, lang)
	if err != nil {
		return "", nil, err
	}
	if !SameEntries(p, q) {
		return "", nil, fail("the generated text parses into different entries")
	}
	return text, q, nil
}

// write renders the sentences and parses the result back.
func write(p *syntax.Program, lang Lang) (string, *syntax.Program, error) {
	r := &renderer{p: p, lang: lang}
	sentences := []func() (sentence, error){
		r.numbers, r.jumps, r.outputs, r.inputs, r.sums, r.conditions, r.labels,
		r.diffs, r.assigns, r.prods, r.implementation, r.ratios, r.nands, r.cool,
	}
	var b strings.Builder
	f := fillers{}
	for i, next := range sentences {
		s, err := next()
		if err != nil {
			return "", nil, err
		}
		if lang == French && i%3 == 2 {
			s.head = f.after(s.head)
		}
		b.WriteString(s.text())
		b.WriteString("\n")
	}
	text := b.String()
	if p.Verys > 0 {
		text = verySpelling(text)
	}
	q, err := syntax.Parse([]byte(text))
	if err != nil {
		return "", nil, fail("the generated text does not parse (%v)", err)
	}
	return text, q, nil
}

// fillers sprinkle Vaudois French with what it loves to say (#29, Gerson's
// rule): every third sentence has one after "Ce programme", "eh", "hein",
// "quoi" and "voilà" in turn ("Ce programme, eh, écrit ..."), and the
// implementation ends ", voilà". The parser reads past them.
type fillers struct{ n int }

func (f *fillers) after(head string) string {
	rest, ok := strings.CutPrefix(head, "Ce programme")
	if !ok {
		return head
	}
	w := []string{"eh", "hein", "quoi", "voilà"}[f.n%4]
	f.n++
	return "Ce programme, " + w + "," + rest
}

// verySpelling writes German as Very Sorted! reads it, in UTF-8 (#28): the
// number words with umlauts and ß ("fünf", "zwölf", "dreißig"),
// "Verhältnis" and "Verknüpfung". "weisst" is the original's word and stays.
func verySpelling(text string) string {
	return numbers.VerySpelling(strings.NewReplacer("Verhaeltnis", "Verhältnis", "Verknuepfung", "Verknüpfung").Replace(text))
}

// SameEntries reports whether two programs declare the same numbers and
// labels and have the same entries in every table, wherever in Code those
// tables lie.
func SameEntries(p, q *syntax.Program) bool {
	if p.Verys != q.Verys || p.LabelsCount != q.LabelsCount || fmt.Sprint(p.Data) != fmt.Sprint(q.Data) {
		return false
	}
	for c := syntax.Sums; c < syntax.NumCategories; c++ {
		if fmt.Sprint(p.Entries(c)) != fmt.Sprint(q.Entries(c)) {
			return false
		}
	}
	return true
}

// Equal reports whether two programs are the same in every respect the
// interpreter can observe: declared numbers, label count, element count,
// table positions, and every slot of the static Code array (zero past what
// was written), since references past a table read neighbouring slots.
func Equal(p, q *syntax.Program) bool {
	if p.Verys != q.Verys || p.LabelsCount != q.LabelsCount || p.TypeCount != q.TypeCount || p.Tables != q.Tables ||
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
// item follows ", and"/", und", or in Italian "e" and French "et", then a
// period. Italian and French keep the comma before the conjunction only
// where the items are pairs that contain one themselves (comma). A French
// sentence may end with a filler (tail).
type sentence struct {
	head  string
	items []string
	conj  string
	comma bool
	tail  string
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
		return s.head + s.tail + "."
	case 1:
		line = s.head + " " + s.items[0] + s.tail + "."
	default:
		sep := " "
		if s.comma {
			sep = ", "
		}
		line = s.head + " " + strings.Join(s.items[:n-1], ", ") + sep + s.conj + " " + s.items[n-1] + s.tail + "."
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
			b.WriteString(item + s.tail + ".")
		case i == n-2 && !s.comma:
			b.WriteString(item)
		case i < n-1:
			b.WriteString(item + ",")
		default:
			b.WriteString(s.conj + " " + item + s.tail + ".")
		}
	}
	return b.String()
}

type renderer struct {
	p    *syntax.Program
	lang Lang
}

// say picks the English, German, Italian or French text.
func (r *renderer) say(en, de, it, fr string) string {
	switch r.lang {
	case German:
		return de
	case Italian:
		return it
	case French:
		return fr
	}
	return en
}

// list makes a sentence in language l.
func list(l Lang, head string, items []string) sentence {
	switch l {
	case German:
		return sentence{head: head, items: items, conj: "und", comma: true}
	case Italian:
		return sentence{head: head, items: items, conj: "e"}
	case French:
		return sentence{head: head, items: items, conj: "et"}
	}
	return sentence{head: head, items: items, conj: "and", comma: true}
}

func (r *renderer) entries(c syntax.Category) []syntax.Slide { return r.p.Entries(c) }

// --- numbers ---

func (r *renderer) numbers() (sentence, error) {
	data := r.p.Data
	if len(data) == 0 {
		return list(r.lang, r.say("This code does not use any numbers", "Dieses Programm benutzt keine Zahlen", "Questo programma non usa numeri", "Ce programme n'utilise aucun nombre"), nil), nil
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
		return list(r.lang, r.say("This code uses the number", "Dieses Programm benutzt die Zahl", "Questo programma usa il numero", "Ce programme utilise le nombre"), items), nil
	}
	return list(r.lang, r.say("This code uses the numbers", "Dieses Programm benutzt die Zahlen", "Questo programma usa i numeri", "Ce programme utilise les nombres"), items), nil
}

// cardinal writes a declarable number: zero, or 1 to 999999999 (Italian
// and French could say more, but the others cannot, and a program reads the same in
// every language).
func cardinal(l Lang, n int32) (string, error) {
	switch {
	case n == 0 && l == German:
		return "null", nil
	case n == 0 && l == French:
		return "zéro", nil
	case n == 0:
		return "zero", nil
	case n < 0:
		return "", fail("negative numbers cannot be declared (%d)", n)
	case n >= 1000000000:
		return "", fail("numbers from 1000000000 on cannot be declared (%d)", n)
	case l == German:
		return numbers.GermanCardinal(n)
	case l == Italian:
		return numbers.ItalianCardinal(n), nil
	case l == French:
		return numbers.VaudoisCardinal(n), nil
	}
	return numbers.EnglishCardinal(n)
}

// --- references ---

// grammatical case of a German reference, and the preposition an Italian
// or French one fuses with its article
type gcase int

const (
	nominative gcase = iota
	accusative
	dative
	itDi // "del primo numero"
	itA  // "al primo numero"
	itDa // "dal primo numero"
	frA  // "au premier nombre"
	frDe // "du premier nombre"
)

type noun struct {
	en, de string
	gender byte // 'f', 'm' or 'n'
	it     string
	itFem  bool
	fr     string
	frFem  bool
}

var nouns = map[syntax.OperandType]noun{
	syntax.Number:    {"number", "Zahl", 'f', "numero", false, "nombre", false},
	syntax.Sum:       {"sum", "Summe", 'f', "somma", true, "somme", true},
	syntax.Diff:      {"ordered difference", "geordnete Differenz", 'f', "differenza ordinata", true, "différence ordonnée", true},
	syntax.Prod:      {"product", "Produkt", 'n', "prodotto", false, "produit", false},
	syntax.Ratio:     {"ratio", "Verhaeltnis", 'n', "rapporto", false, "rapport", false},
	syntax.Assign:    {"assignment", "Zuweisung", 'f', "assegnamento", false, "affectation", true},
	syntax.Jump:      {"jump", "Sprungbefehl", 'm', "salto", false, "saut", false},
	syntax.Label:     {"label", "Sprungziel", 'n', "etichetta", true, "étiquette", true},
	syntax.Condition: {"condition", "Bedingung", 'f', "condizione", true, "condition", true},
	syntax.Write:     {"output", "Ausgabe", 'f', "uscita", true, "sortie", true},
	syntax.Read:      {"input", "Eingabe", 'f', "ingresso", false, "entrée", true},                                              // Very Sorted! only (the parser decides)
	syntax.Nand:      {"logical operation", "logische Verknuepfung", 'f', "operazione logica", true, "opération logique", true}, // likewise
}

// ref writes a reference such as "the third number", "der dritten Zahl" or
// "della terza somma".
func (r *renderer) ref(l Lang, op syntax.Operand, c gcase) (string, error) {
	switch l {
	case Italian:
		return r.itRef(op, c)
	case French:
		return r.frRef(op, c)
	}
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
	// reads it back; some endings it does not ("zwanzigsten").
	inflect := c == dative && nn.gender == 'f' || c != nominative && nn.gender == 'm'
	if inflect && ordinalParses(ord+"n", n, nn.de) {
		ord += "n"
	} else if !ordinalParses(ord, n, nn.de) {
		return "", fail("no German ordinal for %d reads back", n)
	}
	de := nn.de
	if inflect && op.Type == syntax.Nand {
		de = "logischen Verknuepfung" // "der ersten logischen Verknüpfung"
	}
	return article + " " + ord + " " + de, nil
}

// itRef writes an Italian reference: the article, fused with the preposition
// c asks for and elided before a vowel ("dell'ottava cella"), the ordinal,
// which agrees with the noun, and the noun.
func (r *renderer) itRef(op syntax.Operand, c gcase) (string, error) {
	if op.Type&syntax.Indirect != 0 {
		inner, err := r.itRef(syntax.Operand{Type: op.Type &^ syntax.Indirect, Index: op.Index}, itDa)
		if err != nil {
			return "", err
		}
		return itArticle(c, true, "cella") + "cella indicizzata " + inner, nil
	}
	nn, ok := nouns[op.Type]
	if !ok {
		return "", fail("there is no way to refer to an operand of type %d", op.Type)
	}
	n := op.Index + 1
	if n < 1 {
		return "", fail("reference index %d out of range", op.Index)
	}
	ord := numbers.ItalianOrdinal(n, nn.itFem)
	return itArticle(c, nn.itFem, ord) + ord + " " + nn.it, nil
}

// itArticle is the definite article before word, fused with the preposition
// of c, with the space that follows it, or none after an apostrophe.
func itArticle(c gcase, fem bool, word string) string {
	forms := map[gcase][3]string{
		itDi: {"del ", "della ", "dell'"},
		itA:  {"al ", "alla ", "all'"},
		itDa: {"dal ", "dalla ", "dall'"},
	}[c]
	if forms[0] == "" {
		forms = [3]string{"il ", "la ", "l'"}
	}
	switch {
	case strings.ContainsRune("aeiou", rune(word[0])):
		return forms[2]
	case fem:
		return forms[1]
	}
	return forms[0]
}

// frRef writes a French reference: the article, fused with the preposition
// c asks for ("au", "de la"), the ordinal, which agrees with the noun only
// as "premier"/"première" (no ordinal elides its article: "le onzième",
// "le huitième"), and the noun.
func (r *renderer) frRef(op syntax.Operand, c gcase) (string, error) {
	if op.Type&syntax.Indirect != 0 {
		inner, err := r.frRef(syntax.Operand{Type: op.Type &^ syntax.Indirect, Index: op.Index}, nominative)
		if err != nil {
			return "", err
		}
		return frArticle(c, true) + "cellule indexée par " + inner, nil
	}
	nn, ok := nouns[op.Type]
	if !ok {
		return "", fail("there is no way to refer to an operand of type %d", op.Type)
	}
	n := op.Index + 1
	if n < 1 {
		return "", fail("reference index %d out of range", op.Index)
	}
	return frArticle(c, nn.frFem) + numbers.VaudoisOrdinal(n, nn.frFem) + " " + nn.fr, nil
}

// frArticle is the definite article, fused with the preposition of c, with
// the space that follows it.
func frArticle(c gcase, fem bool) string {
	forms := map[gcase][2]string{frA: {"au ", "à la "}, frDe: {"du ", "de la "}}[c]
	if forms[0] == "" {
		forms = [2]string{"le ", "la "}
	}
	if fem {
		return forms[1]
	}
	return forms[0]
}

// frFeminine reports whether a French reference is feminine (an indexed
// cell is "la cellule").
func frFeminine(op syntax.Operand) bool {
	return op.Type&syntax.Indirect != 0 || nouns[op.Type].frFem
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
		return list(r.lang, r.say("This code does never go anywhere", "Dieses Programm geht nirgendwo hin", "Questo programma non va mai da nessuna parte", "Y'a pas le feu au lac"), nil), nil
	}
	items := make([]string, len(js))
	for i, j := range js {
		if j.Ops[0].Type != syntax.Label {
			return sentence{}, fail("jump %d does not go to a label", i+1)
		}
		label, err := r.ref(r.lang, j.Ops[0], r.caseOf(accusative, itA, frA))
		if err != nil {
			return sentence{}, err
		}
		if j.Flags == syntax.UnconditionalJump {
			items[i] = r.say("always goes to ", "springt immer an ", "va sempre ", "va toujours ") + label
			continue
		}
		if j.Ops[1].Type != syntax.Condition {
			return sentence{}, fail("jump %d does not depend on a condition", i+1)
		}
		cond, err := r.ref(r.lang, j.Ops[1], nominative)
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.say("sometimes goes to "+label+" if "+cond+" is true", "springt manchmal an "+label+" wenn "+cond+" wahr ist", "va talvolta "+label+" se "+cond+" è vera", "va parfois "+label+" si "+cond+" est vraie")
	}
	return list(r.lang, r.say("This code", "Dieses Programm", "Questo programma", "Ce programme"), items), nil
}

// caseOf picks the German case or the Italian or French preposition.
func (r *renderer) caseOf(de, it, fr gcase) gcase {
	switch r.lang {
	case Italian:
		return it
	case French:
		return fr
	}
	return de
}

// --- output and input ---

func (r *renderer) outputs() (sentence, error) {
	ws := r.entries(syntax.Writes)
	switch len(ws) {
	case 0:
		return list(r.lang, r.say("This code cannot write", "Dieses Programm kann nicht schreiben", "Questo programma non può scrivere", "Ce programme ne peut pas écrire"), nil), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one output")
	}
	w := ws[0]
	l := r.lang
	formats := map[int32][4]string{
		syntax.FormatCharacter:       {"as a character", "als ein Zeichen", "come carattere", "comme caractère"},
		syntax.FormatEnglishCardinal: {"as a english cardinal", "als ein ein englischer Kardinal", "come cardinale inglese", "comme cardinal anglais"},
		syntax.FormatEnglishOrdinal:  {"as a english english ordinal", "als ein ein ein englische Ordinalzahl", "come ordinale inglese", "comme ordinal anglais"},
		syntax.FormatGermanCardinal:  {"as a german cardinal", "als ein ein ein ein deutscher Kardinal", "come cardinale tedesco", "comme cardinal allemand"},
		syntax.FormatGermanOrdinal:   {"as a german german ordinal", "als eine deutsche Ordinalzahl", "come ordinale tedesco", "comme ordinal allemand"},
		syntax.FormatItalianCardinal: {"as an italian cardinal", "als ein italienischer Kardinal", "come cardinale italiano", "comme cardinal italien"},
		syntax.FormatItalianOrdinal:  {"as an italian ordinal", "als eine italienische Ordinalzahl", "come ordinale italiano", "comme ordinal italien"},
		syntax.FormatVaudoisCardinal: {"as a vaudois cardinal", "als ein waadtländischer Kardinal", "come cardinale vodese", "comme cardinal vaudois"},
		syntax.FormatVaudoisOrdinal:  {"as a vaudois ordinal", "als eine waadtländische Ordinalzahl", "come ordinale vodese", "comme ordinal vaudois"},
	}
	phrases, ok := formats[w.Flags]
	if !ok || r.p.Verys < 2 && w.Flags >= syntax.FormatItalianCardinal {
		return sentence{}, fail("output format %d does not exist", w.Flags)
	}
	what, err := r.ref(l, w.Ops[0], accusative)
	if err != nil {
		return sentence{}, err
	}
	return list(l, r.say("This code writes", "Dieses Programm schreibt", "Questo programma scrive", "Ce programme écrit"), []string{what + " " + phrases[l]}), nil
}

func (r *renderer) inputs() (sentence, error) {
	rs := r.entries(syntax.Reads)
	switch len(rs) {
	case 0:
		return list(r.lang, r.say("This code cannot read", "Dieses Programm kann nicht lesen", "Questo programma non può leggere", "Ce programme ne peut pas lire"), nil), nil
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
	return list(r.lang, r.say("This code reads", "Dieses Programm liest", "Questo programma legge", "Ce programme lit"), []string{what + r.say(" as a character", " als ein Zeichen", " come carattere", " comme caractère")}), nil
}

// --- expressions ---

// binary writes the sentence of an expression table. single and plural are
// the heads for one entry and for a list; prep starts each entry ("of",
// "between"), word joins its operands ("and", "to") in case c. The entries
// are pairs, so an Italian or French list keeps its comma before "e"/"et".
func (r *renderer) binary(l Lang, c syntax.Category, none, single, plural, prep, word string, gc gcase) (sentence, error) {
	es := r.entries(c)
	if len(es) == 0 {
		return list(l, none, nil), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		s, err := r.pair(l, e, word, gc)
		if err != nil {
			return sentence{}, err
		}
		items[i] = strings.TrimPrefix(prep+" "+s, " ")
	}
	s := list(l, plural, items)
	if len(items) == 1 {
		s.head = single
	}
	s.comma = true
	return s, nil
}

func (r *renderer) sums() (sentence, error) {
	switch r.lang {
	case German:
		return r.binary(German, syntax.Sums, "Dieses Programm benutzt keine Summen", "Dieses Programm benutzt die Summe", "Dieses Programm benutzt die Summen", "aus", "und", dative)
	case Italian:
		return r.binary(Italian, syntax.Sums, "Questo programma non usa somme", "Questo programma usa la somma", "Questo programma usa le somme", "", "e", itDi)
	case French:
		return r.binary(French, syntax.Sums, "Ce programme n'utilise aucune somme", "Ce programme utilise la somme", "Ce programme utilise les sommes", "", "et", frDe)
	}
	return r.binary(English, syntax.Sums, "This code does not use any sums", "This code uses the sum", "This code uses the sums", "of", "and", dative)
}

func (r *renderer) diffs() (sentence, error) {
	switch {
	case r.lang == German && len(r.entries(syntax.Diffs)) < 2:
		return r.binary(German, syntax.Diffs, "Dieses Programm benutzt keine geordneten Differenzen", "Dieses Programm benutzt die geordnete Differenz", "", "zwischen", "und", dative)
	case r.lang == Italian:
		return r.binary(Italian, syntax.Diffs, "Questo programma non usa differenze ordinate", "Questo programma usa la differenza ordinata", "Questo programma usa le differenze ordinate", "tra", "e", nominative)
	case r.lang == French:
		return r.binary(French, syntax.Diffs, "Ce programme n'utilise aucune différence ordonnée", "Ce programme utilise la différence ordonnée", "Ce programme utilise les différences ordonnées", "entre", "et", nominative)
	}
	return r.binary(English, syntax.Diffs, "This code does not use any ordered differences", "This code uses the ordered difference", "This code uses the ordered differences", "between", "and", dative)
}

func (r *renderer) prods() (sentence, error) {
	switch r.lang {
	case German:
		return r.binary(German, syntax.Prods, "Dieses Programm benutzt keine Produkte", "Dieses Programm benutzt das Produkt", "Dieses Programm benutzt die Produkte", "von", "und", dative)
	case Italian:
		return r.binary(Italian, syntax.Prods, "Questo programma non usa prodotti", "Questo programma usa il prodotto", "Questo programma usa i prodotti", "", "e", itDi)
	case French:
		return r.binary(French, syntax.Prods, "Ce programme n'utilise aucun produit", "Ce programme utilise le produit", "Ce programme utilise les produits", "", "et", frDe)
	}
	return r.binary(English, syntax.Prods, "This code does not use any products", "This code uses the product", "This code uses the products", "of", "and", dative)
}

func (r *renderer) ratios() (sentence, error) {
	switch {
	case r.lang == German && len(r.entries(syntax.Ratios)) == 0:
		return list(German, "Dieses Programm benutzt keine Verhaeltnisse", nil), nil
	case r.lang == Italian:
		return r.binary(Italian, syntax.Ratios, "Questo programma non usa rapporti", "Questo programma usa il rapporto", "Questo programma usa i rapporti", "tra", "e", nominative)
	case r.lang == French:
		return r.frRatios()
	}
	return r.binary(English, syntax.Ratios, "This code does not use any ratios", "This code uses the ratio", "This code uses the ratios", "of", "to", dative)
}

// nands writes the logical operations: the original's NOR ("of not a and not
// b"), and in Very Sorted! the NAND ("of not both a and b") and German for
// both (#26), and in Very Very Sorted! Italian ("né a né b", "non entrambi
// a e b").
func (r *renderer) nands() (sentence, error) {
	es := r.entries(syntax.Nands)
	if len(es) == 0 {
		return list(r.lang, r.say("This code does not use any logical operations", "Dieses Programm ist unlogisch", "Questo programma è illogico", "Ce programme est illogique"), nil), nil
	}
	l := English
	if r.p.Verys > 0 {
		l = r.lang
	}
	items := make([]string, len(es))
	for i, e := range es {
		if e.Flags != syntax.LogicalNor && (e.Flags != syntax.LogicalNand || r.p.Verys == 0) {
			return sentence{}, fail("logical operation %d has flags %d, which only Very Sorted! NAND (1) or NOR (0) has", i, e.Flags)
		}
		c := nominative
		if l == German {
			c = dative
		}
		a, err := r.ref(l, e.Ops[0], c)
		if err != nil {
			return sentence{}, err
		}
		b, err := r.ref(l, e.Ops[1], c)
		if err != nil {
			return sentence{}, err
		}
		nand := e.Flags == syntax.LogicalNand
		switch {
		case l == German && nand:
			items[i] = "von nicht beiden, " + a + " und " + b
		case l == German:
			items[i] = "von nicht " + a + " und nicht " + b
		case l == Italian && nand:
			items[i] = "non entrambi " + a + " e " + b
		case l == Italian:
			items[i] = "né " + a + " né " + b
		case l == French && nand:
			items[i] = "pas à la fois " + a + " et " + b
		case l == French:
			items[i] = "ni " + a + " ni " + b
		case nand:
			items[i] = "of not both " + a + " and " + b
		default:
			items[i] = "of not " + a + " and not " + b
		}
	}
	heads := map[Lang][2]string{
		English: {"This code uses the logical operation", "This code uses the logical operations"},
		German:  {"Dieses Programm benutzt die logische Verknuepfung", "Dieses Programm benutzt die logischen Verknuepfungen"},
		Italian: {"Questo programma usa l'operazione logica", "Questo programma usa le operazioni logiche"},
		French:  {"Ce programme utilise l'opération logique", "Ce programme utilise les opérations logiques"},
	}[l]
	s := list(l, heads[1], items)
	if len(items) == 1 {
		s.head = heads[0]
	}
	s.comma = true
	return s, nil
}

// frRatios writes French ratios, "du a au b": the prepositions fuse with
// both articles.
func (r *renderer) frRatios() (sentence, error) {
	es := r.entries(syntax.Ratios)
	if len(es) == 0 {
		return list(French, "Ce programme n'utilise aucun rapport", nil), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		a, err := r.ref(French, e.Ops[0], frDe)
		if err != nil {
			return sentence{}, err
		}
		b, err := r.ref(French, e.Ops[1], frA)
		if err != nil {
			return sentence{}, err
		}
		items[i] = a + " " + b
	}
	s := list(French, "Ce programme utilise les rapports", items)
	if len(items) == 1 {
		s.head = "Ce programme utilise le rapport"
	}
	return s, nil
}

// --- conditions, labels, assignments, implementation ---

func (r *renderer) conditions() (sentence, error) {
	cs := r.entries(syntax.Conditions)
	if len(cs) == 0 {
		return list(r.lang, r.say("This code does not use any conditions", "Dieses Programm benutzt keine Bedingungen", "Questo programma non usa condizioni", "Ce programme n'utilise aucune condition"), nil), nil
	}
	items := make([]string, len(cs))
	for i, c := range cs {
		a, err := r.ref(r.lang, c.Ops[0], nominative)
		if err != nil {
			return sentence{}, err
		}
		// "gleich" takes the dative, "kleiner als" the nominative; "uguale
		// a", "minore di".
		// French agrees with the subject: "égal", "égale".
		e := ""
		if frFeminine(c.Ops[0]) {
			e = "e"
		}
		cmp, bc := r.say("is equal to", "ist gleich", "sia uguale", "soit égal"+e), r.caseOf(dative, itA, frA)
		if c.Flags != syntax.CompareEqual {
			cmp, bc = r.say("is less than", "ist kleiner als", "sia minore", "soit inférieur"+e), r.caseOf(nominative, itDi, frA)
		}
		b, err := r.ref(r.lang, c.Ops[1], bc)
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.say("the condition that ", "die Bedingung dass ", "la condizione che ", "la condition que ") + a + " " + cmp + " " + b
	}
	return list(r.lang, r.say("This code uses", "Dieses Programm benutzt", "Questo programma usa", "Ce programme utilise"), items), nil
}

func (r *renderer) labels() (sentence, error) {
	n := r.p.LabelsCount
	switch {
	case n == 0:
		return list(r.lang, r.say("This code does not use any labels", "Dieses Programm benutzt keine Sprungziele", "Questo programma non usa etichette", "Ce programme n'utilise aucune étiquette"), nil), nil
	case n < 0:
		return sentence{}, fail("negative label count %d", n)
	case n >= 1000000000:
		return sentence{}, fail("label count %d cannot be written", n)
	}
	w, _ := cardinal(r.lang, int32(n))
	var item string
	switch {
	case n == 1:
		item = r.say(w+" label", w+" Sprungziel", "un'etichetta", "une étiquette")
	case r.lang == French && strings.HasSuffix(w, "-un"):
		item = w + "e étiquettes" // "vingt-et-une"
	default:
		item = w + r.say(" labels", " Sprungziele", " etichette", " étiquettes")
	}
	return list(r.lang, r.say("This code uses", "Dieses Programm benutzt", "Questo programma usa", "Ce programme utilise"), []string{item}), nil
}

func (r *renderer) assigns() (sentence, error) {
	as := r.entries(syntax.Assigns)
	if len(as) == 0 {
		return list(r.lang, r.say("This code does not use any assignments", "Dieses Programm benutzt keine Zuweisungen", "Questo programma non usa assegnamenti", "Ce programme n'utilise aucune affectation"), nil), nil
	}
	items := make([]string, len(as))
	for i, a := range as {
		// Very Sorted! stores into the cell any value indexes (#27)
		if t := a.Ops[1].Type; t&0xFF != syntax.Number && !(r.p.Verys > 0 && t&syntax.Indirect != 0) {
			return sentence{}, fail("assignment %d does not store into a cell", i+1)
		}
		from, err := r.ref(r.lang, a.Ops[0], accusative)
		if err != nil {
			return sentence{}, err
		}
		to, err := r.ref(r.lang, a.Ops[1], r.caseOf(accusative, itA, frA))
		if err != nil {
			return sentence{}, err
		}
		items[i] = from + r.say(" to ", " an ", " ", " ") + to
	}
	return list(r.lang, r.say("This code assigns", "Dieses Programm weisst zu", "Questo programma assegna", "Ce programme affecte"), items), nil
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
	s := list(r.lang, r.say("This code implements", "Dieses Programm implementiert", "Questo programma implementa", "Ce programme implémente"), items)
	if r.lang == French {
		s.tail = ", voilà"
	}
	return s, nil
}

// cool writes the last sentence, which also names the dialect: the code is
// as very cool as its dialect has verys ("This code is very very cool.",
// "Questo programma è molto molto figo.", "Ce programme est très très
// chouette."); the original's is just "Cool.".
func (r *renderer) cool() (sentence, error) {
	n := r.p.Verys
	if n == 0 {
		return sentence{head: r.say("Cool", "Hervorragend", "", "")}, nil // Italian and French are never the original's
	}
	return sentence{head: r.say("This code is "+strings.Repeat("very ", n)+"cool", "Dieses Programm ist "+strings.Repeat("ganz ", n)+"hervorragend",
		"Questo programma è "+strings.Repeat("molto ", n)+"figo", "Ce programme est "+strings.Repeat("très ", n)+"chouette")}, nil
}
