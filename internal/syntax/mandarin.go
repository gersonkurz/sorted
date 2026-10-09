package syntax

import (
	"strings"

	"github.com/gersonkurz/sorted/internal/numbers"
)

// Mandarin (#33), from Very Very Sorted! on, in hanzi and in pinyin, like
// the other Very Very languages: every sentence has a form, mixed per
// sentence with the other languages, with no quirks and the tables of its
// English twin. Its wording is Gerson's (decided on #33):
//
//	这个程序使用数字一、二和三。
//	这个程序总是跳到第一个标签，并在第一个条件为真时跳到第二个标签。
//	这个程序把第一个数字作为字符写出。
//	这个程序把第二个数字所指的单元作为字符读入。
//	这个程序使用第一个数字和第二个数字的和。
//	这个程序使用第一个数字等于第三个数字的条件。
//	这个程序使用一个标签。
//	这个程序使用第一个数字和第二个数字的差。
//	这个程序把第一个和赋给第一个数字。
//	这个程序使用第一个数字和第二个数字的积。
//	这个程序实现第一个赋值和第一个输出。
//	这个程序使用第一个数字和第二个数字的比。
//	这个程序使用既非第一个数字也非第二个数字的逻辑运算。
//	这个程序非常非常酷。
//
// and "不使用数字" (和, 条件, ...) for none, "哪儿也不去", "不能写", "不能读",
// "不赋值", "不合逻辑". NAND is "并非a和b都成立". Mandarin is SVO like
// English; actions put their object first with 把 ("把a赋给b"), and a list of
// them repeats it ("把a赋给b，把c赋给d"). 和 is both "and" and the sum
// ("…的和和…的和"). Hanzi have no spaces, so each phrase matches as a
// prefix (zw), and the filter has made 。，、 a period and commas and the
// traditional characters simplified (FilterVeryVery). Pinyin is read
// without tones, a word at a time ("zhege chengxu shiyong shuzi yi, er he
// san"). Numbers and ordinals are numbers.ParseChineseCardinal and
// numbers.ParseChineseOrdinal. The parser is lenient where the renderer is
// exact: 个 after an ordinal or not, 并 between jumps or not, 两 anywhere.

// zp is a Mandarin phrase in hanzi and in pinyin without tones.
type zp struct{ hz, py string }

// zw matches a phrase: its hanzi as a prefix (hzPrefix), or its pinyin
// word by word.
func (ps *parser) zw(p zp) bool {
	if p.hz == "" && p.py == "" {
		return true
	}
	save := ps.p
	if p.hz != "" && ps.hzPrefix(p.hz) {
		ps.skipWhitespaces()
		return true
	}
	ps.p = save
	if p.py != "" && ps.seq(strings.Fields(p.py)...) {
		return true
	}
	ps.p = save
	return false
}

// hzPrefix matches hanzi character by character, with any whitespace
// before each ("这个程序 使用"), and leaves the cursor after the last one, or
// where it was if they do not match.
func (ps *parser) hzPrefix(hz string) bool {
	save := ps.p
	for _, r := range hz {
		ps.skipWhitespaces()
		if !strings.HasPrefix(ps.s[ps.p:], string(r)) {
			ps.p = save
			return false
		}
		ps.p += len(string(r))
	}
	return true
}

// zhHead parses "这个程序" and the phrase that follows, then spec,
// restoring the cursor unless all succeed.
func (ps *parser) zhHead(p zp, spec func() bool) bool {
	save := ps.p
	if ps.zw(zp{"这个程序", "zhege chengxu"}) && ps.zw(p) && spec() {
		return true
	}
	ps.p = save
	return false
}

// zhSays parses a sentence that says it all ("这个程序不使用数字").
func (ps *parser) zhSays(hz, py string) bool { return ps.zhHead(zp{hz, py}, always) }

// zhUses parses "这个程序使用" (and the noun of numbers) and a list.
func (ps *parser) zhUses(p zp, spec func() bool) bool {
	return ps.zhHead(p, func() bool { return ps.zhList(spec) })
}

// zhStatements parses "这个程序<verb>" and one statement-like item (single,
// which withdraws an entry not followed by the period, as the original's
// do) or a list.
func (ps *parser) zhStatements(verb zp, single, spec func() bool) bool {
	return ps.zhHead(verb, func() bool { return single() || ps.zhList(spec) })
}

// zhActions parses "这个程序" and one action (single) or a chain of them,
// each after a comma and the join ("把", "并").
func (ps *parser) zhActions(first, join zp, single, spec func() bool) bool {
	return ps.zhHead(first, func() bool { return single() || ps.zhChain(join, spec) })
}

