package emit

import (
	"fmt"
	"slices"
	"strings"

	"github.com/gersonkurz/sorted/internal/syntax"
)

// Exact renders p as a C program that behaves exactly like the interpreter
// (internal/interp) running p: the same output, the same input handling, the
// same runtime errors (on stderr, exit status 1). It keeps the interpreter's
// semantics where the original's own translation (/C, not ported, see the
// package comment) did not: 0-based indirect writes and every output format;
// and so 32-bit wrap-around, evaluation order and depth, and references past
// the end of a table, which read whatever the static Code array holds there.
//
// The tables are resolved while the C is written: every expression a program
// can evaluate becomes a function named after its code slot, and every
// statement a case of a switch inside the program counter loop, so the
// result is a C program without a trace of the tables, which makes
// C -> Sorted! -> C a fair obfuscator.
func Exact(p *syntax.Program) string {
	x := &exact{p: p, labels: map[int32]int32{}, funcs: map[fnKey]bool{}}
	return x.render()
}

type fnKey struct {
	t    syntax.OperandType
	slot int
}

type exact struct {
	p       *syntax.Program
	labels  map[int32]int32
	funcs   map[fnKey]bool // expression functions needed
	queue   []fnKey
	words   bool // a write prints a cardinal or an ordinal
	italian bool // ... in Italian
	vaudois bool // ... in Vaudois French
	input   bool // a read can run
	dynamic bool // a jump target is not a label, so pc can be anything
}

// slot returns code slot i and whether it exists; slots past what the parser
// wrote hold zeros.
func (x *exact) slot(i int) (syntax.Slide, bool) {
	if i < 0 || i >= codeSlots {
		return syntax.Slide{}, false
	}
	return code(x.p, i), true
}

// entry returns the slot number of entry index of category c.
func (x *exact) entry(c syntax.Category, index int32) int {
	return x.p.Tables[c].Index + int(index)
}

// fn names the function that evaluates slot as an expression of type t,
// queueing it for output.
func (x *exact) fn(t syntax.OperandType, slot int) string {
	k := fnKey{t, slot}
	if !x.funcs[k] {
		x.funcs[k] = true
		x.queue = append(x.queue, k)
	}
	name := fmt.Sprintf("%c%d", "NSDPRL___C"[t], slot)
	return strings.ReplaceAll(name, "-", "m")
}

// operand renders the evaluation of an operand (GetDataValue): every
// evaluation counts towards the depth limit, as in the interpreter.
func (x *exact) operand(o syntax.Operand) string {
	var s string
	switch t := o.Type & 0xFF; t {
	case syntax.Number:
		s = fmt.Sprintf("n_(%d)", o.Index)
	case syntax.Sum, syntax.Diff, syntax.Prod, syntax.Ratio, syntax.Nand, syntax.Condition:
		s = fmt.Sprintf("x_(%s)", x.fn(t, x.entry(syntax.Category(t-1), o.Index)))
	case syntax.Label:
		s = fmt.Sprintf("l_(%d)", x.labels[o.Index])
	default:
		s = "l_(0)"
	}
	if o.Type&syntax.Indirect != 0 {
		s = "i_(" + s + ")"
	}
	return s
}

// expression renders the function for k.
func (x *exact) expression(k fnKey) string {
	name := x.fn(k.t, k.slot)
	s, ok := x.slot(k.slot)
	if !ok {
		return fmt.Sprintf("static I %s(void) { Z(\"code slot out of range\"); return 0; }\n", name)
	}
	a, b := x.operand(s.Ops[0]), x.operand(s.Ops[1])
	var r string
	switch k.t {
	case syntax.Sum:
		r = "add_(a, b)"
	case syntax.Diff:
		r = "sub_(a, b)"
	case syntax.Prod:
		r = "mul_(a, b)"
	case syntax.Ratio:
		r = "div_(a, b)"
	case syntax.Nand:
		r = "~a & ~b"
		if s.Flags == syntax.LogicalNand {
			r = "~(a & b)"
		}
	default:
		r = "a < b"
		if s.Flags == syntax.CompareEqual {
			r = "a == b"
		}
	}
	return fmt.Sprintf("static I %s(void) { I a = %s, b = %s; return %s; }\n", name, a, b, r)
}

