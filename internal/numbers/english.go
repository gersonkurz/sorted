package numbers

// Word tables from EnglishNumbers.cpp, misspellings included ("fourty",
// "fiveteen", "nineth", "twelveth"). HundredOrdinals, declared but never read
// in the C file, is left out.
var (
	enSingleCardinal         = [10]string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
	enDecFactorCardinal      = [10]string{"", "", "twenty", "thirty", "fourty", "fifty", "sixty", "seventy", "eighty", "ninety"}
	enZeroTillTwentyCardinal = [20]string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fiveteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	enHundredCardinals       = [10]string{"", "onehundred", "twohundred", "threehundred", "fourhundred", "fivehundred", "sixhundred", "sevenhundred", "eighthundred", "ninehundred"}
	enSingleOrdinal          = [10]string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "nineth"}
	enDecFactorOrdinal       = [10]string{"", "", "twentieth", "thirtieth", "fourtieth", "fiftieth", "sixtieth", "seventieth", "eightieth", "ninetieth"}
	enZeroTillTwentyOrdinal  = [20]string{"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "ninth", "tenth", "eleventh", "twelveth", "thirteenth", "fourteenth", "fifteenth", "sixteenth", "seventeenth", "eighteenth", "nineteenth"}
)

// enSkipth skips an ordinal suffix: "th" followed by a blank, or an "h" that is
// the entire rest of the text.
func (c *cursor) enSkipth() {
	if c.at("th") && isBlank(c.byteAt(2)) {
		c.p += 2
	} else if c.rest("h") {
		c.p++
	}
}

// enHalf appends the words for 0 <= n <= 999 followed by suffix; n == 0
// appends nothing (AddNumericHalfCardinal).
func enHalf(b []byte, n int32, suffix string) []byte {
	if n == 0 {
		return b
	}
	b = append(b, enHundredCardinals[n/100]...)
	n %= 100
	if n != 0 {
		if n < 20 {
			b = append(b, enZeroTillTwentyCardinal[n]...)
		} else {
			b = append(b, enDecFactorCardinal[n/10]...)
			b = append(b, enSingleCardinal[n%10]...)
		}
	}
	return append(b, suffix...)
}

// EnglishCardinal returns n in English words, written as one word
// ("onehundredtwentythree"). Zero, and any multiple of 1000000000, yields the
// empty string. Negative numbers crash the original (see ErrCrash).
func EnglishCardinal(n int32) (string, error) {
	if n < 0 {
		return "", ErrCrash
	}
	var b []byte
	n %= 1000000000
	if n >= 1000000 {
		b = enHalf(b, n/1000000, "million")
		n %= 1000000
	}
	b = enHalf(b, n/1000, "thousand")
	b = enHalf(b, n%1000, "")
	return string(b), nil
}

// EnglishOrdinal returns n as an English ordinal, derived from the cardinal:
// a final "y" becomes "ieth", a final single-digit word its ordinal, a final
// "t" gets "h" and anything else "th". Below 1 it returns OrdinalError.
func EnglishOrdinal(n int32) (string, error) {
	if n < 1 {
		return OrdinalError, nil
	}
	s, _ := EnglishCardinal(n) // cannot fail for n >= 1
	// For an empty cardinal the original inspects the byte before its static
	// buffer, assumed NUL: neither "y" nor "t", so "th" is appended.
	if s != "" && s[len(s)-1] == 'y' {
		return s[:len(s)-1] + "ieth", nil
	}
	for i := 1; i < 10; i++ {
		if hasSuffixFold(s, enSingleCardinal[i]) {
			return s[:len(s)-len(enSingleCardinal[i])] + enSingleOrdinal[i], nil
		}
	}
	if s != "" && s[len(s)-1] == 't' {
		return s + "h", nil
	}
	return s + "th", nil
}

