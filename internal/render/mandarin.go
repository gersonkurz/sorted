package render

import (
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
	"github.com/gersonkurz/sorted/internal/syntax"
)

// Mandarin (#33) is written in two scripts, as two languages: Mandarin in
// simplified hanzi, with no spaces and full-width punctuation ("这个程序使用
// 数字一、二和三。"), and Pinyin, the same words in pinyin with tone marks
// ("Zhège chéngxù shǐyòng shùzì yī, èr hé sān."). The sentences are SVO like
// English, actions put their object first with 把 ("把第一个和赋给第一个数字"),
// jumps are joined by "，并", and 和 is both "and" and the sum ("…的和和…的
// 和"). The wording is Gerson's (#33). Hanzi lay out in their own way, so
// these writers build the text themselves (sentence.raw).

// zh is a text in hanzi and in pinyin.
type zh struct{ hz, py string }

// pick returns the text in the renderer's script.
func (r *renderer) pick(t zh) string {
	if r.lang == Pinyin {
		return t.py
	}
	return t.hz
}

// zhSentences are the fourteen sentences in Mandarin, in order.
func (r *renderer) zhSentences() []func() (sentence, error) {
	return []func() (sentence, error){
		r.zhNumbers, r.zhJumps, r.zhOutputs, r.zhInputs,
		func() (sentence, error) { return r.zhBinary(syntax.Sums, zh{"和", "hé"}) },
		r.zhConditions, r.zhLabels,
		func() (sentence, error) { return r.zhBinary(syntax.Diffs, zh{"差", "chā"}) },
		r.zhAssigns,
		func() (sentence, error) { return r.zhBinary(syntax.Prods, zh{"积", "jī"}) },
		r.zhImplementation,
		func() (sentence, error) { return r.zhBinary(syntax.Ratios, zh{"比", "bǐ"}) },
		r.zhNands, r.zhCool,
	}
}

// zhNouns are the Mandarin nouns of the operand types.
var zhNouns = map[syntax.OperandType]zh{
	syntax.Number: {"数字", "shùzì"}, syntax.Sum: {"和", "hé"}, syntax.Diff: {"差", "chā"}, syntax.Prod: {"积", "jī"},
	syntax.Ratio: {"比", "bǐ"}, syntax.Assign: {"赋值", "fùzhí"}, syntax.Jump: {"跳转", "tiàozhuǎn"},
	syntax.Label: {"标签", "biāoqiān"}, syntax.Condition: {"条件", "tiáojiàn"}, syntax.Write: {"输出", "shūchū"},
	syntax.Read: {"输入", "shūrù"}, syntax.Nand: {"逻辑运算", "luójí yùnsuàn"},
}

// cat joins words: without spaces in hanzi, with them in pinyin.
func (r *renderer) cat(words ...string) string {
	if r.lang == Pinyin {
		return strings.Join(words, " ")
	}
	return strings.Join(words, "")
}

// zhRef writes a Mandarin reference: "第三个数字" / "dì-sān gè shùzì", and
// for the cell a value indexes, "第一个和所指的单元".
func (r *renderer) zhRef(op syntax.Operand) (string, error) {
	if op.Type&syntax.Indirect != 0 {
		inner, err := r.zhRef(syntax.Operand{Type: op.Type &^ syntax.Indirect, Index: op.Index})
		if err != nil {
			return "", err
		}
		return r.cat(inner, r.pick(zh{"所指的单元", "suǒ zhǐ de dānyuán"})), nil
	}
	noun, ok := zhNouns[op.Type]
	if !ok {
		return "", fail("there is no way to refer to an operand of type %d", op.Type)
	}
	n := op.Index + 1
	if n < 1 {
		return "", fail("reference index %d out of range", op.Index)
	}
	return r.cat(r.number(numbers.ChineseOrdinal(n)), r.pick(zh{"个", "gè"}), r.pick(noun)), nil
}

// number writes a number word in the renderer's script.
func (r *renderer) number(hz string) string {
	if r.lang == Pinyin {
		return numbers.Pinyin(hz)
	}
	return hz
}