// pointer renders the code that sets p_ to the cell a store goes to
// (GetDataPointer), or a runtime error.
func (x *exact) pointer(o syntax.Operand) string {
	if x.p.Verys > 0 && o.Type&syntax.Indirect != 0 && o.Type&0xFF != syntax.Number {
		// Very Sorted! stores into the cell any value indexes (see interp)
		return fmt.Sprintf("p_ = %s; N(p_); ", x.operand(syntax.Operand{Type: o.Type &^ syntax.Indirect, Index: o.Index}))
	}
	if o.Type&0xFF != syntax.Number {
		return `Z("store into something that is not a cell"); `
	}
	if o.Type&syntax.Indirect != 0 {
		return fmt.Sprintf("p_ = N(%d); N(p_); ", o.Index)
	}
	return fmt.Sprintf("p_ = %d; N(p_); ", o.Index)
}

// statement renders what the statement in slot i does when the program
// counter reaches it. The statements can run past the last code slot.
func (x *exact) statement(i int) string {
	s, ok := x.slot(i)
	if !ok {
		return `Z("code slot out of range");`
	}
	op := s.Ops[0]
	var c syntax.Category
	switch op.Type {
	case syntax.Assign:
		c = syntax.Assigns
	case syntax.Read:
		c = syntax.Reads
	case syntax.Write:
		c = syntax.Writes
	case syntax.Jump:
		c = syntax.Jumps
	default:
		return "" // labels, and anything else, do nothing
	}
	e, ok := x.slot(x.entry(c, op.Index))
	if !ok {
		return `Z("code slot out of range");`
	}
	switch op.Type {
	case syntax.Assign:
		return x.pointer(e.Ops[1]) + "_[p_] = " + x.operand(e.Ops[0]) + ";"
	case syntax.Read:
		x.input = true
		target := e.Ops[1] // sic, as the original (see interp)
		if x.p.Verys > 0 {
			target = e.Ops[0]
		}
		return x.pointer(target) + "fflush(stdout); _[p_] = g_();"
	case syntax.Write:
		v := x.operand(e.Ops[0])
		switch e.Flags {
		case syntax.FormatEnglishCardinal, syntax.FormatEnglishOrdinal, syntax.FormatGermanCardinal, syntax.FormatGermanOrdinal:
			x.words = true
			return fmt.Sprintf("w_(%d, %s);", e.Flags, v)
		case syntax.FormatItalianCardinal, syntax.FormatItalianOrdinal:
			x.words, x.italian = true, true
			return fmt.Sprintf("wi_(%d, %s);", e.Flags, v)
		case syntax.FormatVaudoisCardinal, syntax.FormatVaudoisOrdinal:
			x.vaudois = true
			return fmt.Sprintf("wf_(%d, %s);", e.Flags, v)
		}
		return fmt.Sprintf("putchar((unsigned char)%s);", v)
	}
	if e.Ops[0].Type != syntax.Label {
		x.dynamic = true
	}
	if e.Flags == syntax.UnconditionalJump {
		return fmt.Sprintf("pc = %s;", x.operand(e.Ops[0]))
	}
	return fmt.Sprintf("if (%s) pc = %s;", x.operand(e.Ops[1]), x.operand(e.Ops[0]))
}

