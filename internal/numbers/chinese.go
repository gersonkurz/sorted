package numbers

import "strings"

// Chinese number words (#33), new in Very Very Sorted! like the other new
// languages, and like them without quirks: every int32 has its words, zero
// ("零", "第零") and negative numbers ("负七") included. A cardinal is one
// word in hanzi, grouped by 万 (10^4) and 亿 (10^8), with 零 marking a gap
// ("一百零一", "一万零一") and "十" for "一十" only at the very start ("十一",
// but "一百一十"). An ordinal is 第 and the cardinal, a prefix like
// Japanese dai-. In pinyin a number is one word too ("èrbǎisānshísì"); with
// its tones stripped, as Very Very Sorted! reads it, 一 and 亿 are both
// "yi", and a "yi" right after a digit is 亿 ("yiyi" is 一亿).
var (
	zhDigits = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}
	zhPinyin = map[string]string{
		"零": "líng", "一": "yī", "二": "èr", "三": "sān", "四": "sì", "五": "wǔ", "六": "liù", "七": "qī",
		"八": "bā", "九": "jiǔ", "十": "shí", "百": "bǎi", "千": "qiān", "万": "wàn", "亿": "yì", "两": "liǎng",
		"第": "dì-", "负": "fù ",
	}
)

// zhSection returns 1 <= n <= 9999; inner says whether a higher group came
// before, where ten is "一十".
func zhSection(n int64, inner bool) string {
	var b strings.Builder
	wrote, zero := false, false
	for _, p := range []struct {
		value int64
		unit  string
	}{{1000, "千"}, {100, "百"}, {10, "十"}, {1, ""}} {
		d := n / p.value % 10
		if d == 0 {
			zero = zero || wrote
			continue
		}
		if zero {
			b.WriteString("零")
			zero = false
		}
		if !(p.value == 10 && d == 1 && !inner && !wrote) {
			b.WriteString(zhDigits[d])
		}
		b.WriteString(p.unit)
		wrote = true
	}
	return b.String()
}

// zhWord returns the cardinal of n >= 1 in hanzi.
func zhWord(n int64) string {
	var b strings.Builder
	started, gap := false, false
	for _, g := range []struct {
		value int64
		unit  string
	}{{100000000, "亿"}, {10000, "万"}, {1, ""}} {
		s := n / g.value % 10000
		if g.value == 100000000 {
			s = n / g.value
		}
		if s == 0 {
			gap = gap || started
			continue
		}
		if started && (s < 1000 || gap) {
			b.WriteString("零")
		}
		b.WriteString(zhSection(s, started) + g.unit)
		started, gap = true, false
	}
	return b.String()
}

// ChineseCardinal returns n in hanzi: "零", "二十三", "一百零一",
// "二十一亿四千七百四十八万三千六百四十七", "负七".
func ChineseCardinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "负", -m
	}
	if m == 0 {
		return s + "零"
	}
	return s + zhWord(m)
}

// ChineseOrdinal returns n as a Chinese ordinal: 第 and the cardinal
// ("第一", "第二十三", "第零"), 负 before a negative one ("负第七").
func ChineseOrdinal(n int32) string {
	if n < 0 {
		return "负第" + ChineseCardinal(n)[len("负"):]
	}
	return "第" + ChineseCardinal(n)
}

// ChineseCount returns n as it is counted with a measure word: 两 for two
// ("两个"), the cardinal otherwise.
func ChineseCount(n int32) string {
	if n == 2 {
		return "两"
	}
	return ChineseCardinal(n)
}

// Pinyin writes Chinese number words (cardinal, ordinal, count) in pinyin,
// one word with tone marks: "èrshísān", "dì-yī", "liǎng", "fù qī".
func Pinyin(hanzi string) string {
	var b strings.Builder
	for _, r := range hanzi {
		b.WriteString(zhPinyin[string(r)])
	}
	return b.String()
}

// zhChars are the characters of a number in hanzi, with their value: a
// digit (两 is 2) or a unit.
var zhChars = map[rune]int64{
	'零': 0, '一': 1, '二': 2, '两': 2, '三': 3, '四': 4, '五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
	'十': 10, '百': 100, '千': 1000, '万': 10000, '亿': 100000000,
}

// zhSyllables are the pinyin of zhChars without tones, longest first, and
// the characters they stand for (yi is 一 or 亿, see zhFromPinyin).
var zhSyllables = []struct{ py, hz string }{
	{"liang", "两"}, {"ling", "零"}, {"qian", "千"}, {"jiu", "九"}, {"liu", "六"}, {"san", "三"}, {"bai", "百"},
	{"wan", "万"}, {"shi", "十"}, {"er", "二"}, {"si", "四"}, {"wu", "五"}, {"qi", "七"}, {"ba", "八"}, {"yi", "一"},
}

