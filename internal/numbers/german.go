package numbers

import "strings"

// Word tables from GermanNumbers.cpp, in its ASCII spelling ("fuenf",
// "dreissig", "zwoelf"). The C file also declares SingleDigitCardinal,
// SingleDigitOrdinal and DecFactorOrdinal, which no German function reads;
// they are left out.
var (
	deSingleCardinalCombined = [10]string{"", "einund", "zweiund", "dreiund", "vierund", "fuenfund", "sechsund", "siebenund", "achtund", "neunund"}
	deDecFactorCardinal      = [10]string{"", "", "zwanzig", "dreissig", "vierzig", "fuenfzig", "sechzig", "siebzig", "achtzig", "neunzig"}
	deZeroTillTwentyCardinal = [20]string{"null", "eins", "zwei", "drei", "vier", "fuenf", "sechs", "sieben", "acht", "neun", "zehn", "elf", "zwoelf", "dreizehn", "vierzehn", "fuenfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"}
	deHundredCardinals       = [10]string{"", "einhundert", "zweihundert", "dreihundert", "vierhundert", "fuenfhundert", "sechshundert", "siebenhundert", "achthundert", "neunhundert"}
	deZeroTillTwentyOrdinal  = [20]string{"", "erste", "zweite", "dritte", "vierte", "fuenfte", "sechste", "siebente", "achte", "neunte", "zehnte", "elfte", "zwoelfte", "dreizehnte", "vierzehnte", "fuenfzehnte", "sechzehnte", "siebzehnte", "achtzehnte", "neunzehnte"}
)

// deSkipth skips an ordinal ending followed by a blank: "ste", "te" or the
// inflected "n" ("die sechsten Zahl").
func (c *cursor) deSkipth() {
	if c.at("ste") && isBlank(c.byteAt(3)) {
		c.p += 3
	} else if c.at("te") && isBlank(c.byteAt(2)) {
		c.p += 2
	} else if c.at("n") && isBlank(c.byteAt(1)) {
		c.p++
	}
}

// deHalf appends the words for 0 <= n <= 999 followed by suffix; n == 0
// appends nothing (AddNumericHalfCardinal).
func deHalf(b []byte, n int32, suffix string) []byte {
	if n == 0 {
		return b
	}
	b = append(b, deHundredCardinals[n/100]...)
	n %= 100
	if n != 0 {
		if n < 20 {
			b = append(b, deZeroTillTwentyCardinal[n]...)
		} else {
			b = append(b, deSingleCardinalCombined[n%10]...)
			b = append(b, deDecFactorCardinal[n/10]...)
		}
	}
	return append(b, suffix...)
}

// GermanCardinal returns n in German words, written as one word
// ("einhundertdreiundzwanzig", but also "einstausend" and "einsmillionen").
// Zero, and any multiple of 1000000000, yields the empty string. A negative
// number yields ErrCrash.
func GermanCardinal(n int32) (string, error) {
	if n < 0 {
		return "", ErrCrash
	}
	var b []byte
	n %= 1000000000
	if n >= 1000000 {
		b = deHalf(b, n/1000000, "millionen")
		n %= 1000000
	}
	b = deHalf(b, n/1000, "tausend")
	b = deHalf(b, n%1000, "")
	return string(b), nil
}

// GermanOrdinal returns n as a German ordinal, derived from the cardinal: a
// final word from "eins" to "neunzehn" (searched in that order) becomes its
// ordinal, anything else gets "ste" ("millionenste"). Below 1 it returns
// OrdinalError.
func GermanOrdinal(n int32) (string, error) {
	if n < 1 {
		return OrdinalError, nil
	}
	s, _ := GermanCardinal(n) // cannot fail for n >= 1
	for i := 1; i < 20; i++ {
		if hasSuffixFold(s, deZeroTillTwentyCardinal[i]) {
			return s[:len(s)-len(deZeroTillTwentyCardinal[i])] + deZeroTillTwentyOrdinal[i], nil
		}
	}
	return s + "ste", nil
}

