package cc

import (
	"fmt"
	"strconv"
	"strings"
)

// TokenKind says what a token is.
type TokenKind int

// Token kinds, as in chibicc.
const (
	TkIdent   TokenKind = iota // identifiers
	TkPunct                    // punctuators
	TkKeyword                  // keywords
	TkNum                      // numeric literals
	TkStr                      // string literals
	TkEOF                      // end of input
)

// Token is a lexical token.
type Token struct {
	Kind TokenKind
	Text string
	Val  int32 // TkNum (an unsigned value as its bit pattern)
	// Unsigned marks an unsigned int literal: suffix u, or a hexadecimal or
	// octal literal too big for int.
	Unsigned bool
	Str      []byte // TkStr: the characters, escapes resolved, without the NUL
	Pos      Pos

	bol   bool     // first token on its line (a '#' there starts a directive)
	space bool     // whitespace or a comment comes before it
	line  int      // logical line: a newline inside a comment does not end one
	hide  []string // macros not to expand again in this token (see Preprocess)
}

// keywords are all keywords of C17, the dialect clang compiles by default.
// Those outside the subset are tokenized as keywords too, so that using one
// gives a clear message instead of making it a variable name. (C23's bool,
// true, false and friends are identifiers in C17.)
var keywords = map[string]bool{}

func init() {
	for _, k := range strings.Fields(`auto break case char const continue default do double else enum
		extern float for goto if inline int long register restrict return short signed sizeof
		static struct switch typedef union unsigned void volatile while _Alignas _Alignof
		_Atomic _Bool _Complex _Generic _Imaginary _Noreturn _Static_assert _Thread_local`) {
		keywords[k] = true
	}
}

// punctuators, longest first. Those beyond subset 1 are recognised so the
// parser can name them.
var puncts = []string{
	"<<=", ">>=", "...",
	"##", "==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=", "/=", "%=",
	"&=", "|=", "^=", "<<", ">>", "->",
}

// Tokenize splits C source into tokens, ending with a TkEOF token.
func Tokenize(src string) ([]Token, error) {
	var toks []Token
	line, col := 1, 1
	bol := true  // nothing but whitespace and comments so far on this line
	logical := 1 // lines, not counting newlines inside comments
	advance := func(n int) {
		for _, c := range src[:n] {
			if c == '\n' {
				line, col = line+1, 1
				bol = true
			} else {
				col++
			}
		}
		src = src[n:]
	}
	space := false
	add := func(t Token) {
		t.bol, t.line, t.space = bol, logical, space
		toks = append(toks, t)
		bol, space = false, false
	}
	for len(src) > 0 {
		pos := Pos{line, col}
		c := src[0]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f':
			if c == '\n' {
				logical++
			}
			advance(1)
			space = true
		case strings.HasPrefix(src, "//"):
			n := strings.IndexByte(src, '\n')
			if n < 0 {
				n = len(src)
			}
			advance(n)
			space = true
		case strings.HasPrefix(src, "/*"):
			n := strings.Index(src[2:], "*/")
			if n < 0 {
				return nil, errorAt(pos, "unclosed block comment")
			}
			advance(n + 4)
			space = true
		case isDigit(c):
			n := 1
			for n < len(src) && isIdent2(src[n]) {
				n++
			}
			v, unsigned, err := parseNumber(src[:n])
			if err != nil {
				return nil, errorAt(pos, "%v", err)
			}
			add(Token{Kind: TkNum, Text: src[:n], Val: v, Unsigned: unsigned, Pos: pos})
			advance(n)
		case isIdent1(c):
			n := 1
			for n < len(src) && isIdent2(src[n]) {
				n++
			}
			kind := TkIdent
			if keywords[src[:n]] {
				kind = TkKeyword
			}
			add(Token{Kind: kind, Text: src[:n], Pos: pos})
			advance(n)
		case c == '"':
			str, n, err := readStringLiteral(src, pos)
			if err != nil {
				return nil, err
			}
			add(Token{Kind: TkStr, Text: src[:n], Str: str, Pos: pos})
			advance(n)
		case c == '\'':
			v, n, err := readCharLiteral(src, pos)
			if err != nil {
				return nil, err
			}
			add(Token{Kind: TkNum, Text: src[:n], Val: v, Pos: pos})
			advance(n)
		default:
			n := 0
			for _, p := range puncts {
				if strings.HasPrefix(src, p) {
					n = len(p)
					break
				}
			}
			if n == 0 && strings.IndexByte("+-*/%=<>!&|^~?:;,.(){}[]#", c) >= 0 {
				n = 1
			}
			if n == 0 {
				return nil, errorAt(pos, "invalid token")
			}
			add(Token{Kind: TkPunct, Text: src[:n], Pos: pos})
			advance(n)
		}
	}
	return append(toks, Token{Kind: TkEOF, Pos: Pos{line, col}}), nil
}

