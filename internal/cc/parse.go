package cc

// This is chibicc's parse.c (at commit a4d3223), slimmed to subset 1. Each
// function reads the grammar symbol it is named after, starting at the
// current token, and returns its node. chibicc threads the position through
// "rest" pointers and exits on errors; here the position lives in the parser
// and errors unwind to Parse.

// Parse parses a C translation unit.
func Parse(src string) (prog *Program, err error) {
	toks, err := Tokenize(src)
	if err != nil {
		return nil, err
	}
	toks, err = Preprocess(toks)
	if err != nil {
		return nil, err
	}
	ps := &parser{toks: toks}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(*Error)
			if !ok {
				panic(r)
			}
			prog, err = nil, e
		}
	}()
	return ps.program(), nil
}

type parser struct {
	toks    []Token
	i       int
	scopes  []map[string]*Obj
	globals map[string]*Obj
	prog    *Program
	loops   int       // nesting depth of loops, for break and continue
	fn      *Function // the function being parsed, nil in main
}

func (ps *parser) tok() Token  { return ps.toks[ps.i] }
func (ps *parser) next() Token { t := ps.toks[ps.i]; ps.i++; return t }

// equal reports whether the current token is s.
func (ps *parser) equal(s string) bool {
	t := ps.tok()
	return (t.Kind == TkPunct || t.Kind == TkKeyword) && t.Text == s
}

// skip consumes s or fails.
func (ps *parser) skip(s string) Token {
	if !ps.equal(s) {
		ps.fail(ps.tok().Pos, "expected '%s'", s)
	}
	return ps.next()
}

// consume consumes s if it is the current token.
func (ps *parser) consume(s string) bool {
	if ps.equal(s) {
		ps.i++
		return true
	}
	return false
}

func (ps *parser) fail(p Pos, format string, args ...any) {
	panic(errorAt(p, format, args...))
}

// unsupported rejects a construct outside the subset.
func (ps *parser) unsupported(p Pos, what string) {
	ps.fail(p, "not supported in Sorted! (yet): %s", what)
}

// program = ((declspec | "void") (function | global-variable))*
func (ps *parser) program() *Program {
	ps.prog = &Program{Funcs: map[string]*Function{}}
	ps.globals = map[string]*Obj{}
	for ps.tok().Kind != TkEOF {
		void := ps.consume("void")
		char := !void && ps.declspec()
		d := ps.declarator()
		if ps.equal("(") {
			if d.array {
				ps.unsupported(d.name.Pos, "functions returning arrays")
			}
			if d.name.Text == "main" {
				if char || void {
					ps.fail(d.name.Pos, "main must return int")
				}
				ps.function(d.name)
			} else {
				ps.otherFunction(d.name, void, char)
			}
			continue
		}
		if void {
			ps.fail(d.name.Pos, "a variable cannot be void")
		}
		ps.globalVariable(d, char)
	}
	if ps.prog.Main == nil {
		ps.fail(ps.tok().Pos, "no main function")
	}
	return ps.prog
}

// declspec = "int" | "char"; it reports whether the type is char.
func (ps *parser) declspec() bool {
	t := ps.tok()
	if t.Kind == TkKeyword && t.Text != "int" && t.Text != "char" {
		ps.unsupported(t.Pos, "the type or specifier '"+t.Text+"' (int and char are the only types)")
	}
	if ps.consume("char") {
		return true
	}
	ps.skip("int")
	return false
}

// decl is a declarator: a name, and for an array its length (0 when it is
// left to the initializer).
type decl struct {
	name  Token
	array bool
	len   int
	lpos  Pos // where the length is, for errors
}

// declarator = ident ("[" num? "]")?
//
// Pointers and arrays of arrays are not in the subset.
func (ps *parser) declarator() decl {
	if ps.equal("*") {
		ps.unsupported(ps.tok().Pos, "pointers")
	}
	t := ps.next()
	if t.Kind != TkIdent {
		ps.fail(t.Pos, "expected a variable name")
	}
	d := decl{name: t}
	if !ps.equal("[") {
		return d
	}
	d.array, d.lpos = true, ps.next().Pos
	if !ps.equal("]") {
		pos := ps.tok().Pos
		n := ps.constant("array lengths other than integer constants")
		if n <= 0 {
			ps.fail(pos, "the length of an array must be positive")
		}
		d.len = int(n)
	}
	ps.skip("]")
	if ps.equal("[") {
		ps.unsupported(ps.tok().Pos, "arrays of arrays")
	}
	return d
}

