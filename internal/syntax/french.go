package syntax

import "github.com/gersonkurz/sorted/internal/numbers"

// French (#29), as spoken in Vaud, from Very Very Sorted! on, like Italian
// (italian.go): every sentence has a French form, mixed per sentence with
// the other languages, with no quirks and the tables of its English twin.
// Its wording is Gerson's (decided on #29):
//
//	Ce programme utilise les nombres un, deux et trois.
//	Ce programme va toujours à la première étiquette et va parfois à la
//	    deuxième étiquette si la première condition est vraie.
//	Ce programme écrit le premier nombre comme caractère.
//	Ce programme lit la cellule indexée par le deuxième nombre comme caractère.
//	Ce programme utilise la somme du premier nombre et du deuxième nombre.
//	Ce programme utilise la condition que le premier nombre soit égal au troisième nombre.
//	Ce programme utilise une étiquette.
//	Ce programme utilise la différence ordonnée entre le premier nombre et le deuxième nombre.
//	Ce programme affecte la première somme au premier nombre.
//	Ce programme utilise le produit du premier nombre et du deuxième nombre.
//	Ce programme implémente la première affectation et la première sortie.
//	Ce programme utilise le rapport du premier nombre au deuxième nombre.
//	Ce programme utilise l'opération logique ni le premier nombre ni le deuxième nombre.
//	Ce programme est très très chouette.
//
// and "n'utilise aucun nombre" (aucune somme, ...) for none, "Y'a pas le feu
// au lac." (or "ne va nulle part") for no jumps, "ne peut pas écrire", "ne
// peut pas lire", "est illogique". NAND is "pas à la fois a et b". Numbers
// are Vaud's, one word joined by hyphens ("septante-et-un"), and the parser
// reads past the fillers "eh", "hein", "quoi" and "voilà" (FilterVeryVery).
// As in Italian, the parser is lenient where the renderer is exact: any
// article before a reference, "soit" or "est", "égal" or "égale".

// frArticle parses an article that may start a reference, alone or fused
// with à or de.
func (ps *parser) frArticle() bool {
	for _, a := range []string{"le", "la", "l", "du", "au", "de la", "a la", "de l", "a l"} {
		if ps.with(always, a) {
			return true
		}
	}
	return false
}

// frIdentifier is a French reference: "le troisième nombre", "la cellule
// indexée par la première somme".
func (ps *parser) frIdentifier(op *Operand) bool {
	return ps.frIndirect(op) || ps.frDirect(op)
}

