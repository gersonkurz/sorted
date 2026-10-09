package numbers

import "strings"

// Japanese number words in romaji (#32), new in Very Very Sorted! like the
// other new languages, and like them without quirks: every int32 has its
// words, zero ("zero", "dai-zero") and negative numbers ("mainasu nana")
// included. A cardinal is one word, written in Hepburn with long vowels
// marked ("nihyakusanjūyon"), grouped by ten thousand (man) and a hundred
// million (oku), with the sound changes of the hundreds and thousands
// (sanbyaku, roppyaku, happyaku, sanzen, hassen, and issen before man or
// oku). An ordinal is the cardinal after the prefix dai- ("dai-san"), built
// from the front unlike any other language Sorted! speaks.
var (
	jaDigits    = [10]string{"", "ichi", "ni", "san", "yon", "go", "roku", "nana", "hachi", "kyū"}
	jaHundreds  = [10]string{"", "hyaku", "nihyaku", "sanbyaku", "yonhyaku", "gohyaku", "roppyaku", "nanahyaku", "happyaku", "kyūhyaku"}
	jaThousands = [10]string{"", "sen", "nisen", "sanzen", "yonsen", "gosen", "rokusen", "nanasen", "hassen", "kyūsen"}
)

// jaBelow10000 returns 1 <= n <= 9999; before says whether man or oku
// follows, where a lone thousand is "issen" ("issenman").
func jaBelow10000(n int64, before bool) string {
	var b strings.Builder
	switch {
	case n == 1000 && before:
		b.WriteString("issen")
	case n >= 1000:
		b.WriteString(jaThousands[n/1000])
	}
	b.WriteString(jaHundreds[n/100%10])
	if t := n / 10 % 10; t > 0 {
		if t > 1 {
			b.WriteString(jaDigits[t])
		}
		b.WriteString("jū")
	}
	b.WriteString(jaDigits[n%10])
	return b.String()
}

// jaWord returns the cardinal of n >= 1.
func jaWord(n int64) string {
	var b strings.Builder
	if g := n / 100000000; g > 0 {
		b.WriteString(jaBelow10000(g, true) + "oku")
	}
	if g := n / 10000 % 10000; g > 0 {
		b.WriteString(jaBelow10000(g, true) + "man")
	}
	if g := n % 10000; g > 0 {
		b.WriteString(jaBelow10000(g, false))
	}
	return b.String()
}

// JapaneseCardinal returns n in Japanese romaji: "zero", "nijūsan",
// "ichimannisensanbyakuyonjūgo", "mainasu nana".
func JapaneseCardinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "mainasu ", -m
	}
	if m == 0 {
		return s + "zero"
	}
	return s + jaWord(m)
}

// JapaneseOrdinal returns n as a Japanese ordinal: "dai-" and the cardinal
// ("dai-ichi", "dai-nijūsan", "dai-zero"), "mainasu" before a negative one.
func JapaneseOrdinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "mainasu ", -m
	}
	if m == 0 {
		return s + "dai-zero"
	}
	return s + "dai-" + jaWord(m)
}

// JapaneseCount returns n with the counter -ko, which counts things, with
// its sound changes: "ikko", "niko", "rokko", "hakko", "jukko", "hyakko",
// "sanbyakko", "roppyakko", "happyakko", "nijūikko".
func JapaneseCount(n int32) string {
	w := JapaneseCardinal(n)
	for _, c := range jaCounts {
		if stem, ok := strings.CutSuffix(w, c.number); ok {
			return stem + c.count
		}
	}
	return w + "ko"
}

// jaCounts are the endings the counter changes.
var jaCounts = []struct{ number, count string }{
	{"ichi", "ikko"}, {"roku", "rokko"}, {"hachi", "hakko"}, {"jū", "jukko"},
	{"hyaku", "hyakko"}, {"byaku", "byakko"}, {"pyaku", "pyakko"},
}

// jaCountsRead are the endings read: those written, and "hachiko" and
// "jikko".
var jaCountsRead = append(append([]struct{ number, count string }{}, jaCounts...),
	struct{ number, count string }{"hachi", "hachiko"}, struct{ number, count string }{"jū", "jikko"})

// JaPlain spells Japanese romaji as Very Very Sorted! compares it: without
// accents, a long u or o written as typed ("juu", "jouken") or with a
// macron ("jū", "jōken") the same as a short one, and "wo" as "o". "oo"
// stays: "gooku" is 500000000.
func JaPlain(w string) string {
	w = strings.ToLower(strings.NewReplacer("ū", "u", "ō", "o", "â", "a", "û", "u", "ô", "o").Replace(w))
	w = strings.NewReplacer("uu", "u", "ou", "o").Replace(w)
	if w == "wo" {
		return "o"
	}
	return w
}

