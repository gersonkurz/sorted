package syntax

import (
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
)

// Japanese in romaji (#32), from Very Very Sorted! on, like Italian,
// French and Portuguese: every sentence has a Japanese form, mixed per
// sentence with the other languages, with no quirks and the tables of its
// English twin. Its wording is Gerson's (decided on #32):
//
//	Kono puroguramu wa kazu ichi, ni to san o tsukaimasu.
//	Kono puroguramu wa itsumo dai-ichi no raberu ni tobi,
//	    dai-ichi no jōken ga shin nara dai-ni no raberu ni tobimasu.
//	Kono puroguramu wa dai-ichi no kazu o moji to shite kakimasu.
//	Kono puroguramu wa dai-ni no kazu ga sasu seru o moji to shite yomimasu.
//	Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no wa o tsukaimasu.
//	Kono puroguramu wa dai-ichi no kazu ga dai-san no kazu to hitoshii to iu jōken o tsukaimasu.
//	Kono puroguramu wa raberu o ikko tsukaimasu.
//	Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no sa o tsukaimasu.
//	Kono puroguramu wa dai-ichi no wa o dai-ichi no kazu ni dainyū shimasu.
//	Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no seki o tsukaimasu.
//	Kono puroguramu wa dai-ichi no dainyū to dai-ichi no shutsuryoku o jissō shimasu.
//	Kono puroguramu wa dai-ichi no kazu to dai-ni no kazu no hi o tsukaimasu.
//	Kono puroguramu wa dai-ichi no kazu demo dai-ni no kazu demo nai ronri enzan o tsukaimasu.
//	Kono puroguramu wa totemo totemo kakkoii desu.
//
// and "kazu o tsukaimasen" (wa, sa, ...) for none, "doko ni mo ikimasen",
// "kakemasen", "yomemasen", "dainyū shimasen", "hironriteki desu". NAND is
// "a to b no ryōhō de wa nai". Japanese is verb-final: the sentences that
// list things end with their verb after the list ("... o tsukaimasu"), and
// those that list actions chain the verb, every item but the last ending
// in its stem and a comma ("... ni tobi, ... ni tobimasu"). Ordinals are a
// prefix (numbers.ParseJapaneseOrdinal), numbers one word. Words are
// compared as numbers.JaPlain spells them, so a long vowel may have a
// macron, be doubled or neither ("jōken", "jouken", "joken"). The parser
// is lenient where the renderer is exact: the stem or the polite form of a
// verb anywhere in a chain, a comma before or after the "to" of a list.

// jw matches Japanese words, all or nothing, each compared as JaPlain
// spells it.
func (ps *parser) jw(words string) bool {
	save := ps.p
	for _, w := range strings.Fields(words) {
		ps.skipWhitespaces()
		end := ps.p
		for end < len(ps.s) && isLetter(ps.s[end]) {
			end++
		}
		if end == ps.p || numbers.JaPlain(ps.s[ps.p:end]) != w {
			ps.p = save
			return false
		}
		ps.p = end
		ps.skipWhitespaces()
	}
	return true
}

// jpHead parses "kono puroguramu wa" and the words that follow, then spec,
// restoring the cursor unless both succeed.
func (ps *parser) jpHead(words string, spec func() bool) bool {
	save := ps.p
	if ps.jw("kono puroguramu wa "+words) && spec() {
		return true
	}
	ps.p = save
	return false
}

// jpSays parses a sentence that says it all ("kono puroguramu wa kazu o
// tsukaimasen").
func (ps *parser) jpSays(words string) bool { return ps.jpHead(words, always) }

// jpUses parses "kono puroguramu wa", a list, and "o tsukaimasu".
func (ps *parser) jpUses(words string, spec func() bool) bool {
	return ps.jpHead(words, func() bool { return ps.jpList(spec) && ps.jw("o tsukaimasu") })
}

// jpStatements parses "kono puroguramu wa", one statement-like item or a
// list, and the verb: the single item, as the original's, is withdrawn
// unless the verb and the period follow it directly (jpSingle).
func (ps *parser) jpStatements(c Category, spec func() bool, verb string) bool {
	return ps.jpHead("", func() bool {
		return (ps.jpSingle(c, spec, verb) || ps.jpList(spec)) && ps.jw(verb)
	})
}