func (ps *parser) frIndirect(op *Operand) bool {
	save := ps.p
	if ps.frArticle() && ps.seq("cellule", "indexee", "par") && ps.frDirect(op) {
		op.Type |= Indirect
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) frDirect(op *Operand) bool {
	save := ps.p
	if ps.frArticle() {
		if n, next := numbers.ParseVaudoisOrdinal(ps.s, ps.p); n > 0 {
			ps.p = next
			ps.skipWhitespaces()
			if t, ok := ps.frNoun(); ok {
				op.Type = t
				op.Index = n - 1
				return true
			}
		}
	}
	ps.p = save
	return false
}

// frNoun is what a reference refers to; sortie and entrée are output and
// input, affectation an assignment.
func (ps *parser) frNoun() (OperandType, bool) {
	switch {
	case ps.kw("nombre"):
		return Number, true
	case ps.kw("cellule"):
		return Cell, true
	case ps.kw("somme"):
		return Sum, true
	case ps.seq("difference", "ordonnee"):
		return Diff, true
	case ps.kw("produit"):
		return Prod, true
	case ps.kw("rapport"):
		return Ratio, true
	case ps.kw("affectation"):
		return Assign, true
	case ps.kw("saut"):
		return Jump, true
	case ps.kw("etiquette"):
		return Label, true
	case ps.kw("condition"):
		return Condition, true
	case ps.kw("sortie"):
		return Write, true
	case ps.kw("entree"):
		return Read, true
	case ps.seq("operation", "logique"):
		return Nand, true
	}
	return 0, false
}

// frCardinal parses a Vaudois cardinal a program may declare (below
// 1000000000, as in every language).
func (ps *parser) frCardinal() (int32, bool) {
	n, next, ok := numbers.ParseVaudoisCardinal(ps.s, ps.p)
	if ok = ok && n < 1000000000; ok {
		ps.p = next
		ps.skipWhitespaces()
	}
	return n, ok
}

// frList parses a French list: "un, deux et trois", "..., et ...".
func (ps *parser) frList(spec func() bool) bool { return ps.listOf(spec, "et") }

// frHead parses "ce programme" and the words that follow, then spec.
func (ps *parser) frHead(words string, spec func() bool) bool {
	return ps.with(spec, "ce programme "+words)
}

// frSays parses a sentence that says it all ("ce programme n'utilise aucun
// nombre").
func (ps *parser) frSays(words string) bool { return ps.frHead(words, always) }

// frUses parses "ce programme utilise" and one item after the singular ("la
// somme ...") or a list after the plural ("les sommes ...").
func (ps *parser) frUses(spec func() bool, one, many string) bool {
	return ps.frHead("utilise", func() bool {
		return ps.with(spec, one) || ps.with(func() bool { return ps.frList(spec) }, many)
	})
}

// frStatements parses "ce programme <verb>" and one statement-like item
// (single, as the original's) or a list.
func (ps *parser) frStatements(verb string, single, spec func() bool) bool {
	return ps.frHead(verb, func() bool { return single() || ps.frList(spec) })
}

// frPair parses "<a> et <b>" into a cell's operands.
func (ps *parser) frPair(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.kw("et") && ps.identifier(&cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// frBetween parses "entre <a> et <b>".
func (ps *parser) frBetween(cell *Slide) bool {
	save := ps.p
	if ps.kw("entre") && ps.frPair(cell) {
		return true
	}
	ps.p = save
	return false
}

// frRatio parses "du <a> au <b>": the second article says "à".
func (ps *parser) frRatio(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.frIdentifier(&cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// frNand parses "ni <a> ni <b>" (NOR) and "pas à la fois <a> et <b>"
// (NAND).
func (ps *parser) frNand(cell *Slide) bool {
	save := ps.p
	if ps.kw("ni") && ps.identifier(&cell.Ops[0]) && ps.kw("ni") && ps.identifier(&cell.Ops[1]) {
		cell.Flags = LogicalNor
		return true
	}
	ps.p = save
	if ps.seq("pas", "a", "la", "fois") && ps.frPair(cell) {
		cell.Flags = LogicalNand
		return true
	}
	ps.p = save
	return false
}

// frCondition parses "la condition que <a> soit égal <b>" (or "inférieur",
// either gender, and "est" for "soit"); the second operand's article says
// "à" ("au").
func (ps *parser) frCondition(cell *Slide) bool {
	save := ps.p
	if ps.seq("la", "condition", "que") && ps.identifier(&cell.Ops[0]) && (ps.kw("soit") || ps.kw("est")) {
		switch {
		case ps.kw("egal") || ps.kw("egale"):
			cell.Flags = CompareEqual
		case ps.kw("inferieur") || ps.kw("inferieure"):
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

// frJump parses "va toujours <label>" and "va parfois <label> si
// <condition> est vraie".
func (ps *parser) frJump(cell *Slide) bool {
	save := ps.p
	if ps.seq("va", "toujours") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label {
		cell.Flags = UnconditionalJump
		return true
	}
	ps.p = save
	if ps.seq("va", "parfois") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label &&
		ps.kw("si") && ps.identifier(&cell.Ops[1]) && cell.Ops[1].Type == Condition &&
		(ps.kw("est") || ps.kw("soit")) && ps.kw("vraie") {
		cell.Flags = ConditionalJump
		return true
	}
	ps.p = save
	return false
}

// frCharacter parses the one input format, "comme caractère".
func (ps *parser) frCharacter(cell *Slide) bool {
	if ps.seq("comme", "caractere") || ps.seq("comme", "un", "caractere") {
		cell.Flags = FormatCharacter
		return true
	}
	return false
}

// frNumber parses one declared number.
func (ps *parser) frNumber() bool {
	n, ok := ps.frCardinal()
	return ok && ps.storeSingleNumber(n)
}

// frLabels parses "ce programme utilise une étiquette" / "deux étiquettes".
func (ps *parser) frLabels() bool {
	return ps.frHead("utilise", func() bool {
		n, ok := ps.frCardinal()
		if ok && (n == 1 && ps.kw("etiquette") || ps.kw("etiquettes")) {
			ps.code.LabelsCount = int(n)
			return true
		}
		return false
	})
}
