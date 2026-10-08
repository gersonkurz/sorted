package syntax

import (
	"bytes"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Filter prepares raw source the way ReadSortedSourcecode does before parsing:
// every byte outside [A-Za-z.,] becomes a space.
//
// It also reproduces how the original reads the file, with fopen(..., "rt")
// and fgets(line, 1024, ...) under the Win32 C runtime: text mode turns CRLF
// into LF, a Ctrl-Z (0x1A) ends the file, and a NUL byte ends the line buffer,
// dropping the rest of that fgets chunk (up to and including its newline, or
// 1023 bytes in all).
func Filter(raw []byte) string {
	const chunk = 1023 // fgets(line, 1024, fp) reads at most 1023 bytes
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	out := make([]byte, 0, len(raw))
	for len(raw) > 0 {
		// One fgets call: up to and including '\n', at most chunk bytes,
		// and nothing at or after a Ctrl-Z.
		n := 0
		for n < len(raw) && n < chunk {
			if raw[n] == 0x1A {
				break
			}
			n++
			if raw[n-1] == '\n' {
				break
			}
		}
		line := raw[:n]
		eof := n < len(raw) && raw[n] == 0x1A
		for _, b := range line {
			if b == 0 {
				break
			}
			if isValidChar(b) {
				out = append(out, b)
			} else {
				out = append(out, ' ')
			}
		}
		if eof || n == 0 {
			break
		}
		raw = raw[n:]
	}
	return string(out)
}

// FilterVery prepares raw source for Very Sorted! (#28), which reads its text
// as UTF-8: every letter (and combining mark) of any script is kept, so are
// '.' and ',', and everything else becomes a space, as do bytes that are not
// UTF-8. The text is normalised first, so that keywords match ignoring case
// and encoding: NFC (an "ü" may be one code point or "u" and a combining
// diaeresis), full Unicode case folding ("STRASSE", "Straße" and "strasse"
// meet as "strasse"), and the German umlauts written out the way the 2000
// word tables spell them (ä ö ü as ae oe ue), so that "fünf", "zwölf",
// "dreißig" and "Verhältnis" read as "fuenf", "zwoelf", "dreissig" and
// "verhaeltnis", and the ASCII spellings still work.
//
// The Win32 text-mode reading that Filter reproduces belongs to 2000: here a
// Ctrl-Z or a NUL is just another character that is not a letter.
func FilterVery(raw []byte) string {
	s := norm.NFC.String(cases.Fold().String(norm.NFC.String(string(raw))))
	s = umlauts.Replace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '.' || r == ',', unicode.IsLetter(r), unicode.Is(unicode.M, r):
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	return b.String()
}

var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue")

// FilterVeryVery prepares raw source for Very Very Sorted! (#30): as
// FilterVery, and then without accents, so that "è", "può" and "ventitré"
// read as "e", "puo" and "ventitre", the way its keywords are spelled
// (written with or without accents, they mean the same). The German
// umlauts are written out before (FilterVery), so "fünf" is still "fuenf".
func FilterVeryVery(raw []byte) string { return Unaccent(FilterVery(raw)) }

// Unaccent strips accents: decompose (NFD), drop the combining marks,
// recompose (NFC). The voicing marks of kana (U+3099, U+309A) are not
// accents and stay: ド is not ト.
func Unaccent(s string) string {
	t, _, _ := transform.String(unaccent, s)
	return t
}

var unaccent = transform.Chain(norm.NFD, runes.Remove(runes.Predicate(isAccent)), norm.NFC)

func isAccent(r rune) bool { return unicode.Is(unicode.Mn, r) && r != '\u3099' && r != '\u309a' }

func isChar(b byte) bool { return 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' }

func isValidChar(b byte) bool { return isChar(b) || b == '.' || b == ',' }

// isLetter reports whether the parser takes b as part of a word: an ASCII
// letter, or a byte of a UTF-8 letter that FilterVery kept. Filter leaves no
// byte above 0x7F, so for 2000 text this is isChar.
func isLetter(b byte) bool { return isChar(b) || b >= 0x80 }

// isText reports whether the parser stops skipping at b (isValidChar).
func isText(b byte) bool { return isLetter(b) || b == '.' || b == ',' }
