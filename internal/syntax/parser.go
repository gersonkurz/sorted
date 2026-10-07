package syntax

import "github.com/gersonkurz/sorted/internal/numbers"

// Error is a parse failure. Its text is the line the original prints.
type Error struct {
	// What names the sentence that failed, e.g. "number declaration".
	What string
}

func (e *Error) Error() string { return "ERROR, missing or invalid " + e.What }

// Parse parses filtered source text (see Filter) into a Program. A program is
// exactly fourteen sentences in a fixed order; the first one that does not
// parse is reported. Text after the final "Cool." is ignored.
//
// A program that does not parse as the original's Sorted! may be Very
// Sorted! (#25): the same fourteen sentences, ending with "This code is very
// cool." instead, in which a statement can also be an input ("the first
// input"). Parse tries the original grammar first and the very one only
// when that fails, so everything the original accepts parses exactly as
// before, and when neither does, the original's error is reported.
func Parse(src string) (*Program, error) {
	p, err := parseDialect(src, false)
	if err == nil {
		return p, nil
	}
	if v, verr := parseDialect(src, true); verr == nil {
		return v, nil
	}
	return nil, err
}

func parseDialect(src string, very bool) (*Program, error) {
	ps := &parser{s: src, code: &Program{Very: very}, very: very}
	sentences := []struct {
		parse func() bool
		what  string
	}{
		{ps.numberDeclaration, "number declaration"},
		{ps.jumpDeclaration, "declaration of jumps"},
		{ps.outputDeclaration, "declaration of output"},
		{ps.inputDeclaration, "declaration of input"},
		{ps.sumDeclaration, "sum declaration"},
		{ps.conditionDeclaration, "declaration of conditions"},
		{ps.labelDeclaration, "label declaration"},
		{ps.diffDeclaration, "declaration of ordered differences"},
		{ps.assignDeclaration, "declaration of assignments"},
		{ps.prodDeclaration, "declaration of products"},
		{ps.implementationDeclaration, "declaration of implementation"},
		{ps.ratioDeclaration, "declaration of ratios"},
		{ps.nandDeclaration, "declaration of logical operations"},
		{ps.cool, "coolness"},
	}
	for _, s := range sentences {
		if !s.parse() {
			return nil, &Error{s.what}
		}
	}
	return ps.code, nil
}

// parser holds the cursor (m_pszExpression) and the tables being built.
// Saving and restoring p is pushExpression/popExpression.
type parser struct {
	s    string
	p    int
	code *Program
	very bool // the Very Sorted! dialect (see Parse)
}

// at returns the byte under the cursor, NUL at the end.
func (ps *parser) at() byte {
	if ps.p < len(ps.s) {
		return ps.s[ps.p]
	}
	return 0
}

// skipWhitespaces skips everything that is not [A-Za-z.,]. The Win32 build
// lacks the NUL check and runs past the end of the text (undefined
// behaviour); the port stops at the end.
func (ps *parser) skipWhitespaces() {
	for ps.p < len(ps.s) && !isValidChar(ps.s[ps.p]) {
		ps.p++
	}
}

// kw matches a keyword, case-insensitively, that is not followed by a letter,
// and skips the whitespace around it (IS_KEYWORD). It consumes nothing on
// failure, apart from leading whitespace.
func (ps *parser) kw(k string) bool {
	ps.skipWhitespaces()
	end := ps.p + len(k)
	if end > len(ps.s) || (end < len(ps.s) && isChar(ps.s[end])) {
		return false
	}
	if !equalFold(ps.s[ps.p:end], k) {
		return false
	}
	ps.p = end
	ps.skipWhitespaces()
	return true
}

// equalFold compares ASCII case-insensitively, like stricmp.
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// seq matches a sequence of keywords, all or nothing (IS_KEYWORD_SEQUENCE).
func (ps *parser) seq(ks ...string) bool {
	save := ps.p
	for _, k := range ks {
		if !ps.kw(k) {
			ps.p = save
			return false
		}
	}
	return true
}

// cardinal parses a number declaration's value: English first, then German
// from wherever the English attempt left the cursor (CARDINAL).
func (ps *parser) cardinal() (int32, bool) {
	n, next, ok := numbers.ParseEnglishCardinal(ps.s, ps.p)
	ps.p = next
	if ok {
		return n, true
	}
	n, next, ok = numbers.ParseGermanCardinal(ps.s, ps.p)
	ps.p = next
	return n, ok
}