// function = "(" "void"? ")" "{" compound-stmt, for main.
func (ps *parser) function(name Token) {
	if ps.prog.Main != nil {
		ps.fail(name.Pos, "redefinition of main")
	}
	if _, dup := ps.globals[name.Text]; dup {
		ps.fail(name.Pos, "redefinition of '%s' as a function", name.Text)
	}
	ps.skip("(")
	ps.consume("void")
	if !ps.equal(")") {
		ps.unsupported(ps.tok().Pos, "parameters of main")
	}
	ps.skip(")")
	if !ps.equal("{") {
		ps.fail(ps.tok().Pos, "expected the body of main")
	}
	ps.next()
	ps.prog.Main = ps.compoundStmt()
}

// otherFunction = "(" params ")" ("{" compound-stmt | ";")
// params        = "void" | param ("," param)*
// param         = declspec ident?   (names may be left out in a prototype)
//
// A prototype declares the function for calls before its definition; the
// definition must agree with it.
func (ps *parser) otherFunction(name Token, void, char bool) {
	if _, dup := ps.globals[name.Text]; dup {
		ps.fail(name.Pos, "redefinition of '%s' as a function", name.Text)
	}
	fn := &Function{Name: name.Text, Void: void, Char: char, Pos: name.Pos}
	ps.skip("(")
	if ps.equal("void") && ps.toks[ps.i+1].Text == ")" {
		ps.next()
	}
	var names []Token
	for !ps.equal(")") {
		if len(fn.Params) > 0 {
			ps.skip(",")
		}
		if ps.equal("void") {
			ps.fail(ps.tok().Pos, "a parameter cannot be void")
		}
		pchar := ps.declspec()
		if ps.equal("*") {
			ps.unsupported(ps.tok().Pos, "pointers")
		}
		var pname Token
		if ps.tok().Kind == TkIdent {
			pname = ps.next()
		}
		if ps.equal("[") {
			ps.unsupported(ps.tok().Pos, "array parameters (no pointers)")
		}
		fn.Params = append(fn.Params, &Obj{Name: pname.Text, Char: pchar, Pos: pname.Pos})
		names = append(names, pname)
	}
	ps.skip(")")
	prev, declared := ps.prog.Funcs[name.Text]
	if declared {
		if prev.Void != fn.Void || prev.Char != fn.Char || len(prev.Params) != len(fn.Params) {
			ps.fail(name.Pos, "conflicting declarations of '%s'", name.Text)
		}
		for i := range fn.Params {
			if prev.Params[i].Char != fn.Params[i].Char {
				ps.fail(name.Pos, "conflicting declarations of '%s'", name.Text)
			}
		}
	}
	if ps.consume(";") {
		if !declared {
			ps.prog.Funcs[name.Text] = fn
		}
		return
	}
	if declared && prev.Defined {
		ps.fail(name.Pos, "redefinition of '%s'", name.Text)
	}
	if !ps.equal("{") {
		ps.fail(ps.tok().Pos, "expected the body of '%s'", name.Text)
	}
	scope := map[string]*Obj{}
	for i, n := range names {
		if n.Text == "" {
			ps.fail(name.Pos, "parameter %d of '%s' has no name", i+1, name.Text)
		}
		if _, dup := scope[n.Text]; dup {
			ps.fail(n.Pos, "redefinition of parameter '%s'", n.Text)
		}
		scope[n.Text] = fn.Params[i]
	}
	if declared { // calls parsed so far point at the prototype: complete it
		prev.Params = fn.Params
		fn = prev
	}
	fn.Defined = true
	ps.prog.Funcs[name.Text] = fn
	ps.next()
	ps.fn = fn
	fn.Body = ps.block(scope) // the parameters live in the body's own scope
	ps.fn = nil
}

// newObj makes the variable a declarator declares.
func (ps *parser) newObj(d decl, char, global bool) *Obj {
	return &Obj{Name: d.name.Text, IsGlobal: global, Char: char, Len: d.len, Pos: d.name.Pos}
}

