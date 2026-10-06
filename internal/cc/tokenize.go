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
	TkEOF                      // end of input
)

// Token is a lexical token.
type Token struct {
	Kind TokenKind
	Text string
	Val  int32 // TkNum
	Pos  Pos
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
	"==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=", "/=", "%=",
	"&=", "|=", "^=", "<<", ">>", "->",
}

// Tokenize splits C source into tokens, ending with a TkEOF token.
func Tokenize(src string) ([]Token, error) {
	var toks []Token
	line, col := 1, 1
	bol := true // nothing but whitespace and comments so far on this line
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
	add := func(t Token) {
		toks = append(toks, t)
		bol = false
	}
	for len(src) > 0 {
		pos := Pos{line, col}
		c := src[0]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f':
			advance(1)
		case strings.HasPrefix(src, "//"):
			n := strings.IndexByte(src, '\n')
			if n < 0 {
				n = len(src)
			}
			advance(n)
		case strings.HasPrefix(src, "/*"):
			n := strings.Index(src[2:], "*/")
			if n < 0 {
				return nil, errorAt(pos, "unclosed block comment")
			}
			advance(n + 4)
		case c == '#' && !bol:
			return nil, errorAt(pos, "stray '#' (a preprocessing line must start with it)")
		case c == '#':
			// "#include <stdio.h>" is allowed and ignored, so that a program
			// for Sorted! is also a C program that declares putchar.
			n := strings.IndexByte(src, '\n')
			if n < 0 {
				n = len(src)
			}
			if strings.Join(strings.Fields(strings.Replace(src[:n], "include", " include ", 1)), " ") != "# include <stdio.h>" {
				return nil, errorAt(pos, "not supported in Sorted! (yet): the preprocessor (except #include <stdio.h>)")
			}
			advance(n)
		case isDigit(c):
			n := 1
			for n < len(src) && isIdent2(src[n]) {
				n++
			}
			v, err := parseNumber(src[:n])
			if err != nil {
				return nil, errorAt(pos, "%v", err)
			}
			add(Token{TkNum, src[:n], v, pos})
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
			add(Token{kind, src[:n], 0, pos})
			advance(n)
		case c == '\'' || c == '"':
			return nil, errorAt(pos, "not supported in Sorted! (yet): character and string literals")
		default:
			n := 0
			for _, p := range puncts {
				if strings.HasPrefix(src, p) {
					n = len(p)
					break
				}
			}
			if n == 0 && strings.IndexByte("+-*/%=<>!&|^~?:;,.(){}[]", c) >= 0 {
				n = 1
			}
			if n == 0 {
				return nil, errorAt(pos, "invalid token")
			}
			add(Token{TkPunct, src[:n], 0, pos})
			advance(n)
		}
	}
	return append(toks, Token{TkEOF, "", 0, Pos{line, col}}), nil
}

// parseNumber reads a decimal, hexadecimal (0x) or octal (leading 0) int
// literal. Suffixes and values beyond int are not part of the subset.
func parseNumber(s string) (int32, error) {
	base, digits := 10, s
	switch {
	case len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X"):
		base, digits = 16, s[2:]
	case len(s) > 1 && s[0] == '0':
		base, digits = 8, s[1:]
	}
	v, err := strconv.ParseInt(digits, base, 64)
	switch {
	case err != nil && err.(*strconv.NumError).Err == strconv.ErrRange || err == nil && v > 2147483647:
		return 0, fmt.Errorf("integer literal out of range: %s", s)
	case err != nil:
		return 0, fmt.Errorf("invalid integer literal: %s", s)
	}
	return int32(v), nil
}

func isDigit(c byte) bool  { return '0' <= c && c <= '9' }
func isIdent1(c byte) bool { return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '_' }
func isIdent2(c byte) bool { return isIdent1(c) || isDigit(c) }
