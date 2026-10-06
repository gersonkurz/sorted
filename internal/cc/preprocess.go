package cc

import (
	"fmt"
	"slices"
	"strings"
)

// macro is a #define: object-like (NAME body) or function-like
// (NAME(params) body).
type macro struct {
	funcLike bool
	params   []string
	body     []Token
}

// Preprocess runs the directives in toks and expands macros, as a much
// smaller cousin of chibicc's preprocess.c:
//
//   - #include <stdio.h> is allowed and ignored, so that a program for
//     Sorted! is also a C program that declares putchar;
//   - #define defines object-like and function-like macros, and #undef
//     removes one; a later #define replaces an earlier one;
//   - a line holding only # is a null directive;
//   - everything else (#if, #include "file", # and ## in macros, ...) is not
//     part of the subset.
//
// Expansion rescans the result, and a macro is not expanded again inside its
// own expansion: each token carries a hide set, the macros it came from, as
// in Prosser's algorithm, which chibicc uses too. Arguments
// are expanded when they are substituted. Expanded tokens take the position
// of the macro's name where it is used.
func Preprocess(toks []Token) (out []Token, err error) {
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			out, err = nil, e
		}
	}()
	pp := &preprocessor{macros: map[string]*macro{}}
	return pp.expand(toks), nil
}

type preprocessor struct {
	macros map[string]*macro
}

func ppFail(p Pos, format string, args ...any) {
	panic(&Error{p, fmt.Sprintf(format, args...)})
}

func ppUnsupported(p Pos, what string) {
	ppFail(p, "not supported in Sorted! (yet): %s", what)
}

// expand expands the macros in toks and runs the directives among them
// (arguments rejects directives inside macro arguments).
func (pp *preprocessor) expand(toks []Token) []Token {
	var out []Token
	for len(toks) > 0 {
		t := toks[0]
		if t.Kind == TkPunct && t.Text == "#" {
			if !t.bol {
				ppFail(t.Pos, "stray '#' (a preprocessing line must start with it)")
			}
			n := 1
			for n < len(toks) && toks[n].Kind != TkEOF && toks[n].line == t.line {
				n++
			}
			pp.directive(t, toks[1:n])
			toks = toks[n:]
			continue
		}
		m := pp.macros[t.Text]
		if t.Kind != TkIdent || m == nil || slices.Contains(t.hide, t.Text) {
			out = append(out, t)
			toks = toks[1:]
			continue
		}
		// Every token of the expansion gets the hide set hs added: for an
		// object-like macro, the name's own set plus the name; for a
		// function-like one, only what both the name and the closing
		// parenthesis carry, plus the name (Prosser's algorithm, as in
		// chibicc). Arguments are expanded when they are substituted, so an
		// argument the body does not use is never expanded.
		place := func(hs []string, args [][]Token) []Token {
			expanded := make([][]Token, len(args))
			var r []Token
			for _, b := range m.body {
				part := []Token{b}
				if i := slices.Index(m.params, b.Text); i >= 0 && b.Kind == TkIdent {
					if expanded[i] == nil {
						expanded[i] = append([]Token{}, pp.expand(args[i])...)
					}
					part = expanded[i]
				}
				for _, a := range part {
					a.Pos, a.bol = t.Pos, false
					a.hide = union(a.hide, hs)
					r = append(r, a)
				}
			}
			return r
		}
		if !m.funcLike {
			toks = append(place(union(t.hide, []string{t.Text}), nil), toks[1:]...)
			continue
		}
		if len(toks) < 2 || toks[1].Kind != TkPunct || toks[1].Text != "(" {
			out = append(out, t) // a function-like macro's name without arguments
			toks = toks[1:]
			continue
		}
		args, rparen, rest := pp.arguments(t, toks[2:])
		if len(m.params) == 0 && len(args) == 1 && len(args[0]) == 0 {
			args = nil
		}
		if len(args) != len(m.params) {
			ppFail(t.Pos, "macro '%s' takes %d argument(s), not %d", t.Text, len(m.params), len(args))
		}
		var hs []string
		for _, h := range t.hide {
			if slices.Contains(rparen.hide, h) {
				hs = append(hs, h)
			}
		}
		toks = append(place(union(hs, []string{t.Text}), args), rest...)
	}
	return out
}

