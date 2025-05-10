package parser

import (
	"fmt"
	"slug/internal/ast"
	"slug/internal/lexer"
	"slug/internal/token"
	"strconv"
)

const (
	_           int = iota
	LOWEST          // assignment
	LOGICAL_OR      // logical or
	LOGICAL_AND     // logical and
	EQUALS          // ==
	COMPARISON      // > or <
	BITWISE_OR
	BITWISE_XOR
	BITWISE_AND // bitwise operators
	SHIFT       // bit shifting
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
	INDEX       // array[index]
)

var precedences = map[token.TokenType]int{
	token.ASSIGN:       LOWEST, // Assignment has lowest precedence
	token.EQ:           EQUALS,
	token.NOT_EQ:       EQUALS,
	token.LOGICAL_AND:  LOGICAL_AND,
	token.LOGICAL_OR:   LOGICAL_OR,
	token.BITWISE_AND:  BITWISE_AND,
	token.BITWISE_OR:   BITWISE_OR,
	token.BITWISE_XOR:  BITWISE_XOR,
	token.SHIFT_LEFT:   SHIFT,
	token.SHIFT_RIGHT:  SHIFT,
	token.LT:           COMPARISON,
	token.LT_EQ:        COMPARISON,
	token.GT:           COMPARISON,
	token.GT_EQ:        COMPARISON,
	token.PLUS:         SUM,
	token.MINUS:        SUM,
	token.SLASH:        PRODUCT,
	token.ASTERISK:     PRODUCT,
	token.PERCENT:      PRODUCT,
	token.APPEND_ITEM:  PRODUCT,
	token.PREPEND_ITEM: PRODUCT,
	token.PERIOD:       CALL,
	token.LPAREN:       CALL,
	token.LBRACKET:     INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	src    string // source code here
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer, source string) *Parser {
	p := &Parser{
		l:      l,
		src:    source,
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.NIL, p.parseNil)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.COMPLEMENT, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.PERCENT, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LOGICAL_AND, p.parseInfixExpression)
	p.registerInfix(token.LOGICAL_OR, p.parseInfixExpression)
	p.registerInfix(token.BITWISE_AND, p.parseInfixExpression)
	p.registerInfix(token.BITWISE_OR, p.parseInfixExpression)
	p.registerInfix(token.BITWISE_XOR, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_RIGHT, p.parseInfixExpression)
	p.registerInfix(token.SHIFT_LEFT, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.LT_EQ, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.GT_EQ, p.parseInfixExpression)
	p.registerInfix(token.APPEND_ITEM, p.parseInfixExpression)
	p.registerInfix(token.PREPEND_ITEM, p.parseInfixExpression)

	p.registerInfix(token.PERIOD, p.parseFunctionFirstCallExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) addError(message string, args ...interface{}) {
	line, col := GetLineAndColumn(p.src, p.curToken.Position)
	m := fmt.Sprintf(message, args...)
	msg := fmt.Sprintf("[%3d:%2d] %s", line, col, m)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekError(t token.TokenType) {
	// Line and column are extracted using the position of the peek token.
	p.addError("expected next token to be %s, got %s instead", t, p.peekToken.Type)
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	// Line and column are extracted using the position of the current token.
	p.addError("no prefix parse function for %s found", t)
}

// Update `expectPeek` to include line and column context when a peek error happens
func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.VAR:
		return p.parseVarStatement()
	case token.VAL:
		return p.parseValStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.IMPORT:
		return p.parseImportStatement()
	case token.NOT_IMPLEMENTED:
		return p.parseNotImplemented()
	case token.TRY:
		return p.parseTryCatchStatement()
	case token.THROW:
		return p.parseThrowStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVarStatement() *ast.VarStatement {
	stmt := &ast.VarStatement{Token: p.curToken}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE)) {
		p.addError("expected identifier, array, or hash literal after 'var'")
		return nil
	}

	// consume var
	p.nextToken()

	stmt.Pattern = p.parseMatchPattern()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseValStatement() *ast.ValStatement {
	stmt := &ast.ValStatement{Token: p.curToken}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE)) {
		p.addError("expected identifier, array, or hash literal after 'val'")
		return nil
	}

	// consume var
	p.nextToken()

	stmt.Pattern = p.parseMatchPattern()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// Enhance the error message in parseImportStatement for invalid imports