func (x *exact) render() string {
	stmts := x.p.Tables[syntax.Statements]
	for j := 0; j < stmts.Count; j++ {
		if s := code(x.p, stmts.Index+j); s.Ops[0].Type == syntax.Label {
			x.labels[s.Ops[0].Index] = int32(j)
		}
	}

	// main first, which finds the expressions it needs.
	var m strings.Builder
	m.WriteString("int main(void) {\n\tI pc, p_;\n")
	m.WriteString("#ifdef _WIN32\n\t_setmode(_fileno(stdin), _O_BINARY);\n\t_setmode(_fileno(stdout), _O_BINARY);\n#endif\n")
	fmt.Fprintf(&m, "\tfor (pc = 0; pc < %d; pc = add_(pc, 1)) {\n\t\tswitch (pc) {\n", stmts.Count)
	for j := 0; j < stmts.Count; j++ {
		if s := x.statement(stmts.Index + j); s != "" {
			fmt.Fprintf(&m, "\t\tcase %d: %s break;\n", j, s)
		}
	}
	if x.dynamic {
		// A jump to something other than a label can send pc below zero,
		// where the interpreter runs the slots before the statements.
		m.WriteString("\t\tdefault:\n\t\t\tif (pc < 0) {\n")
		fmt.Fprintf(&m, "\t\t\t\tif (pc < %d) Z(\"code slot out of range\");\n", -stmts.Index)
		m.WriteString("\t\t\t\tswitch (pc) {\n")
		for i := 0; i < stmts.Index && i < len(x.p.Code); i++ {
			if s := x.statement(i); s != "" {
				fmt.Fprintf(&m, "\t\t\t\tcase %d: %s break;\n", i-stmts.Index, s)
			}
		}
		m.WriteString("\t\t\t\t}\n\t\t\t}\n")
	}
	m.WriteString("\t\t}\n\t}\n\treturn 0;\n}\n")

	var fns []string
	for i := 0; i < len(x.queue); i++ { // expressions may queue more
		fns = append(fns, x.expression(x.queue[i]))
	}

	var b strings.Builder
	b.WriteString(exactHeader)
	n := min(len(x.p.Data), dataCells)
	if n == 0 {
		fmt.Fprintf(&b, "static I _[%d];\n", dataCells)
	} else {
		fmt.Fprintf(&b, "static I _[%d] = {%s};\n", dataCells, joinData(x.p.Data[:n]))
	}
	b.WriteString(exactRuntime)
	if x.words {
		b.WriteString(words(x.p.Verys > 0))
	}
	if x.italian {
		b.WriteString(exactItalian)
	}
	if x.vaudois {
		b.WriteString(exactVaudois)
	}
	if x.input {
		b.WriteString(exactInput)
	}
	if len(fns) > 0 {
		for i, k := range x.queue { // declarations, six to a line
			b.WriteString("static I " + x.fn(k.t, k.slot) + "(void);")
			if i%6 == 5 || i == len(x.queue)-1 {
				b.WriteString("\n")
			} else {
				b.WriteString(" ")
			}
		}
		slices.Sort(fns)
		b.WriteString(strings.Join(fns, ""))
	}
	b.WriteString(m.String())
	return b.String()
}

const exactHeader = `/* Written by sorted --to-c: behaves like the Sorted! interpreter. */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#ifdef _WIN32
#include <fcntl.h>
#include <io.h>
#endif

typedef int32_t I;
typedef uint32_t U;
`

const exactRuntime = `static I depth_;
static void Z(const char *what) { fflush(stdout); fprintf(stderr, "runtime error: %s\n", what); exit(1); }
static I N(I i) { if (i < 0 || i >= 193719) Z("cell out of range"); return _[i]; }
static void in_(void) { if (++depth_ > 65536) Z("recursion too deep"); }
static I n_(I i) { I r; in_(); r = N(i); depth_--; return r; }
static I x_(I (*f)(void)) { I r; in_(); r = f(); depth_--; return r; }
static I l_(I v) { in_(); depth_--; return v; }
static I i_(I v) { return N((I)((U)v - 1)); }
static I add_(I a, I b) { return (I)((U)a + (U)b); }
static I sub_(I a, I b) { return (I)((U)a - (U)b); }
static I mul_(I a, I b) { return (I)((U)a * (U)b); }
static I div_(I a, I b) { if (b == 0) Z("division by zero"); return a == INT32_MIN && b == -1 ? a : a / b; }
`