// ParseGermanCardinal parses a German cardinal at s[pos:]. It reports the
// value, the position the original leaves its cursor at (which moves even
// when parsing fails), and whether a positive number was found. "null" is
// recognised only before any whitespace.
func ParseGermanCardinal(s string, pos int) (value int32, next int, ok bool) {
	c := cursor{s, pos}
	if c.at("null") {
		c.p += 4
		c.skipws()
		return 0, c.p, true
	}
	var result, total int32
	c.skipws()
	for c.more() {
		c.skipws()
		for i := 1; i < 10; i++ {
			if c.at(deHundredCardinals[i]) {
				result += int32(i * 100)
				c.p += len(deHundredCardinals[i])
				c.skipword("und")
				break
			}
		}
		found := false
		for i := 1; i < 10; i++ {
			if c.at(deSingleCardinalCombined[i]) {
				result += int32(i)
				c.p += len(deSingleCardinalCombined[i])
				c.skipws()
				found = true
				break
			}
		}
		// After "<digit>und" the original reuses the same decade loop as
		// without it; only the fallback to "eins".."neunzehn" differs.
		decade := func() bool {
			for i := 2; i < 10; i++ {
				if c.at(deDecFactorCardinal[i]) {
					result += int32(i * 10)
					c.p += len(deDecFactorCardinal[i])
					c.skipws()
					return true
				}
			}
			return false
		}
		if found {
			decade()
		} else if !decade() {
			for i := 19; i >= 1; i-- {
				if c.at(deZeroTillTwentyCardinal[i]) {
					result += int32(i)
					c.p += len(deZeroTillTwentyCardinal[i])
					c.skipws()
					break
				}
			}
		}
		if c.at("millionen") {
			total += result * 1000000
			c.p += 9
			c.skipword("und")
		} else if c.at("tausend") {
			total += result * 1000
			c.p += 7
			c.skipword("und")
		} else if c.at("hundert") {
			total += result * 100
			c.p += 7
			c.skipword("und")
		} else {
			total += result
			c.skipws()
			break
		}
		result = 0
	}
	return total, c.p, total > 0
}

// ParseGermanOrdinal parses a German ordinal ("erste", "zweihundertelfte",
// "einundzwanzigste", "sechsten", but also plain cardinals) at s[pos:]. It
// returns the value, 0 meaning none, and the position the original leaves its
// cursor at, which moves even when parsing fails.
//
// Two oddities of the original are kept: an ordinal from "erste" to
// "neunzehnte" does not stop the search, so a decade or cardinal directly
// after it is added too; and "millionen" is followed by an if, not an else
// if, so the number before it is counted once more by the final branch.
func ParseGermanOrdinal(s string, pos int) (value int32, next int) {
	c := cursor{s, pos}
	var result, total int32
	c.skipws()
	for c.more() {
		c.skipws()
		found := false
		for i := 1; i < 10; i++ {
			if c.at(deHundredCardinals[i]) {
				result += int32(i * 100)
				c.p += len(deHundredCardinals[i])
				c.skipword("und")
				found = true
				break
			}
		}
		if found && c.at("ste") {
			c.p += 3
			total += result
			c.skipws()
			break
		}
		found = false
		for i := 1; i < 10; i++ {
			if c.at(deSingleCardinalCombined[i]) {
				result += int32(i)
				c.p += len(deSingleCardinalCombined[i])
				c.skipws()
				found = true
				break
			}
		}
		if found {
			for i := 2; i < 10; i++ {
				if c.at(deDecFactorCardinal[i]) {
					result += int32(i * 10)
					c.p += len(deDecFactorCardinal[i])
					c.skipws()
					found = true
					break
				}
			}
			if found && c.at("ste") {
				c.p += 3
				total += result
				c.skipws()
				break
			}
		} else {
			for i := 19; i >= 1; i-- {
				if c.at(deZeroTillTwentyOrdinal[i]) {
					result += int32(i)
					c.p += len(deZeroTillTwentyOrdinal[i])
					c.deSkipth()
					break // found stays false, as in the original
				}
			}
			for i := 2; i < 10; i++ {
				if c.at(deDecFactorCardinal[i]) {
					result += int32(i * 10)
					c.p += len(deDecFactorCardinal[i])
					c.deSkipth()
					found = true
					break
				}
			}
			if !found {
				for i := 19; i >= 1; i-- {
					if c.at(deZeroTillTwentyCardinal[i]) {
						result += int32(i)
						c.p += len(deZeroTillTwentyCardinal[i])
						c.deSkipth()
						break
					}
				}
			}
		}
		if c.at("millionen") {
			total += result * 1000000
			c.p += 9
			c.skipword("und")
		}
		if c.at("millionste") {
			total += result * 1000000
			c.p += 10
			c.skipws()
			break
		} else if c.at("tausend") {
			total += result * 1000
			c.p += 7
			if c.at("ste") {
				c.p += 3
				c.skipws()
				break
			}
			c.skipword("und")
		} else if c.at("hundert") {
			total += result * 100
			c.p += 7
			if c.at("ste") {
				c.p += 3
				c.skipws()
				break
			}
			c.skipword("und")
		} else {
			total += result
			if c.at("ste") {
				c.p += 3
				c.skipws()
				break
			}
			c.skipws()
			break
		}
		result = 0
	}
	return total, c.p
}

// verySpelling writes the umlauts and the ß that the 2000 tables spell out.
// Each replacement has the same length in bytes as what it replaces, which
// the C port in internal/emit relies on.
var verySpelling = strings.NewReplacer("fuenf", "fünf", "zwoelf", "zwölf", "dreissig", "dreißig")

// VerySpelling spells German number words (a cardinal or an ordinal from
// this package) as Very Sorted! prints them, in UTF-8: "fünf", "zwölf",
// "dreißig" instead of "fuenf", "zwoelf", "dreissig" (#28). The words are
// otherwise the original's, "einstausend" and "siebente" included.
func VerySpelling(s string) string { return verySpelling.Replace(s) }