// zhSep separates two items: the end of the one before ("、", "，") and the
// beginning of the next ("和", "并").
type zhSep struct{ end, begin zh }

var (
	zhListMid  = zhSep{zh{"、", ","}, zh{}}
	zhListLast = zhSep{zh{}, zh{"和", "hé"}}
	zhJumpSep  = zhSep{zh{"，", ","}, zh{"并", "bìng"}}
	zhActSep   = zhSep{zh{"，", ","}, zh{}}
)

// zhLine lays a sentence out: "这个程序" and the verb, the items with their
// separators (mid between all but the last two, last before the last), on
// one line if it fits, otherwise one item per indented line.
func (r *renderer) zhLine(verb zh, items []string, mid, last zhSep) sentence {
	head := r.cat(r.pick(zh{"这个程序", "Zhège chéngxù"}), r.pick(verb))
	head = strings.TrimSpace(head)
	period := r.pick(zh{"。", "."})
	parts := make([]string, len(items)) // each item with its separators
	for i, item := range items {
		sep := mid
		if i == len(items)-1 {
			sep = last
		}
		if i > 0 && r.pick(sep.begin) != "" {
			item = r.cat(r.pick(sep.begin), item)
		}
		if i < len(items)-1 {
			next := mid
			if i == len(items)-2 {
				next = last
			}
			item += r.pick(next.end)
		} else {
			item += period
		}
		parts[i] = item
	}
	if len(items) == 0 {
		return sentence{raw: head + period}
	}
	line := r.cat(append([]string{head}, parts...)...)
	if zhWidth(line) <= width {
		return sentence{raw: line}
	}
	return sentence{raw: head + "\n\t" + strings.Join(parts, "\n\t")}
}