// global-variable = (declarator ("=" global-init)? ("," declarator ("=" global-init)?)*)? ";"
// global-init     = constant-expr | "{" constant-expr ("," constant-expr)* ","? "}" | string
//
// The first declarator has been read already.
func (ps *parser) globalVariable(d decl, char bool) {
	for {
		name := d.name
		if _, dup := ps.globals[name.Text]; dup {
			ps.fail(name.Pos, "redefinition of '%s'", name.Text)
		}
		if name.Text == "main" && ps.prog.Main != nil {
			ps.fail(name.Pos, "redefinition of 'main' as a variable")
		}
		if _, fn := ps.prog.Funcs[name.Text]; fn {
			ps.fail(name.Pos, "redefinition of '%s' as a variable", name.Text)
		}
		v := ps.newObj(d, char, true)
		if ps.consume("=") {
			if d.array {
				v.Init = arrayInit(ps, v, d, ps.initializer, func(c int32) int32 { return c })
			} else {
				v.Init = []int32{ps.initializer()}
			}
			if !ps.equal(",") && !ps.equal(";") {
				ps.unsupported(ps.tok().Pos, "global initializers other than integer constants")
			}
		} else if d.array && d.len == 0 {
			ps.fail(d.lpos, "an array without a length needs an initializer")
		}
		ps.globals[v.Name] = v
		ps.prog.Globals = append(ps.prog.Globals, v)
		if ps.consume(";") {
			return
		}
		ps.skip(",")
		d = ps.declarator()
	}
}

// initializer parses the initializer of a global: a constant expression.
func (ps *parser) initializer() int32 {
	return ps.constant("global initializers other than integer constants")
}

// constant parses an integer constant expression (see Fold); anything else
// is what.
func (ps *parser) constant(what string) int32 {
	pos := ps.tok().Pos
	v, ok := Fold(ps.logor())
	if !ok {
		ps.unsupported(pos, what)
	}
	return v
}

// arrayInit reads the initializer of array v, "{" item ("," item)* ","? "}"
// or, for a char array, a string literal (adjacent literals are joined), and
// returns the values, setting v.Len when the declarator left it open. A
// string's terminating NUL is included when the array has room for it.
// item reads one list element; char turns a string's character into one.
func arrayInit[T any](ps *parser, v *Obj, d decl, item func() T, char func(int32) T) []T {
	var items []T
	t := ps.tok()
	if t.Kind == TkStr {
		if !v.Char {
			ps.fail(t.Pos, "a string literal can only initialize a char array")
		}
		var str []byte
		for ps.tok().Kind == TkStr {
			str = append(str, ps.next().Str...)
		}
		if d.len == 0 || len(str) < d.len {
			str = append(str, 0)
		}
		if d.len > 0 && len(str) > d.len {
			ps.fail(t.Pos, "the string is longer than the array")
		}
		for _, c := range str {
			items = append(items, char(int32(int8(c))))
		}
	} else {
		ps.skip("{")
		for !ps.equal("}") {
			items = append(items, item())
			if !ps.consume(",") {
				break
			}
		}
		ps.skip("}")
		if len(items) == 0 {
			ps.fail(t.Pos, "empty initializer")
		}
		if d.len > 0 && len(items) > d.len {
			ps.fail(t.Pos, "too many initializers for the array")
		}
	}
	if d.len == 0 {
		v.Len = len(items)
	}
	return items
}

// declaration = declspec (declarator ("=" init)? ("," declarator ("=" init)?)*)? ";"
// init        = assign | "{" assign ("," assign)* ","? "}" | string
//
// It becomes a block of assignment statements for the initializers. An array
// initializer assigns every element: those it leaves out become 0, as in C.
func (ps *parser) declaration() *Node {
	block := &Node{Kind: NdBlock, Pos: ps.tok().Pos}
	char := ps.declspec()
	first := true
	for !ps.consume(";") {
		if !first {
			ps.skip(",")
		}
		first = false
		d := ps.declarator()
		scope := ps.scopes[len(ps.scopes)-1]
		if _, dup := scope[d.name.Text]; dup {
			ps.fail(d.name.Pos, "redefinition of '%s'", d.name.Text)
		}
		v := ps.newObj(d, char, false)
		scope[d.name.Text] = v
		if !ps.equal("=") {
			if d.array && d.len == 0 {
				ps.fail(d.lpos, "an array without a length needs an initializer")
			}
			continue
		}
		eq := ps.next()
		assign := func(lhs, rhs *Node) {
			n := &Node{Kind: NdAssign, Pos: eq.Pos, Lhs: lhs, Rhs: rhs}
			block.Body = append(block.Body, &Node{Kind: NdExprStmt, Pos: d.name.Pos, Lhs: n})
		}
		if !d.array {
			assign(&Node{Kind: NdVar, Pos: d.name.Pos, Var: v}, ps.assign())
			continue
		}
		items := arrayInit(ps, v, d, ps.assign, func(c int32) *Node { return &Node{Kind: NdNum, Pos: eq.Pos, Val: c} })
		for i := 0; i < v.Len; i++ {
			rhs := &Node{Kind: NdNum, Pos: eq.Pos}
			if i < len(items) {
				rhs = items[i]
			}
			idx := &Node{Kind: NdNum, Pos: eq.Pos, Val: int32(i)}
			assign(&Node{Kind: NdIndex, Pos: d.name.Pos, Var: v, Lhs: idx}, rhs)
		}
	}
	return block
}

