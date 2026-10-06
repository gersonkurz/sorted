package cc

import "fmt"

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
	strs    map[string]*Obj // string literals by content, so equal ones share their cells
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
	ps.strs = map[string]*Obj{}
	for ps.tok().Kind != TkEOF {
		var base *Type // nil for void
		if !ps.consume("void") {
			base = ps.declspec()
		} else if ps.equal("*") {
			ps.unsupported(ps.tok().Pos, "void pointers")
		}
		d := ps.declarator(base)
		if ps.equal("(") {
			if d.array {
				ps.unsupported(d.name.Pos, "functions returning arrays")
			}
			if d.name.Text == "main" {
				if d.ty == nil || d.ty.Kind != TyInt {
					ps.fail(d.name.Pos, "main must return int")
				}
				ps.function(d.name)
			} else {
				ps.otherFunction(d.name, d.ty)
			}
			continue
		}
		if base == nil {
			ps.fail(d.name.Pos, "a variable cannot be void")
		}
		ps.globalVariable(d)
	}
	if ps.prog.Main == nil {
		ps.fail(ps.tok().Pos, "no main function")
	}
	return ps.prog
}

// declspec = "int" | "char"
func (ps *parser) declspec() *Type {
	t := ps.tok()
	if t.Kind == TkKeyword && t.Text != "int" && t.Text != "char" {
		ps.unsupported(t.Pos, "the type or specifier '"+t.Text+"' (int and char are the only types)")
	}
	if ps.consume("char") {
		return tyChar
	}
	ps.skip("int")
	return tyInt
}

// decl is a declarator: a name and its type, and for an array its length (0
// when it is left to the initializer).
type decl struct {
	name  Token
	ty    *Type // nil for void (a function's result)
	array bool
	len   int
	lpos  Pos // where the length is, for errors
}