// exactWords ports the number formatting of internal/numbers (and so of
// EnglishNumbers.cpp and GermanNumbers.cpp), misspellings included.
const exactWords = `static const char *const enS[10] = {"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"};
static const char *const enD[10] = {"", "", "twenty", "thirty", "fourty", "fifty", "sixty", "seventy", "eighty", "ninety"};
static const char *const enT[20] = {"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fiveteen", "sixteen", "seventeen", "eighteen", "nineteen"};
static const char *const enH[10] = {"", "onehundred", "twohundred", "threehundred", "fourhundred", "fivehundred", "sixhundred", "sevenhundred", "eighthundred", "ninehundred"};
static const char *const enO[10] = {"", "first", "second", "third", "fourth", "fifth", "sixth", "seventh", "eighth", "nineth"};
static const char *const deS[10] = {"", "einund", "zweiund", "dreiund", "vierund", "fuenfund", "sechsund", "siebenund", "achtund", "neunund"};
static const char *const deD[10] = {"", "", "zwanzig", "dreissig", "vierzig", "fuenfzig", "sechzig", "siebzig", "achtzig", "neunzig"};
static const char *const deT[20] = {"null", "eins", "zwei", "drei", "vier", "fuenf", "sechs", "sieben", "acht", "neun", "zehn", "elf", "zwoelf", "dreizehn", "vierzehn", "fuenfzehn", "sechzehn", "siebzehn", "achtzehn", "neunzehn"};
static const char *const deH[10] = {"", "einhundert", "zweihundert", "dreihundert", "vierhundert", "fuenfhundert", "sechshundert", "siebenhundert", "achthundert", "neunhundert"};
static const char *const deO[20] = {"", "erste", "zweite", "dritte", "vierte", "fuenfte", "sechste", "siebente", "achte", "neunte", "zehnte", "elfte", "zwoelfte", "dreizehnte", "vierzehnte", "fuenfzehnte", "sechzehnte", "siebzehnte", "achtzehnte", "neunzehnte"};
static void half_(char *b, I n, const char *suffix, int de) {
	if (n == 0) return;
	strcat(b, de ? deH[n / 100] : enH[n / 100]);
	n %= 100;
	if (n != 0) {
		if (n < 20) strcat(b, de ? deT[n] : enT[n]);
		else if (de) { strcat(b, deS[n % 10]); strcat(b, deD[n / 10]); }
		else { strcat(b, enD[n / 10]); strcat(b, enS[n % 10]); }
	}
	strcat(b, suffix);
}
static void card_(char *b, I n, int de) {
	b[0] = 0;
	n %= 1000000000;
	if (n >= 1000000) { half_(b, n / 1000000, de ? "millionen" : "million", de); n %= 1000000; }
	half_(b, n / 1000, de ? "tausend" : "thousand", de);
	half_(b, n % 1000, "", de);
}
static int ends_(const char *s, const char *w) {
	size_t n = strlen(s), k = strlen(w);
	return k <= n && strcmp(s + n - k, w) == 0;
}
static void w_(I format, I v) {
	char b[512];
	int i;
	size_t n;
	if (format == 1 || format == 3) {
		if (v < 0) Z("cannot write a negative number as a cardinal");
		card_(b, v, format == 3);
	} else if (v < 1) {
		strcpy(b, "ERROR, ORDINALS ARE POSITIVE INTEGERS");
	} else if (format == 2) {
		card_(b, v, 0);
		n = strlen(b);
		if (n > 0 && b[n - 1] == 'y') strcpy(b + n - 1, "ieth");
		else {
			for (i = 1; i < 10 && !ends_(b, enS[i]); i++) {}
			if (i < 10) strcpy(b + n - strlen(enS[i]), enO[i]);
			else strcat(b, n > 0 && b[n - 1] == 't' ? "h" : "th");
		}
	} else {
		card_(b, v, 1);
		n = strlen(b);
		for (i = 1; i < 20 && !ends_(b, deT[i]); i++) {}
		if (i < 20) strcpy(b + n - strlen(deT[i]), deO[i]);
		else strcat(b, "ste");
	}
	SPELL
	fputs(b, stdout);
	putchar('\n');
}
`