func (p *Parser) parseImportStatement() *ast.ImportStatement {
	stmt := &ast.ImportStatement{Token: p.curToken}

	// Parse the module path
	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.PathParts = append(stmt.PathParts, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})

	for p.peekTokenIs(token.PERIOD) {
		p.nextToken() // Consume '.'
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			stmt.PathParts = append(stmt.PathParts, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal})
		}
	}

	// Check for wildcard or braces with symbols
	if p.curTokenIs(token.PERIOD) {
		p.nextToken()
		if p.curTokenIs(token.ASTERISK) {
			stmt.Wildcard = true
		} else if p.curTokenIs(token.LBRACE) {
			stmt.Symbols = p.parseImportSymbols()
		}
	}

	if !stmt.Wildcard && len(stmt.Symbols) == 0 {
		p.addError("invalid import: must specify `*` or `{symbols}`")
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseImportSymbols() []*ast.ImportSymbol {
	var symbols []*ast.ImportSymbol

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		symbol := &ast.ImportSymbol{Name: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}}

		// Check for alias using "as"
		if p.peekTokenIs(token.AS) {
			p.nextToken() // Consume "as"
			p.nextToken() // Consume "as"
			symbol.Alias = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}

		symbols = append(symbols, symbol)

		// Handle comma between symbols
		if p.peekTokenIs(token.COMMA) {
			p.nextToken()
		}
	}
	p.expectPeek(token.RBRACE)

	return symbols
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) parseIdentifier() ast.Expression {
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken()
		return p.parseAssignmentExpression(ident)
	}

	return ident
}

// Modify parseIntegerLiteral to include line and column
func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.addError("could not parse %q as integer", p.curToken.Literal)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()

	if p.peekTokenIs(token.PREPEND_ITEM) {
		// prepend is right-associative
		expression.Right = p.parseExpression(precedence - 1)
	} else {
		expression.Right = p.parseExpression(precedence)
	}

	return expression
}

func (p *Parser) parseAssignmentExpression(left *ast.Identifier) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseNil() ast.Expression {
	return &ast.Nil{Token: p.curToken}
}

func (p *Parser) parseMatchExpression() ast.Expression {
	match := &ast.MatchExpression{Token: p.curToken}

	// Check if match has a value to match against
	if !p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		match.Value = p.parseExpression(LOWEST)
	} else {
		p.addError("match expression must be followed by an expression")
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		p.addError("'{' expected after match expression")
		return nil
	}

	match.Cases = []*ast.MatchCase{}

	// Skip the opening brace
	p.nextToken()

	// Parse cases until we hit the closing brace
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		matchCase := p.parseMatchCase()
		if matchCase != nil {
			match.Cases = append(match.Cases, matchCase)
		}
		p.nextToken() // Move to the next case or closing brace
	}

	return match
}

func (p *Parser) parseMatchCase() *ast.MatchCase {
	matchCase := &ast.MatchCase{Token: p.curToken}

	// Parse the pattern
	var pattern ast.MatchPattern
	if p.peekTokenIs(token.COMMA) {
		// Multi-pattern case - comma-separated list of patterns
		pattern = p.parseMultiPattern()
	} else {
		// Single-pattern case - single pattern
		pattern = p.parseMatchPattern()
	}

	if pattern == nil {
		return nil
	}
	matchCase.Pattern = pattern

	// Check for guard condition with 'if'
	if p.peekTokenIs(token.IF) {
		p.nextToken() // Consume 'if'
		p.nextToken() // Move to the guard expression
		matchCase.Guard = p.parseExpression(LOWEST)
	}

	// Expect => followed by a block statement
	if !p.expectPeek(token.ROCKET) {
		return nil
	}

	// Parse block statement or expression
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		matchCase.Body = p.parseBlockStatement()
	} else {
		// For single-expression cases
		p.nextToken()
		stmt := p.parseStatement()

		// Create a block with a single statement
		matchCase.Body = &ast.BlockStatement{
			Token:      p.curToken,
			Statements: []ast.Statement{stmt},
		}

		// Expect a semicolon at the end of the expression
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
	}

	return matchCase
}