// stmt = "return" expr? ";"
//
//	| "if" "(" expr ")" stmt ("else" stmt)?
//	| "while" "(" expr ")" stmt
//	| "for" "(" (declaration | expr-stmt) expr? ";" expr? ")" stmt
//	| "break" ";" | "continue" ";"
//	| "{" compound-stmt
//	| expr-stmt
func (ps *parser) stmt() *Node {
	t := ps.tok()
	switch {
	case ps.equal("return"):
		ps.next()
		n := &Node{Kind: NdReturn, Pos: t.Pos}
		if !ps.equal(";") {
			if ps.fn != nil && ps.fn.Void {
				ps.fail(ps.tok().Pos, "a void function cannot return a value")
			}
			n.Lhs = ps.expr()
		}
		ps.skip(";")
		return n
	case ps.equal("if"):
		ps.next()
		n := &Node{Kind: NdIf, Pos: t.Pos}
		ps.skip("(")
		n.Cond = ps.expr()
		ps.skip(")")
		n.Then = ps.stmt()
		if ps.consume("else") {
			n.Els = ps.stmt()
		}
		return n
	case ps.equal("while"):
		ps.next()
		n := &Node{Kind: NdWhile, Pos: t.Pos}
		ps.skip("(")
		n.Cond = ps.expr()
		ps.skip(")")
		n.Then = ps.loopBody()
		return n
	case ps.equal("for"):
		ps.next()
		n := &Node{Kind: NdFor, Pos: t.Pos}
		ps.skip("(")
		ps.scopes = append(ps.scopes, map[string]*Obj{}) // for (int i = ...)
		if ps.equal("int") || ps.equal("char") || ps.tok().Kind == TkKeyword && isTypeKeyword(ps.tok().Text) {
			n.Init = ps.declaration()
		} else {
			n.Init = ps.exprStmt()
		}
		if !ps.equal(";") {
			n.Cond = ps.expr()
		}
		ps.skip(";")
		if !ps.equal(")") {
			n.Inc = ps.expr()
		}
		ps.skip(")")
		n.Then = ps.loopBody()
		ps.scopes = ps.scopes[:len(ps.scopes)-1]
		return n
	case ps.equal("break"), ps.equal("continue"):
		ps.next()
		if ps.loops == 0 {
			ps.fail(t.Pos, "'%s' outside a loop", t.Text)
		}
		ps.skip(";")
		if t.Text == "break" {
			return &Node{Kind: NdBreak, Pos: t.Pos}
		}
		return &Node{Kind: NdContinue, Pos: t.Pos}
	case ps.equal("{"):
		ps.next()
		return ps.compoundStmt()
	case t.Kind == TkKeyword && t.Text != "int" && t.Text != "void":
		ps.unsupported(t.Pos, "'"+t.Text+"'")
	}
	return ps.exprStmt()
}

// loopBody parses the statement of a loop, where break and continue apply.
func (ps *parser) loopBody() *Node {
	ps.loops++
	defer func() { ps.loops-- }()
	return ps.stmt()
}

// compound-stmt = (declaration | stmt)* "}"
func (ps *parser) compoundStmt() *Node { return ps.block(map[string]*Obj{}) }

// block parses a compound statement whose declarations go into scope.
func (ps *parser) block(scope map[string]*Obj) *Node {
	n := &Node{Kind: NdBlock, Pos: ps.tok().Pos}
	ps.scopes = append(ps.scopes, scope)
	for !ps.equal("}") {
		if ps.tok().Kind == TkEOF {
			ps.fail(ps.tok().Pos, "expected '}'")
		}
		if ps.equal("int") || ps.equal("char") || ps.tok().Kind == TkKeyword && isTypeKeyword(ps.tok().Text) {
			n.Body = append(n.Body, ps.declaration())
		} else {
			n.Body = append(n.Body, ps.stmt())
		}
	}
	ps.next()
	ps.scopes = ps.scopes[:len(ps.scopes)-1]
	return n
}

