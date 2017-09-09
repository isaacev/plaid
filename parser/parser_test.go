package parser

import (
	"plaid/lexer"
	"testing"
)

func TestPeekTokenIsNot(t *testing.T) {
	expectTokenToMatch := func(source string, tests ...lexer.Type) {
		p := makeParser(source)
		got := p.peekTokenIsNot(tests[0], tests[1:]...)

		if got != false {
			t.Errorf("Expected %t, got %t\n", false, got)
		}
	}

	expectTokenToNotMatch := func(source string, tests ...lexer.Type) {
		p := makeParser(source)
		got := p.peekTokenIsNot(tests[0], tests[1:]...)

		if got != true {
			t.Errorf("Expected %t, got %t\n", false, got)
		}
	}

	expectTokenToMatch("abc", lexer.Ident)
	expectTokenToMatch("abc", lexer.Number, lexer.Ident)
	expectTokenToMatch("123", lexer.Number, lexer.Ident)
	expectTokenToMatch("", lexer.EOF, lexer.Error)
	expectTokenToMatch("#", lexer.EOF, lexer.Error)
	expectTokenToNotMatch("abc", lexer.EOF, lexer.Error)
}

func TestRegisterPrecedence(t *testing.T) {
	parser := makeParser("")
	parser.registerPrecedence(lexer.Ident, Sum)

	got := parser.precedenceTable[lexer.Ident]
	if got != Sum {
		t.Errorf("Expected %v, got %v\n", Sum, got)
	}
}

func TestRegisterPrefix(t *testing.T) {
	parser := makeParser("")
	parser.registerPrefix(lexer.Ident, parseIdent)

	if _, exists := parser.prefixParseFuncs[lexer.Ident]; exists == false {
		t.Error("Expected prefix parse function, got nothing")
	}
}

func TestRegisterPostfix(t *testing.T) {
	parser := makeParser("")
	parser.registerPostfix(lexer.Plus, parseInfix, Sum)

	if _, exists := parser.postfixParseFuncs[lexer.Plus]; exists == false {
		t.Error("Expected postfix parse function, got nothing")
	}

	level, exists := parser.precedenceTable[lexer.Plus]
	if (exists == false) || (level != Sum) {
		t.Errorf("Expected Plus precedence to be %v, got %v\n", Sum, level)
	}
}

func TestPeekPrecedence(t *testing.T) {
	parser := makeParser("+*")
	parser.registerPrecedence(lexer.Plus, Sum)

	level := parser.peekPrecedence()
	if level != Sum {
		t.Errorf("Expected Plus precedence to be %v, got %v\n", Sum, level)
	}

	parser.lexer.Next()

	level = parser.peekPrecedence()
	if level != Lowest {
		t.Errorf("Expected Star precedence to be %v, got %v\n", Lowest, level)
	}
}

func TestParseFunction(t *testing.T) {
	prog, err := Parse("let a := 123; let b := 456;")
	expectNoErrors(t, "(let a 123)\n(let b 456)", prog, err)
}

func TestParseProgram(t *testing.T) {
	p := makeParser("let a := 123; let b := 456;")
	loadGrammar(p)
	prog, err := parseProgram(p)
	expectNoErrors(t, "(let a 123)\n(let b 456)", prog, err)

	p = makeParser("let a = 123; let b := 456;")
	loadGrammar(p)
	prog, err = parseProgram(p)
	expectAnError(t, "(1:7) expected :=", prog, err)
}

func TestParseStmt(t *testing.T) {
	expectStmt := func(source string, ast string) {
		parser := makeParser(source)
		loadGrammar(parser)
		stmt, err := parseStmt(parser)
		expectNoErrors(t, ast, stmt, err)
	}

	expectStmtError := func(source string, msg string) {
		parser := makeParser(source)
		loadGrammar(parser)
		stmt, err := parseStmt(parser)
		expectAnError(t, msg, stmt, err)
	}

	expectStmt("let a := 123;", "(let a 123)")
	expectStmtError("123 + 456", "(1:1) expected start of statement")
}

func TestParseStmtBlock(t *testing.T) {
	expectStmtBlock := func(source string, ast string) {
		parser := makeParser(source)
		loadGrammar(parser)
		block, err := parseStmtBlock(parser)
		expectNoErrors(t, ast, block, err)
	}

	expectStmtBlockError := func(source string, msg string) {
		parser := makeParser(source)
		loadGrammar(parser)
		block, err := parseStmtBlock(parser)
		expectAnError(t, msg, block, err)
	}

	expectStmtBlock("{ let a := 123; }", "{\n  (let a 123)}")
	expectStmtBlockError("let a := 123; }", "(1:1) expected left brace")
	expectStmtBlockError("{ let a := 123 }", "(1:16) expected semicolon")
	expectStmtBlockError("{ let a := 123;", "(1:15) expected right brace")
}