func (p *Parser) parseMatchPattern() ast.MatchPattern {

	switch p.curToken.Type {
	case token.UNDERSCORE:
		return &ast.WildcardPattern{Token: p.curToken}
	case token.ELLIPSIS:
		pattern := ast.SpreadPattern{Token: p.curToken}
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			pattern.Value = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			pattern.Value = nil
		}
		return &pattern
	case token.IDENT:
		return &ast.IdentifierPattern{
			Token: p.curToken,
			Value: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
		}
	case token.INT, token.STRING, token.TRUE, token.FALSE, token.NIL:
		// Literal patterns (numbers, strings, booleans, nil)
		expr := p.parseExpression(LOWEST)
		return &ast.LiteralPattern{Token: p.curToken, Value: expr}
	case token.LBRACKET:
		// Array pattern
		return p.parseArrayPattern()
	case token.LBRACE:
		// Hash pattern
		return p.parseHashPattern()
	default:
		p.peekError(p.curToken.Type)
		return nil
	}
}

func (p *Parser) parseMultiPattern() ast.MatchPattern {

	expr := p.parseExpression(LOWEST)

	multiPattern := &ast.MultiPattern{
		Token:    p.curToken,
		Patterns: []ast.MatchPattern{&ast.LiteralPattern{Token: p.curToken, Value: expr}},
	}

	// Check for additional literal patterns separated by commas
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next literal
		expr = p.parseExpression(LOWEST)
		multiPattern.Patterns = append(multiPattern.Patterns, &ast.LiteralPattern{Token: p.curToken, Value: expr})
	}

	// If only one pattern, return as LiteralPattern
	if len(multiPattern.Patterns) == 1 {
		return multiPattern.Patterns[0]
	}

	return multiPattern
}

func (p *Parser) parseArrayPattern() ast.MatchPattern {
	arrayPattern := &ast.ArrayPattern{Token: p.curToken}
	arrayPattern.Elements = []ast.MatchPattern{}

	p.nextToken() // Skip '['

	// Handle empty list: `[]`
	if p.curTokenIs(token.RBRACKET) {
		return arrayPattern
	}

	for !p.curTokenIs(token.RBRACKET) {
		element := p.parseMatchPattern()
		if element == nil {
			return nil
		}

		arrayPattern.Elements = append(arrayPattern.Elements, element)

		if p.peekTokenIs(token.COMMA) {
			p.nextToken() // consume IDENT
			p.nextToken() // consume ,
		} else if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // consume IDENT
		}
	}

	return arrayPattern
}

func (p *Parser) parseHashPattern() *ast.HashPattern {
	hashPattern := &ast.HashPattern{
		Token:  p.curToken,
		Pairs:  make(map[string]ast.MatchPattern),
		Spread: false,
	}

	// Empty hash pattern
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return hashPattern
	}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		readSpread := p.curTokenIs(token.ELLIPSIS)
		if readSpread {
			value := p.parseMatchPattern()
			hashPattern.Pairs[token.ELLIPSIS] = value
			hashPattern.Spread = true
			continue
		}

		readIdent := p.curTokenIs(token.LBRACKET)
		if readIdent {
			p.nextToken() // consume the '['
		}

		key := p.parseExpression(LOWEST)

		if readIdent {
			p.expectPeek(token.RBRACKET)
		}

		_, isIdent := key.(*ast.Identifier)
		if isIdent && !readIdent {
			key = &ast.StringLiteral{
				Token: key.(*ast.Identifier).Token,
				Value: key.(*ast.Identifier).Value}
		}

		if p.peekTokenIs(token.COLON) {
			p.nextToken()
			p.nextToken()
			value := p.parseMatchPattern()
			hashPattern.Pairs[key.String()] = value
		} else {
			hashPattern.Pairs[key.String()] = &ast.IdentifierPattern{
				Token: p.curToken,
				Value: &ast.Identifier{Token: p.curToken, Value: key.String()}}
		}

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hashPattern
}