func isTypeKeyword(s string) bool {
	switch s {
	case "short", "long", "unsigned", "signed", "float", "double", "struct", "union",
		"enum", "typedef", "static", "extern", "const", "volatile", "register", "auto", "void",
		"inline", "restrict", "_Alignas", "_Atomic", "_Bool", "_Complex", "_Imaginary",
		"_Noreturn", "_Thread_local":
		return true
	}
	return false
}

// expr-stmt = expr? ";"
func (ps *parser) exprStmt() *Node {
	t := ps.tok()
	if ps.consume(";") {
		return &Node{Kind: NdBlock, Pos: t.Pos}
	}
	n := &Node{Kind: NdExprStmt, Pos: t.Pos, Lhs: ps.expr()}
	ps.skip(";")
	return n
}

// expr = assign; the comma operator is not in the subset.
func (ps *parser) expr() *Node {
	n := ps.assign()
	if ps.equal(",") {
		ps.unsupported(ps.tok().Pos, "the comma operator")
	}
	return n
}

// operators that are C but not (yet) in the subset, rejected where they
// would continue an expression.
var notYet = map[string]string{"?": "?:", "->": "->", ".": "structs"}

// compound assignment operators and the operation each applies
var assignOps = map[string]NodeKind{
	"+=": NdAdd, "-=": NdSub, "*=": NdMul, "/=": NdDiv, "%=": NdMod,
	"&=": NdBitAnd, "|=": NdBitOr, "^=": NdBitXor, "<<=": NdShl, ">>=": NdShr,
}

// assign    = logor (assign-op assign)?
// assign-op = "=" | "+=" | "-=" | "*=" | "/=" | "%=" | "&=" | "|=" | "^=" | "<<=" | ">>="
//
// As in chibicc's to_assign, x op= e is x = x op e.
func (ps *parser) assign() *Node {
	n := ps.logor()
	t := ps.tok()
	if t.Kind == TkPunct {
		if what, ok := notYet[t.Text]; ok {
			ps.unsupported(t.Pos, what)
		}
	}
	if ps.equal("=") {
		ps.next()
		ps.lvalue(n, t)
		return &Node{Kind: NdAssign, Pos: t.Pos, Lhs: n, Rhs: ps.assign()}
	}
	if op, ok := assignOps[t.Text]; ok && t.Kind == TkPunct {
		ps.next()
		ps.lvalue(n, t)
		rhs := &Node{Kind: op, Pos: t.Pos, Lhs: n, Rhs: ps.assign()}
		return &Node{Kind: NdAssign, Pos: t.Pos, Lhs: n, Rhs: rhs}
	}
	return n
}

// lvalue checks that n can be assigned to by the operator at t: a scalar
// variable or an array element (a whole array is not assignable).
func (ps *parser) lvalue(n *Node, t Token) {
	if n.Kind == NdVar && n.Var.Len > 0 {
		ps.unsupported(n.Pos, "using an array without an index (no pointers)")
	}
	if n.Kind != NdVar && n.Kind != NdIndex || n.Rvalue {
		ps.fail(t.Pos, "the left side of '%s' must be a variable", t.Text)
	}
}

// logor = logand ("||" logand)*
func (ps *parser) logor() *Node {
	n := ps.logand()
	for ps.equal("||") {
		t := ps.next()
		n = &Node{Kind: NdLogOr, Pos: t.Pos, Lhs: n, Rhs: ps.logand()}
	}
	return n
}

// logand = bitor ("&&" bitor)*
func (ps *parser) logand() *Node {
	n := ps.bitor()
	for ps.equal("&&") {
		t := ps.next()
		n = &Node{Kind: NdLogAnd, Pos: t.Pos, Lhs: n, Rhs: ps.bitor()}
	}
	return n
}

// binary parses a left-associative level: next (op next)*.
func (ps *parser) binary(next func() *Node, ops map[string]NodeKind) *Node {
	n := next()
	for {
		t := ps.tok()
		kind, ok := ops[t.Text]
		if !ok || t.Kind != TkPunct {
			return n
		}
		ps.next()
		n = &Node{Kind: kind, Pos: t.Pos, Lhs: n, Rhs: next()}
	}
}