func TestParseDeclarationStmt(t *testing.T) {
	p := makeParser("let a := 123;")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err := parseDeclarationStmt(p)
	expectNoErrors(t, "(let a 123)", stmt, err)

	p = makeParser("a := 123;")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err = parseDeclarationStmt(p)
	expectAnError(t, "(1:1) expected LET keyword", stmt, err)

	p = makeParser("let 0 := 123;")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err = parseDeclarationStmt(p)
	expectAnError(t, "(1:5) expected identifier", stmt, err)

	p = makeParser("let a = 123;")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err = parseDeclarationStmt(p)
	expectAnError(t, "(1:7) expected :=", stmt, err)

	p = makeParser("let a :=;")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err = parseDeclarationStmt(p)
	expectAnError(t, "(1:9) unexpected symbol", stmt, err)

	p = makeParser("let a := 123")
	p.registerPrefix(lexer.Number, parseNumber)
	stmt, err = parseDeclarationStmt(p)
	expectAnError(t, "(1:12) expected semicolon", stmt, err)
}

func TestParseTypeSig(t *testing.T) {
	expectTypeSig(t, parseTypeSig, "Int", "Int")
	expectTypeSig(t, parseTypeSig, "[Int]", "[Int]")
	expectTypeSig(t, parseTypeSig, "Int?", "Int?")
	expectTypeSig(t, parseTypeSig, "Int??", "Int??")
	expectTypeSig(t, parseTypeSig, "Int???", "Int???")
	expectTypeSig(t, parseTypeSig, "[Int?]", "[Int?]")
	expectTypeSig(t, parseTypeSig, "[Int?]?", "[Int?]?")
	expectTypeSig(t, parseTypeSig, "[Int]?", "[Int]?")

	expectTypeSigError(t, parseTypeSig, "[?]", "(1:2) unexpected symbol")
	expectTypeSigError(t, parseTypeSig, "[Int", "(1:4) expected right bracket")
	expectTypeSigError(t, parseTypeSig, "?", "(1:1) unexpected symbol")
}

func TestParseTypeIdent(t *testing.T) {
	expectTypeSig(t, parseTypeIdent, "Int", "Int")
	expectTypeSigError(t, parseTypeIdent, "123", "(1:1) expected identifier")
}

func TestParseTypeList(t *testing.T) {
	expectTypeSig(t, parseTypeList, "[Int]", "[Int]")
	expectTypeSigError(t, parseTypeList, "Int]", "(1:1) expected left bracket")
	expectTypeSigError(t, parseTypeList, "[?]", "(1:2) unexpected symbol")
}

func TestParseTypeOptional(t *testing.T) {
	expectTypeOpt := func(fn typeSigParser, source string, ast string) {
		p := makeParser(source)
		loadGrammar(p)
		sig, err := fn(p)
		expectNoErrors(t, sig.String(), sig, err)
		sig, err = parseTypeOptional(p, sig)
		expectNoErrors(t, ast, sig, err)
	}

	expectTypeOptError := func(fn typeSigParser, source string, msg string) {
		p := makeParser(source)
		loadGrammar(p)
		sig, err := fn(p)
		expectNoErrors(t, sig.String(), sig, err)
		sig, err = parseTypeOptional(p, sig)
		expectAnError(t, msg, sig, err)
	}

	expectTypeOpt(parseTypeIdent, "Int?", "Int?")
	expectTypeOpt(parseTypeList, "[Int]?", "[Int]?")

	expectTypeOptError(parseTypeIdent, "Int", "(1:3) expected question mark")
}

func TestParseExpr(t *testing.T) {
	p := makeParser("+a")
	p.registerPrefix(lexer.Plus, parsePrefix)
	p.registerPrefix(lexer.Ident, parseIdent)
	expr, err := parseExpr(p, Lowest)
	expectNoErrors(t, "(+ a)", expr, err)

	p = makeParser("+")
	p.registerPrefix(lexer.Plus, parsePrefix)
	p.registerPrefix(lexer.Ident, parseIdent)
	expr, err = parseExpr(p, Lowest)
	expectAnError(t, "(1:1) unexpected symbol", expr, err)

	p = makeParser("a + b + c")
	p.registerPostfix(lexer.Plus, parseInfix, Sum)
	p.registerPrefix(lexer.Ident, parseIdent)
	expr, err = parseExpr(p, Lowest)
	expectNoErrors(t, "(+ (+ a b) c)", expr, err)

	p = makeParser("a + b * c")
	p.registerPostfix(lexer.Plus, parseInfix, Sum)
	p.registerPostfix(lexer.Star, parseInfix, Product)
	p.registerPrefix(lexer.Ident, parseIdent)
	expr, err = parseExpr(p, Lowest)
	expectNoErrors(t, "(+ a (* b c))", expr, err)

	p = makeParser("a +")
	p.registerPostfix(lexer.Plus, parseInfix, Sum)
	p.registerPrefix(lexer.Ident, parseIdent)
	expr, err = parseExpr(p, Lowest)
	expectAnError(t, "(1:3) unexpected symbol", expr, err)
}

