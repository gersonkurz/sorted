package numbers

import "strings"

// Italian number words (#30), new in Very Very Sorted!. Unlike the English and
// German ones they are not a port and have no quirks: every number from
// math.MinInt32 to math.MaxInt32 has its word, zero ("zero", "zeresimo") and
// negative numbers ("meno sette") included. A number is one word, as on a
// cheque ("unmilioneduecentomila"), and its final "tre" carries the accent
// ("ventitré", but "ventitreesimo").
var (
	itUnits    = [20]string{"", "uno", "due", "tre", "quattro", "cinque", "sei", "sette", "otto", "nove", "dieci", "undici", "dodici", "tredici", "quattordici", "quindici", "sedici", "diciassette", "diciotto", "diciannove"}
	itTens     = [10]string{"", "", "venti", "trenta", "quaranta", "cinquanta", "sessanta", "settanta", "ottanta", "novanta"}
	itOrdinals = [11]string{"", "primo", "secondo", "terzo", "quarto", "quinto", "sesto", "settimo", "ottavo", "nono", "decimo"}
)

// itBelow1000 appends 1 <= n <= 999. A ten drops its vowel before "uno" and
// "otto" ("ventuno", "trentotto"), "cento" before "ottanta" ("centottanta");
// "centouno" and "centootto" keep theirs.
func itBelow1000(b []byte, n int64) []byte {
	h, r := n/100, n%100
	if h > 1 {
		b = append(b, itUnits[h]...)
	}
	if h > 0 {
		if r/10 == 8 {
			b = append(b, "cent"...)
		} else {
			b = append(b, "cento"...)
		}
	}
	if r < 20 {
		return append(b, itUnits[r]...)
	}
	tens := itTens[r/10]
	if u := r % 10; u == 1 || u == 8 {
		tens = tens[:len(tens)-1]
	}
	return append(append(b, tens...), itUnits[r%10]...)
}

// itGroup appends n times a power of a thousand: one ("mille", "unmilione")
// or more ("duemila", "duemilioni").
func itGroup(b []byte, n int64, one, more string) []byte {
	if n == 1 {
		return append(b, one...)
	}
	return append(itBelow1000(b, n), more...)
}

// itPlain is the cardinal of n >= 0 without the accent.
func itPlain(n int64) string {
	if n == 0 {
		return "zero"
	}
	var b []byte
	if n >= 1000000000 {
		b = itGroup(b, n/1000000000, "unmiliardo", "miliardi")
		n %= 1000000000
	}
	if n >= 1000000 {
		b = itGroup(b, n/1000000, "unmilione", "milioni")
		n %= 1000000
	}
	if n >= 1000 {
		b = itGroup(b, n/1000, "mille", "mila")
		n %= 1000
	}
	if n > 0 {
		b = itBelow1000(b, n)
	}
	return string(b)
}

// ItalianCardinal returns n in Italian words: "zero", "ventitré",
// "duemilacentottanta", "meno sette".
func ItalianCardinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "meno ", -m
	}
	w := itPlain(m)
	if len(w) > len("tre") && strings.HasSuffix(w, "tre") {
		w = w[:len(w)-1] + "é"
	}
	return s + w
}

// ItalianOrdinal returns n as an Italian ordinal, masculine ("ventitreesimo")
// or feminine ("ventitreesima"): primo to decimo, then the cardinal without
// its last vowel and "-esimo" ("undicesimo", "ventunesimo", "centesimo"),
// where a final "tre" or "sei" keeps it ("ventitreesimo", "ventiseiesimo"),
// "-mila" becomes "-millesimo", "centouno" "centunesimo" and "centootto"
// "centottesimo", and a million is "milionesimo". Zero is "zeresimo", and a
// negative number "meno" and the ordinal of its magnitude.
func ItalianOrdinal(n int32, feminine bool) string {
	m := int64(n)
	s := ""
	if m < 0 {
		s, m = "meno ", -m
	}
	s += itOrdinal(m)
	if feminine {
		s = s[:len(s)-1] + "a"
	}
	return s
}