// zhFromPinyin turns a toneless pinyin number into hanzi, or "" if it is
// not one: "ershisan" is 二十三, and "yi" after a digit (not 零) is 亿. After
// 十, "yi" may be either ("shiyi" is 十一, "shiyilingwu" 十亿零五); it is 一
// unless only 亿 makes a number, so 1000000000 alone does not read back in
// pinyin (it reads 11), which no declaration, reference or count needs.
func zhFromPinyin(w string) string {
	var rs []rune
	var after10 []int // where a "yi" follows 十
	digit := false
	for w != "" {
		found := false
		for _, s := range zhSyllables {
			if rest, ok := strings.CutPrefix(w, s.py); ok {
				r := []rune(s.hz)[0]
				if r == '一' && digit {
					r = '亿'
				}
				if r == '一' && len(rs) > 0 && rs[len(rs)-1] == '十' {
					after10 = append(after10, len(rs))
				}
				v := zhChars[r]
				digit = v >= 1 && v <= 9
				rs = append(rs, r)
				w, found = rest, true
				break
			}
		}
		if !found {
			return ""
		}
	}
	if _, ok := zhValue(string(rs)); !ok {
		for _, i := range after10 {
			alt := append([]rune{}, rs...)
			alt[i] = '亿'
			if _, ok := zhValue(string(alt)); ok {
				return string(alt)
			}
		}
	}
	return string(rs)
}

// zhSectionValue reads a group below 10000 in hanzi: digits each before
// its unit (千, 百, 十) in falling order, 零 for a gap, a last digit alone
// only after 十 or a gap ("一百零一", not the colloquial "二百五"), and 十
// alone at the start for 一十. After a higher group (inner), a lone digit
// needs its 零 too ("一万零五", not the colloquial "一万五").
func zhSectionValue(rs []rune, inner bool) (int64, bool) {
	var n int64
	last, gap := int64(100000), false // the last unit read
	for i := 0; i < len(rs); i++ {
		v, ok := zhChars[rs[i]]
		switch {
		case !ok || v >= 10000:
			return 0, false
		case v == 0:
			gap = true
		case v >= 10:
			if v != 10 || n != 0 {
				return 0, false
			}
			n, last, gap = 10, 10, false
		case i+1 < len(rs) && zhChars[rs[i+1]] >= 10 && zhChars[rs[i+1]] < 10000:
			u := zhChars[rs[i+1]]
			if u >= last {
				return 0, false
			}
			n, last, gap = n+v*u, u, false
			i++
		case i == len(rs)-1 && (last == 10 || last == 100000 && !inner || gap):
			n += v
		default:
			return 0, false
		}
	}
	return n, n > 0
}

// zhValue reads a whole cardinal in hanzi: the 亿 and 万 groups, then the
// rest.
func zhValue(s string) (int32, bool) {
	if s == "零" {
		return 0, true
	}
	rs := []rune(s)
	var total int64
	for _, g := range []struct {
		value int64
		unit  rune
	}{{100000000, '亿'}, {10000, '万'}} {
		for i, r := range rs {
			if r == g.unit {
				v, ok := zhSectionValue(rs[:i], total > 0)
				if !ok {
					return 0, false
				}
				total += v * g.value
				rs = rs[i+1:]
				break
			}
		}
	}
	if len(rs) > 0 {
		v, ok := zhSectionValue(rs, total > 0)
		if !ok {
			return 0, false
		}
		total += v
	}
	if total == 0 || total > 1<<31-1 {
		return 0, false
	}
	return int32(total), true
}

// zhNumberAt returns the number at s[pos:] after any spaces, as hanzi (a
// run of number characters, or a pinyin word turned into them), and the
// position after it.
func zhNumberAt(s string, pos int) (string, int) {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	start := pos
	var b strings.Builder
	for _, r := range s[pos:] {
		if _, ok := zhChars[r]; !ok {
			break
		}
		b.WriteRune(r)
		pos += len(string(r))
	}
	if pos > start {
		return b.String(), pos
	}
	for pos < len(s) && ('a' <= lower(s[pos]) && lower(s[pos]) <= 'z' || s[pos] == '-') {
		pos++
	}
	return zhFromPinyin(strings.ToLower(s[start:pos])), pos
}

// ParseChineseCardinal parses a Chinese cardinal of zero or more at
// s[pos:], after any spaces: a run of hanzi number characters ("二十三",
// "两百") or a pinyin word without tones ("ershisan", "liangbai"). It
// returns the value, the position after it, and whether there was one; on
// failure the position is pos.
func ParseChineseCardinal(s string, pos int) (value int32, next int, ok bool) {
	hz, end := zhNumberAt(s, pos)
	v, ok := zhValue(hz)
	if hz == "" || !ok {
		return 0, pos, false
	}
	return v, end, true
}

// ParseChineseOrdinal parses a Chinese ordinal at s[pos:], after any
// spaces: 第 and a cardinal ("第二十三"), or in pinyin "di-ershisan",
// "diershisan" or "di ershisan". It returns the value, 0 for none (and for
// 第零), and the position after it.
func ParseChineseOrdinal(s string, pos int) (value int32, next int) {
	p := pos
	for p < len(s) && (s[p] == ' ' || s[p] == '\t') {
		p++
	}
	if rest, ok := strings.CutPrefix(s[p:], "第"); ok {
		if v, end, ok := ParseChineseCardinal(s, len(s)-len(rest)); ok && v > 0 {
			return v, end
		}
		return 0, pos
	}
	if !strings.HasPrefix(strings.ToLower(s[p:]), "di") {
		return 0, pos
	}
	p += 2
	if p < len(s) && s[p] == '-' {
		p++
	}
	if v, end, ok := ParseChineseCardinal(s, p); ok && v > 0 {
		return v, end
	}
	return 0, pos
}