// union returns the hide set a plus the names in b it lacks.
func union(a, b []string) []string {
	r := slices.Clone(a)
	for _, h := range b {
		if !slices.Contains(r, h) {
			r = append(r, h)
		}
	}
	return r
}

// arguments splits the tokens after a function-like macro's "(" into its
// arguments, at the commas outside nested parentheses, and returns the
// closing ")" and the tokens after it.
func (pp *preprocessor) arguments(name Token, toks []Token) ([][]Token, Token, []Token) {
	args := [][]Token{nil}
	depth := 0
	for i, t := range toks {
		if t.Kind == TkEOF {
			break
		}
		if t.Kind == TkPunct && t.Text == "#" && t.bol {
			ppUnsupported(t.Pos, "directives inside the arguments of a macro")
		}
		if t.Kind == TkPunct {
			switch t.Text {
			case "(":
				depth++
			case ")":
				if depth == 0 {
					return args, t, toks[i+1:]
				}
				depth--
			case ",":
				if depth == 0 {
					args = append(args, nil)
					continue
				}
			}
		}
		args[len(args)-1] = append(args[len(args)-1], t)
	}
	ppFail(name.Pos, "unterminated call of macro '%s'", name.Text)
	return nil, Token{}, nil
}

// directive runs the directive "# line...".
func (pp *preprocessor) directive(hash Token, line []Token) {
	if len(line) == 0 {
		return // the null directive
	}
	name := line[0]
	switch name.Text {
	case "include":
		var text []string
		for _, t := range line[1:] {
			text = append(text, t.Text)
		}
		if strings.Join(text, "") != "<stdio.h>" {
			ppUnsupported(hash.Pos, "#include other than <stdio.h>")
		}
	case "define":
		if len(line) < 2 || line[1].Kind != TkIdent {
			ppFail(name.Pos, "#define needs a macro name (an identifier)")
		}
		id := line[1]
		m := &macro{}
		body := line[2:]
		// NAME( with nothing between, not even a comment, is a
		// function-like macro.
		if len(body) > 0 && body[0].Text == "(" && body[0].Kind == TkPunct && !body[0].space {
			m.funcLike = true
			body = pp.params(id, m, body[1:])
		}
		for _, t := range body {
			if t.Kind == TkPunct && (t.Text == "#" || t.Text == "##") {
				ppUnsupported(t.Pos, "'"+t.Text+"' in macros")
			}
		}
		m.body = body
		pp.macros[id.Text] = m
	case "undef":
		if len(line) != 2 || line[1].Kind != TkIdent {
			ppFail(name.Pos, "#undef needs exactly one macro name")
		}
		delete(pp.macros, line[1].Text)
	default:
		ppUnsupported(hash.Pos, "#"+name.Text+" (the preprocessor knows #define, #undef and #include <stdio.h>)")
	}
}

// params reads "a, b)" of a function-like macro and returns the body after it.
func (pp *preprocessor) params(id Token, m *macro, toks []Token) []Token {
	if len(toks) > 0 && toks[0].Text == ")" {
		return toks[1:]
	}
	for i := 0; i < len(toks); i += 2 {
		p := toks[i]
		if p.Text == "..." {
			ppUnsupported(p.Pos, "variadic macros")
		}
		if p.Kind != TkIdent {
			ppFail(p.Pos, "expected a parameter name in macro '%s'", id.Text)
		}
		if slices.Contains(m.params, p.Text) {
			ppFail(p.Pos, "duplicate parameter '%s' in macro '%s'", p.Text, id.Text)
		}
		m.params = append(m.params, p.Text)
		if i+1 < len(toks) && toks[i+1].Text == ")" {
			return toks[i+2:]
		}
		if i+1 >= len(toks) || toks[i+1].Text != "," {
			break
		}
	}
	ppFail(id.Pos, "expected ')' after the parameters of macro '%s'", id.Text)
	return nil
}
