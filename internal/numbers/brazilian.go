package numbers

import "strings"

// Brazilian Portuguese number words (#31), new in Very Very Sorted! like the
// Italian and Vaudois ones, and like them without quirks: every int32 has
// its words, zero ("zero", "zerésimo") and negative numbers ("menos sete")
// included. The counting is Brazil's: dezesseis, dezessete, dezenove, and
// the short scale (bilhão); Portugal's dezasseis and mil milhões are not
// numbers here. The words follow CLDR's Portuguese spellout rules: several
// words joined by "e" ("duzentos e trinta e quatro"), "e" before the rest
// of a thousand or more only where that rest is below 100 or a multiple of
// 100 ("mil e cem", "mil cento e um"), and ordinals as compound words that
// agree in gender ("vigésimo terceiro", "vigésima terceira", "dois
// milésimo"), with a masculine cardinal counting millions. Since a list
// joins its last item with "e" too, the parser reads greedily ("vinte e um"
// is 21) and the renderer writes "vinte, e um" for the list.
var (
	ptUnits    = [20]string{"", "um", "dois", "três", "quatro", "cinco", "seis", "sete", "oito", "nove", "dez", "onze", "doze", "treze", "catorze", "quinze", "dezesseis", "dezessete", "dezoito", "dezenove"}
	ptTens     = [10]string{"", "dez", "vinte", "trinta", "quarenta", "cinquenta", "sessenta", "setenta", "oitenta", "noventa"}
	ptHundreds = [10]string{"", "cento", "duzentos", "trezentos", "quatrocentos", "quinhentos", "seiscentos", "setecentos", "oitocentos", "novecentos"}

	ptOrdUnits    = [10]string{"", "primeiro", "segundo", "terceiro", "quarto", "quinto", "sexto", "sétimo", "oitavo", "nono"}
	ptOrdTens     = [10]string{"", "décimo", "vigésimo", "trigésimo", "quadragésimo", "quinquagésimo", "sexagésimo", "septuagésimo", "octogésimo", "nonagésimo"}
	ptOrdHundreds = [10]string{"", "centésimo", "ducentésimo", "tricentésimo", "quadringentésimo", "quingentésimo", "sexcentésimo", "septingentésimo", "octingentésimo", "noningentésimo"}
)

// ptScales are the powers of a thousand that are nouns: the cardinal's
// singular and plural, and the ordinal.
var ptScales = []struct {
	value              int64
	one, many, ordinal string
}{{1000000000, "bilhão", "bilhões", "bilionésimo"}, {1000000, "milhão", "milhões", "milionésimo"}}

// ptBelow1000 returns 1 <= n <= 999: "cem", "cento e um", "duzentos e
// trinta e quatro", feminine "uma", "duas", "duzentas".
func ptBelow1000(n int64, fem bool) string {
	if n == 100 {
		return "cem"
	}
	h, r := n/100, n%100
	var w []string
	if h > 0 {
		w = append(w, ptHundreds[h])
		if fem && h > 1 {
			w[0] = strings.TrimSuffix(w[0], "os") + "as"
		}
	}
	switch {
	case r == 0:
	case r < 20:
		w = append(w, ptUnit(r, fem))
	case r%10 == 0:
		w = append(w, ptTens[r/10])
	default:
		w = append(w, ptTens[r/10], ptUnit(r%10, fem))
	}
	return strings.Join(w, " e ")
}

func ptUnit(n int64, fem bool) string {
	switch {
	case fem && n == 1:
		return "uma"
	case fem && n == 2:
		return "duas"
	}
	return ptUnits[n]
}

// ptCardinal returns the cardinal of n >= 1.
func ptCardinal(n int64, fem bool) string {
	for _, s := range ptScales {
		if c := n / s.value; c > 0 {
			noun := s.many
			if c == 1 {
				noun = s.one
			}
			return ptBelow1000(c, false) + " " + noun + ptRest(n%s.value, fem)
		}
	}
	if k := n / 1000; k > 0 {
		w := "mil"
		if k > 1 {
			w = ptBelow1000(k, fem) + " mil"
		}
		return w + ptRest(n%1000, fem)
	}
	return ptBelow1000(n, fem)
}

// ptRest joins the rest r of a thousand or more: with "e" where r is below
// 100 or a multiple of 100, as CLDR's %%spellout-cardinal-masculine-with-e.
func ptRest(r int64, fem bool) string {
	switch {
	case r == 0:
		return ""
	case r < 100 || r%100 == 0:
		return " e " + ptCardinal(r, fem)
	}
	return " " + ptCardinal(r, fem)
}