func itOrdinal(n int64) string {
	switch {
	case n == 0:
		return "zeresimo"
	case n <= 10:
		return itOrdinals[n]
	case n == 1000000:
		return "milionesimo"
	case n == 1000000000:
		return "miliardesimo"
	}
	s := itPlain(n)
	switch {
	case strings.HasSuffix(s, "tre"), strings.HasSuffix(s, "sei"):
	case strings.HasSuffix(s, "mila"):
		s = s[:len(s)-len("mila")] + "mill"
	case strings.HasSuffix(s, "centouno"):
		s = s[:len(s)-len("ouno")] + "un"
	case strings.HasSuffix(s, "centootto"):
		s = s[:len(s)-len("ootto")] + "ott"
	default:
		s = s[:len(s)-1]
	}
	return s + "esimo"
}

// ParseItalianCardinal parses an Italian cardinal of zero or more at s[pos:],
// after any spaces: a word such as "ventitre" (the parser reads Very Very
// Sorted! without accents) or "un" or "una" for one, written as one word or
// with spaces around "milione", "milioni", "miliardo" and "miliardi" ("un
// milione duecentomila"). Elided and full forms both read ("centuno" and
// "centouno", "ventiotto" and "ventotto"). It returns the value, the position
// after the number, and whether there was one; on failure the position is
// pos.
func ParseItalianCardinal(s string, pos int) (value int32, next int, ok bool) {
	w, end := itWord(s, pos)
	for {
		n, after := itWord(s, end)
		if n == "un" { // "un miliardo un milione"
			if m, past := itWord(s, after); m == "milione" {
				n, after = n+m, past
			}
		}
		if n == "" || !itBig(w) && n != "milione" && n != "milioni" && n != "miliardo" && n != "miliardi" {
			break
		}
		if _, ok := itValue(w + n); !ok {
			break
		}
		w, end = w+n, after
	}
	if w == "un" || w == "una" {
		return 1, end, true
	}
	v, ok := itValue(w)
	if !ok || v > 1<<31-1 {
		return 0, pos, false
	}
	return int32(v), end, true
}

// itBig reports whether w ends with a power of a thousand written as a noun,
// after which the number may go on in the next word.
func itBig(w string) bool {
	for _, b := range []string{"milione", "milioni", "miliardo", "miliardi"} {
		if strings.HasSuffix(w, b) {
			return true
		}
	}
	return false
}

// ParseItalianOrdinal parses an Italian ordinal of either gender at s[pos:],
// after any spaces, one word: "primo", "terza", "ventitreesimo",
// "centunesima", "milionesimo". It returns the value, 0 for none (and for
// "zeresimo"), and the position after it.
func ParseItalianOrdinal(s string, pos int) (value int32, next int) {
	w, end := itWord(s, pos)
	if len(w) < 2 || (w[len(w)-1] != 'o' && w[len(w)-1] != 'a') {
		return 0, pos
	}
	w = w[:len(w)-1] + "o"
	for i, o := range itOrdinals {
		if i > 0 && w == o {
			return int32(i), end
		}
	}
	stem, ok := strings.CutSuffix(w, "esimo")
	if !ok {
		return 0, pos
	}
	switch stem {
	case "milion":
		return 1000000, end
	case "miliard":
		return 1000000000, end
	}
	// the cardinal is the stem and its lost vowel, if any ("ventitre-",
	// "vent-i", "cent-o"); "-mill-" is "mila" or "mille" ("duemillesimo")
	var cards []string
	if k, ok := strings.CutSuffix(stem, "mill"); ok {
		cards = []string{k + "mila", k + "mille"}
	} else {
		for _, v := range []string{"", "o", "e", "i", "a"} {
			cards = append(cards, stem+v)
		}
	}
	for _, c := range cards {
		if n, ok := itValue(c); ok && n > 10 && n <= 1<<31-1 {
			return int32(n), end
		}
	}
	return 0, pos
}

