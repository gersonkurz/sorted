// Package numbers converts between integers and English, German, Italian,
// Vaudois French, Brazilian Portuguese, Japanese or Chinese number words,
// as Sorted! does when it parses number declarations and ordinal
// references, and when it writes a value as a cardinal or ordinal. Italian,
// Vaudois French, Brazilian Portuguese, Japanese and Chinese (italian.go,
// vaudois.go, brazilian.go, japanese.go, chinese.go, Very Very Sorted!) are
// new and have no quirks.
//
// English and German are a line-by-line port of EnglishNumbers.cpp and
// GermanNumbers.cpp from legacy/sorted.win32, quirks included: the
// misspellings in the word tables ("fiveteen", "fourty", "nineth",
// "twelveth", ...), prefix matching without word boundaries, and cursors
// that move even when parsing fails.
//
// Undefined behaviour of the C code is not emulated (Gerson's ruling): it gets
// a simple, documented result instead. Formatting a negative cardinal indexes
// the word tables with negative subscripts, so it is reported as ErrCrash; the
// few reads before a static result buffer are taken to find NUL bytes.
//
// Parsers take the text and a byte offset and return the offset where the
// original left its cursor. Like the C code they treat the end of the text as
// a NUL terminator and skip only spaces and tabs.
package numbers

import "errors"

// ErrCrash is the port-defined result of formatting a negative cardinal. The
// original indexes its word tables with negative subscripts there, which is
// undefined behaviour and, by the maintainer's ruling, not emulated.
var ErrCrash = errors.New("negative cardinal (undefined behaviour in the original)")

// OrdinalError is what the original prints as the ordinal of a number below 1.
const OrdinalError = "ERROR, ORDINALS ARE POSITIVE INTEGERS"

// cursor walks the text like the C code's char pointer: reading past the end
// yields NUL.
type cursor struct {
	s string
	p int
}

func (c *cursor) byteAt(k int) byte {
	if c.p+k < len(c.s) {
		return c.s[c.p+k]
	}
	return 0
}

// more reports whether the cursor is not at the terminating NUL (while(*p)).
func (c *cursor) more() bool { return c.p < len(c.s) }

// at reports whether the text at the cursor starts with w, ignoring ASCII case
// (!strnicmp(p, w, strlen(w))).
func (c *cursor) at(w string) bool {
	if len(c.s)-c.p < len(w) {
		return false
	}
	return equalFold(c.s[c.p:c.p+len(w)], w)
}

// rest reports whether the whole remaining text equals w, ignoring ASCII case
// (!stricmp(p, w)).
func (c *cursor) rest(w string) bool { return equalFold(c.s[c.p:], w) }

// skipws skips spaces and tabs only; the parser's own whitespace notion is
// wider.
func (c *cursor) skipws() {
	for c.byteAt(0) == ' ' || c.byteAt(0) == '\t' {
		c.p++
	}
}

// skipword skips whitespace and then the connector word ("and"/"und"), which
// is matched as a prefix without a word boundary.
func (c *cursor) skipword(w string) {
	c.skipws()
	if c.at(w) {
		c.p += len(w)
		c.skipws()
	}
}

// isBlank reports whether b is one of the two bytes skipws skips.
func isBlank(b byte) bool { return b == ' ' || b == '\t' }

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if lower(a[i]) != lower(b[i]) {
			return false
		}
	}
	return true
}

func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + 'a' - 'A'
	}
	return b
}

// hasSuffixFold reports whether s ends with w, ignoring ASCII case. When w is
// longer than s, the original compares from before the start of its static
// result buffer; those bytes are assumed to be NUL, which never matches.
func hasSuffixFold(s, w string) bool {
	return len(w) <= len(s) && equalFold(s[len(s)-len(w):], w)
}