// declarator = "*"* ident ("[" num? "]")?
//
// base is the declspec's type. Arrays of arrays, and declarators in
// parentheses (pointers to arrays, function pointers), are not in the subset.
func (ps *parser) declarator(base *Type) decl {
	ty := base
	for ps.equal("*") {
		ty = pointerTo(ty)
		ps.next()
	}
	if ps.equal("(") {
		ps.unsupported(ps.tok().Pos, "declarators in parentheses (pointers to arrays, function pointers)")
	}
	t := ps.next()
	if t.Kind != TkIdent {
		ps.fail(t.Pos, "expected a variable name")
	}
	d := decl{name: t, ty: ty}
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
	if ty != nil {
		d.ty = arrayOf(ty, d.len)
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
// param         = declspec "*"* ident? ("[" num? "]")?
//
// Names may be left out in a prototype. An array parameter is a pointer, as
// in C. ret is the result type, nil for void. A prototype declares the
// function for calls before its definition; the definition must agree with
// it.
func (ps *parser) otherFunction(name Token, ret *Type) {
	if _, dup := ps.globals[name.Text]; dup {
		ps.fail(name.Pos, "redefinition of '%s' as a function", name.Text)
	}
	fn := &Function{Name: name.Text, Ret: ret, Void: ret == nil, Char: ret != nil && ret.Kind == TyChar, Pos: name.Pos}
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
		ty := ps.declspec()
		for ps.consume("*") {
			ty = pointerTo(ty)
		}
		if ps.equal("(") {
			ps.unsupported(ps.tok().Pos, "declarators in parentheses (pointers to arrays, function pointers)")
		}
		var pname Token
		if ps.tok().Kind == TkIdent {
			pname = ps.next()
		}
		if ps.consume("[") { // int a[] and int a[10] are int *a
			if !ps.equal("]") {
				ps.constant("array lengths other than integer constants")
			}
			ps.skip("]")
			if ps.equal("[") {
				ps.unsupported(ps.tok().Pos, "arrays of arrays")
			}
			ty = pointerTo(ty)
		}
		fn.Params = append(fn.Params, &Obj{Name: pname.Text, Ty: ty, Char: ty.Kind == TyChar, Pos: pname.Pos})
		names = append(names, pname)
	}
	ps.skip(")")
	prev, declared := ps.prog.Funcs[name.Text]
	if declared {
		if (prev.Ret == nil) != (ret == nil) || ret != nil && !sameType(prev.Ret, ret) || len(prev.Params) != len(fn.Params) {
			ps.fail(name.Pos, "conflicting declarations of '%s'", name.Text)
		}
		for i := range fn.Params {
			if !sameType(prev.Params[i].Ty, fn.Params[i].Ty) {
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
func (ps *parser) newObj(d decl, global bool) *Obj {
	elem := d.ty
	if d.array {
		elem = d.ty.Base
	}
	return &Obj{Name: d.name.Text, IsGlobal: global, Ty: d.ty, Char: elem.Kind == TyChar, Len: d.len, Pos: d.name.Pos}
}

// global-variable = (declarator ("=" global-init)? ("," declarator ("=" global-init)?)*)? ";"
// global-init     = init | "{" init ("," init)* ","? "}" | string
//
// An init is a constant expression, or for a pointer an address: 0, a string
// literal, or the address of a global (&g, &a[2], a, a + 2). The first
// declarator has been read already.
func (ps *parser) globalVariable(d decl) {
	base := d.ty
	for base.Kind == TyPtr || base.Kind == TyArray {
		base = base.Base
	}
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
		v := ps.newObj(d, true)
		if ps.consume("=") {
			elem := v.Ty
			if d.array {
				elem = v.Ty.Base
			}
			item := func() ginit { return ps.initializer(elem) }
			var items []ginit
			if d.array {
				items = arrayInit(ps, v, d, item, func(c int32) ginit { return ginit{val: c} })
			} else {
				items = []ginit{item()}
			}
			for i, it := range items {
				v.Init = append(v.Init, it.val)
				if it.ref != nil {
					if v.InitRef == nil {
						v.InitRef = make([]*Obj, len(items))
					}
					v.InitRef[i] = it.ref
				}
			}
			if !ps.equal(",") && !ps.equal(";") {
				ps.unsupported(ps.tok().Pos, "global initializers other than constants and addresses")
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
		d = ps.declarator(base)
	}
}

// ginit is one initial value of a global: a number, or the address of ref
// plus val.
type ginit struct {
	val int32
	ref *Obj
}

// initializer parses one initial value of a global of type t: a constant
// expression, or for a pointer 0 or an address.
func (ps *parser) initializer(t *Type) ginit {
	pos := ps.tok().Pos
	n := ps.logor()
	ps.typed(n)
	if v, ok := Fold(n); ok && (t.IsInteger() || v == 0) {
		return ginit{val: v}
	}
	if t.IsInteger() && n.Ty.IsInteger() {
		ps.unsupported(pos, "global initializers other than constants and addresses")
	}
	ps.assignable(t, n, pos, "the initializer")
	ref, off, ok := addrConst(n)
	if !ok {
		ps.unsupported(pos, "global initializers other than constants and addresses")
	}
	return ginit{val: off, ref: ref}
}

// addrConst evaluates the address of a global plus a constant offset: &g,
// &a[k], a (an array), and those plus or minus a constant.
func addrConst(n *Node) (*Obj, int32, bool) {
	switch n.Kind {
	case NdAddr:
		switch x := n.Lhs; x.Kind {
		case NdVar:
			return x.Var, 0, true
		case NdIndex:
			k, ok := Fold(x.Lhs)
			return x.Var, k, ok
		}
	case NdAdd, NdSub:
		if ref, off, ok := addrConst(n.Lhs); ok {
			k, okK := Fold(n.Rhs)
			if n.Kind == NdSub {
				k = -k
			}
			return ref, off + k, okK
		}
		if ref, off, ok := addrConst(n.Rhs); ok && n.Kind == NdAdd {
			k, okK := Fold(n.Lhs)
			return ref, off + k, okK
		}
	}
	return nil, 0, false
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
		v.Ty.Len = v.Len
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
	base := ps.declspec()
	first := true
	for !ps.consume(";") {
		if !first {
			ps.skip(",")
		}
		first = false
		d := ps.declarator(base)
		scope := ps.scopes[len(ps.scopes)-1]
		if _, dup := scope[d.name.Text]; dup {
			ps.fail(d.name.Pos, "redefinition of '%s'", d.name.Text)
		}
		v := ps.newObj(d, false)
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
			ps.typed(n)
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
			pos := ps.tok().Pos
			n.Lhs = ps.expr()
			ret := tyInt // main
			if ps.fn != nil {
				ret = ps.fn.Ret
			}
			ps.assignable(ret, n.Lhs, pos, "the return value")
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

// expr = assign; the comma operator is not in the subset. The expression
// comes back typed (see typed).
func (ps *parser) expr() *Node {
	n := ps.assign()
	if ps.equal(",") {
		ps.unsupported(ps.tok().Pos, "the comma operator")
	}
	ps.typed(n)
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
// variable, an array element, or *p (a whole array is not assignable).
func (ps *parser) lvalue(n *Node, t Token) {
	if decayed(n) {
		ps.fail(n.Pos, "an array cannot be assigned to, only its elements")
	}
	if n.Kind != NdVar && n.Kind != NdIndex && n.Kind != NdDeref || n.Rvalue {
		ps.fail(t.Pos, "the left side of '%s' cannot be assigned to", t.Text)
	}
}

// decayed reports whether n is an array used as a value, which is the
// address of its first element.
func decayed(n *Node) bool {
	return n.Kind == NdAddr && n.Lhs.Kind == NdVar && n.Lhs.Var.Ty.Kind == TyArray
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

// unary = ("+" | "-" | "!" | "~" | "&" | "*") unary
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
		ps.integer(n, "the operand of unary '+'")
		if n.Kind == NdVar || n.Kind == NdIndex || n.Kind == NdDeref {
			copied := *n
			copied.Rvalue = true
			n = &copied
		}
		return n
	case ps.consume("-"):
		return &Node{Kind: NdNeg, Pos: t.Pos, Lhs: ps.unary()}
	case ps.consume("&"):
		n := ps.unary()
		if decayed(n) {
			ps.unsupported(t.Pos, "pointers to arrays (&array; the array itself is the address of its first element)")
		}
		if n.Kind != NdVar && n.Kind != NdIndex && n.Kind != NdDeref || n.Rvalue {
			ps.fail(t.Pos, "cannot take the address of a value, only of a variable or an element")
		}
		return &Node{Kind: NdAddr, Pos: t.Pos, Lhs: n}
	case ps.consume("*"):
		n := &Node{Kind: NdDeref, Pos: t.Pos, Lhs: ps.unary()}
		ps.typed(n)
		return n
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
// a[i] on an array variable is an NdIndex; on anything else, a pointer, it
// is *(p + i), as chibicc does for all of them. As in chibicc's new_inc_dec,
// x++ is (x = x + 1) - 1: the hoisted assignment runs first, and the value
// is the new x minus one. An array used as a value decays to the address of
// its first element.
func (ps *parser) postfix() *Node {
	n := ps.primary()
	for ps.equal("[") || ps.equal("++") || ps.equal("--") {
		t := ps.next()
		if t.Text == "[" {
			if n.Kind == NdVar && n.Var.Ty.Kind == TyArray && !n.Rvalue {
				n = &Node{Kind: NdIndex, Pos: t.Pos, Var: n.Var, Lhs: ps.expr()}
			} else {
				if ps.typed(n).Kind != TyPtr {
					ps.fail(t.Pos, "subscripted value is not an array or a pointer")
				}
				sum := &Node{Kind: NdAdd, Pos: t.Pos, Lhs: n, Rhs: ps.expr()}
				n = &Node{Kind: NdDeref, Pos: t.Pos, Lhs: sum}
			}
			ps.skip("]")
			ps.typed(n)
			continue
		}
		n = decay(n)
		ps.lvalue(n, t)
		undo := NdSub
		if t.Text == "--" {
			undo = NdAdd
		}
		one := &Node{Kind: NdNum, Pos: t.Pos, Val: 1}
		n = &Node{Kind: undo, Pos: t.Pos, Lhs: incDec(n, t), Rhs: one, WrapChar: ps.typed(n).Kind == TyChar}
	}
	return decay(n)
}

// decay turns an array variable used as a value into the address of its
// first element; anything else stays as it is.
func decay(n *Node) *Node {
	if n.Kind == NdVar && n.Var.Ty.Kind == TyArray {
		return &Node{Kind: NdAddr, Pos: n.Pos, Lhs: n}
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
		return ps.stringLiteral()
	case t.Kind == TkKeyword:
		ps.unsupported(t.Pos, "'"+t.Text+"'")
	}
	ps.fail(t.Pos, "expected an expression")
	return nil
}

// stringLiteral reads a string literal (adjacent ones are joined) and
// returns the anonymous char array that holds it, NUL included, as a global.
// Equal literals share one array.
func (ps *parser) stringLiteral() *Node {
	t := ps.tok()
	var str []byte
	for ps.tok().Kind == TkStr {
		str = append(str, ps.next().Str...)
	}
	str = append(str, 0)
	v, ok := ps.strs[string(str)]
	if !ok {
		v = &Obj{Name: fmt.Sprintf(".str%d", len(ps.strs)), IsGlobal: true, Ty: arrayOf(tyChar, len(str)), Char: true, Len: len(str), Pos: t.Pos}
		for _, c := range str {
			v.Init = append(v.Init, int32(int8(c)))
		}
		ps.strs[string(str)] = v
		ps.prog.Globals = append(ps.prog.Globals, v)
	}
	return &Node{Kind: NdVar, Pos: t.Pos, Var: v}
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