// ParseEnglishCardinal parses an English cardinal at s[pos:]. It reports the
// value, the position the original leaves its cursor at (which moves even
// when parsing fails), and whether a number was found. "zero" is recognised
// only before any whitespace; any other text yielding 0 is a failure.
func ParseEnglishCardinal(s string, pos int) (value int32, next int, ok bool) {
	c := cursor{s, pos}
	if c.at("zero") {
		c.p += 4
		c.skipws()
		return 0, c.p, true
	}
	var result, total int32
	c.skipws()
	for c.more() {
		c.skipws()
		for i := 1; i < 10; i++ {
			if c.at(enHundredCardinals[i]) {
				result += int32(i * 100)
				c.p += len(enHundredCardinals[i])
				c.skipword("and")
				break
			}
		}
		found := false
		for i := 2; i < 10; i++ {
			if c.at(enDecFactorCardinal[i]) {
				result += int32(i * 10)
				c.p += len(enDecFactorCardinal[i])
				c.skipws()
				found = true
				break
			}
		}
		if found {
			for i := 1; i < 10; i++ {
				if c.at(enSingleCardinal[i]) {
					result += int32(i)
					c.p += len(enSingleCardinal[i])
					c.skipws()
					break
				}
			}
		} else {
			for i := 19; i >= 1; i-- {
				if c.at(enZeroTillTwentyCardinal[i]) {
					result += int32(i)
					c.p += len(enZeroTillTwentyCardinal[i])
					c.skipws()
					break
				}
			}
		}
		if c.at("million") {
			total += result * 1000000
			c.p += 7
			c.skipword("and")
		} else if c.at("thousand") {
			total += result * 1000
			c.p += 8
			c.skipword("and")
		} else if c.at("hundred") {
			total += result * 100
			c.p += 7
			c.skipword("and")
		} else {
			total += result
			c.skipws()
			break
		}
		result = 0
	}
	return total, c.p, total != 0
}

// ParseEnglishOrdinal parses an English ordinal ("first", "eleventh",
// "twentysecond", "one hundred and first", but also plain cardinals such as
// "eight") at s[pos:]. It returns the value, 0 meaning none, and the position
// the original leaves its cursor at, which moves even when parsing fails.
func ParseEnglishOrdinal(s string, pos int) (value int32, next int) {
	c := cursor{s, pos}
	var result, total int32
	c.skipws()
	for c.more() {
		c.skipws()
		for i := 1; i < 10; i++ {
			if c.at(enHundredCardinals[i]) {
				result += int32(i * 100)
				c.p += len(enHundredCardinals[i])
				c.skipword("and")
				break
			}
		}
		found := false
		for i := 2; i < 10; i++ {
			if c.at(enDecFactorCardinal[i]) {
				result += int32(i * 10)
				c.p += len(enDecFactorCardinal[i])
				c.skipws()
				found = true
				break
			}
		}
		if !found {
			for i := 2; i < 10; i++ {
				if c.at(enDecFactorOrdinal[i]) {
					result += int32(i * 10)
					c.p += len(enDecFactorOrdinal[i])
					c.skipws()
					found = true
					break
				}
			}
		}
		if found {
			found = false
			for i := 1; i < 10; i++ {
				if c.at(enSingleCardinal[i]) {
					result += int32(i)
					c.p += len(enSingleCardinal[i])
					c.skipws()
					found = true
					break
				}
			}
			if !found {
				for i := 1; i < 10; i++ {
					if c.at(enSingleOrdinal[i]) {
						result += int32(i)
						c.p += len(enSingleOrdinal[i])
						c.skipws()
						break
					}
				}
			}
		} else {
			for i := 19; i >= 1; i-- {
				if c.at(enZeroTillTwentyCardinal[i]) {
					result += int32(i)
					c.p += len(enZeroTillTwentyCardinal[i])
					c.skipws()
					found = true
					break
				}
			}
			if !found {
				for i := 19; i >= 1; i-- {
					if c.at(enZeroTillTwentyOrdinal[i]) {
						result += int32(i)
						c.p += len(enZeroTillTwentyOrdinal[i])
						c.skipws()
						break
					}
				}
			}
		}
		if c.at("million") {
			total += result * 1000000
			c.p += 7
			c.skipword("and")
			c.enSkipth()
		} else if c.at("thousand") {
			total += result * 1000
			c.p += 8
			c.skipword("and")
			c.enSkipth()
		} else if c.at("hundred") {
			total += result * 100
			c.p += 7
			c.skipword("and")
			c.enSkipth()
		} else {
			c.enSkipth()
			total += result
			c.skipws()
			break
		}
		result = 0
	}
	return total, c.p
}