// itWord returns the word at s[pos:] after any spaces, and the position after
// it: letters, and the bytes of non-ASCII letters.
func itWord(s string, pos int) (string, int) {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	start := pos
	for pos < len(s) && ('a' <= lower(s[pos]) && lower(s[pos]) <= 'z' || s[pos] >= 0x80) {
		pos++
	}
	return strings.ToLower(s[start:pos]), pos
}

// itValue reads a whole word as a cardinal (zero included).
func itValue(w string) (int64, bool) {
	if w == "zero" {
		return 0, true
	}
	return itPowers(w, []itPower{{"miliard", 1000000000, "o", "i"}, {"milion", 1000000, "e", "i"}})
}

// itPower is a power of a thousand written as a noun: "unmilione",
// "duemilioni".
type itPower struct {
	stem       string
	value      int64
	one, other string
}

// itPowers reads w, whose powers from powers[0] down may be present.
func itPowers(w string, powers []itPower) (int64, bool) {
	if len(powers) == 0 {
		return itThousands(w)
	}
	p := powers[0]
	i := strings.Index(w, p.stem)
	if i < 0 {
		return itPowers(w, powers[1:])
	}
	head, rest := w[:i], w[i+len(p.stem):]
	var count int64
	switch {
	case head == "un" && strings.HasPrefix(rest, p.one):
		count, rest = 1, rest[len(p.one):]
	case strings.HasPrefix(rest, p.other):
		c, ok := itBelow1000Value(head)
		if !ok || c < 2 {
			return 0, false
		}
		count, rest = c, rest[len(p.other):]
	default:
		return 0, false
	}
	if rest == "" {
		return count * p.value, true
	}
	r, ok := itPowers(rest, powers[1:])
	if !ok || r == 0 {
		return 0, false
	}
	return count*p.value + r, true
}

// itThousands reads 1 to 999999.
func itThousands(w string) (int64, bool) {
	var k int64
	switch {
	case strings.HasPrefix(w, "mille"):
		k, w = 1000, w[len("mille"):]
	case strings.Contains(w, "mila"):
		i := strings.Index(w, "mila")
		c, ok := itBelow1000Value(w[:i])
		if !ok || c < 2 {
			return 0, false
		}
		k, w = c*1000, w[i+len("mila"):]
	}
	if w == "" {
		return k, k > 0
	}
	r, ok := itBelow1000Value(w)
	return k + r, ok
}

// itBelow1000Value reads 1 to 999, elided or not.
func itBelow1000Value(w string) (int64, bool) {
	for h := 9; h >= 1; h-- {
		prefix := "cent"
		if h > 1 {
			prefix = itUnits[h] + "cent"
		}
		rest, ok := strings.CutPrefix(w, prefix)
		if !ok {
			continue
		}
		// "cento", "centodue", "centootto", and elided "centotto",
		// "centuno", "centottanta"
		if rest == "o" {
			return int64(h * 100), true
		}
		if full, ok := strings.CutPrefix(rest, "o"); ok {
			if r, ok := itBelow100Value(full); ok {
				return int64(h*100) + r, true
			}
		}
		if strings.HasPrefix(rest, "o") || strings.HasPrefix(rest, "u") {
			if r, ok := itBelow100Value(rest); ok {
				return int64(h*100) + r, true
			}
		}
		return 0, false
	}
	return itBelow100Value(w)
}

// itBelow100Value reads 1 to 99.
func itBelow100Value(w string) (int64, bool) {
	for i := 1; i < 20; i++ {
		if w == itUnits[i] {
			return int64(i), true
		}
	}
	for t := 2; t < 10; t++ {
		tens := itTens[t]
		if w == tens {
			return int64(t * 10), true
		}
		for _, stem := range []string{tens, tens[:len(tens)-1]} {
			if rest, ok := strings.CutPrefix(w, stem); ok {
				for u := 1; u < 10; u++ {
					if rest == itUnits[u] {
						return int64(t*10 + u), true
					}
				}
			}
		}
	}
	return 0, false
}
