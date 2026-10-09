package syntax

import "github.com/gersonkurz/sorted/internal/numbers"

// Brazilian Portuguese (#31), from Very Very Sorted! on, like Italian
// (italian.go) and French (french.go): every sentence has a Portuguese form,
// mixed per sentence with the other languages, with no quirks and the tables
// of its English twin. Its wording is Gerson's (decided on #31):
//
//	Este programa usa os números um, dois e três.
//	Este programa sempre vai para o primeiro rótulo e às vezes vai para o
//	    segundo rótulo se a primeira condição for verdadeira.
//	Este programa escreve o primeiro número como caractere.
//	Este programa lê a célula indexada pelo segundo número como caractere.
//	Este programa usa a soma do primeiro número e do segundo número.
//	Este programa usa a condição de que o primeiro número seja igual ao terceiro número.
//	Este programa usa um rótulo.
//	Este programa usa a diferença ordenada entre o primeiro número e o segundo número.
//	Este programa atribui a primeira soma ao primeiro número.
//	Este programa usa o produto do primeiro número e do segundo número.
//	Este programa implementa a primeira atribuição e a primeira saída.
//	Este programa usa a razão entre o primeiro número e o segundo número.
//	Este programa usa a operação lógica nem o primeiro número nem o segundo número.
//	Este programa é muito muito legal.
//
// and "não usa nenhum número" (nenhuma soma, ...) for none, "não vai a lugar
// nenhum", "não pode escrever", "não pode ler", "não faz nenhuma
// atribuição", "é ilógico". NAND is "não ambos a e b". Numbers are Brazil's,
// several words joined by "e" ("vinte e um"), read greedily (see
// numbers.ParseBrazilianCardinal), so the list "vinte e um" is 21 and the
// renderer writes 20 and 1 as "vinte, e um". As in Italian, the parser is
// lenient where the renderer is exact: any article before a reference,
// "seja" or "é", "for" or "é", "menor que" or "menor do que", an ordinal
// of either gender.

// ptArticles are the articles that may start a reference, alone or fused
// with de, a or por ("à" is "a" without its accent).
var ptArticles = []string{"o", "a", "do", "da", "ao", "pelo", "pela"}

func (ps *parser) ptArticle() bool {
	for _, a := range ptArticles {
		if ps.kw(a) {
			return true
		}
	}
	return false
}

// ptIdentifier is a Portuguese reference: "o terceiro número", "a célula
// indexada pela primeira soma".
func (ps *parser) ptIdentifier(op *Operand) bool {
	return ps.ptIndirect(op) || ps.ptDirect(op)
}