// zhList parses a Mandarin list: items separated by commas (、), the last
// after 和, with a comma before it or not. A single item is a list too.
func (ps *parser) zhList(spec func() bool) bool {
	save := ps.p
	if spec() {
		for {
			comma := ps.kw(",")
			if ps.zw(zp{"和", "he"}) {
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

// zhChain parses actions separated by commas, each but the first after the
// join if it is there ("，把…", "，并…").
func (ps *parser) zhChain(join zp, spec func() bool) bool {
	save := ps.p
	if !spec() {
		return false
	}
	for ps.kw(",") {
		ps.zw(join)
		if !spec() {
			ps.p = save
			return false
		}
	}
	return true
}

// zhIdentifier is a Mandarin reference: "第三个数字", "第一个和所指的单元"
// (the cell the first sum points to).
func (ps *parser) zhIdentifier(op *Operand) bool {
	save := ps.p
	n, next := numbers.ParseChineseOrdinal(ps.s, ps.p)
	if n == 0 {
		return false
	}
	ps.p = next
	ps.skipWhitespaces()
	ps.zw(zp{"个", "ge"})
	for _, noun := range []struct {
		p zp
		t OperandType
	}{
		{zp{"数字", "shuzi"}, Number}, {zp{"单元", "danyuan"}, Cell}, {zp{"和", "he"}, Sum}, {zp{"差", "cha"}, Diff},
		{zp{"积", "ji"}, Prod}, {zp{"比", "bi"}, Ratio}, {zp{"赋值", "fuzhi"}, Assign}, {zp{"跳转", "tiaozhuan"}, Jump},
		{zp{"标签", "biaoqian"}, Label}, {zp{"条件", "tiaojian"}, Condition}, {zp{"输出", "shuchu"}, Write},
		{zp{"输入", "shuru"}, Read}, {zp{"逻辑运算", "luoji yunsuan"}, Nand},
	} {
		if ps.zw(noun.p) {
			op.Type, op.Index = noun.t, n-1
			if ps.zw(zp{"所指的单元", "suo zhi de danyuan"}) {
				op.Type |= Indirect
			}
			return true
		}
	}
	ps.p = save
	return false
}

// zhPair parses "<a>和<b>的<noun>": a sum (和), an ordered difference (差),
// a product (积) or a ratio (比).
func (ps *parser) zhPair(cell *Slide, noun zp) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && ps.zw(zp{"和", "he"}) && ps.identifier(&cell.Ops[1]) && ps.zw(zp{"的" + noun.hz, "de " + noun.py}) {
		return true
	}
	ps.p = save
	return false
}

// zhNand parses "既非<a>也非<b>的逻辑运算" (NOR) and "并非<a>和<b>都成立的逻辑运算"
// (NAND).
func (ps *parser) zhNand(cell *Slide) bool {
	save := ps.p
	if ps.zw(zp{"既非", "ji fei"}) && ps.identifier(&cell.Ops[0]) && ps.zw(zp{"也非", "ye fei"}) && ps.identifier(&cell.Ops[1]) &&
		ps.zw(zp{"的逻辑运算", "de luoji yunsuan"}) {
		cell.Flags = LogicalNor
		return true
	}
	ps.p = save
	if (ps.zw(zp{"并非", "bingfei"}) || ps.zw(zp{"", "bing fei"})) && ps.identifier(&cell.Ops[0]) && ps.zw(zp{"和", "he"}) &&
		ps.identifier(&cell.Ops[1]) && ps.zw(zp{"都成立的逻辑运算", "dou chengli de luoji yunsuan"}) {
		cell.Flags = LogicalNand
		return true
	}
	ps.p = save
	return false
}

// zhCondition parses "<a>等于<b>的条件" and "<a>小于<b>的条件".
func (ps *parser) zhCondition(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) {
		switch {
		case ps.zw(zp{"等于", "dengyu"}):
			cell.Flags = CompareEqual
		case ps.zw(zp{"小于", "xiaoyu"}):
			cell.Flags = CompareLess
		default:
			ps.p = save
			return false
		}
		if ps.identifier(&cell.Ops[1]) && ps.zw(zp{"的条件", "de tiaojian"}) {
			return true
		}
	}
	ps.p = save
	return false
}

// zhJump parses "总是跳到<label>" and "在<condition>为真时跳到<label>".
func (ps *parser) zhJump(cell *Slide) bool {
	save := ps.p
	if ps.zw(zp{"总是跳到", "zongshi tiaodao"}) && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label {
		cell.Flags = UnconditionalJump
		return true
	}
	ps.p = save
	if ps.zw(zp{"在", "zai"}) && ps.identifier(&cell.Ops[1]) && cell.Ops[1].Type == Condition &&
		ps.zw(zp{"为真时跳到", "wei zhen shi tiaodao"}) && ps.identifier(&cell.Ops[0]) && cell.Ops[0].Type == Label {
		cell.Flags = ConditionalJump
		return true
	}
	ps.p = save
	return false
}

// zhAssign parses "<a>赋给<b>" (after 把).
func (ps *parser) zhAssign(cell *Slide) bool {
	save := ps.p
	if ps.identifier(&cell.Ops[0]) && (ps.zw(zp{"赋给", "fu gei"}) || ps.zw(zp{"", "fugei"})) &&
		ps.zhIdentifier(&cell.Ops[1]) && ps.assignable(cell.Ops[1]) {
		return true
	}
	ps.p = save
	return false
}

// zhFormats name the output formats in Mandarin; 語 reads as 语.
var zhFormats = func() []struct {
	p      zp
	format int32
} {
	type f = struct {
		p      zp
		format int32
	}
	var fs []f
	for _, l := range []struct {
		hz, py            string
		cardinal, ordinal int32
	}{
		{"英语", "yingyu", FormatEnglishCardinal, FormatEnglishOrdinal},
		{"德语", "deyu", FormatGermanCardinal, FormatGermanOrdinal},
		{"意大利语", "yidaliyu", FormatItalianCardinal, FormatItalianOrdinal},
		{"沃州", "wozhou", FormatVaudoisCardinal, FormatVaudoisOrdinal},
		{"巴西", "baxi", FormatBrazilianCardinal, FormatBrazilianOrdinal},
		{"日语", "riyu", FormatJapaneseCardinal, FormatJapaneseOrdinal},
		{"中文", "zhongwen", FormatChineseCardinal, FormatChineseOrdinal},
	} {
		for _, hz := range []string{l.hz, strings.ReplaceAll(l.hz, "语", "語")} {
			fs = append(fs, f{zp{hz + "基数", l.py + " jishu"}, l.cardinal}, f{zp{hz + "序数", l.py + " xushu"}, l.ordinal})
		}
	}
	return append(fs, f{zp{"字符", "zifu"}, FormatCharacter})
}()

// zhWrite parses what follows the reference in an output: "作为<format>写出".
func (ps *parser) zhWrite(cell *Slide) bool {
	save := ps.p
	if ps.zw(zp{"作为", "zuowei"}) {
		for _, f := range zhFormats {
			if ps.zw(f.p) && ps.zw(zp{"写出", "xiechu"}) {
				cell.Flags = f.format
				return true
			}
		}
	}
	ps.p = save
	return false
}

// zhRead parses what follows the reference in an input: "作为字符读入".
func (ps *parser) zhRead(cell *Slide) bool {
	if ps.zw(zp{"作为字符读入", "zuowei zifu duru"}) {
		cell.Flags = FormatCharacter
		return true
	}
	return false
}

// zhNumber parses one declared number (below 1000000000, as in every
// language).
func (ps *parser) zhNumber() bool {
	n, next, ok := numbers.ParseChineseCardinal(ps.s, ps.p)
	if !ok || n >= 1000000000 {
		return false
	}
	ps.p = next
	ps.skipWhitespaces()
	return ps.storeSingleNumber(n)
}

// zhLabels parses "这个程序使用两个标签", the count with its measure word.
func (ps *parser) zhLabels() bool {
	return ps.zhHead(zp{"使用", "shiyong"}, func() bool {
		n, next, ok := numbers.ParseChineseCardinal(ps.s, ps.p)
		if !ok || n >= 1000000000 {
			return false
		}
		ps.p = next
		ps.skipWhitespaces()
		if ps.zw(zp{"个", "ge"}) && ps.zw(zp{"标签", "biaoqian"}) {
			ps.code.LabelsCount = int(n)
			return true
		}
		return false
	})
}

// zhCool parses the marker in hanzi, "这个程序非常非常酷。" or "非常非常酷。"
// (the pinyin is in cool's table).
func (ps *parser) zhCool() bool {
	save := ps.p
	very := strings.Repeat("非常", ps.verys) + "酷"
	if (ps.zw(zp{"这个程序" + very, ""}) || ps.zw(zp{very, ""})) && ps.kw(".") {
		return true
	}
	ps.p = save
	return false
}