// bitor = bitxor ("|" bitxor)*
func (ps *parser) bitor() *Node {
	return ps.binary(ps.bitxor, map[string]NodeKind{"|": NdBitOr})
}

// bitxor = bitand ("^" bitand)*
func (ps *parser) bitxor() *Node {
	return ps.binary(ps.bitand, map[string]NodeKind{"^": NdBitXor})
}

// bitand = equality ("&" equality)*
func (ps *parser) bitand() *Node {
	return ps.binary(ps.equality, map[string]NodeKind{"&": NdBitAnd})
}

// equality = relational ("==" relational | "!=" relational)*
func (ps *parser) equality() *Node {
	n := ps.relational()
	for {
		t := ps.tok()
		switch {
		case ps.consume("=="):
			n = &Node{Kind: NdEq, Pos: t.Pos, Lhs: n, Rhs: ps.relational()}
		case ps.consume("!="):
			n = &Node{Kind: NdNe, Pos: t.Pos, Lhs: n, Rhs: ps.relational()}
		default:
			return n
		}
	}
}

// relational = shift ("<" shift | "<=" shift | ">" shift | ">=" shift)*
// As in chibicc, a > b is b < a and a >= b is b <= a.
func (ps *parser) relational() *Node {
	n := ps.shift()
	for {
		t := ps.tok()
		switch {
		case ps.consume("<"):
			n = &Node{Kind: NdLt, Pos: t.Pos, Lhs: n, Rhs: ps.shift()}
		case ps.consume("<="):
			n = &Node{Kind: NdLe, Pos: t.Pos, Lhs: n, Rhs: ps.shift()}
		case ps.consume(">"):
			n = &Node{Kind: NdLt, Pos: t.Pos, Lhs: ps.shift(), Rhs: n}
		case ps.consume(">="):
			n = &Node{Kind: NdLe, Pos: t.Pos, Lhs: ps.shift(), Rhs: n}
		default:
			return n
		}
	}
}

// shift = add ("<<" add | ">>" add)*
func (ps *parser) shift() *Node {
	return ps.binary(ps.add, map[string]NodeKind{"<<": NdShl, ">>": NdShr})
}

// add = mul ("+" mul | "-" mul)*
func (ps *parser) add() *Node {
	n := ps.mul()
	for {
		t := ps.tok()
		switch {
		case ps.consume("+"):
			n = &Node{Kind: NdAdd, Pos: t.Pos, Lhs: n, Rhs: ps.mul()}
		case ps.consume("-"):
			n = &Node{Kind: NdSub, Pos: t.Pos, Lhs: n, Rhs: ps.mul()}
		default:
			return n
		}
	}
}

// mul = unary ("*" unary | "/" unary | "%" unary)*
func (ps *parser) mul() *Node {
	n := ps.unary()
	for {
		t := ps.tok()
		switch {
		case ps.consume("*"):
			n = &Node{Kind: NdMul, Pos: t.Pos, Lhs: n, Rhs: ps.unary()}
		case ps.consume("/"):
			n = &Node{Kind: NdDiv, Pos: t.Pos, Lhs: n, Rhs: ps.unary()}
		case ps.consume("%"):
			n = &Node{Kind: NdMod, Pos: t.Pos, Lhs: n, Rhs: ps.unary()}
		default:
			return n
		}
	}
}

// unary = ("+" | "-" | "!" | "~") unary
//
//	| ("++" | "--") unary
//	| postfix
func (ps *parser) unary() *Node {
	t := ps.tok()
	switch {
	case ps.consume("!"):
		return &Node{Kind: NdNot, Pos: t.Pos, Lhs: ps.unary()}
	case ps.consume("~"):
		return &Node{Kind: NdBitNot, Pos: t.Pos, Lhs: ps.unary()}
	case ps.consume("++"), ps.consume("--"):
		// ++x is x = x + 1
		n := ps.unary()
		ps.lvalue(n, t)
		return incDec(n, t)
	case ps.consume("+"):
		n := ps.unary()
		if n.Kind == NdVar || n.Kind == NdIndex {
			copied := *n
			copied.Rvalue = true
			n = &copied
		}
		return n
	case ps.consume("-"):
		return &Node{Kind: NdNeg, Pos: t.Pos, Lhs: ps.unary()}
	case ps.equal("&"), ps.equal("*"):
		ps.unsupported(t.Pos, "unary '"+t.Text+"'")
	}
	return ps.postfix()
}