func (ps *parser) ptIndirect(op *Operand) bool {
	save := ps.p
	if ps.ptArticle() && ps.seq("celula", "indexada") {
		ps.kw("por") // "por o", or fused: "pelo"
		if ps.ptDirect(op) {
			op.Type |= Indirect
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) ptDirect(op *Operand) bool {
	save := ps.p
	if ps.ptArticle() {
		if n, next := numbers.ParseBrazilianOrdinal(ps.s, ps.p); n > 0 {
			ps.p = next
			ps.skipWhitespaces()
			if t, ok := ps.ptNoun(); ok {
				op.Type = t
				op.Index = n - 1
				return true
			}
		}
	}
	ps.p = save
	return false
}

// ptNoun is what a reference refers to; saída and entrada are output and
// input, rótulo a label.
func (ps *parser) ptNoun() (OperandType, bool) {
	switch {
	case ps.kw("numero"):
		return Number, true
	case ps.kw("celula"):
		return Cell, true
	case ps.kw("soma"):
		return Sum, true
	case ps.seq("diferenca", "ordenada"):
		return Diff, true
	case ps.kw("produto"):
		return Prod, true
	case ps.kw("razao"):
		return Ratio, true
	case ps.kw("atribuicao"):
		return Assign, true
	case ps.kw("salto"):
		return Jump, true
	case ps.kw("rotulo"):
		return Label, true
	case ps.kw("condicao"):
		return Condition, true
	case ps.kw("saida"):
		return Write, true
	case ps.kw("entrada"):
		return Read, true
	case ps.seq("operacao", "logica"):
		return Nand, true
	}
	return 0, false
}

// ptCardinal parses a Brazilian cardinal a program may declare (below
// 1000000000, as in every language).
func (ps *parser) ptCardinal() (int32, bool) {
	n, next, ok := numbers.ParseBrazilianCardinal(ps.s, ps.p)
	if ok = ok && n < 1000000000; ok {
		ps.p = next
		ps.skipWhitespaces()
	}
	return n, ok
}

// ptList parses a Portuguese list: "um, dois e três", "..., e ...".
func (ps *parser) ptList(spec func() bool) bool { return ps.listOf(spec, "e") }

// ptHead parses "este programa" and the words that follow, then spec.
func (ps *parser) ptHead(words string, spec func() bool) bool {
	return ps.with(spec, "este programa "+words)
}

// ptSays parses a sentence that says it all ("este programa não usa nenhum
// número").
func (ps *parser) ptSays(words string) bool { return ps.ptHead(words, always) }

// ptUses parses "este programa usa" and one item after the singular ("a
// soma ...") or a list after the plural ("as somas ...").
func (ps *parser) ptUses(spec func() bool, one, many string) bool {
	return ps.ptHead("usa", func() bool {
		return ps.with(spec, one) || ps.with(func() bool { return ps.ptList(spec) }, many)
	})
}

// ptStatements parses "este programa <verb>" and one statement-like item
// (single, as the original's) or a list.
func (ps *parser) ptStatements(verb string, single, spec func() bool) bool {
	return ps.ptHead(verb, func() bool { return single() || ps.ptList(spec) })
}

// ptPair parses "<a> e <b>" into a cell's operands.
func (ps *parser) ptPair(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.kw("e") && ps.identifier(&cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// ptBetween parses "entre <a> e <b>", the ordered differences and the
// ratios.
func (ps *parser) ptBetween(cell *Slide) bool {
	save := ps.p
	if ps.kw("entre") && ps.ptPair(cell) {
		return true
	}
	ps.p = save
	return false
}

// ptNand parses "nem <a> nem <b>" (NOR) and "não ambos <a> e <b>" (NAND).
func (ps *parser) ptNand(cell *Slide) bool {
	save := ps.p
	if ps.kw("nem") && ps.identifier(&cell.Ops[0]) && ps.kw("nem") && ps.identifier(&cell.Ops[1]) {
		cell.Flags = LogicalNor
		return true
	}
	ps.p = save
	if ps.seq("nao", "ambos") && ps.ptPair(cell) {
		cell.Flags = LogicalNand
		return true
	}
	ps.p = save
	return false
}

// ptCondition parses "a condição de que <a> seja igual <b>" ("ao b") and
// "... seja menor que <b>" (or "menor do que"), "é" for "seja" and "de"
// optional.
func (ps *parser) ptCondition(cell *Slide) bool {
	save := ps.p
	if ps.seq("a", "condicao", "de", "que") || ps.seq("a", "condicao", "que") {
		if !ps.identifier(&cell.Ops[0]) || !ps.kw("seja") && !ps.kw("e") {
			ps.p = save
			return false
		}
		switch {
		case ps.kw("igual"):
			cell.Flags = CompareEqual
		case ps.kw("menor") && (ps.seq("do", "que") || ps.kw("que")):
			cell.Flags = CompareLess
		default:
			ps.p = save
			return false
		}
		if ps.identifier(&cell.Ops[1]) {
			return true
		}
	}
	ps.p = save
	return false
}

// ptJump parses "sempre vai para <label>" and "às vezes vai para <label> se
// <condition> for verdadeira" ("é verdadeira" too).
func (ps *parser) ptJump(cell *Slide) bool {
	save := ps.p
	if ps.seq("sempre", "vai", "para") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label {
		cell.Flags = UnconditionalJump
		return true
	}
	ps.p = save
	if ps.seq("as", "vezes", "vai", "para") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label &&
		ps.kw("se") && ps.identifier(&cell.Ops[1]) && cell.Ops[1].Type == Condition &&
		(ps.kw("for") || ps.kw("e")) && ps.kw("verdadeira") {
		cell.Flags = ConditionalJump
		return true
	}
	ps.p = save
	return false
}

// ptCharacter parses the one input format, "como caractere".
func (ps *parser) ptCharacter(cell *Slide) bool {
	if ps.seq("como", "caractere") || ps.seq("como", "um", "caractere") {
		cell.Flags = FormatCharacter
		return true
	}
	return false
}

// ptNumber parses one declared number.
func (ps *parser) ptNumber() bool {
	n, ok := ps.ptCardinal()
	return ok && ps.storeSingleNumber(n)
}

// ptLabels parses "este programa usa um rótulo" / "dois rótulos".
func (ps *parser) ptLabels() bool {
	return ps.ptHead("usa", func() bool {
		n, ok := ps.ptCardinal()
		if ok && (n == 1 && ps.kw("rotulo") || ps.kw("rotulos")) {
			ps.code.LabelsCount = int(n)
			return true
		}
		return false
	})
}