// readCharLiteral reads a character constant such as 'a' or '\n' and returns
// its value and length. Its type is int, its value that of the char, which is
// signed: '\xff' is -1. Multi-character constants and non-ASCII characters
// are not part of the subset.
func readCharLiteral(src string, pos Pos) (int32, int, error) {
	p := 1
	if p >= len(src) || src[p] == '\n' {
		return 0, 0, errorAt(pos, "unclosed char literal")
	}
	var c int
	if src[p] == '\\' {
		var err error
		c, p, err = readEscapedChar(src, p+1, pos)
		if err != nil {
			return 0, 0, err
		}
	} else {
		if src[p] >= 0x80 {
			return 0, 0, errorAt(pos, "not supported in Sorted! (yet): non-ASCII characters")
		}
		if src[p] == '\'' {
			return 0, 0, errorAt(pos, "empty char literal")
		}
		c, p = int(src[p]), p+1
	}
	if p >= len(src) || src[p] != '\'' {
		if p < len(src) && src[p] != '\n' && strings.IndexByte(src[p:], '\'') > 0 {
			return 0, 0, errorAt(pos, "not supported in Sorted! (yet): multi-character constants")
		}
		return 0, 0, errorAt(pos, "unclosed char literal")
	}
	return int32(int8(c)), p + 1, nil
}

// readStringLiteral reads a string literal and returns its characters
// (escapes resolved, no terminating NUL) and its length in the source.
func readStringLiteral(src string, pos Pos) ([]byte, int, error) {
	var str []byte
	p := 1
	for {
		if p >= len(src) || src[p] == '\n' {
			return nil, 0, errorAt(pos, "unclosed string literal")
		}
		switch c := src[p]; {
		case c == '"':
			return str, p + 1, nil
		case c == '\\':
			v, next, err := readEscapedChar(src, p+1, pos)
			if err != nil {
				return nil, 0, err
			}
			str, p = append(str, byte(v)), next
		case c >= 0x80:
			return nil, 0, errorAt(pos, "not supported in Sorted! (yet): non-ASCII characters")
		default:
			str, p = append(str, c), p+1
		}
	}
}

// readEscapedChar reads the escape sequence after a backslash at src[p], as
// chibicc's read_escaped_char does: up to three octal digits, \x and hex
// digits, or a letter.
func readEscapedChar(src string, p int, pos Pos) (int, int, error) {
	at := func(i int) byte {
		if i < len(src) {
			return src[i]
		}
		return 0
	}
	isOct := func(b byte) bool { return '0' <= b && b <= '7' }
	switch {
	case isOct(at(p)):
		c := 0
		for n := 0; n < 3 && isOct(at(p)); n++ {
			c = c<<3 + int(at(p)-'0')
			p++
		}
		if c > 0xff {
			return 0, 0, errorAt(pos, "octal escape sequence out of range")
		}
		return c, p, nil
	case at(p) == 'x':
		p++
		start := p
		c := 0
		for strings.IndexByte("0123456789abcdefABCDEF", at(p)) >= 0 && at(p) != 0 {
			d, _ := strconv.ParseInt(string(at(p)), 16, 32)
			c = c<<4 + int(d)
			if c > 0xff {
				return 0, 0, errorAt(pos, "hex escape sequence out of range")
			}
			p++
		}
		if p == start {
			return 0, 0, errorAt(pos, "invalid hex escape sequence")
		}
		return c, p, nil
	}
	escapes := map[byte]int{'a': 7, 'b': 8, 't': 9, 'n': 10, 'v': 11, 'f': 12, 'r': 13, 'e': 27}
	if c, ok := escapes[at(p)]; ok {
		return c, p + 1, nil
	}
	if at(p) == 0 || at(p) == '\n' {
		return 0, 0, errorAt(pos, "unclosed char literal")
	}
	// \' \" \? \\ and, like chibicc and clang (with a warning), any other
	// character stand for themselves.
	return int(at(p)), p + 1, nil
}

// parseNumber reads a decimal, hexadecimal (0x) or octal (leading 0)
// literal of type int or unsigned int, as C17 types it: a suffix u makes it
// unsigned, and a hexadecimal or octal literal too big for int is unsigned
// too. Anything bigger, or a decimal literal beyond int (a long), is not
// part of the subset.
func parseNumber(s string) (int32, bool, error) {
	unsigned := false
	if strings.HasSuffix(s, "u") || strings.HasSuffix(s, "U") {
		unsigned, s = true, s[:len(s)-1]
	}
	base, digits := 10, s
	switch {
	case len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X"):
		base, digits = 16, s[2:]
	case len(s) > 1 && s[0] == '0':
		base, digits = 8, s[1:]
	}
	v, err := strconv.ParseUint(digits, base, 64)
	switch {
	case err != nil && err.(*strconv.NumError).Err != strconv.ErrRange:
		return 0, false, fmt.Errorf("invalid integer literal: %s", s)
	case err != nil || v > 4294967295 || v > 2147483647 && base == 10 && !unsigned:
		return 0, false, fmt.Errorf("integer literal out of range: %s", s)
	}
	return int32(uint32(v)), unsigned || v > 2147483647, nil
}

func isDigit(c byte) bool  { return '0' <= c && c <= '9' }
func isIdent1(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_' }
func isIdent2(c byte) bool { return isIdent1(c) || isDigit(c) }