func (p *Parser) parseHashPatternPair(hp *ast.HashPattern) bool {
	// For hash patterns, keys are always identifiers
	if !p.curTokenIs(token.IDENT) {
		p.addError("expected identifier as hash pattern key, got %s", p.curToken.Type)
		return false
	}

	key := p.curToken.Literal

	// Check if this is a shorthand notation (just the key)
	if p.peekTokenIs(token.COMMA) || p.peekTokenIs(token.RBRACE) {
		// Shorthand notation - { name } is the same as { name: name }
		hp.Pairs[key] = &ast.IdentifierPattern{
			Token: p.curToken,
			Value: &ast.Identifier{Token: p.curToken, Value: key},
		}
		return true
	}

	// Otherwise, expect colon followed by a pattern
	if !p.expectPeek(token.COLON) {
		return false
	}

	p.nextToken() // Move to the pattern

	pattern := p.parseMatchPattern()
	if pattern == nil {
		return false
	}

	hp.Pairs[key] = pattern
	return true
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.ThenBranch = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		// Check if we have an "else if" construction
		if p.peekTokenIs(token.IF) {
			p.nextToken() // consume the IF token
			// Parse the else-if as an if expression and set it as the else branch
			elseIfExpression := p.parseIfExpression()
			// Wrap the else-if expression in a block statement
			elseBlock := &ast.BlockStatement{
				Token: p.curToken,
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Token:      p.curToken,
						Expression: elseIfExpression,
					},
				},
			}
			expression.ElseBranch = elseBlock
		} else if !p.expectPeek(token.LBRACE) {
			return nil
		} else {
			expression.ElseBranch = p.parseBlockStatement()
		}
	}

	return expression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.FunctionParameter {
	parameters := []*ast.FunctionParameter{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return parameters
	}

	p.nextToken()

	for {
		param := &ast.FunctionParameter{}

		if p.curTokenIs(token.ELLIPSIS) {
			// Handle variadic parameter (e.g., ...b)
			p.nextToken()
			param.IsVariadic = true
			param.Name = p.parseIdentifier().(*ast.Identifier)
			parameters = append(parameters, param)
			break // Variadic must be the last parameter.
		}

		if p.peekTokenIs(token.ASSIGN) {
			param.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken() // consume identifier
			p.nextToken() // consume =
			param.Default = p.parseExpression(LOWEST)
		} else {
			param.Name = p.parseIdentifier().(*ast.Identifier)
		}

		parameters = append(parameters, param)

		// Stop if no more parameters
		if !p.peekTokenIs(token.COMMA) {
			break
		}
		p.nextToken() // Consume comma
		p.nextToken() // Move to the next token
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return parameters
}

// Modify parseFunctionFirstCallExpression to include enhanced context
func (p *Parser) parseFunctionFirstCallExpression(left ast.Expression) ast.Expression {
	if !p.expectPeek(token.IDENT) {
		p.addError("expected function identifier after '.', got %s instead", p.peekToken.Type)
		return nil
	}

	function := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		args := p.parseExpressionList(token.RPAREN)
		args = append([]ast.Expression{left}, args...)
		return &ast.CallExpression{
			Token:     function.Token,
			Function:  function,
			Arguments: args,
		}
	} else {
		return &ast.IndexExpression{
			Token: function.Token,
			Left:  left,
			Index: &ast.StringLiteral{Token: function.Token, Value: function.Value},
		}
	}
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	// Construct the IndexExpression node
	expr := &ast.IndexExpression{
		Token: p.curToken, // The '[' token
		Left:  left,
	}

	p.nextToken() // Consume '[' and parse the index or slice

	// Check for slice syntax start:end or start:end:step
	if p.curTokenIs(token.COLON) || p.peekTokenIs(token.COLON) {
		expr.Index = p.parseSliceExpression()
	} else {
		expr.Index = p.parseExpression(LOWEST)
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return expr
}

func (p *Parser) parseSliceExpression() ast.Expression {
	// Create the slice node
	slice := &ast.SliceExpression{
		Token: p.curToken,
	}

	if !p.peekTokenIs(token.RBRACKET) {

		if !p.curTokenIs(token.COLON) {
			slice.Start = p.parseExpression(LOWEST)
			p.nextToken()
		}

		if !p.peekTokenIs(token.RBRACKET) {
			p.nextToken()

			if !p.curTokenIs(token.COLON) {
				slice.End = p.parseExpression(LOWEST)
			}

			if !p.peekTokenIs(token.RBRACKET) {
				p.nextToken()

				if !p.peekTokenIs(token.RBRACKET) {
					p.nextToken()
					slice.Step = p.parseExpression(LOWEST)
				} else if !p.curTokenIs(token.COLON) {
					slice.Step = p.parseExpression(LOWEST)
				}
			}
		}
	}
	return slice
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		readIdent := p.curTokenIs(token.LBRACKET)
		if readIdent {
			p.nextToken() // consume the '['
		}

		key := p.parseExpression(LOWEST)

		if readIdent {
			p.expectPeek(token.RBRACKET)
		}

		_, isIdent := key.(*ast.Identifier)
		if isIdent && !readIdent {
			key = &ast.StringLiteral{Token: key.(*ast.Identifier).Token, Value: key.(*ast.Identifier).Value}
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		hash.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseTryCatchStatement() *ast.TryCatchStatement {
	stmt := &ast.TryCatchStatement{Token: p.curToken}

	// Parse the try block
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.TryBlock = p.parseBlockStatement()

	// Parse the catch block
	if !p.expectPeek(token.CATCH) {
		return nil
	}
	stmt.CatchToken = p.curToken

	// this seems hacky, i wonder if it's idiomatic Go...
	expression := p.parseMatchExpression().(*ast.MatchExpression)

	// add a default case to the CatchBlock expression to rethrow value
	// todo: maybe we can check for the default case already in the expression?
	expression.Cases = append(expression.Cases, &ast.MatchCase{
		Token:   p.curToken,
		Pattern: &ast.SpreadPattern{Token: p.curToken},
		Body: &ast.BlockStatement{
			Token: p.curToken,
			Statements: []ast.Statement{
				&ast.ThrowStatement{
					Token: p.curToken,
					Value: expression.Value,
				},
			},
		},
	})
	stmt.CatchBlock = expression

	return stmt
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	throw := &ast.ThrowStatement{Token: p.curToken}

	// Advance to the expression after `throw`
	p.nextToken()
	ident := p.parseIdentifier()

	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	var ef *ast.HashLiteral = nil
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume RPAREN
		ef = p.parseHashLiteral().(*ast.HashLiteral)
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if ef == nil {
		pairs := make(map[ast.Expression]ast.Expression)
		pairs[&ast.StringLiteral{Token: p.curToken, Value: "type"}] = &ast.StringLiteral{Token: p.curToken, Value: ident.String()}

		throw.Value = &ast.HashLiteral{
			Token: p.curToken,
			Pairs: pairs,
		}
	} else {
		ef.Pairs[&ast.StringLiteral{Token: p.curToken, Value: "type"}] = &ast.StringLiteral{Token: p.curToken, Value: ident.String()}
		throw.Value = ef
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return throw
}

func (p *Parser) parseNotImplemented() *ast.ThrowStatement {
	throw := &ast.ThrowStatement{Token: p.curToken}
	pairs := make(map[ast.Expression]ast.Expression)
	pairs[&ast.StringLiteral{Token: p.curToken, Value: "type"}] = &ast.StringLiteral{Token: p.curToken, Value: "NotImplementedError"}

	throw.Value = &ast.HashLiteral{
		Token: p.curToken,
		Pairs: pairs,
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return throw
}

// todo put this in a utils file
func GetLineAndColumn(src string, pos int) (line int, column int) {
	line = 1
	column = 1
	for i, char := range src {
		if i == pos {
			break
		}
		if char == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return
}