// jpSingle is the original's single entry in a verb-final sentence: spec,
// kept if the verb and the period follow, otherwise withdrawn (cancel,
// TypeCount stays), as singleCondition and singleStatement do.
func (ps *parser) jpSingle(c Category, spec func() bool, verb string) bool {
	save := ps.p
	if spec() {
		after := ps.p
		if ps.jw(verb) && ps.at() == '.' {
			ps.p = after
			return true
		}
		ps.cancel(c)
	}
	ps.p = save
	return false
}

// jpActions parses "kono puroguramu wa" and one action (single, as the
// original's) or a chain of them, each with its verb.
func (ps *parser) jpActions(single, spec func() bool) bool {
	return ps.jpHead("", func() bool { return single() || ps.jpChain(spec) })
}

// jpList parses a Japanese list: items separated by commas, the last after
// "to", with a comma before or after it or none ("ichi, ni to san"). A
// single item is a list too.
func (ps *parser) jpList(spec func() bool) bool {
	save := ps.p
	if spec() {
		for {
			comma := ps.kw(",")
			if ps.jw("to") {
				ps.kw(",")
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

// jpChain parses actions separated by commas ("... ni tobi, ... ni
// tobimasu"). A single action is a chain too.
func (ps *parser) jpChain(spec func() bool) bool {
	save := ps.p
	if !spec() {
		return false
	}
	for ps.kw(",") {
		if !spec() {
			ps.p = save
			return false
		}
	}
	return true
}

// jpVerb parses a verb in its stem or polite form ("tobi", "tobimasu").
func (ps *parser) jpVerb(stem, polite string) bool { return ps.jw(polite) || ps.jw(stem) }

// jpIdentifier is a Japanese reference: "dai-san no kazu", or "dai-ichi no
// wa ga sasu seru", the cell the first sum points to.
func (ps *parser) jpIdentifier(op *Operand) bool {
	save := ps.p
	n, next := numbers.ParseJapaneseOrdinal(ps.s, ps.p)
	if n == 0 {
		return false
	}
	ps.p = next
	ps.skipWhitespaces()
	if t, ok := ps.jpNoun(); ok {
		op.Type, op.Index = t, n-1
		if ps.jw("ga sasu seru") {
			op.Type |= Indirect
		}
		return true
	}
	ps.p = save
	return false
}

// jpNoun parses "no" and what a reference refers to.
func (ps *parser) jpNoun() (OperandType, bool) {
	save := ps.p
	if ps.jw("no") {
		for _, n := range []struct {
			words string
			t     OperandType
		}{
			{"kazu", Number}, {"seru", Cell}, {"wa", Sum}, {"sa", Diff}, {"seki", Prod}, {"hi", Ratio},
			{"dainyu", Assign}, {"janpu", Jump}, {"raberu", Label}, {"joken", Condition},
			{"shutsuryoku", Write}, {"nyuryoku", Read}, {"ronri enzan", Nand},
		} {
			if ps.jw(n.words) {
				return n.t, true
			}
		}
	}
	ps.p = save
	return 0, false
}

// jpPair parses "<a> to <b> no <noun>", a sum (wa), an ordered difference
// (sa), a product (seki) or a ratio (hi).
func (ps *parser) jpPair(cell *Slide, noun string) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.jw("to") && ps.identifier(&cell.Ops[1]) && ps.jw("no "+noun) {
		return true
	}
	ps.p = save
	return false
}

// jpNand parses "<a> demo <b> demo nai ronri enzan" (NOR) and "<a> to <b>
// no ryōhō de wa nai ronri enzan" (NAND).
func (ps *parser) jpNand(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.jw("demo") && ps.identifier(&cell.Ops[1]) && ps.jw("demo nai ronri enzan") {
		cell.Flags = LogicalNor
		return true
	}
	ps.p = save
	if ps.identifier(&cell.Ops[0]) && ps.jw("to") && ps.identifier(&cell.Ops[1]) && ps.jw("no ryoho de wa nai ronri enzan") {
		cell.Flags = LogicalNand
		return true
	}
	ps.p = save
	return false
}

// jpCondition parses "<a> ga <b> to hitoshii to iu jōken" and "<a> ga <b>
// yori chiisai to iu jōken".
func (ps *parser) jpCondition(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.jw("ga") && ps.identifier(&cell.Ops[1]) {
		switch {
		case ps.jw("to hitoshii to iu joken"):
			cell.Flags = CompareEqual
			return true
		case ps.jw("yori chiisai to iu joken"):
			cell.Flags = CompareLess
			return true
		}
	}
	ps.p = save
	return false
}

// jpJump parses "itsumo <label> ni tobi(masu)" and "<condition> ga shin
// nara <label> ni tobi(masu)".
func (ps *parser) jpJump(cell *Slide) bool {
	save := ps.p
	if ps.jw("itsumo") && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label && ps.jw("ni") && ps.jpVerb("tobi", "tobimasu") {
		cell.Flags = UnconditionalJump
		return true
	}
	ps.p = save
	if ps.identifier(&cell.Ops[1]) && cell.Ops[1].Type == Condition && ps.jw("ga shin nara") &&
		ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label && ps.jw("ni") && ps.jpVerb("tobi", "tobimasu") {
		cell.Flags = ConditionalJump
		return true
	}
	ps.p = save
	return false
}

// jpAssign parses "<a> o <b> ni dainyū shi(masu)".
func (ps *parser) jpAssign(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.jw("o") && ps.jpIdentifier(&cell.Ops[1]) && ps.assignable(cell.Ops[1]) &&
		ps.jw("ni dainyu") && ps.jpVerb("shi", "shimasu") {
		return true
	}
	ps.p = save
	return false
}

// jpFormats name the output formats in Japanese, as JaPlain spells them.
var jpFormats = []struct {
	words  string
	format int32
}{
	{"moji", FormatCharacter},
	{"eigo no kisu", FormatEnglishCardinal}, {"eigo no josu", FormatEnglishOrdinal},
	{"doitsugo no kisu", FormatGermanCardinal}, {"doitsugo no josu", FormatGermanOrdinal},
	{"itariago no kisu", FormatItalianCardinal}, {"itariago no josu", FormatItalianOrdinal},
	{"voshu no kisu", FormatVaudoisCardinal}, {"voshu no josu", FormatVaudoisOrdinal},
	{"burajiru no kisu", FormatBrazilianCardinal}, {"burajiru no josu", FormatBrazilianOrdinal},
	{"nihongo no kisu", FormatJapaneseCardinal}, {"nihongo no josu", FormatJapaneseOrdinal},
	{"chugokugo no kisu", FormatChineseCardinal}, {"chugokugo no josu", FormatChineseOrdinal},
}

// jpWrite parses what follows the reference in an output: "o <format> to
// shite kaki(masu)".
func (ps *parser) jpWrite(cell *Slide) bool {
	save := ps.p
	for _, f := range jpFormats {
		if ps.jw("o "+f.words+" to shite") && ps.jpVerb("kaki", "kakimasu") {
			cell.Flags = f.format
			return true
		}
		ps.p = save
	}
	return false
}

// jpRead parses what follows the reference in an input: "o moji to shite
// yomi(masu)".
func (ps *parser) jpRead(cell *Slide) bool {
	save := ps.p
	if ps.jw("o moji to shite") && ps.jpVerb("yomi", "yomimasu") {
		cell.Flags = FormatCharacter
		return true
	}
	ps.p = save
	return false
}

// jpNumber parses one declared number (below 1000000000, as in every
// language).
func (ps *parser) jpNumber() bool {
	n, next, ok := numbers.ParseJapaneseCardinal(ps.s, ps.p)
	if !ok || n >= 1000000000 {
		return false
	}
	ps.p = next
	ps.skipWhitespaces()
	return ps.storeSingleNumber(n)
}

// jpLabels parses "kono puroguramu wa raberu o niko tsukaimasu", the count
// with its counter (numbers.ParseJapaneseCount).
func (ps *parser) jpLabels() bool {
	return ps.jpHead("raberu o", func() bool {
		n, next, ok := numbers.ParseJapaneseCount(ps.s, ps.p)
		if !ok || n >= 1000000000 {
			return false
		}
		ps.p = next
		ps.skipWhitespaces()
		if ps.jw("tsukaimasu") {
			ps.code.LabelsCount = int(n)
			return true
		}
		return false
	})
}
