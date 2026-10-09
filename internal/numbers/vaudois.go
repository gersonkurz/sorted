package numbers

import "strings"

// Vaudois French number words (#29), new in Very Very Sorted! like the
// Italian ones, and like them without quirks: every int32 has its words,
// zero ("zéro", "zéroième") and negative numbers ("moins sept") included.
// The counting is Vaud's, decimal: septante, huitante, nonante. A number is
// one word, its parts joined by hyphens as the 1990 rectified spelling
// writes it ("deux-cent-vingt-et-un", "deux-millions-trois-cents"); the
// parser in syntax keeps hyphens inside words in Very Very Sorted!.
var (
	frUnits = [17]string{"", "un", "deux", "trois", "quatre", "cinq", "six", "sept", "huit", "neuf", "dix", "onze", "douze", "treize", "quatorze", "quinze", "seize"}
	frTens  = [10]string{"", "dix", "vingt", "trente", "quarante", "cinquante", "soixante", "septante", "huitante", "nonante"}
)

// frBelow100 returns 1 <= n <= 99: "dix-sept", "vingt-et-un", "septante-deux".
func frBelow100(n int64) []string {
	switch {
	case n <= 16:
		return []string{frUnits[n]}
	case n < 20:
		return []string{"dix", frUnits[n-10]}
	}
	t, u := n/10, n%10
	switch u {
	case 0:
		return []string{frTens[t]}
	case 1:
		return []string{frTens[t], "et", "un"}
	}
	return []string{frTens[t], frUnits[u]}
}

// frBelow1000 returns 1 <= n <= 999. A multiplied hundred takes an s when
// nothing follows it (plural says whether a noun such as "millions" may):
// "deux-cents", "deux-cent-un", "deux-cent-mille", "deux-cents-millions".
func frBelow1000(n int64, plural bool) []string {
	h, r := n/100, n%100
	var w []string
	switch {
	case h == 1:
		w = []string{"cent"}
	case h > 1 && r == 0 && plural:
		w = []string{frUnits[h], "cents"}
	case h > 1:
		w = []string{frUnits[h], "cent"}
	}
	if r > 0 {
		w = append(w, frBelow100(r)...)
	}
	return w
}

// frWords returns the parts of the cardinal of n >= 0.
func frWords(n int64) []string {
	if n == 0 {
		return []string{"zéro"}
	}
	var w []string
	for _, p := range []struct {
		value      int64
		one, other string
	}{{1000000000, "milliard", "milliards"}, {1000000, "million", "millions"}} {
		if c := n / p.value; c > 0 {
			if c == 1 {
				w = append(w, "un", p.one)
			} else {
				w = append(append(w, frBelow1000(c, true)...), p.other)
			}
			n %= p.value
		}
	}
	if k := n / 1000; k > 0 {
		if k > 1 {
			w = append(w, frBelow1000(k, false)...)
		}
		w = append(w, "mille")
		n %= 1000
	}
	if n > 0 {
		w = append(w, frBelow1000(n, true)...)
	}
	return w
}

// VaudoisCardinal returns n in Vaudois French: "zéro", "vingt-et-un",
// "nonante-neuf", "deux-cents", "moins sept". The masculine "un" stands for
// one, as French counts.
func VaudoisCardinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "moins ", -m
	}
	return s + strings.Join(frWords(m), "-")
}

// VaudoisOrdinal returns n as a Vaudois French ordinal: "premier" (feminine
// "première"), then the cardinal with "-ième" ("deuxième", "vingt-et-unième",
// "deux-centième", "millième"), cinq and neuf as "cinquième" and
// "neuvième", a million "millionième", and a final hundred loses its plural
// ("deux-centième", "deux-cent-millionième"). Zero is "zéroième", a negative number
// "moins" and the ordinal of its magnitude.
func VaudoisOrdinal(n int32, feminine bool) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "moins ", -m
	}
	switch {
	case m == 1 && feminine:
		return s + "première"
	case m == 1:
		return s + "premier"
	case m == 0:
		return s + "zéroième"
	}
	w := frWords(m)
	if len(w) == 2 && w[0] == "un" { // "un-million": "millionième"
		w = w[1:]
	}
	last := w[len(w)-1]
	switch {
	case last == "cents":
		last = "cent"
	case last == "millions" || last == "milliards":
		// the noun becomes the ordinal, so the hundred before it no
		// longer multiplies a noun: "deux-cent-millionième" (but
		// "deux-cents-millions-troisième")
		last = last[:len(last)-1]
		if len(w) >= 2 && w[len(w)-2] == "cents" {
			w[len(w)-2] = "cent"
		}
	case last == "cinq":
		last = "cinqu"
	case last == "neuf":
		last = "neuv"
	case strings.HasSuffix(last, "e"):
		last = last[:len(last)-1]
	}
	w[len(w)-1] = last + "ième"
	return s + strings.Join(w, "-")
}