// ParseJapaneseCardinal parses a Japanese cardinal of zero or more at
// s[pos:], after any spaces: one word, read without accents and with long
// vowels as short ones (JaPlain), so "nijuusan" and "nijusan" are both
// 23. It is lenient where the formatter is exact: "shi", "shichi" and "ku"
// for 4, 7 and 9, "issen" or "sen", "rei" for zero. It returns the value,
// the position after the word, and whether there was one; on failure the
// position is pos.
func ParseJapaneseCardinal(s string, pos int) (value int32, next int, ok bool) {
	w, end := jaWordAt(s, pos)
	v, ok := jaValue(w)
	if !ok {
		return 0, pos, false
	}
	return v, end, true
}

// ParseJapaneseOrdinal parses a Japanese ordinal at s[pos:], after any
// spaces: "dai-" and a cardinal, written "dai-san", "daisan" or "dai san".
// It returns the value, 0 for none (and for "dai-zero"), and the position
// after it.
func ParseJapaneseOrdinal(s string, pos int) (value int32, next int) {
	w, end := jaWordAt(s, pos)
	rest, ok := strings.CutPrefix(w, "dai")
	if !ok {
		return 0, pos
	}
	if rest == "" { // "dai san"
		rest, end = jaWordAt(s, end)
	}
	v, ok := jaValue(strings.TrimPrefix(rest, "-"))
	if !ok || v == 0 {
		return 0, pos
	}
	return v, end
}

// ParseJapaneseCount parses a count with the counter -ko ("niko", "rokko",
// "hachiko") at s[pos:], after any spaces. It returns the value, the
// position after the word, and whether there was one; on failure the
// position is pos.
func ParseJapaneseCount(s string, pos int) (value int32, next int, ok bool) {
	w, end := jaWordAt(s, pos)
	stem, ok := strings.CutSuffix(w, "ko")
	if !ok {
		return 0, pos, false
	}
	for _, c := range jaCountsRead {
		if st, ok := strings.CutSuffix(w, JaPlain(c.count)); ok {
			if v, ok := jaValue(st + JaPlain(c.number)); ok && v > 0 {
				return v, end, true
			}
		}
	}
	if v, ok := jaValue(stem); ok && v > 0 {
		return v, end, true
	}
	return 0, pos, false
}

// jaWordAt returns the word at s[pos:] after any spaces, hyphens included,
// as JaPlain spells it, and the position after it.
func jaWordAt(s string, pos int) (string, int) {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	start := pos
	for pos < len(s) && ('a' <= lower(s[pos]) && lower(s[pos]) <= 'z' || s[pos] >= 0x80 || s[pos] == '-') {
		pos++
	}
	return JaPlain(s[start:pos]), pos
}

// jaPlaces are the forms of each place of a group below 10000, as JaPlain
// spells them, with what they are worth; the readings the formatter does
// not write are here too.
var jaPlaces = func() [4]map[string]int64 {
	var p [4]map[string]int64
	for i := range p {
		p[i] = map[string]int64{}
	}
	for d := int64(1); d <= 9; d++ {
		p[0][JaPlain(jaThousands[d])] = d * 1000
		p[1][JaPlain(jaHundreds[d])] = d * 100
		tens := JaPlain(jaDigits[d]) + "ju"
		if d == 1 {
			tens = "ju"
		}
		p[2][tens] = d * 10
		p[3][JaPlain(jaDigits[d])] = d
	}
	p[0]["issen"] = 1000
	for d, alt := range map[int64]string{4: "shi", 7: "shichi", 9: "ku"} {
		p[2][alt+"ju"] = d * 10
		p[3][alt] = d
	}
	return p
}()

// jaGroup reads a group of 1 to 9999 at the start of w, each place by its
// longest form, and returns it and the rest of w.
func jaGroup(w string) (int64, string) {
	var n int64
	for _, place := range jaPlaces {
		best := ""
		for form := range place {
			if len(form) > len(best) && strings.HasPrefix(w, form) {
				best = form
			}
		}
		n += place[best]
		w = w[len(best):]
	}
	return n, w
}

// jaValue reads a whole word as a cardinal: the oku and man groups, then
// the rest.
func jaValue(w string) (int32, bool) {
	if w == "zero" || w == "rei" {
		return 0, true
	}
	var total int64
	for _, sc := range []struct {
		value int64
		name  string
	}{{100000000, "oku"}, {10000, "man"}} {
		if g, rest := jaGroup(w); g > 0 && strings.HasPrefix(rest, sc.name) {
			total += g * sc.value
			w = rest[len(sc.name):]
		}
	}
	g, rest := jaGroup(w)
	total += g
	if rest != "" || total == 0 || total > 1<<31-1 {
		return 0, false
	}
	return int32(total), true
}
