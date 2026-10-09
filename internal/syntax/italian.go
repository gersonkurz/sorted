package syntax

import (
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
)

// Italian (#30), spoken from Very Very Sorted! on: every sentence has an
// Italian form, mixed per sentence with English and German like those two
// with each other. Its wording is Gerson's (decided on #30):
//
//	Questo programma usa i numeri zero, uno e due.
//	Questo programma va sempre alla prima etichetta e va talvolta alla
//	    seconda etichetta se la prima condizione è vera.
//	Questo programma scrive il primo numero come carattere.
//	Questo programma legge la cella indicizzata dal secondo numero come carattere.
//	Questo programma usa la somma del primo numero e del secondo numero.
//	Questo programma usa la condizione che il primo numero sia uguale al terzo numero.
//	Questo programma usa un'etichetta.
//	Questo programma usa la differenza ordinata tra il primo numero e il secondo numero.
//	Questo programma assegna la prima somma al primo numero.
//	Questo programma usa il prodotto del primo numero e del secondo numero.
//	Questo programma implementa il primo assegnamento e la prima uscita.
//	Questo programma usa il rapporto tra il primo numero e il secondo numero.
//	Questo programma usa l'operazione logica né il primo numero né il secondo numero.
//	Questo programma è molto molto figo.
//
// and "non usa numeri" (somme, ...) for none, "non va mai da nessuna parte",
// "non può scrivere", "non può leggere", "è illogico". Italian has no
// quirks: every form parses as written. The parser is lenient where the
// renderer is exact: any article, fused with a preposition or not, may
// stand before a reference ("del", "all'", "dalla", ...), a list may have a
// comma before its final "e" or not, and a condition may say "sia" or "è".
// The text arrives without accents (FilterVeryVery), so the keywords are
// spelled without them.

// itArticles are the articles that may start a reference, alone or fused
// with di, a or da ("l'" and its kin lose the apostrophe in the filter).
var itArticles = []string{"il", "lo", "la", "l", "del", "dello", "della", "dell", "al", "allo", "alla", "all", "dal", "dallo", "dalla", "dall"}

func (ps *parser) itArticle() bool {
	for _, a := range itArticles {
		if ps.kw(a) {
			return true
		}
	}
	return false
}

// itIdentifier is an Italian reference: "il terzo numero", "la cella
// indicizzata dalla prima somma".
func (ps *parser) itIdentifier(op *Operand) bool {
	return ps.itIndirect(op) || ps.itDirect(op)
}

