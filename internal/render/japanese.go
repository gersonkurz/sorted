package render

import (
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// Japanese (#32) is verb-final, so it has sentence writers of its own:
// lists of things end with their verb ("Kono puroguramu wa kazu ichi, ni
// to san o tsukaimasu."), lists of actions chain it ("... ni tobi, ... ni
// tobimasu."), and a list of pairs puts its comma after the final "to"
// ("... no wa to, ... no wa"). References are "dai-san no kazu", and "...
// ga sasu seru" for the cell a value points to. The wording is Gerson's
// (#32), romaji in Hepburn with long vowels marked.

// jpHead starts every Japanese sentence.
const jpHead = "Kono puroguramu wa"

// jpSentences are the fourteen sentences in Japanese, in order.
func (r *renderer) jpSentences() []func() (sentence, error) {
	return []func() (sentence, error){
		r.jpNumbers, r.jpJumps, r.jpOutputs, r.jpInputs,
		func() (sentence, error) { return r.jpBinary(syntax.Sums, "wa") },
		r.jpConditions, r.jpLabels,
		func() (sentence, error) { return r.jpBinary(syntax.Diffs, "sa") },
		r.jpAssigns,
		func() (sentence, error) { return r.jpBinary(syntax.Prods, "seki") },
		r.jpImplementation,
		func() (sentence, error) { return r.jpBinary(syntax.Ratios, "hi") },
		r.jpNands, r.jpCool,
	}
}

// jaNouns are the Japanese nouns of the operand types.
var jaNouns = map[syntax.OperandType]string{
	syntax.Number: "kazu", syntax.Sum: "wa", syntax.Diff: "sa", syntax.Prod: "seki", syntax.Ratio: "hi",
	syntax.Assign: "dainyū", syntax.Jump: "janpu", syntax.Label: "raberu", syntax.Condition: "jōken",
	syntax.Write: "shutsuryoku", syntax.Read: "nyūryoku", syntax.Nand: "ronri enzan",
}

// jpRef writes a Japanese reference: "dai-san no kazu", and for the cell a
// value indexes, "dai-ichi no wa ga sasu seru".
func (r *renderer) jpRef(op syntax.Operand) (string, error) {
	if op.Type&syntax.Indirect != 0 {
		inner, err := r.jpRef(syntax.Operand{Type: op.Type &^ syntax.Indirect, Index: op.Index})
		if err != nil {
			return "", err
		}
		return inner + " ga sasu seru", nil
	}
	noun, ok := jaNouns[op.Type]
	if !ok {
		return "", fail("there is no way to refer to an operand of type %d", op.Type)
	}
	n := op.Index + 1
	if n < 1 {
		return "", fail("reference index %d out of range", op.Index)
	}
	return numbers.JapaneseOrdinal(n) + " no " + noun, nil
}

// jpSays is a sentence that says it all.
func jpSays(words string) sentence { return sentence{head: jpHead + " " + words} }

// jpList is a list of things, joined by "to", and its verb.
func jpList(head string, items []string, verb string) sentence {
	return sentence{head: head, items: items, conj: "to", tail: " " + verb}
}

// jpChain is a chain of actions: every item but the last in the stem of
// the verb, the last in the polite form, all after commas.
func jpChain(items []string, stem, polite string) sentence {
	for i := range items {
		if i < len(items)-1 {
			items[i] += " " + stem
		} else {
			items[i] += " " + polite
		}
	}
	return sentence{head: jpHead, items: items}
}

func (r *renderer) jpNumbers() (sentence, error) {
	if len(r.p.Data) == 0 {
		return jpSays("kazu o tsukaimasen"), nil
	}
	items := make([]string, len(r.p.Data))
	for i, d := range r.p.Data {
		w, err := cardinal(Japanese, d)
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	return jpList(jpHead+" kazu", items, "o tsukaimasu"), nil
}

func (r *renderer) jpJumps() (sentence, error) {
	js := r.entries(syntax.Jumps)
	if len(js) == 0 {
		return jpSays("doko ni mo ikimasen"), nil
	}
	items := make([]string, len(js))
	for i, j := range js {
		if j.Ops[0].Type != syntax.Label {
			return sentence{}, fail("jump %d does not go to a label", i+1)
		}
		label, err := r.jpRef(j.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		if j.Flags == syntax.UnconditionalJump {
			items[i] = "itsumo " + label + " ni"
			continue
		}
		if j.Ops[1].Type != syntax.Condition {
			return sentence{}, fail("jump %d does not depend on a condition", i+1)
		}
		cond, err := r.jpRef(j.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = cond + " ga shin nara " + label + " ni"
	}
	return jpChain(items, "tobi", "tobimasu"), nil
}

// jaFormats name the output formats in Japanese.
var jaFormats = map[int32]string{
	syntax.FormatCharacter:         "moji",
	syntax.FormatEnglishCardinal:   "eigo no kisū",
	syntax.FormatEnglishOrdinal:    "eigo no josū",
	syntax.FormatGermanCardinal:    "doitsugo no kisū",
	syntax.FormatGermanOrdinal:     "doitsugo no josū",
	syntax.FormatItalianCardinal:   "itariago no kisū",
	syntax.FormatItalianOrdinal:    "itariago no josū",
	syntax.FormatVaudoisCardinal:   "vōshū no kisū",
	syntax.FormatVaudoisOrdinal:    "vōshū no josū",
	syntax.FormatBrazilianCardinal: "burajiru no kisū",
	syntax.FormatBrazilianOrdinal:  "burajiru no josū",
	syntax.FormatJapaneseCardinal:  "nihongo no kisū",
	syntax.FormatJapaneseOrdinal:   "nihongo no josū",
	syntax.FormatChineseCardinal:   "chūgokugo no kisū",
	syntax.FormatChineseOrdinal:    "chūgokugo no josū",
}

func (r *renderer) jpOutputs() (sentence, error) {
	ws := r.entries(syntax.Writes)
	switch len(ws) {
	case 0:
		return jpSays("kakemasen"), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one output")
	}
	format, ok := jaFormats[ws[0].Flags]
	if !ok {
		return sentence{}, fail("output format %d does not exist", ws[0].Flags)
	}
	what, err := r.jpRef(ws[0].Ops[0])
	if err != nil {
		return sentence{}, err
	}
	return sentence{head: jpHead, items: []string{what + " o " + format + " to shite kakimasu"}}, nil
}

func (r *renderer) jpInputs() (sentence, error) {
	rs := r.entries(syntax.Reads)
	switch len(rs) {
	case 0:
		return jpSays("yomemasen"), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one input")
	}
	if rs[0].Flags != syntax.FormatCharacter {
		return sentence{}, fail("input format %d does not exist", rs[0].Flags)
	}
	what, err := r.jpRef(rs[0].Ops[0])
	if err != nil {
		return sentence{}, err
	}
	return sentence{head: jpHead, items: []string{what + " o moji to shite yomimasu"}}, nil
}

// jpBinary writes an expression table, each entry "<a> to <b> no <noun>":
// sums (wa), ordered differences (sa), products (seki), ratios (hi).
func (r *renderer) jpBinary(c syntax.Category, noun string) (sentence, error) {
	es := r.entries(c)
	if len(es) == 0 {
		return jpSays(noun + " o tsukaimasen"), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		a, err := r.jpRef(e.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.jpRef(e.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = a + " to " + b + " no " + noun
	}
	s := jpList(jpHead, items, "o tsukaimasu")
	s.after = true
	return s, nil
}

func (r *renderer) jpConditions() (sentence, error) {
	cs := r.entries(syntax.Conditions)
	if len(cs) == 0 {
		return jpSays("jōken o tsukaimasen"), nil
	}
	items := make([]string, len(cs))
	for i, c := range cs {
		a, err := r.jpRef(c.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.jpRef(c.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		cmp := " to hitoshii"
		if c.Flags != syntax.CompareEqual {
			cmp = " yori chiisai"
		}
		items[i] = a + " ga " + b + cmp + " to iu jōken"
	}
	s := jpList(jpHead, items, "o tsukaimasu")
	s.after = true
	return s, nil
}

func (r *renderer) jpLabels() (sentence, error) {
	n := r.p.LabelsCount
	switch {
	case n == 0:
		return jpSays("raberu o tsukaimasen"), nil
	case n < 0:
		return sentence{}, fail("negative label count %d", n)
	case n >= 1000000000:
		return sentence{}, fail("label count %d cannot be written", n)
	}
	return sentence{head: jpHead + " raberu o", items: []string{numbers.JapaneseCount(int32(n))}, tail: " tsukaimasu"}, nil
}

func (r *renderer) jpAssigns() (sentence, error) {
	as := r.entries(syntax.Assigns)
	if len(as) == 0 {
		return jpSays("dainyū shimasen"), nil
	}
	items := make([]string, len(as))
	for i, a := range as {
		if t := a.Ops[1].Type; t&0xFF != syntax.Number && t&syntax.Indirect == 0 {
			return sentence{}, fail("assignment %d does not store into a cell", i+1)
		}
		from, err := r.jpRef(a.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		to, err := r.jpRef(a.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = from + " o " + to + " ni dainyū"
	}
	return jpChain(items, "shi", "shimasu"), nil
}

func (r *renderer) jpImplementation() (sentence, error) {
	ss := r.entries(syntax.Statements)
	if len(ss) == 0 {
		return sentence{}, fail("a program needs at least one statement")
	}
	items := make([]string, len(ss))
	for i, s := range ss {
		w, err := r.jpRef(s.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	return jpList(jpHead, items, "o jissō shimasu"), nil
}

// jpNands writes the logical operations: "<a> demo <b> demo nai ronri
// enzan" (NOR) and "<a> to <b> no ryōhō de wa nai ronri enzan" (NAND).
func (r *renderer) jpNands() (sentence, error) {
	es := r.entries(syntax.Nands)
	if len(es) == 0 {
		return jpSays("hironriteki desu"), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		if e.Flags != syntax.LogicalNor && e.Flags != syntax.LogicalNand {
			return sentence{}, fail("logical operation %d has flags %d, which only Very Sorted! NAND (1) or NOR (0) has", i, e.Flags)
		}
		a, err := r.jpRef(e.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.jpRef(e.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		if e.Flags == syntax.LogicalNand {
			items[i] = a + " to " + b + " no ryōhō de wa nai ronri enzan"
		} else {
			items[i] = a + " demo " + b + " demo nai ronri enzan"
		}
	}
	s := jpList(jpHead, items, "o tsukaimasu")
	s.after = true
	return s, nil
}

// jpCool is the marker: "Kono puroguramu wa totemo totemo kakkoii desu."
func (r *renderer) jpCool() (sentence, error) {
	return jpSays(strings.Repeat("totemo ", r.p.Verys) + "kakkoii desu"), nil
}