// ptOrdinal returns the ordinal of n >= 1, masculine or feminine.
func ptOrdinal(n int64, fem bool) string {
	var w string
	switch {
	case n >= 1000:
		scale, ord := int64(1000), "milésimo"
		for _, s := range ptScales {
			if n >= s.value {
				scale, ord = s.value, s.ordinal
				break
			}
		}
		if c := n / scale; scale > 1000 || c > 1 {
			w = ptCardinal(c, fem) + " "
		}
		w += ptGender(ord, fem)
		n %= scale
	case n >= 100:
		w, n = ptGender(ptOrdHundreds[n/100], fem), n%100
	case n >= 10:
		w, n = ptGender(ptOrdTens[n/10], fem), n%10
	default:
		w, n = ptGender(ptOrdUnits[n], fem), 0
	}
	if n > 0 {
		w += " " + ptOrdinal(n, fem)
	}
	return w
}

// ptGender makes a masculine ordinal word feminine: every one ends in -o.
func ptGender(w string, fem bool) string {
	if fem {
		return strings.TrimSuffix(w, "o") + "a"
	}
	return w
}

// BrazilianCardinal returns n in Brazilian Portuguese: "zero", "vinte e um",
// "duzentos e trinta e quatro", "um milhão", "menos sete". It is masculine,
// as Portuguese counts.
func BrazilianCardinal(n int32) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "menos ", -m
	}
	if m == 0 {
		return s + "zero"
	}
	return s + ptCardinal(m, false)
}

// BrazilianOrdinal returns n as a Brazilian Portuguese ordinal, masculine or
// feminine: "primeiro"/"primeira", "vigésimo terceiro", "dois milésimo",
// "um milionésimo", "duas milésima". Zero is "zerésimo", a negative number
// "menos" and the ordinal of its magnitude.
func BrazilianOrdinal(n int32, feminine bool) string {
	m, s := int64(n), ""
	if m < 0 {
		s, m = "menos ", -m
	}
	if m == 0 {
		return s + ptGender("zerésimo", feminine)
	}
	return s + ptOrdinal(m, feminine)
}

// ParseBrazilianCardinal parses a Brazilian Portuguese cardinal of zero or
// more at s[pos:], after any spaces, read without accents ("tres",
// "milhao"): one or more words, read greedily, so "vinte e um" is 21, but
// "e" joins only what can follow ("dois e tres" is two). It is lenient
// where the formatter is exact: feminine forms ("uma", "duzentas"),
// "quatorze", "milhao" and "milhoes" either way, "e" between any two
// groups or none. It returns the value, the position after the last word,
// and whether there was one; on failure the position is pos.
func ParseBrazilianCardinal(s string, pos int) (value int32, next int, ok bool) {
	r := ptRead(s, pos)
	if r.at("zero") {
		return 0, r.end(pos), true
	}
	v, ok := r.cardinal()
	if !ok || v > 1<<31-1 {
		return 0, pos, false
	}
	return int32(v), r.end(pos), true
}

// ParseBrazilianOrdinal parses a Brazilian Portuguese ordinal at s[pos:],
// after any spaces, read without accents, either gender: "primeiro",
// "vigesima terceira", "dois milesimo", "um milionesimo". It returns the
// value, 0 for none (and for "zeresimo"), and the position after it.
func ParseBrazilianOrdinal(s string, pos int) (value int32, next int) {
	r := ptRead(s, pos)
	var total int64
	for _, sc := range ptScales {
		save := r.i
		c, ok := r.below1000()
		if !ok {
			c = 1 // "milionesimo" alone
		}
		if r.ordinal(plain(sc.ordinal)) {
			total += c * sc.value
		} else {
			r.i = save
		}
	}
	save := r.i
	if r.ordinal("milesimo") {
		total += 1000
	} else if c, ok := r.below1000(); ok && c > 1 && r.ordinal("milesimo") {
		total += c * 1000
	} else {
		r.i = save
	}
	for _, t := range []struct {
		words [10]string
		value int64
	}{{ptOrdHundreds, 100}, {ptOrdTens, 10}, {ptOrdUnits, 1}} {
		for d := 1; d <= 9; d++ {
			if r.ordinal(plain(t.words[d])) {
				total += int64(d) * t.value
				break
			}
		}
	}
	if total == 0 || total > 1<<31-1 {
		return 0, pos
	}
	return int32(total), r.end(pos)
}

// plain spells a word as Very Very Sorted! reads it, without accents.
func plain(w string) string {
	return strings.NewReplacer("é", "e", "ê", "e", "ã", "a", "õ", "o").Replace(w)
}