func (ps *parser) itIndirect(op *Operand) bool {
	save := ps.p
	if ps.itArticle() && ps.seq("cella", "indicizzata") && ps.itDirect(op) {
		op.Type |= Indirect
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) itDirect(op *Operand) bool {
	save := ps.p
	if ps.itArticle() {
		if n, next := numbers.ParseItalianOrdinal(ps.s, ps.p); n > 0 {
			ps.p = next
			ps.skipWhitespaces()
			if t, ok := ps.itNoun(); ok {
				op.Type = t
				op.Index = n - 1
				return true
			}
		}
	}
	ps.p = save
	return false
}

// itNoun is what a reference refers to; uscita and ingresso are output and
// input.
func (ps *parser) itNoun() (OperandType, bool) {
	switch {
	case ps.kw("numero"):
		return Number, true
	case ps.kw("cella"):
		return Cell, true
	case ps.kw("somma"):
		return Sum, true
	case ps.seq("differenza", "ordinata"):
		return Diff, true
	case ps.kw("prodotto"):
		return Prod, true
	case ps.kw("rapporto"):
		return Ratio, true
	case ps.kw("assegnamento"):
		return Assign, true
	case ps.kw("salto"):
		return Jump, true
	case ps.kw("etichetta"):
		return Label, true
	case ps.kw("condizione"):
		return Condition, true
	case ps.kw("uscita"):
		return Write, true
	case ps.kw("ingresso"):
		return Read, true
	case ps.seq("operazione", "logica"):
		return Nand, true
	}
	return 0, false
}

// itCardinal parses an Italian cardinal ("ventitre", "un milione") that a
// program may declare: below 1000000000, as in English and German (whose
// words stop there), so a program reads the same in every language.
// Italian output still prints any number.
func (ps *parser) itCardinal() (int32, bool) {
	n, next, ok := numbers.ParseItalianCardinal(ps.s, ps.p)
	if ok = ok && n < 1000000000; ok {
		ps.p = next
		ps.skipWhitespaces()
	}
	return n, ok
}

// itList parses an Italian list: items separated by commas, the last after
// "e" (or "ed"), with or without a comma before it ("uno, due e tre",
// "..., e ..."). A single item is a list too.
func (ps *parser) itList(spec func() bool) bool { return ps.listOf(spec, "e", "ed") }

// listOf parses a list of Very Very Sorted!'s Italian or French: items
// separated by commas, the last after one of the conjunctions, with or
// without a comma before it. A single item is a list too.
func (ps *parser) listOf(spec func() bool, conjs ...string) bool {
	conj := func() bool {
		for _, c := range conjs {
			if ps.kw(c) {
				return true
			}
		}
		return false
	}
	save := ps.p
	if spec() {
		for {
			comma := ps.kw(",")
			if conj() {
				if spec() {
					return true
				}
				break
			}
			if !comma {
				return true
			}
			if !spec() {
				break
			}
		}
	}
	ps.p = save
	return false
}

// with parses the words, then spec, restoring the cursor unless both
// succeed.
func (ps *parser) with(spec func() bool, words string) bool {
	save := ps.p
	if ps.seq(strings.Fields(words)...) && spec() {
		return true
	}
	ps.p = save
	return false
}

// itHead parses "questo programma" and the words that follow, then spec.
func (ps *parser) itHead(words string, spec func() bool) bool {
	return ps.with(spec, "questo programma "+words)
}

// always is a spec that needs nothing more.
func always() bool { return true }

// itNone parses a sentence that says there is none.
func (ps *parser) itNone(words string) bool { return ps.itHead("non "+words, always) }

// itUses parses "questo programma usa" and one item after the singular
// ("la somma ...") or a list after the plural ("le somme ...").
func (ps *parser) itUses(spec func() bool, one, many string) bool {
	return ps.itHead("usa", func() bool {
		return ps.with(spec, one) || ps.with(func() bool { return ps.itList(spec) }, many)
	})
}

// itStatements parses "questo programma <verb>" and one statement-like item
// (single, which withdraws an entry not followed by the period, as the
// original's do) or a list.
func (ps *parser) itStatements(verb string, single, spec func() bool) bool {
	return ps.itHead(verb, func() bool { return single() || ps.itList(spec) })
}

// itPair parses "<a> e <b>" into a cell's operands.
func (ps *parser) itPair(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && (ps.kw("e") || ps.kw("ed")) && ps.identifier(&cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// itBetween parses "tra <a> e <b>" (or "fra").
func (ps *parser) itBetween(cell *Slide) bool {
	save := ps.p
	if (ps.kw("tra") || ps.kw("fra")) && ps.itPair(cell) {
		return true
	}
	ps.p = save
	return false
}

// itNand parses "né <a> né <b>" (NOR) and "non entrambi <a> e <b>" (NAND).
func (ps *parser) itNand(cell *Slide) bool {
	save := ps.p
	if ps.kw("ne") && ps.identifier(&cell.Ops[0]) && ps.kw("ne") && ps.identifier(&cell.Ops[1]) {
		cell.Flags = LogicalNor
		return true
	}
	ps.p = save
	if ps.seq("non", "entrambi") && ps.itPair(cell) {
		cell.Flags = LogicalNand
		return true
	}
	ps.p = save
	return false
}

// itComparison parses "sia uguale" and "sia minore" ("è" for "sia" too);
// the second operand's article says "a" or "di" ("al", "del").
func (ps *parser) itComparison(cell *Slide) bool {
	save := ps.p
	if ps.kw("sia") || ps.kw("e") {
		switch {
		case ps.kw("uguale"):
			cell.Flags = CompareEqual
			return true
		case ps.kw("minore"):
			cell.Flags = CompareLess
			return true
		}
	}
	ps.p = save
	return false
}

// itCondition parses "la condizione che <a> sia uguale <b>".
func (ps *parser) itCondition(cell *Slide) bool {
	save := ps.p
	if ps.seq("la", "condizione", "che") && ps.identifier(&cell.Ops[0]) && ps.itComparison(cell) && ps.identifier(&cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// itJump parses "va sempre <label>" and "va talvolta <label> se <condition>
// è vera".
func (ps *parser) itJump(cell *Slide) bool {
	save := ps.p
	if ps.seq("va", "sempre") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label {
		cell.Flags = UnconditionalJump
		return true
	}
	ps.p = save
	if ps.seq("va", "talvolta") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label &&
		ps.kw("se") && ps.identifier(&cell.Ops[1]) && cell.Ops[1].Type == Condition &&
		(ps.kw("sia") || ps.kw("e")) && ps.kw("vera") {
		cell.Flags = ConditionalJump
		return true
	}
	ps.p = save
	return false
}

// veryVeryFormat parses the output formats Very Very Sorted! adds: all of
// them in Italian ("come carattere", "come cardinale italiano") and French
// ("comme caractère", "comme ordinal vaudois"), and the Italian and Vaudois
// numbers in the other languages ("as an italian ordinal", "als ein
// waadtländischer Kardinal", "come cardinale vodese"). Unlike the original's formats, each is parsed
// whole or not at all.
func (ps *parser) veryVeryFormat(cell *Slide) bool {
	for _, f := range []struct {
		words  string
		format int32
	}{
		{"come carattere", FormatCharacter},
		{"come un carattere", FormatCharacter},
		{"come cardinale inglese", FormatEnglishCardinal},
		{"come ordinale inglese", FormatEnglishOrdinal},
		{"come cardinale tedesco", FormatGermanCardinal},
		{"come ordinale tedesco", FormatGermanOrdinal},
		{"come cardinale italiano", FormatItalianCardinal},
		{"come ordinale italiano", FormatItalianOrdinal},
		{"as an italian cardinal", FormatItalianCardinal},
		{"as a italian cardinal", FormatItalianCardinal},
		{"as an italian ordinal", FormatItalianOrdinal},
		{"as a italian ordinal", FormatItalianOrdinal},
		{"als ein italienischer kardinal", FormatItalianCardinal},
		{"als eine italienische ordinalzahl", FormatItalianOrdinal},
		{"come cardinale vodese", FormatVaudoisCardinal},
		{"come ordinale vodese", FormatVaudoisOrdinal},
		{"comme caractere", FormatCharacter},
		{"comme un caractere", FormatCharacter},
		{"comme cardinal anglais", FormatEnglishCardinal},
		{"comme ordinal anglais", FormatEnglishOrdinal},
		{"comme cardinal allemand", FormatGermanCardinal},
		{"comme ordinal allemand", FormatGermanOrdinal},
		{"comme cardinal italien", FormatItalianCardinal},
		{"comme ordinal italien", FormatItalianOrdinal},
		{"comme cardinal vaudois", FormatVaudoisCardinal},
		{"comme ordinal vaudois", FormatVaudoisOrdinal},
		{"as a vaudois cardinal", FormatVaudoisCardinal},
		{"as a vaudois ordinal", FormatVaudoisOrdinal},
		{"als ein waadtlaendischer kardinal", FormatVaudoisCardinal},
		{"als eine waadtlaendische ordinalzahl", FormatVaudoisOrdinal},
	} {
		if ps.seq(strings.Fields(f.words)...) {
			cell.Flags = f.format
			return true
		}
	}
	return false
}

// itCharacter parses the one input format, "come carattere".
func (ps *parser) itCharacter(cell *Slide) bool {
	if ps.seq("come", "carattere") || ps.seq("come", "un", "carattere") {
		cell.Flags = FormatCharacter
		return true
	}
	return false
}

// itNumber parses one declared number.
func (ps *parser) itNumber() bool {
	n, ok := ps.itCardinal()
	return ok && ps.storeSingleNumber(n)
}

// itLabels parses "questo programma usa un'etichetta" / "due etichette".
func (ps *parser) itLabels() bool {
	return ps.itHead("usa", func() bool {
		n, ok := ps.itCardinal()
		if ok && (n == 1 && ps.kw("etichetta") || ps.kw("etichette")) {
			ps.code.LabelsCount = int(n)
			return true
		}
		return false
	})
}