// zhWidth is the width of s on a terminal: two columns for a wide (CJK)
// character.
func zhWidth(s string) int {
	w := 0
	for _, c := range s {
		if c >= 0x1100 {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// zhSays is a sentence that says it all.
func (r *renderer) zhSays(t zh) sentence { return r.zhLine(t, nil, zhListMid, zhListLast) }

// zhCardinal writes a declarable number (as cardinal does for the other
// languages).
func (r *renderer) zhCardinal(n int32) (string, error) {
	if _, err := cardinal(English, n); err != nil {
		return "", err
	}
	return r.number(numbers.ChineseCardinal(n)), nil
}

func (r *renderer) zhNumbers() (sentence, error) {
	if len(r.p.Data) == 0 {
		return r.zhSays(zh{"不使用数字", "bù shǐyòng shùzì"}), nil
	}
	items := make([]string, len(r.p.Data))
	for i, d := range r.p.Data {
		w, err := r.zhCardinal(d)
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	return r.zhLine(zh{"使用数字", "shǐyòng shùzì"}, items, zhListMid, zhListLast), nil
}

func (r *renderer) zhJumps() (sentence, error) {
	js := r.entries(syntax.Jumps)
	if len(js) == 0 {
		return r.zhSays(zh{"哪儿也不去", "nǎr yě bú qù"}), nil
	}
	items := make([]string, len(js))
	for i, j := range js {
		if j.Ops[0].Type != syntax.Label {
			return sentence{}, fail("jump %d does not go to a label", i+1)
		}
		label, err := r.zhRef(j.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		if j.Flags == syntax.UnconditionalJump {
			items[i] = r.cat(r.pick(zh{"总是跳到", "zǒngshì tiàodào"}), label)
			continue
		}
		if j.Ops[1].Type != syntax.Condition {
			return sentence{}, fail("jump %d does not depend on a condition", i+1)
		}
		cond, err := r.zhRef(j.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.cat(r.pick(zh{"在", "zài"}), cond, r.pick(zh{"为真时跳到", "wéi zhēn shí tiàodào"}), label)
	}
	return r.zhLine(zh{}, items, zhJumpSep, zhJumpSep), nil
}

// zhFormatNames name the output formats in Mandarin.
var zhFormatNames = map[int32]zh{
	syntax.FormatCharacter:         {"字符", "zìfú"},
	syntax.FormatEnglishCardinal:   {"英语基数", "Yīngyǔ jīshù"},
	syntax.FormatEnglishOrdinal:    {"英语序数", "Yīngyǔ xùshù"},
	syntax.FormatGermanCardinal:    {"德语基数", "Déyǔ jīshù"},
	syntax.FormatGermanOrdinal:     {"德语序数", "Déyǔ xùshù"},
	syntax.FormatItalianCardinal:   {"意大利语基数", "Yìdàlìyǔ jīshù"},
	syntax.FormatItalianOrdinal:    {"意大利语序数", "Yìdàlìyǔ xùshù"},
	syntax.FormatVaudoisCardinal:   {"沃州基数", "Wòzhōu jīshù"},
	syntax.FormatVaudoisOrdinal:    {"沃州序数", "Wòzhōu xùshù"},
	syntax.FormatBrazilianCardinal: {"巴西基数", "Bāxī jīshù"},
	syntax.FormatBrazilianOrdinal:  {"巴西序数", "Bāxī xùshù"},
	syntax.FormatJapaneseCardinal:  {"日语基数", "Rìyǔ jīshù"},
	syntax.FormatJapaneseOrdinal:   {"日语序数", "Rìyǔ xùshù"},
	syntax.FormatChineseCardinal:   {"中文基数", "Zhōngwén jīshù"},
	syntax.FormatChineseOrdinal:    {"中文序数", "Zhōngwén xùshù"},
}

func (r *renderer) zhOutputs() (sentence, error) {
	ws := r.entries(syntax.Writes)
	switch len(ws) {
	case 0:
		return r.zhSays(zh{"不能写", "bù néng xiě"}), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one output")
	}
	format, ok := zhFormatNames[ws[0].Flags]
	if !ok {
		return sentence{}, fail("output format %d does not exist", ws[0].Flags)
	}
	what, err := r.zhRef(ws[0].Ops[0])
	if err != nil {
		return sentence{}, err
	}
	item := r.cat(r.pick(zh{"把", "bǎ"}), what, r.pick(zh{"作为", "zuòwéi"}), r.pick(format), r.pick(zh{"写出", "xiěchū"}))
	return r.zhLine(zh{}, []string{item}, zhActSep, zhActSep), nil
}

func (r *renderer) zhInputs() (sentence, error) {
	rs := r.entries(syntax.Reads)
	switch len(rs) {
	case 0:
		return r.zhSays(zh{"不能读", "bù néng dú"}), nil
	case 1:
	default:
		return sentence{}, fail("a program can declare only one input")
	}
	if rs[0].Flags != syntax.FormatCharacter {
		return sentence{}, fail("input format %d does not exist", rs[0].Flags)
	}
	what, err := r.zhRef(rs[0].Ops[0])
	if err != nil {
		return sentence{}, err
	}
	item := r.cat(r.pick(zh{"把", "bǎ"}), what, r.pick(zh{"作为字符读入", "zuòwéi zìfú dúrù"}))
	return r.zhLine(zh{}, []string{item}, zhActSep, zhActSep), nil
}

// zhBinary writes an expression table, each entry "<a>和<b>的<noun>": sums
// (和), ordered differences (差), products (积), ratios (比).
func (r *renderer) zhBinary(c syntax.Category, noun zh) (sentence, error) {
	es := r.entries(c)
	if len(es) == 0 {
		return r.zhSays(zh{"不使用" + noun.hz, "bù shǐyòng " + noun.py}), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		a, err := r.zhRef(e.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.zhRef(e.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.cat(a, r.pick(zh{"和", "hé"}), b, r.pick(zh{"的", "de"}), r.pick(noun))
	}
	return r.zhLine(zh{"使用", "shǐyòng"}, items, zhListMid, zhListLast), nil
}

func (r *renderer) zhConditions() (sentence, error) {
	cs := r.entries(syntax.Conditions)
	if len(cs) == 0 {
		return r.zhSays(zh{"不使用条件", "bù shǐyòng tiáojiàn"}), nil
	}
	items := make([]string, len(cs))
	for i, c := range cs {
		a, err := r.zhRef(c.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.zhRef(c.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		cmp := zh{"等于", "děngyú"}
		if c.Flags != syntax.CompareEqual {
			cmp = zh{"小于", "xiǎoyú"}
		}
		items[i] = r.cat(a, r.pick(cmp), b, r.pick(zh{"的条件", "de tiáojiàn"}))
	}
	return r.zhLine(zh{"使用", "shǐyòng"}, items, zhListMid, zhListLast), nil
}

func (r *renderer) zhLabels() (sentence, error) {
	n := r.p.LabelsCount
	switch {
	case n == 0:
		return r.zhSays(zh{"不使用标签", "bù shǐyòng biāoqiān"}), nil
	case n < 0:
		return sentence{}, fail("negative label count %d", n)
	case n >= 1000000000:
		return sentence{}, fail("label count %d cannot be written", n)
	}
	item := r.cat(r.number(numbers.ChineseCount(int32(n))), r.pick(zh{"个标签", "gè biāoqiān"}))
	return r.zhLine(zh{"使用", "shǐyòng"}, []string{item}, zhListMid, zhListLast), nil
}

func (r *renderer) zhAssigns() (sentence, error) {
	as := r.entries(syntax.Assigns)
	if len(as) == 0 {
		return r.zhSays(zh{"不赋值", "bú fùzhí"}), nil
	}
	items := make([]string, len(as))
	for i, a := range as {
		if t := a.Ops[1].Type; t&0xFF != syntax.Number && t&syntax.Indirect == 0 {
			return sentence{}, fail("assignment %d does not store into a cell", i+1)
		}
		from, err := r.zhRef(a.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		to, err := r.zhRef(a.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		items[i] = r.cat(r.pick(zh{"把", "bǎ"}), from, r.pick(zh{"赋给", "fù gěi"}), to)
	}
	return r.zhLine(zh{}, items, zhActSep, zhActSep), nil
}

func (r *renderer) zhImplementation() (sentence, error) {
	ss := r.entries(syntax.Statements)
	if len(ss) == 0 {
		return sentence{}, fail("a program needs at least one statement")
	}
	items := make([]string, len(ss))
	for i, s := range ss {
		w, err := r.zhRef(s.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		items[i] = w
	}
	return r.zhLine(zh{"实现", "shíxiàn"}, items, zhListMid, zhListLast), nil
}

// zhNands writes the logical operations: "既非<a>也非<b>的逻辑运算" (NOR) and
// "并非<a>和<b>都成立的逻辑运算" (NAND).
func (r *renderer) zhNands() (sentence, error) {
	es := r.entries(syntax.Nands)
	if len(es) == 0 {
		return r.zhSays(zh{"不合逻辑", "bù hé luójí"}), nil
	}
	items := make([]string, len(es))
	for i, e := range es {
		if e.Flags != syntax.LogicalNor && e.Flags != syntax.LogicalNand {
			return sentence{}, fail("logical operation %d has flags %d, which only Very Sorted! NAND (1) or NOR (0) has", i, e.Flags)
		}
		a, err := r.zhRef(e.Ops[0])
		if err != nil {
			return sentence{}, err
		}
		b, err := r.zhRef(e.Ops[1])
		if err != nil {
			return sentence{}, err
		}
		if e.Flags == syntax.LogicalNand {
			items[i] = r.cat(r.pick(zh{"并非", "bìngfēi"}), a, r.pick(zh{"和", "hé"}), b, r.pick(zh{"都成立的逻辑运算", "dōu chénglì de luójí yùnsuàn"}))
		} else {
			items[i] = r.cat(r.pick(zh{"既非", "jì fēi"}), a, r.pick(zh{"也非", "yě fēi"}), b, r.pick(zh{"的逻辑运算", "de luójí yùnsuàn"}))
		}
	}
	return r.zhLine(zh{"使用", "shǐyòng"}, items, zhListMid, zhListLast), nil
}

// zhCool is the marker: "这个程序非常非常酷。"
func (r *renderer) zhCool() (sentence, error) {
	n := r.p.Verys
	return r.zhSays(zh{strings.Repeat("非常", n) + "酷", strings.Repeat("fēicháng ", n) + "kù"}), nil
}
