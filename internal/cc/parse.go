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
	loops   int // nesting depth of loops, for break and continue
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

// program = ("int" (function-definition | global-variable))*
func (ps *parser) program() *Program {
	ps.prog = &Program{}
	ps.globals = map[string]*Obj{}
	for ps.tok().Kind != TkEOF {
		ps.declspec()
		name := ps.declarator()
		if ps.equal("(") {
			ps.function(name)
			continue
		}
		ps.globalVariable(name)
	}
	if ps.prog.Main == nil {
		ps.fail(ps.tok().Pos, "no main function")
	}
	return ps.prog
}

// declspec = "int"
func (ps *parser) declspec() {
	t := ps.tok()
	if t.Kind == TkKeyword && t.Text != "int" {
		ps.unsupported(t.Pos, "the type or specifier '"+t.Text+"' (int is the only type)")
	}
	ps.skip("int")
}

// declarator = ident; pointers and arrays are not in the subset.
func (ps *parser) declarator() Token {
	if ps.equal("*") {
		ps.unsupported(ps.tok().Pos, "pointers")
	}
	t := ps.next()
	if t.Kind != TkIdent {
		ps.fail(t.Pos, "expected a variable name")
	}
	if ps.equal("[") {
		ps.unsupported(ps.tok().Pos, "arrays")
	}
	return t
}