// ParseVaudoisCardinal parses a Vaudois French cardinal of zero or more at
// s[pos:], after any spaces: one word, its parts joined by hyphens, read
// without accents ("vingt-et-un", "deux-cents", "zero"), "une" for one
// too ("vingt-et-une"), "cent" and "cents", "million" and "millions" either
// way. It returns the value, the position after the word, and whether
// there was one; on failure the position is pos.
func ParseVaudoisCardinal(s string, pos int) (value int32, next int, ok bool) {
	w, end := frWord(s, pos)
	v, ok := frValue(strings.Split(w, "-"))
	if !ok || v > 1<<31-1 {
		return 0, pos, false
	}
	return int32(v), end, true
}

// ParseVaudoisOrdinal parses a Vaudois French ordinal at s[pos:], after any
// spaces, read without accents: "premier", "premiere", "second", "seconde",
// or a cardinal with "-ieme" ("deuxieme", "vingt-et-unieme",
// "millionieme"). It returns the value, 0 for none (and for "zeroieme"),
// and the position after it.
func ParseVaudoisOrdinal(s string, pos int) (value int32, next int) {
	w, end := frWord(s, pos)
	switch w {
	case "premier", "premiere":
		return 1, end
	case "second", "seconde":
		return 2, end
	}
	stem, ok := strings.CutSuffix(w, "ieme")
	if !ok || stem == "" {
		return 0, pos
	}
	parts := strings.Split(stem, "-")
	last := parts[len(parts)-1]
	var cards []string // the last part as the cardinal writes it
	switch {
	case last == "cinqu":
		cards = []string{"cinq"}
	case last == "neuv":
		cards = []string{"neuf"}
	case last == "million" || last == "milliard":
		cards = []string{last, last + "s"}
	default:
		cards = []string{last, last + "e", last + "s"}
	}
	for _, c := range cards {
		p := append(append([]string{}, parts[:len(parts)-1]...), c)
		if len(p) == 1 && (c == "million" || c == "milliard") {
			p = []string{"un", c}
		}
		if n, ok := frValue(p); ok && n > 1 && n <= 1<<31-1 {
			return int32(n), end
		}
	}
	return 0, pos
}

// frWord returns the word at s[pos:] after any spaces, hyphens included, and
// the position after it.
func frWord(s string, pos int) (string, int) {
	for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
		pos++
	}
	start := pos
	for pos < len(s) && ('a' <= lower(s[pos]) && lower(s[pos]) <= 'z' || s[pos] >= 0x80 || s[pos] == '-') {
		pos++
	}
	return strings.ToLower(s[start:pos]), pos
}

// frValue reads the parts of a cardinal, all of them.
func frValue(p []string) (int64, bool) {
	if len(p) == 1 && p[0] == "zero" {
		return 0, true
	}
	r := frReader{p: p}
	var total int64
	for _, scale := range []struct {
		value     int64
		one, many string
	}{{1000000000, "milliard", "milliards"}, {1000000, "million", "millions"}} {
		save := r.i
		c, ok := r.below1000()
		if ok && (r.at(scale.one) || r.at(scale.many)) {
			total += c * scale.value
			continue
		}
		r.i = save
	}
	save := r.i
	if r.at("mille") {
		total += 1000
	} else if c, ok := r.below1000(); ok && c > 1 && r.at("mille") {
		total += c * 1000
	} else {
		r.i = save
	}
	if c, ok := r.below1000(); ok {
		total += c
	}
	return total, r.i == len(p) && len(p) > 0 && total > 0
}

// frReader walks the parts of a cardinal.
type frReader struct {
	p []string
	i int
}

// at consumes the part w if it comes next.
func (r *frReader) at(w string) bool {
	if r.i < len(r.p) && r.p[r.i] == w {
		r.i++
		return true
	}
	return false
}

// unit consumes a unit word from lo to hi ("une" is "un"), returning it.
func (r *frReader) unit(lo, hi int) (int64, bool) {
	if r.i < len(r.p) {
		w := r.p[r.i]
		if w == "une" {
			w = "un"
		}
		for u := lo; u <= hi; u++ {
			if w == frUnits[u] {
				r.i++
				return int64(u), true
			}
		}
	}
	return 0, false
}

// below1000 consumes 1 to 999: hundreds, then tens and units.
func (r *frReader) below1000() (int64, bool) {
	save := r.i
	var n int64
	if h, ok := r.unit(2, 9); ok && (r.at("cent") || r.at("cents")) {
		n = h * 100
	} else {
		r.i = save
		if r.at("cent") || r.at("cents") {
			n = 100
		}
	}
	if b, ok := r.below100(); ok {
		n += b
	}
	return n, n > 0
}

// below100 consumes 1 to 99: "seize", "dix-sept", "vingt", "vingt-et-un",
// "septante-deux".
func (r *frReader) below100() (int64, bool) {
	save := r.i
	for t := 9; t >= 1; t-- {
		if r.at(frTens[t]) {
			if r.at("et") {
				if u, ok := r.unit(1, 1); ok {
					return int64(t*10) + u, true
				}
				r.i--
			}
			if u, ok := r.unit(1, 9); ok && (t > 1 || u >= 7) {
				return int64(t*10) + u, true
			}
			r.i = save + 1
			return int64(t * 10), true
		}
	}
	r.i = save
	return r.unit(1, 16)
}
