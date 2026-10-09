package numbers

import (
	"math"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func TestChineseCardinal(t *testing.T) {
	for _, tt := range []struct {
		n          int32
		hz, pinyin string
	}{
		{0, "零", "líng"}, {1, "一", "yī"}, {10, "十", "shí"}, {11, "十一", "shíyī"}, {20, "二十", "èrshí"}, {23, "二十三", "èrshísān"},
		{101, "一百零一", "yībǎilíngyī"}, {110, "一百一十", "yībǎiyīshí"}, {234, "二百三十四", "èrbǎisānshísì"},
		{1001, "一千零一", "yīqiānlíngyī"}, {1010, "一千零一十", "yīqiānlíngyīshí"}, {10000, "一万", "yīwàn"},
		{10001, "一万零一", "yīwànlíngyī"}, {11000, "一万一千", "yīwànyīqiān"}, {100000, "十万", "shíwàn"},
		{100100, "十万零一百", "shíwànlíngyībǎi"}, {1000000, "一百万", "yībǎiwàn"}, {10000000, "一千万", "yīqiānwàn"},
		{100000000, "一亿", "yīyì"}, {100000010, "一亿零一十", "yīyìlíngyīshí"}, {120000000, "一亿二千万", "yīyìèrqiānwàn"},
		{1000000001, "十亿零一", "shíyìlíngyī"},
		{math.MaxInt32, "二十一亿四千七百四十八万三千六百四十七", ""},
		{-7, "负七", "fù qī"}, {math.MinInt32, "负二十一亿四千七百四十八万三千六百四十八", ""},
	} {
		got := ChineseCardinal(tt.n)
		if got != tt.hz {
			t.Errorf("ChineseCardinal(%d) = %q, want %q", tt.n, got, tt.hz)
		}
		if tt.pinyin != "" && Pinyin(got) != tt.pinyin {
			t.Errorf("Pinyin(%q) = %q, want %q", got, Pinyin(got), tt.pinyin)
		}
	}
	for _, tt := range []struct {
		n             int32
		ord, count, p string
	}{
		{1, "第一", "一", "dì-yī"}, {2, "第二", "两", "dì-èr"}, {23, "第二十三", "二十三", "dì-èrshísān"}, {0, "第零", "零", "dì-líng"},
		{-7, "负第七", "负七", "fù dì-qī"},
	} {
		if got := ChineseOrdinal(tt.n); got != tt.ord || Pinyin(got) != tt.p {
			t.Errorf("ChineseOrdinal(%d) = %q (%q), want %q (%q)", tt.n, got, Pinyin(got), tt.ord, tt.p)
		}
		if got := ChineseCount(tt.n); got != tt.count {
			t.Errorf("ChineseCount(%d) = %q, want %q", tt.n, got, tt.count)
		}
	}
}

// toneless strips the tone marks, as Very Very Sorted! reads pinyin.
func toneless(s string) string {
	t, _, _ := transform.String(transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC), s)
	return t
}

// The parser reads what the formatter writes, in hanzi and in toneless
// pinyin.
func TestChineseRoundTrip(t *testing.T) {
	for _, n := range italianSamples() {
		hz := ChineseCardinal(n)
		ws := []string{hz, toneless(Pinyin(hz))}
		if n == 1000000000 {
			ws = ws[:1] // "shiyi" reads as 11 (see zhFromPinyin)
		}
		for _, w := range ws {
			if v, next, ok := ParseChineseCardinal(w+"个", 0); !ok || v != n || next != len(w) {
				t.Fatalf("ParseChineseCardinal(%q) = %d, %d, %v", w, v, next, ok)
			}
		}
		if n < 1 {
			continue
		}
		for _, o := range []string{ChineseOrdinal(n), toneless(Pinyin(ChineseOrdinal(n)))} {
			if v, next := ParseChineseOrdinal(" "+o+" ge", 0); v != n || next != len(o)+1 {
				t.Fatalf("ParseChineseOrdinal(%q) = %d, %d", o, v, next)
			}
		}
		c := ChineseCount(n)
		if v, _, ok := ParseChineseCardinal(c, 0); !ok || v != n {
			t.Fatalf("count %q = %d, %v", c, v, ok)
		}
	}
}

func TestParseChinese(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int32
		ok   bool
		rest string
	}{
		{"两百", 200, true, ""}, {"两万", 20000, true, ""}, {"一十一", 11, true, ""}, {"liangqian", 2000, true, ""},
		{"一,二", 1, true, ",二"}, {"二和三", 2, true, "和三"}, {"二十三个", 23, true, "个"}, {"yiyi", 100000000, true, ""},
		{"shiyi", 11, true, ""}, {"shiyilingwu", 1000000005, true, ""}, {"ershiyi", 21, true, ""}, {"yibaiyishi", 110, true, ""}, {"yiwanlingyi", 10001, true, ""},
		{"二百五", 0, false, "二百五"}, {"一万五", 0, false, "一万五"}, {"十二十", 0, false, "十二十"}, {"百", 0, false, "百"},
		{"数字", 0, false, "数字"}, {"", 0, false, ""}, {"shuzi", 0, false, "shuzi"}, {"三十亿", 0, false, "三十亿"},
	} {
		v, next, ok := ParseChineseCardinal(tt.s, 0)
		if ok != tt.ok || v != tt.want || tt.s[next:] != tt.rest {
			t.Errorf("ParseChineseCardinal(%q) = %d, %q, %v", tt.s, v, tt.s[next:], ok)
		}
	}
	for _, tt := range []struct {
		s    string
		want int32
	}{{"第一个", 1}, {"diyi ge", 1}, {"di yi ge", 1}, {"di-shiyi", 11}, {"第零", 0}, {"第", 0}, {"dianzi", 0}, {"一", 0}} {
		if v, _ := ParseChineseOrdinal(tt.s, 0); v != tt.want {
			t.Errorf("ParseChineseOrdinal(%q) = %d, want %d", tt.s, v, tt.want)
		}
	}
	if strings.Contains(Pinyin(ChineseCardinal(12)), "'") {
		t.Error("pinyin numbers are one word, without apostrophes")
	}
}