// words is the C number formatting, for Very Sorted! with German spelled in
// UTF-8.
func words(very bool) string {
	if very {
		return exactSpelling + strings.Replace(exactWords, "\tSPELL\n", "\tif (format >= 3) spell_(b);\n", 1)
	}
	return strings.Replace(exactWords, "\tSPELL\n", "", 1)
}

// exactSpelling is numbers.VerySpelling: Very Sorted! prints German number
// words in UTF-8. Each replacement is as long as what it replaces, so it is
// written over it.
const exactSpelling = `static void spell_(char *b) {
	static const char *const from[3] = {"fuenf", "zwoelf", "dreissig"};
	static const char *const to[3] = {"f\303\274nf", "zw\303\266lf", "drei\303\237ig"};
	char *p;
	int i;
	for (i = 0; i < 3; i++)
		for (p = b; (p = strstr(p, from[i])) != 0; p += strlen(to[i])) memcpy(p, to[i], strlen(to[i]));
}
`

// exactItalian is the Italian number formatting of internal/numbers (Very
// Very Sorted!, #30), after exactWords (it uses ends_): any number has its
// words, in UTF-8.
const exactItalian = `static const char *const itU[20] = {"", "uno", "due", "tre", "quattro", "cinque", "sei", "sette", "otto", "nove", "dieci", "undici", "dodici", "tredici", "quattordici", "quindici", "sedici", "diciassette", "diciotto", "diciannove"};
static const char *const itD[10] = {"", "", "venti", "trenta", "quaranta", "cinquanta", "sessanta", "settanta", "ottanta", "novanta"};
static const char *const itO[11] = {"", "primo", "secondo", "terzo", "quarto", "quinto", "sesto", "settimo", "ottavo", "nono", "decimo"};
static void it3_(char *b, U n) {
	U h = n / 100, r = n % 100;
	if (h > 1) strcat(b, itU[h]);
	if (h > 0) strcat(b, r / 10 == 8 ? "cent" : "cento");
	if (r < 20) { strcat(b, itU[r]); return; }
	strcat(b, itD[r / 10]);
	if (r % 10 == 1 || r % 10 == 8) b[strlen(b) - 1] = 0;
	strcat(b, itU[r % 10]);
}
static void itg_(char *b, U n, const char *one, const char *more) {
	if (n == 1) strcat(b, one);
	else { it3_(b, n); strcat(b, more); }
}
static void itc_(char *b, U n) {
	if (n == 0) { strcat(b, "zero"); return; }
	if (n >= 1000000000) { itg_(b, n / 1000000000, "unmiliardo", "miliardi"); n %= 1000000000; }
	if (n >= 1000000) { itg_(b, n / 1000000, "unmilione", "milioni"); n %= 1000000; }
	if (n >= 1000) { itg_(b, n / 1000, "mille", "mila"); n %= 1000; }
	if (n > 0) it3_(b, n);
}
static void wi_(I format, I v) {
	char b[512];
	char *w;
	size_t n;
	U m = v < 0 ? 0u - (U)v : (U)v;
	strcpy(b, v < 0 ? "meno " : "");
	w = b + strlen(b);
	if (format == 5) {
		itc_(w, m);
		n = strlen(w);
		if (n > 3 && ends_(w, "tre")) strcpy(w + n - 1, "\303\251");
	} else if (m == 0) strcpy(w, "zeresimo");
	else if (m <= 10) strcpy(w, itO[m]);
	else if (m == 1000000) strcpy(w, "milionesimo");
	else if (m == 1000000000) strcpy(w, "miliardesimo");
	else {
		itc_(w, m);
		n = strlen(w);
		if (ends_(w, "tre") || ends_(w, "sei")) {}
		else if (ends_(w, "mila")) strcpy(w + n - 4, "mill");
		else if (ends_(w, "centouno")) strcpy(w + n - 4, "un");
		else if (ends_(w, "centootto")) strcpy(w + n - 5, "ott");
		else w[n - 1] = 0;
		strcat(w, "esimo");
	}
	fputs(b, stdout);
	putchar('\n');
}
`