// function = "(" "void"? ")" "{" compound-stmt; only main.
func (ps *parser) function(name Token) {
	if name.Text != "main" {
		ps.unsupported(name.Pos, "functions other than main")
	}
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

// global-variable = (declarator ("=" constant)? ("," declarator ("=" constant)?)*)? ";"
// The first declarator has been read already.
func (ps *parser) globalVariable(name Token) {
	for {
		if _, dup := ps.globals[name.Text]; dup {
			ps.fail(name.Pos, "redefinition of '%s'", name.Text)
		}
		if name.Text == "main" && ps.prog.Main != nil {
			ps.fail(name.Pos, "redefinition of 'main' as a variable")
		}
		v := &Obj{Name: name.Text, IsGlobal: true, Pos: name.Pos}
		if ps.consume("=") {
			v.Init = ps.constant()
			if !ps.equal(",") && !ps.equal(";") {
				ps.unsupported(ps.tok().Pos, "global initializers other than integer constants")
			}
		}
		ps.globals[v.Name] = v
		ps.prog.Globals = append(ps.prog.Globals, v)
		if ps.consume(";") {
			return
		}
		ps.skip(",")
		name = ps.declarator()
	}
}

// constant = "-"? num, the initializer of a global.
func (ps *parser) constant() int32 {
	neg := ps.consume("-")
	t := ps.next()
	if t.Kind != TkNum {
		ps.unsupported(t.Pos, "global initializers other than integer constants")
	}
	if neg {
		return -t.Val
	}
	return t.Val
}

// declaration = "int" (declarator ("=" assign)? ("," declarator ("=" assign)?)*)? ";"
// It becomes a block of assignment statements for the initializers.
func (ps *parser) declaration() *Node {
	block := &Node{Kind: NdBlock, Pos: ps.tok().Pos}
	ps.declspec()
	first := true
	for !ps.consume(";") {
		if !first {
			ps.skip(",")
		}
		first = false
		name := ps.declarator()
		scope := ps.scopes[len(ps.scopes)-1]
		if _, dup := scope[name.Text]; dup {
			ps.fail(name.Pos, "redefinition of '%s'", name.Text)
		}
		v := &Obj{Name: name.Text, Pos: name.Pos}
		scope[name.Text] = v
		if !ps.equal("=") {
			continue
		}
		eq := ps.next()
		lhs := &Node{Kind: NdVar, Pos: name.Pos, Var: v}
		assign := &Node{Kind: NdAssign, Pos: eq.Pos, Lhs: lhs, Rhs: ps.assign()}
		block.Body = append(block.Body, &Node{Kind: NdExprStmt, Pos: name.Pos, Lhs: assign})
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
		if ps.equal("int") || ps.tok().Kind == TkKeyword && isTypeKeyword(ps.tok().Text) {
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
func (ps *parser) compoundStmt() *Node {
	n := &Node{Kind: NdBlock, Pos: ps.tok().Pos}
	ps.scopes = append(ps.scopes, map[string]*Obj{})
	for !ps.equal("}") {
		if ps.tok().Kind == TkEOF {
			ps.fail(ps.tok().Pos, "expected '}'")
		}
		if ps.equal("int") || ps.tok().Kind == TkKeyword && isTypeKeyword(ps.tok().Text) {
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
	case "char", "short", "long", "unsigned", "signed", "float", "double", "struct", "union",
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
var notYet = map[string]string{
	"?": "?:", "&": "&", "|": "|", "^": "^", "<<": "<<", ">>": ">>", "&=": "&=", "|=": "|=",
	"^=": "^=", "<<=": "<<=", ">>=": ">>=", "[": "arrays", "->": "->", ".": "structs",
}

// compound assignment operators and the operation each applies
var assignOps = map[string]NodeKind{"+=": NdAdd, "-=": NdSub, "*=": NdMul, "/=": NdDiv, "%=": NdMod}

// assign    = logor (assign-op assign)?
// assign-op = "=" | "+=" | "-=" | "*=" | "/=" | "%="
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

// lvalue checks that n can be assigned to by the operator at t.
func (ps *parser) lvalue(n *Node, t Token) {
	if n.Kind != NdVar || n.Rvalue {
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

// logand = equality ("&&" equality)*
func (ps *parser) logand() *Node {
	n := ps.equality()
	for ps.equal("&&") {
		t := ps.next()
		n = &Node{Kind: NdLogAnd, Pos: t.Pos, Lhs: n, Rhs: ps.equality()}
	}
	return n
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

// relational = add ("<" add | "<=" add | ">" add | ">=" add)*
// As in chibicc, a > b is b < a and a >= b is b <= a.
func (ps *parser) relational() *Node {
	n := ps.add()
	for {
		t := ps.tok()
		switch {
		case ps.consume("<"):
			n = &Node{Kind: NdLt, Pos: t.Pos, Lhs: n, Rhs: ps.add()}
		case ps.consume("<="):
			n = &Node{Kind: NdLe, Pos: t.Pos, Lhs: n, Rhs: ps.add()}
		case ps.consume(">"):
			n = &Node{Kind: NdLt, Pos: t.Pos, Lhs: ps.add(), Rhs: n}
		case ps.consume(">="):
			n = &Node{Kind: NdLe, Pos: t.Pos, Lhs: ps.add(), Rhs: n}
		default:
			return n
		}
	}
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

// unary = ("+" | "-" | "!") unary
//
//	| ("++" | "--") unary
//	| postfix
func (ps *parser) unary() *Node {
	t := ps.tok()
	switch {
	case ps.consume("!"):
		return &Node{Kind: NdNot, Pos: t.Pos, Lhs: ps.unary()}
	case ps.consume("++"), ps.consume("--"):
		// ++x is x = x + 1
		n := ps.unary()
		ps.lvalue(n, t)
		return incDec(n, t)
	case ps.consume("+"):
		n := ps.unary()
		if n.Kind == NdVar {
			copied := *n
			copied.Rvalue = true
			n = &copied
		}
		return n
	case ps.consume("-"):
		return &Node{Kind: NdNeg, Pos: t.Pos, Lhs: ps.unary()}
	case ps.equal("~"), ps.equal("&"), ps.equal("*"):
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

// postfix = primary ("++" | "--")*
//
// As in chibicc's new_inc_dec, x++ is (x = x + 1) - 1: the hoisted
// assignment runs first, and the value is the new x minus one.
func (ps *parser) postfix() *Node {
	n := ps.primary()
	for ps.equal("++") || ps.equal("--") {
		t := ps.next()
		ps.lvalue(n, t)
		undo := NdSub
		if t.Text == "--" {
			undo = NdAdd
		}
		one := &Node{Kind: NdNum, Pos: t.Pos, Val: 1}
		n = &Node{Kind: undo, Pos: t.Pos, Lhs: incDec(n, t), Rhs: one}
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
	case t.Kind == TkKeyword:
		ps.unsupported(t.Pos, "'"+t.Text+"'")
	}
	ps.fail(t.Pos, "expected an expression")
	return nil
}

// funcall = ident "(" assign ")"; putchar is the only function.
func (ps *parser) funcall(name Token) *Node {
	if name.Text != "putchar" {
		ps.unsupported(name.Pos, "calling '"+name.Text+"' (putchar is the only function)")
	}
	ps.skip("(")
	n := &Node{Kind: NdFuncall, Pos: name.Pos, Func: name.Text}
	if ps.equal(")") {
		ps.fail(ps.tok().Pos, "putchar takes one argument")
	}
	n.Args = append(n.Args, ps.assign())
	if !ps.equal(")") {
		ps.fail(ps.tok().Pos, "putchar takes one argument")
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