// ptReader walks the words of a number: each word and where it ends.
type ptReader struct {
	w    []string
	ends []int
	i    int
}

// ptRead reads the words at s[pos:], separated by spaces or tabs, up to
// anything else (a comma, a period, the end).
func ptRead(s string, pos int) *ptReader {
	r := &ptReader{}
	for {
		for pos < len(s) && (s[pos] == ' ' || s[pos] == '\t') {
			pos++
		}
		start := pos
		for pos < len(s) && ('a' <= lower(s[pos]) && lower(s[pos]) <= 'z' || s[pos] >= 0x80) {
			pos++
		}
		if pos == start {
			return r
		}
		r.w = append(r.w, strings.ToLower(s[start:pos]))
		r.ends = append(r.ends, pos)
	}
}

// end is the position after the words consumed, or pos for none.
func (r *ptReader) end(pos int) int {
	if r.i == 0 {
		return pos
	}
	return r.ends[r.i-1]
}

// at consumes one of the words if it comes next.
func (r *ptReader) at(ws ...string) bool {
	for _, w := range ws {
		if r.i < len(r.w) && r.w[r.i] == w {
			r.i++
			return true
		}
	}
	return false
}

// ordinal consumes an ordinal word, given masculine and unaccented, in
// either gender.
func (r *ptReader) ordinal(w string) bool {
	return r.at(w, strings.TrimSuffix(w, "o")+"a")
}

// and consumes "e" if what follows it parses (part), and otherwise leaves
// both alone.
func (r *ptReader) and(part func() bool) bool {
	save := r.i
	if r.at("e") && part() {
		return true
	}
	r.i = save
	return false
}

// cardinal consumes a cardinal of 1 or more: the billions, millions and
// thousands, then the rest, each group after an optional "e".
func (r *ptReader) cardinal() (int64, bool) {
	var total int64
	some := false
	group := func(part func() bool) {
		if some {
			if r.and(part) || part() {
				return
			}
		} else if part() {
			some = true
		}
	}
	for _, sc := range ptScales {
		group(func() bool {
			save := r.i
			if c, ok := r.below1000(); ok && r.at(plain(sc.one), plain(sc.many)) {
				total += c * sc.value
				return true
			}
			r.i = save
			return false
		})
	}
	group(func() bool {
		save := r.i
		if r.at("mil") {
			total += 1000
			return true
		}
		if c, ok := r.below1000(); ok && c > 1 && r.at("mil") {
			total += c * 1000
			return true
		}
		r.i = save
		return false
	})
	group(func() bool {
		c, ok := r.below1000()
		total += c
		return ok
	})
	return total, some
}

// below1000 consumes 1 to 999: "cem", "cento e ...", a hundred ("duzentos",
// "duzentas") and "e" and the rest, or below 100.
func (r *ptReader) below1000() (int64, bool) {
	if r.at("cem") {
		return 100, true
	}
	save := r.i
	if r.at("cento") {
		if b, ok := r.andBelow100(); ok {
			return 100 + b, true
		}
		r.i = save
		return 0, false
	}
	for h := 2; h <= 9; h++ {
		if r.at(ptHundreds[h], strings.TrimSuffix(ptHundreds[h], "os")+"as") {
			b, _ := r.andBelow100()
			return int64(h)*100 + b, true
		}
	}
	return r.below100()
}

// andBelow100 consumes "e" and 1 to 99, or nothing.
func (r *ptReader) andBelow100() (int64, bool) {
	var b int64
	ok := r.and(func() bool {
		var ok bool
		b, ok = r.below100()
		return ok
	})
	return b, ok
}

// below100 consumes 1 to 99: "dezesseis", "vinte", "vinte e um".
func (r *ptReader) below100() (int64, bool) {
	for t := 2; t <= 9; t++ {
		if r.at(ptTens[t]) {
			var u int64
			r.and(func() bool {
				var ok bool
				u, ok = r.unit(1, 9)
				return ok
			})
			return int64(t)*10 + u, true
		}
	}
	return r.unit(1, 19)
}

// unit consumes a unit word from lo to hi ("uma", "duas", "quatorze" too).
func (r *ptReader) unit(lo, hi int) (int64, bool) {
	for u := lo; u <= hi; u++ {
		alt := map[int]string{1: "uma", 2: "duas", 14: "quatorze"}[u]
		if r.at(plain(ptUnits[u])) || alt != "" && r.at(alt) {
			return int64(u), true
		}
	}
	return 0, false
}
