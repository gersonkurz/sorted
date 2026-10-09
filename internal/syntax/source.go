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
func FilterVery(raw []byte) string { return filterUTF8(raw, false) }

// filterUTF8 is FilterVery; with hyphens, a hyphen between two letters stays
// too, as part of the word ("vingt-et-un").
func filterUTF8(raw []byte, hyphens bool) string {
	s := norm.NFC.String(cases.Fold().String(norm.NFC.String(string(raw))))
	rs := []rune(umlauts.Replace(s))
	letter := func(i int) bool {
		return i >= 0 && i < len(rs) && (unicode.IsLetter(rs[i]) || unicode.Is(unicode.M, rs[i]))
	}
	var b strings.Builder
	for i, r := range rs {
		switch {
		case r == '.' || r == ',', letter(i), hyphens && r == '-' && letter(i-1) && letter(i+1):
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
// Two things serve its French (#29): a hyphen between letters stays part of
// the word, so a number is one word ("deux-cent-vingt-et-un"), and the
// fillers French loves, "eh", "hein", "quoi" and "voilà", are dropped with
// the commas around them ("Ce programme, eh, utilise", "…, voilà, quoi."),
// wherever they stand: they carry no meaning. Three serve its Mandarin
// (#33): the full-width 。，、 are a period and commas, traditional
// characters (and Taiwan's 程式) read as the simplified ones the keywords
// are spelled in, and a period or comma gets a space after it before a
// letter outside ASCII, so a keyword ends there ("一,二" is "一, 二").
func FilterVeryVery(raw []byte) string {
	s := filterUTF8([]byte(hanPunctuation.Replace(string(raw))), true)
	return dropFillers(spaceAfterPunctuation(simplified.Replace(Unaccent(s))))
}

// hanPunctuation is the full-width punctuation Mandarin writes.
var hanPunctuation = strings.NewReplacer("。", ". ", "，", ", ", "、", ", ")

// simplified turns the traditional characters of Mandarin's vocabulary
// into the simplified ones (#33). 語 stays, as Japanese writes it (ドイツ語);
// the Mandarin format names read both 语 and 語 (zhFormats).
var simplified = strings.NewReplacer(
	"程式", "程序", "這", "这", "個", "个", "數", "数", "總", "总", "並", "并", "為", "为", "時", "时", "兒", "儿",
	"寫", "写", "讀", "读", "單", "单", "積", "积", "於", "于", "條", "条", "標", "标", "籤", "签", "賦", "赋",
	"給", "给", "實", "实", "現", "现", "邏", "逻", "輯", "辑", "運", "运", "轉", "转", "輸", "输",
	"萬", "万", "億", "亿", "兩", "两", "負", "负",
)

// spaceAfterPunctuation puts a space after a period or comma that a byte
// above 0x7F follows.
func spaceAfterPunctuation(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteByte(s[i])
		if (s[i] == '.' || s[i] == ',') && i+1 < len(s) && s[i+1] >= 0x80 {
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// fillers are the words Very Very Sorted! reads past (FilterVeryVery).
var fillers = map[string]bool{"eh": true, "hein": true, "quoi": true, "voila": true}

// dropFillers removes the fillers from filtered text, each with the commas
// (and spaces) directly before and after it, leaving one space. The rest of
// the text stays as it is.
func dropFillers(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if !isLetter(s[i]) {
			out = append(out, s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && isLetter(s[j]) {
			j++
		}
		if !fillers[s[i:j]] {
			out = append(out, s[i:j]...)
			i = j
			continue
		}
		for len(out) > 0 && (out[len(out)-1] == ' ' || out[len(out)-1] == ',') {
			out = out[:len(out)-1]
		}
		for j < len(s) && (s[j] == ' ' || s[j] == ',') {
			j++
		}
		out = append(out, ' ')
		i = j
	}
	return string(out)
}

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
// letter, a byte of a UTF-8 letter that FilterVery kept, or a hyphen that
// FilterVeryVery kept inside a word. Filter leaves no byte above 0x7F and
// no hyphen, so for 2000 text this is isChar.
func isLetter(b byte) bool { return isChar(b) || b >= 0x80 || b == '-' }

// isText reports whether the parser stops skipping at b (isValidChar).
func isText(b byte) bool { return isLetter(b) || b == '.' || b == ',' }