// exactVaudois is the Vaudois French number formatting of internal/numbers
// (Very Very Sorted!, #29): any number has its words, joined by hyphens,
// in UTF-8.
const exactVaudois = `static const char *const frU[17] = {"", "un", "deux", "trois", "quatre", "cinq", "six", "sept", "huit", "neuf", "dix", "onze", "douze", "treize", "quatorze", "quinze", "seize"};
static const char *const frD[10] = {"", "dix", "vingt", "trente", "quarante", "cinquante", "soixante", "septante", "huitante", "nonante"};
static void frp_(char *w, const char *part) {
	if (*w) strcat(w, "-");
	strcat(w, part);
}
static void fr2_(char *w, U n) {
	if (n <= 16) frp_(w, frU[n]);
	else if (n < 20) { frp_(w, "dix"); frp_(w, frU[n - 10]); }
	else {
		frp_(w, frD[n / 10]);
		if (n % 10 == 1) { frp_(w, "et"); frp_(w, "un"); }
		else if (n % 10 != 0) frp_(w, frU[n % 10]);
	}
}
static void fr3_(char *w, U n, int plural) {
	U h = n / 100, r = n % 100;
	if (h == 1) frp_(w, "cent");
	else if (h > 1) { frp_(w, frU[h]); frp_(w, r == 0 && plural ? "cents" : "cent"); }
	if (r != 0) fr2_(w, r);
}
static void frg_(char *w, U c, const char *one, const char *more) {
	if (c == 1) { frp_(w, "un"); frp_(w, one); }
	else { fr3_(w, c, 1); frp_(w, more); }
}
static void frc_(char *w, U n) {
	if (n == 0) { frp_(w, "z\303\251ro"); return; }
	if (n >= 1000000000) { frg_(w, n / 1000000000, "milliard", "milliards"); n %= 1000000000; }
	if (n >= 1000000) { frg_(w, n / 1000000, "million", "millions"); n %= 1000000; }
	if (n >= 1000) { if (n / 1000 > 1) fr3_(w, n / 1000, 0); frp_(w, "mille"); n %= 1000; }
	if (n > 0) fr3_(w, n, 1);
}
static void wf_(I format, I v) {
	char b[512];
	char *w, *last;
	size_t n;
	U m = v < 0 ? 0u - (U)v : (U)v;
	strcpy(b, v < 0 ? "moins " : "");
	w = b + strlen(b);
	if (format == 7) frc_(w, m);
	else if (m == 1) strcpy(w, "premier");
	else if (m == 0) strcpy(w, "z\303\251roi\303\250me");
	else {
		frc_(w, m);
		if (strcmp(w, "un-million") == 0 || strcmp(w, "un-milliard") == 0) memmove(w, w + 3, strlen(w + 3) + 1);
		last = strrchr(w, '-');
		last = last ? last + 1 : w;
		n = strlen(last);
		if (strcmp(last, "cents") == 0) last[n - 1] = 0;
		else if (strcmp(last, "millions") == 0 || strcmp(last, "milliards") == 0) {
			last[n - 1] = 0;
			if (last - w >= 6 && strncmp(last - 6, "cents-", 6) == 0 && (last - 6 == w || last[-7] == '-')) memmove(last - 2, last - 1, strlen(last - 1) + 1);
		}
		else if (strcmp(last, "cinq") == 0) strcpy(last, "cinqu");
		else if (strcmp(last, "neuf") == 0) strcpy(last, "neuv");
		else if (last[n - 1] == 'e') last[n - 1] = 0;
		strcat(w, "i\303\250me");
	}
	fputs(b, stdout);
	putchar('\n');
}
`

// exactInput reads like the interpreter's stdin: as the Win32 C runtime's
// text mode does, CRLF arrives as LF and Ctrl-Z ends the input for good.
const exactInput = `static int eof_;
static I g_(void) {
	int c;
	if (eof_) return -1;
	c = getchar();
	if (c == EOF || c == 0x1A) { eof_ = 1; return -1; }
	if (c == '\r') {
		int d = getchar();
		if (d == '\n') return '\n';
		if (d != EOF) ungetc(d, stdin);
	}
	return c;
}
`