// incDec builds x = x + 1 (or x - 1 for "--") for the operator at t.
func incDec(n *Node, t Token) *Node {
	kind := NdAdd
	if t.Text == "--" {
		kind = NdSub
	}
	one := &Node{Kind: NdNum, Pos: t.Pos, Val: 1}
	return &Node{Kind: NdAssign, Pos: t.Pos, Lhs: n, Rhs: &Node{Kind: kind, Pos: t.Pos, Lhs: n, Rhs: one}}
}

// postfix = primary ("[" expr "]" | "++" | "--")*
//
// As in chibicc's new_inc_dec, x++ is (x = x + 1) - 1: the hoisted
// assignment runs first, and the value is the new x minus one. An array can
// only be used with an index: without pointers, there is nothing else to do
// with it.
func (ps *parser) postfix() *Node {
	n := ps.primary()
	for ps.equal("[") || ps.equal("++") || ps.equal("--") {
		t := ps.next()
		if t.Text == "[" {
			if n.Kind != NdVar || n.Var.Len == 0 || n.Rvalue {
				ps.fail(t.Pos, "subscripted value is not an array")
			}
			n = &Node{Kind: NdIndex, Pos: t.Pos, Var: n.Var, Lhs: ps.expr()}
			ps.skip("]")
			continue
		}
		ps.lvalue(n, t)
		undo := NdSub
		if t.Text == "--" {
			undo = NdAdd
		}
		one := &Node{Kind: NdNum, Pos: t.Pos, Val: 1}
		n = &Node{Kind: undo, Pos: t.Pos, Lhs: incDec(n, t), Rhs: one, WrapChar: n.Var.Char}
	}
	if n.Kind == NdVar && n.Var.Len > 0 {
		ps.unsupported(n.Pos, "using an array without an index (no pointers)")
	}
	return n
}

// primary = "(" expr ")" | ident func-args? | num
func (ps *parser) primary() *Node {
	t := ps.tok()
	switch {
	case ps.consume("("):
		n := ps.expr()
		ps.skip(")")
		return n
	case t.Kind == TkIdent:
		ps.next()
		v := ps.findVar(t.Text)
		if ps.equal("(") {
			if v != nil {
				ps.fail(t.Pos, "'%s' is a variable, not a function", t.Text)
			}
			return ps.funcall(t)
		}
		if v == nil {
			ps.fail(t.Pos, "undefined variable '%s'", t.Text)
		}
		return &Node{Kind: NdVar, Pos: t.Pos, Var: v}
	case t.Kind == TkNum:
		ps.next()
		return &Node{Kind: NdNum, Pos: t.Pos, Val: t.Val}
	case t.Kind == TkStr:
		ps.unsupported(t.Pos, "string literals outside char array initializers")
	case t.Kind == TkKeyword:
		ps.unsupported(t.Pos, "'"+t.Text+"'")
	}
	ps.fail(t.Pos, "expected an expression")
	return nil
}

// funcall = ident "(" (assign ("," assign)*)? ")"
//
// putchar is the only library function; any other must be declared
// (defined, or a prototype) before it is called.
func (ps *parser) funcall(name Token) *Node {
	n := &Node{Kind: NdFuncall, Pos: name.Pos, Func: name.Text}
	want := 1
	if name.Text != "putchar" {
		fn, ok := ps.prog.Funcs[name.Text]
		if !ok {
			if name.Text == "main" {
				ps.unsupported(name.Pos, "calling main (recursion)")
			}
			ps.unsupported(name.Pos, "calling '"+name.Text+"' (putchar is the only library function, and other functions must be declared first)")
		}
		n.Fn, want = fn, len(fn.Params)
	}
	ps.skip("(")
	for !ps.equal(")") {
		if len(n.Args) > 0 {
			ps.skip(",")
		}
		n.Args = append(n.Args, ps.assign())
	}
	if len(n.Args) != want {
		ps.fail(ps.tok().Pos, "%s takes %d argument(s), not %d", name.Text, want, len(n.Args))
	}
	ps.next()
	return n
}

// findVar looks a name up from the innermost scope out to the globals.
func (ps *parser) findVar(name string) *Obj {
	for i := len(ps.scopes) - 1; i >= 0; i-- {
		if v, ok := ps.scopes[i][name]; ok {
			return v
		}
	}
	return ps.globals[name]
}