func TestParseInfix(t *testing.T) {
	parser := makeParser("a + b")
	parser.registerPostfix(lexer.Plus, parseInfix, Sum)
	parser.registerPrefix(lexer.Ident, parseIdent)

	var left Expr
	var expr Expr
	var err error

	if left, err = parseIdent(parser); err != nil {
		t.Errorf("Expected no errors, got %v\n", err)
	}

	expr, err = parseInfix(parser, left)
	expectNoErrors(t, "(+ a b)", expr, err)

	parser = makeParser("a +")
	parser.registerPostfix(lexer.Plus, parseInfix, Sum)
	parser.registerPrefix(lexer.Ident, parseIdent)

	if left, err = parseIdent(parser); err != nil {
		t.Errorf("Expected no errors, got %v\n", err)
	}

	expr, err = parseInfix(parser, left)
	expectAnError(t, "(1:3) unexpected symbol", expr, err)
}

func TestParsePostfix(t *testing.T) {
	parser := makeParser("a+")
	parser.registerPostfix(lexer.Plus, parsePostfix, Postfix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	var left Expr
	var expr Expr
	var err error

	if left, err = parseIdent(parser); err != nil {
		t.Errorf("Expected no errors, got %v\n", err)
	}

	expr, err = parsePostfix(parser, left)
	expectNoErrors(t, "(+ a)", expr, err)
}

func TestParsePrefix(t *testing.T) {
	parser := makeParser("+a")
	parser.registerPrefix(lexer.Plus, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err := parsePrefix(parser)
	expectNoErrors(t, "(+ a)", expr, err)

	parser = makeParser("+")
	parser.registerPrefix(lexer.Plus, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err = parsePrefix(parser)
	expectAnError(t, "(1:1) unexpected symbol", expr, err)
}

func TestParseGroup(t *testing.T) {
	parser := makeParser("(a)")
	parser.registerPrefix(lexer.ParenL, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err := parseGroup(parser)
	expectNoErrors(t, "a", expr, err)

	parser = makeParser("a)")
	parser.registerPrefix(lexer.ParenL, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err = parseGroup(parser)
	expectAnError(t, "(1:1) expected left paren", expr, err)

	parser = makeParser("(")
	parser.registerPrefix(lexer.ParenL, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err = parseGroup(parser)
	expectAnError(t, "(1:1) unexpected symbol", expr, err)

	parser = makeParser("(a")
	parser.registerPrefix(lexer.ParenL, parsePrefix)
	parser.registerPrefix(lexer.Ident, parseIdent)

	expr, err = parseGroup(parser)
	expectAnError(t, "(1:2) expected right paren", expr, err)
}

func TestParseIdent(t *testing.T) {
	parser := makeParser("abc")

	expr, err := parseIdent(parser)
	expectNoErrors(t, "abc", expr, err)

	parser = makeParser("123")

	expr, err = parseIdent(parser)
	expectAnError(t, "(1:1) expected identifier", expr, err)
}

func TestParseNumber(t *testing.T) {
	parser := makeParser("123")

	expr, err := parseNumber(parser)
	expectNoErrors(t, "123", expr, err)

	parser = makeParser("abc")

	expr, err = parseNumber(parser)
	expectAnError(t, "(1:1) expected number literal", expr, err)

	loc := lexer.Loc{Line: 1, Col: 1}
	expr, err = evalNumber(lexer.Token{Type: lexer.Number, Lexeme: "abc", Loc: loc})
	expectAnError(t, "(1:1) malformed number literal", expr, err)
}

type typeSigParser func(p *Parser) (TypeSig, error)

func expectTypeSig(t *testing.T, fn typeSigParser, source string, ast string) {
	p := makeParser(source)
	loadGrammar(p)
	sig, err := fn(p)
	expectNoErrors(t, ast, sig, err)
}

func expectTypeSigError(t *testing.T, fn typeSigParser, source string, msg string) {
	p := makeParser(source)
	loadGrammar(p)
	sig, err := fn(p)
	expectAnError(t, msg, sig, err)
}

func expectNoErrors(t *testing.T, ast string, node Node, err error) {
	if err != nil {
		t.Errorf("Expected no errors, got '%s'\n", err)
	} else {
		expectAST(t, ast, node)
	}
}

func expectAnError(t *testing.T, msg string, node Node, err error) {
	if err == nil {
		t.Errorf("Expected an error, got %s\n", node)
	} else if err.Error() != msg {
		t.Errorf("Expected '%s', got '%s'\n", msg, err)
	}
}

func expectAST(t *testing.T, ast string, got Node) {
	if ast != got.String() {
		t.Errorf("Expected '%s', got '%s'\n", ast, got)
	}
}