// prepare starts a category's table at the current element count
// (PrepareCodeInfo).
func (ps *parser) prepare(c Category) {
	ps.code.Tables[c] = Table{Count: 0, Index: ps.code.TypeCount}
}

// accept counts the entry just written to category c.
func (ps *parser) accept(c Category) {
	ps.code.Tables[c].Count++
	ps.code.TypeCount++
}

// cancel withdraws the last entry of category c but leaves TypeCount alone,
// as CancelSpec does.
func (ps *parser) cancel(c Category) {
	if ps.code.Tables[c].Count > 0 {
		ps.code.Tables[c].Count--
	}
}

// sequence parses "spec, spec, ..., and spec" (the *_SEQUENCE loops after
// their introducing keywords): specs separated by commas, the last one after
// ", and"/", und". A list without that final form fails.
func (ps *parser) sequence(spec func() bool) bool {
	save := ps.p
	for spec() {
		if !ps.kw(",") {
			break
		}
		if ps.kw("and") || ps.kw("und") {
			if spec() {
				return true
			}
			break
		}
	}
	ps.p = save
	return false
}

// --- numbers ---

func (ps *parser) usesNoNumbers() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "numbers") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "zahlen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesNumbers() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") || ps.seq("dieses", "programm", "benutzt", "die") {
		if ps.singleNumber() || ps.numberSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

// storeSingleNumber appends n to the data unless it is already there
// ("cardinals only once"). Numbers stay stored when the sentence later fails.
func (ps *parser) storeSingleNumber(n int32) bool {
	for _, d := range ps.code.Data {
		if d == n {
			return false
		}
	}
	ps.code.Data = append(ps.code.Data, n)
	return true
}

func (ps *parser) singleNumber() bool {
	save := ps.p
	if ps.kw("number") || ps.kw("zahl") {
		if n, ok := ps.cardinal(); ok {
			return ps.storeSingleNumber(n)
		}
	}
	ps.p = save
	return false
}

func (ps *parser) numberSequence() bool {
	save := ps.p
	if !ps.kw("numbers") && !ps.kw("zahlen") {
		return false
	}
	for {
		n, ok := ps.cardinal()
		if !ok || !ps.storeSingleNumber(n) {
			break
		}
		if !ps.kw(",") {
			break
		}
		if ps.kw("and") || ps.kw("und") {
			n, ok := ps.cardinal()
			if !ok {
				break
			}
			return ps.storeSingleNumber(n)
		}
	}
	ps.p = save
	return false
}

func (ps *parser) numberDeclaration() bool {
	if ps.usesNumbers() || ps.usesNoNumbers() {
		return ps.kw(".")
	}
	return false
}

// --- identifiers ---

func (ps *parser) identifier(op *Operand) bool {
	return ps.indirectUse(op) || ps.directUse(op)
}

func (ps *parser) indirectUse(op *Operand) bool {
	save := ps.p
	if ps.seq("the", "cell", "indexed", "by") ||
		ps.seq("diejenige", "zelle", "die", "indiziert", "wird", "durch") {
		if ps.directUse(op) {
			op.Type |= Indirect
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) directUse(op *Operand) bool {
	save := ps.p
	if ps.kw("the") || ps.kw("die") || ps.kw("das") || ps.kw("der") || ps.kw("den") {
		n, next := numbers.ParseEnglishOrdinal(ps.s, ps.p)
		ps.p = next
		if n == 0 {
			n, next = numbers.ParseGermanOrdinal(ps.s, ps.p)
			ps.p = next
		}
		if n != 0 {
			t, ok := Number, true
			switch {
			case ps.kw("sum") || ps.kw("summe"):
				t = Sum
			case ps.kw("number") || ps.kw("zahl"):
				t = Number
			case (ps.kw("ordered") && ps.kw("difference")) || (ps.kw("geordnete") && ps.kw("differenz")):
				t = Diff
			case ps.kw("product") || ps.kw("produkt"):
				t = Prod
			case ps.kw("ratio") || ps.kw("verhaeltnis"):
				t = Ratio
			case ps.kw("cell") || ps.kw("zelle"):
				t = Cell
			case ps.kw("assignment") || ps.kw("zuweisung"):
				t = Assign
			case ps.kw("jump") || ps.kw("sprungbefehl"):
				t = Jump
			case ps.kw("label") || ps.kw("sprungziel"):
				t = Label
			case ps.kw("condition") || ps.kw("bedingung"):
				t = Condition
			case ps.kw("output") || ps.kw("ausgabe"):
				t = Write
			case ps.very && (ps.kw("input") || ps.kw("eingabe")):
				t = Read
			default:
				ok = false
			}
			if ok {
				op.Type = t
				op.Index = n - 1
				return true
			}
		}
	}
	ps.p = save
	return false
}

// --- sums ---

func (ps *parser) sumSpec() bool {
	save := ps.p
	cell := ps.code.slot(Sums)
	if ps.kw("of") || ps.kw("von") || ps.kw("aus") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.kw("and") || ps.kw("und") {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Sums)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoSums() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "sums") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "summen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesSums() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") || ps.seq("dieses", "programm", "benutzt", "die") {
		if ps.singleSum() || ps.sumSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleSum() bool {
	save := ps.p
	if ps.kw("sum") || ps.kw("summe") {
		if ps.sumSpec() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) sumSequence() bool {
	save := ps.p
	if !ps.kw("sums") && !ps.kw("summen") {
		return false
	}
	if ps.sequence(ps.sumSpec) {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) sumDeclaration() bool {
	ps.prepare(Sums)
	if ps.usesSums() || ps.usesNoSums() {
		return ps.kw(".")
	}
	return false
}

// cool is the last sentence. It ignores whatever follows. In Very Sorted!
// the code is very cool.
func (ps *parser) cool() bool {
	if ps.very {
		return ps.seq("this", "code", "is", "very", "cool", ".") || ps.seq("very", "cool", ".") ||
			ps.seq("dieses", "programm", "ist", "ganz", "hervorragend", ".") || ps.seq("ganz", "hervorragend", ".")
	}
	return ps.seq("this", "code", "is", "cool", ".") || ps.seq("cool", ".") ||
		ps.seq("dieses", "programm", "ist", "hervorragend", ".") || ps.seq("hervorragend", ".")
}

// --- ordered differences ---

func (ps *parser) diffSpec() bool {
	save := ps.p
	cell := ps.code.slot(Diffs)
	if ps.kw("between") || ps.kw("zwischen") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.kw("and") || ps.kw("und") {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Diffs)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoDiffs() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "ordered", "differences") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "geordneten", "differenzen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesDiffs() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") || ps.seq("dieses", "programm", "benutzt", "die") {
		if ps.singleDiff() || ps.diffSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleDiff() bool {
	save := ps.p
	if (ps.kw("ordered") && ps.kw("difference")) || (ps.kw("geordnete") && ps.kw("differenz")) {
		if ps.diffSpec() {
			return true
		}
	}
	ps.p = save
	return false
}

// diffSequence: the German introduction is the singular "geordnete
// differenz", the same as singleDiff's, as in the original.
func (ps *parser) diffSequence() bool {
	save := ps.p
	if !(ps.kw("ordered") && ps.kw("differences")) && !(ps.kw("geordnete") && ps.kw("differenz")) {
		return false
	}
	if ps.sequence(ps.diffSpec) {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) diffDeclaration() bool {
	ps.prepare(Diffs)
	if ps.usesDiffs() || ps.usesNoDiffs() {
		return ps.kw(".")
	}
	return false
}

// --- products ---

func (ps *parser) prodSpec() bool {
	save := ps.p
	cell := ps.code.slot(Prods)
	if ps.kw("of") || ps.kw("von") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.kw("and") || ps.kw("und") {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Prods)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoProds() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "products") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "produkte") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesProds() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") || ps.seq("dieses", "programm", "benutzt") {
		if ps.singleProd() || ps.prodSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleProd() bool {
	save := ps.p
	if ps.kw("product") || (ps.kw("das") && ps.kw("produkt")) {
		if ps.prodSpec() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) prodSequence() bool {
	save := ps.p
	if !ps.kw("products") && !(ps.kw("die") && ps.kw("produkte")) {
		return false
	}
	if ps.sequence(ps.prodSpec) {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) prodDeclaration() bool {
	ps.prepare(Prods)
	if ps.usesProds() || ps.usesNoProds() {
		return ps.kw(".")
	}
	return false
}

// --- ratios (English only) ---

func (ps *parser) ratioSpec() bool {
	save := ps.p
	cell := ps.code.slot(Ratios)
	if ps.kw("of") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.kw("to") {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Ratios)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoRatios() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "ratios") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "verhaeltnisse") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesRatios() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") {
		if ps.singleRatio() || ps.ratioSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleRatio() bool {
	save := ps.p
	if ps.kw("ratio") {
		if ps.ratioSpec() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) ratioSequence() bool {
	save := ps.p
	if !ps.kw("ratios") {
		return false
	}
	if ps.sequence(ps.ratioSpec) {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) ratioDeclaration() bool {
	ps.prepare(Ratios)
	if ps.usesRatios() || ps.usesNoRatios() {
		return ps.kw(".")
	}
	return false
}

// --- logical operations (NAND, English only) ---

func (ps *parser) nandSpec() bool {
	save := ps.p
	cell := ps.code.slot(Nands)
	if ps.kw("of") && ps.kw("not") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.kw("and") && ps.kw("not") {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Nands)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoNands() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "logical", "operations") ||
		ps.seq("dieses", "programm", "ist", "unlogisch") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesNands() bool {
	save := ps.p
	if ps.seq("this", "code", "uses", "the") {
		if ps.singleNand() || ps.nandSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleNand() bool {
	save := ps.p
	if ps.kw("logical") && ps.kw("operation") {
		if ps.nandSpec() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) nandSequence() bool {
	save := ps.p
	if !(ps.kw("logical") && ps.kw("operations")) {
		return false
	}
	if ps.sequence(ps.nandSpec) {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) nandDeclaration() bool {
	ps.prepare(Nands)
	if ps.usesNands() || ps.usesNoNands() {
		return ps.kw(".")
	}
	return false
}

// --- labels ---

func (ps *parser) usesNoLabels() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "labels") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "sprungziele") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesLabels() bool {
	save := ps.p
	if ps.seq("this", "code", "uses") || ps.seq("dieses", "programm", "benutzt") {
		if n, ok := ps.cardinal(); ok {
			if (n == 1 && ps.kw("label")) || ps.kw("labels") {
				ps.code.LabelsCount = int(n)
				return true
			}
			if (n == 1 && ps.kw("sprungziel")) || ps.kw("sprungziele") {
				ps.code.LabelsCount = int(n)
				return true
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) labelDeclaration() bool {
	if ps.usesLabels() || ps.usesNoLabels() {
		return ps.kw(".")
	}
	return false
}

// --- assignments ---

// assignSpec: the target must be a number (cell), direct or indirect.
func (ps *parser) assignSpec() bool {
	save := ps.p
	cell := ps.code.slot(Assigns)
	if ps.identifier(&cell.Ops[0]) {
		if ps.kw("to") || ps.kw("an") {
			if ps.identifier(&cell.Ops[1]) {
				if cell.Ops[1].Type&0xFF == Number {
					ps.accept(Assigns)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoAssigns() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "assignments") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "zuweisungen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesAssigns() bool {
	save := ps.p
	if ps.seq("this", "code", "assigns") || ps.seq("dieses", "programm", "weisst", "zu") {
		if ps.singleAssign() || ps.assignSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

// singleAssign accepts one assignment only if the period follows directly;
// otherwise it withdraws the entry (but not its TypeCount).
func (ps *parser) singleAssign() bool {
	save := ps.p
	if ps.assignSpec() {
		if ps.at() == '.' {
			return true
		}
		ps.cancel(Assigns)
	}
	ps.p = save
	return false
}

func (ps *parser) assignSequence() bool { return ps.sequence(ps.assignSpec) }

func (ps *parser) assignDeclaration() bool {
	ps.prepare(Assigns)
	if ps.usesAssigns() || ps.usesNoAssigns() {
		return ps.kw(".")
	}
	return false
}

// --- output ---

// outputSpec parses "<identifier> as a <format>". The alternatives share a
// cursor that is not restored between them, as in the original: "english"
// consumed by the cardinal alternative is gone when the ordinal alternative
// runs, so "as a english ordinal" and "as a german ordinal" never parse, nor
// do the German "ein ..." forms other than "ein Zeichen". Doubling the eaten
// word gets through: "as a english english ordinal", "als ein ein ein ein
// deutscher Kardinal".
func (ps *parser) outputSpec() bool {
	save := ps.p
	cell := ps.code.slot(Writes)
	if ps.identifier(&cell.Ops[0]) {
		if (ps.kw("as") && ps.kw("a")) || ps.kw("als") {
			switch {
			case ps.kw("character") || (ps.kw("ein") && ps.kw("zeichen")):
				cell.Flags = FormatCharacter
			case (ps.kw("english") && ps.kw("cardinal")) ||
				(ps.kw("ein") && ps.kw("englischer") && ps.kw("Kardinal")):
				cell.Flags = FormatEnglishCardinal
			case (ps.kw("english") && ps.kw("ordinal")) ||
				(ps.kw("ein") && ps.kw("englische") && ps.kw("Ordinalzahl")):
				cell.Flags = FormatEnglishOrdinal
			case (ps.kw("german") && ps.kw("cardinal")) ||
				(ps.kw("ein") && ps.kw("deutscher") && ps.kw("Kardinal")):
				cell.Flags = FormatGermanCardinal
			case (ps.kw("german") && ps.kw("ordinal")) ||
				(ps.kw("eine") && ps.kw("deutsche") && ps.kw("Ordinalzahl")):
				cell.Flags = FormatGermanOrdinal
			default:
				ps.p = save
				return false
			}
			ps.accept(Writes)
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoOutput() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "produce", "any", "output") ||
		ps.seq("this", "code", "cannot", "write") ||
		ps.seq("dieses", "programm", "erzeugt", "keine", "ausgaben") ||
		ps.seq("dieses", "programm", "kann", "nicht", "schreiben") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesOutput() bool {
	save := ps.p
	if ps.seq("this", "code", "writes") || ps.seq("Dieses", "programm", "schreibt") {
		if ps.singleOutput() || ps.outputSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

// singleOutput, unlike singleAssign, does not check for the period: it
// accepts the first output of a list, after which the sentence fails at the
// comma. A program can therefore declare only one output.
func (ps *parser) singleOutput() bool {
	save := ps.p
	if ps.outputSpec() {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) outputSequence() bool { return ps.sequence(ps.outputSpec) }

func (ps *parser) outputDeclaration() bool {
	ps.prepare(Writes)
	if ps.usesOutput() || ps.usesNoOutput() {
		return ps.kw(".")
	}
	return false
}

// --- input ---

func (ps *parser) inputSpec() bool {
	save := ps.p
	cell := ps.code.slot(Reads)
	if ps.identifier(&cell.Ops[0]) {
		if (ps.kw("as") && ps.kw("a")) || ps.kw("als") {
			if ps.kw("character") || (ps.kw("ein") && ps.kw("zeichen")) {
				cell.Flags = FormatCharacter
			} else {
				ps.p = save
				return false
			}
			ps.accept(Reads)
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoInput() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "produce", "any", "input") ||
		ps.seq("this", "code", "cannot", "read") ||
		ps.seq("dieses", "programm", "liest", "keine", "eingaben") ||
		ps.seq("dieses", "programm", "kann", "nicht", "lesen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesInput() bool {
	save := ps.p
	if ps.seq("this", "code", "reads") || ps.seq("dieses", "programm", "liest") {
		if ps.singleInput() || ps.inputSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

// singleInput has no period check either: one input per program.
func (ps *parser) singleInput() bool {
	save := ps.p
	if ps.inputSpec() {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) inputSequence() bool { return ps.sequence(ps.inputSpec) }

func (ps *parser) inputDeclaration() bool {
	ps.prepare(Reads)
	if ps.usesInput() || ps.usesNoInput() {
		return ps.kw(".")
	}
	return false
}

// --- conditions ---

func (ps *parser) comparison(cell *Slide) bool {
	save := ps.p
	if ps.seq("is", "equal", "to") || ps.seq("ist", "gleich") {
		cell.Flags = CompareEqual
		return true
	}
	ps.p = save
	if ps.seq("is", "less", "than") || ps.seq("ist", "kleiner", "als") {
		cell.Flags = CompareLess
		return true
	}
	return false
}

func (ps *parser) conditionSpec() bool {
	save := ps.p
	cell := ps.code.slot(Conditions)
	if ps.seq("the", "condition", "that") || ps.seq("die", "bedingung", "dass") {
		if ps.identifier(&cell.Ops[0]) {
			if ps.comparison(cell) {
				if ps.identifier(&cell.Ops[1]) {
					ps.accept(Conditions)
					return true
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoConditions() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "not", "use", "any", "conditions") ||
		ps.seq("dieses", "programm", "benutzt", "keine", "bedingungen") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesConditions() bool {
	save := ps.p
	if ps.seq("this", "code", "uses") || ps.seq("dieses", "programm", "benutzt") {
		if ps.singleCondition() || ps.conditionSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleCondition() bool {
	save := ps.p
	if ps.conditionSpec() {
		if ps.at() == '.' {
			return true
		}
		ps.cancel(Conditions)
	}
	ps.p = save
	return false
}

func (ps *parser) conditionSequence() bool { return ps.sequence(ps.conditionSpec) }

func (ps *parser) conditionDeclaration() bool {
	ps.prepare(Conditions)
	if ps.usesConditions() || ps.usesNoConditions() {
		return ps.kw(".")
	}
	return false
}

// --- implementation (statements) ---

// statementSpec accepts any identifier as a statement; the interpreter acts
// on assignments, outputs, inputs, jumps and labels.
func (ps *parser) statementSpec() bool {
	save := ps.p
	cell := ps.code.slot(Statements)
	if ps.identifier(&cell.Ops[0]) {
		ps.accept(Statements)
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) singleStatement() bool {
	save := ps.p
	if ps.statementSpec() {
		if ps.at() == '.' {
			return true
		}
		ps.cancel(Statements)
	}
	ps.p = save
	return false
}

func (ps *parser) statementSequence() bool { return ps.sequence(ps.statementSpec) }

func (ps *parser) implementationDeclaration() bool {
	ps.prepare(Statements)
	save := ps.p
	if ps.seq("this", "code", "implements") || ps.seq("dieses", "programm", "implementiert") {
		if ps.singleStatement() || ps.statementSequence() {
			return ps.kw(".")
		}
	}
	ps.p = save
	return false
}

// --- jumps ---

// jumpSpec: the target must be a direct label, the condition a direct
// condition.
func (ps *parser) jumpSpec() bool {
	save := ps.p
	cell := ps.code.slot(Jumps)
	if ps.seq("always", "goes", "to") || ps.seq("springt", "immer", "an") {
		if ps.identifier(&cell.Ops[0]) {
			if cell.Ops[0].Type == Label {
				cell.Flags = UnconditionalJump
				ps.accept(Jumps)
				return true
			}
		}
	} else if ps.seq("sometimes", "goes", "to") || ps.seq("springt", "manchmal", "an") {
		if ps.identifier(&cell.Ops[0]) {
			if cell.Ops[0].Type == Label {
				if ps.kw("if") || ps.kw("wenn") || ps.kw("falls") {
					if ps.identifier(&cell.Ops[1]) {
						if cell.Ops[1].Type == Condition {
							if ps.seq("is", "true") || ps.seq("wahr", "ist") {
								cell.Flags = ConditionalJump
								ps.accept(Jumps)
								return true
							}
						}
					}
				}
			}
		}
	}
	ps.p = save
	return false
}

func (ps *parser) usesNoJumps() bool {
	save := ps.p
	if ps.seq("this", "code", "does", "never", "go", "anywhere") ||
		ps.seq("dieses", "programm", "geht", "nirgendwo", "hin") {
		return true
	}
	ps.p = save
	return false
}

func (ps *parser) usesJumps() bool {
	save := ps.p
	if ps.seq("this", "code") || ps.seq("dieses", "programm") {
		if ps.singleJump() || ps.jumpSequence() {
			return true
		}
	}
	ps.p = save
	return false
}

func (ps *parser) singleJump() bool {
	save := ps.p
	if ps.jumpSpec() {
		if ps.at() == '.' {
			return true
		}
		ps.cancel(Jumps)
	}
	ps.p = save
	return false
}

func (ps *parser) jumpSequence() bool { return ps.sequence(ps.jumpSpec) }

func (ps *parser) jumpDeclaration() bool {
	ps.prepare(Jumps)
	if ps.usesJumps() || ps.usesNoJumps() {
		return ps.kw(".")
	}
	return false
}
