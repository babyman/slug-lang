package parser

import (
	"bytes"
	"fmt"
	"math"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/svc/lexer"
	"slug/internal/token"
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
	CALL_CHAIN  // 10 /> abs
	CALL        // myFunction(X)
	INDEX       // list[index]
)

var precedences = map[token.TokenType]int{
	token.ASSIGN:              LOWEST, // Assignment has lowest precedence
	token.EQ:                  EQUALS,
	token.NOT_EQ:              EQUALS,
	token.LOGICAL_AND:         LOGICAL_AND,
	token.LOGICAL_OR:          LOGICAL_OR,
	token.BITWISE_AND:         BITWISE_AND,
	token.BITWISE_OR:          BITWISE_OR,
	token.BITWISE_XOR:         BITWISE_XOR,
	token.SHIFT_LEFT:          SHIFT,
	token.SHIFT_RIGHT:         SHIFT,
	token.LT:                  COMPARISON,
	token.LT_EQ:               COMPARISON,
	token.GT:                  COMPARISON,
	token.GT_EQ:               COMPARISON,
	token.PLUS:                SUM,
	token.MINUS:               SUM,
	token.SLASH:               PRODUCT,
	token.ASTERISK:            PRODUCT,
	token.PERCENT:             PRODUCT,
	token.APPEND_ITEM:         PRODUCT,
	token.PREPEND_ITEM:        PRODUCT,
	token.PERIOD:              CALL,
	token.LPAREN:              CALL,
	token.INTERPOLATION_START: CALL,
	token.CALL_CHAIN:          CALL_CHAIN,
	token.LBRACKET:            INDEX,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	tokenizer   lexer.Tokenizer
	src         string // source code here
	errors      []string
	pendingTags []*ast.Tag

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l lexer.Tokenizer, source string) *Parser {
	p := &Parser{
		tokenizer: l,
		src:       source,
		errors:    []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.NIL, p.parseNil)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.COMPLEMENT, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseListLiteral)
	p.registerPrefix(token.LBRACE, p.parseMapLiteral)
	p.registerPrefix(token.MATCH, p.parseMatchExpression)
	p.registerPrefix(token.VAR, p.parseVarStatement)
	p.registerPrefix(token.VAL, p.parseValStatement)

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

	p.registerInfix(token.CALL_CHAIN, p.parseCallChainExpression)
	p.registerInfix(token.PERIOD, p.parseDotIdentifierToIndexExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.INTERPOLATION_START, p.parseInterpolationExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.tokenizer.NextToken()
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

	for !p.curTokenIs(token.EOF) && !p.curTokenIs(token.ILLEGAL) {
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
	case token.FOREIGN:
		return p.parseForeignFunctionDeclaration()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.NOT_IMPLEMENTED:
		return p.parseNotImplemented()
	case token.DEFER:
		return p.parseDeferStatement()
	case token.TRY:
		return p.parseTryCatchStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.AT:
		tag := p.parseTag()
		p.pendingTags = append(p.pendingTags, tag)
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseVarStatement() ast.Expression {

	varExp := &ast.VarExpression{
		Token: p.curToken,
	}

	if p.pendingTags != nil {
		varExp.Tags = p.pendingTags
		p.pendingTags = nil
	}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) ||
		p.peekTokenIs(token.LBRACE) || p.peekTokenIs(token.MATCH_KEYS_EXACT)) {
		p.addError("expected identifier, list, or map literal after 'var'")
		return nil
	}

	// consume var
	p.nextToken()

	varExp.Pattern = p.parseMatchPattern()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	//if varExp.Tags != nil {
	//	fmt.Printf("Var adding tags: %v %v\n", varExp, len(varExp.Tags))
	//}
	varExp.Value = p.parseExpression(LOWEST)

	return varExp
}

func (p *Parser) parseValStatement() ast.Expression {
	valExp := &ast.ValExpression{
		Token: p.curToken,
	}

	if p.pendingTags != nil {
		valExp.Tags = p.pendingTags
		p.pendingTags = nil
	}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE)) {
		p.addError("expected identifier, list, or map literal after 'val'")
		return nil
	}

	// consume var
	p.nextToken()

	valExp.Pattern = p.parseMatchPattern()

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	valExp.Value = p.parseExpression(LOWEST)

	//if valExp.Tags != nil {
	//	fmt.Printf("Val adding tags: %v %v\n", valExp, len(valExp.Tags))
	//}

	return valExp
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

// Modify parseNumberLiteral to include line and column
func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Token: p.curToken}

	value, err := dec64.FromString(p.curToken.Literal)
	if err != nil {
		p.addError("could not parse %q as number", p.curToken.Literal)
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

// No commented out code needed here - replaced with the new implementation

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
	case token.NUMBER, token.STRING, token.TRUE, token.FALSE, token.NIL:
		// Literal patterns (numbers, strings, booleans, nil)
		expr := p.parseExpression(LOWEST)
		return &ast.LiteralPattern{Token: p.curToken, Value: expr}
	case token.LBRACKET:
		// List pattern
		return p.parseListPattern()
	case token.LBRACE:
		// Map pattern
		return p.parseMapPattern()
	case token.MATCH_KEYS_EXACT:
		// Map pattern
		return p.parseMapPattern()
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

func (p *Parser) parseListPattern() ast.MatchPattern {
	listPattern := &ast.ListPattern{Token: p.curToken}
	listPattern.Elements = []ast.MatchPattern{}

	p.nextToken() // Skip '['

	// Handle empty list: `[]`
	if p.curTokenIs(token.RBRACKET) {
		return listPattern
	}

	for !p.curTokenIs(token.RBRACKET) {
		element := p.parseMatchPattern()
		if element == nil {
			return nil
		}

		listPattern.Elements = append(listPattern.Elements, element)

		if p.peekTokenIs(token.COMMA) {
			p.nextToken() // consume IDENT
			p.nextToken() // consume ,
		} else if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // consume IDENT
		}
	}

	return listPattern
}

func (p *Parser) parseMapPattern() *ast.MapPattern {
	mapPattern := &ast.MapPattern{
		Token:     p.curToken,
		Pairs:     make(map[string]ast.MatchPattern),
		Spread:    false,
		Exact:     false,
		SelectAll: false,
	}

	// Empty map pattern
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return mapPattern
	}

	if p.curTokenIs(token.MATCH_KEYS_EXACT) {
		mapPattern.Exact = true
	}

	if p.peekTokenIs(token.ASTERISK) {
		p.nextToken()
		p.expectPeek(token.RBRACE)
		mapPattern.SelectAll = true
		return mapPattern
	}

	for !p.peekTokenIs(token.MATCH_KEYS_CLOSE) && !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		readSpread := p.curTokenIs(token.ELLIPSIS)
		if readSpread {
			if mapPattern.Exact {
				p.addError("spread not allowed in exact match")
				return nil
			} else {
				value := p.parseMatchPattern()
				mapPattern.Pairs[token.ELLIPSIS] = value
				mapPattern.Spread = true
				continue
			}
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
			mapPattern.Pairs[key.String()] = value
		} else {
			mapPattern.Pairs[key.String()] = &ast.IdentifierPattern{
				Token: p.curToken,
				Value: &ast.Identifier{Token: p.curToken, Value: key.String()}}
		}

		if p.peekTokenIs(token.MATCH_KEYS_CLOSE) || p.peekTokenIs(token.RBRACE) {
			// ok
		} else if !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if mapPattern.Exact {
		if !p.expectPeek(token.MATCH_KEYS_CLOSE) {
			return nil
		}
	} else if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return mapPattern
}

func (p *Parser) parseMapPatternPair(hp *ast.MapPattern) bool {
	// For map patterns, keys are always identifiers
	if !p.curTokenIs(token.IDENT) {
		p.addError("expected identifier as map pattern key, got %s", p.curToken.Type)
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

func (p *Parser) parseInterpolationExpression(left ast.Expression) ast.Expression {

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: "+",
		Left:     left,
	}
	p.nextToken()

	expression.Right = p.parseExpression(LOWEST)

	if !p.expectPeek(token.INTERPOLATION_END) {
		return nil
	}

	if p.peekTokenIs(token.STRING) {
		p.nextToken()
		expression = &ast.InfixExpression{
			Token:    p.curToken,
			Operator: "+",
			Left:     expression,
			Right:    p.parseStringLiteral(),
		}
	}

	return expression
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
	lit.Signature = p.generateSignature(lit.Parameters)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	// Analyze function body for tail calls
	setTailCallFlags(lit)

	return lit
}

// setTailCallFlags analyzes a function literal and marks call expressions in tail position
func setTailCallFlags(fn *ast.FunctionLiteral) {
	if fn.Body == nil || len(fn.Body.Statements) == 0 {
		return
	}

	// Check for tail calls in the function body
	hasTailCall := checkTailCallsInBlock(fn.Body)
	fn.HasTailCall = hasTailCall
}

// checkTailCallsInBlock checks for tail calls in a block statement
func checkTailCallsInBlock(block *ast.BlockStatement) bool {
	if len(block.Statements) == 0 {
		return false
	}

	// Only the last statement in a block can contain a tail call
	lastStmt := block.Statements[len(block.Statements)-1]
	return checkTailCallsInStatement(lastStmt)
}

// checkTailCallsInStatement checks for tail calls in a statement
func checkTailCallsInStatement(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		// In a return statement, the returned expression might be a tail call
		return markTailCall(s.ReturnValue)

	case *ast.ExpressionStatement:
		// In an expression statement at the end of a block, the expression might be a tail call
		return markTailCall(s.Expression)

	default:
		return false
	}
}

// markTailCall marks call expressions as tail calls and returns whether a tail call was found
func markTailCall(expr ast.Expression) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		// This is a direct tail call
		e.IsTailCall = true
		return true

	case *ast.IfExpression:
		// An if expression has tail calls if both branches have tail calls in their final statements
		thenHasTail := checkTailCallsInBlock(e.ThenBranch)
		elseHasTail := false
		if e.ElseBranch != nil {
			elseHasTail = checkTailCallsInBlock(e.ElseBranch)
		}
		return thenHasTail || elseHasTail

	case *ast.MatchExpression:
		// A match expression has tail calls if any of its cases have tail calls
		hasAnyTailCall := false
		for _, matchCase := range e.Cases {
			if matchCase.Body != nil && checkTailCallsInBlock(matchCase.Body) {
				hasAnyTailCall = true
			}
		}
		return hasAnyTailCall

	default:
		return false
	}
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

		// Collect tags (e.g., @int, @str)
		for p.curTokenIs(token.AT) {
			tag := p.parseTag()
			param.Tags = append(param.Tags, tag)
			p.nextToken()
		}

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

func (p *Parser) parseCallChainExpression(left ast.Expression) ast.Expression {
	precedence := p.curPrecedence()
	p.nextToken()

	right := p.parseExpression(precedence)
	if right == nil {
		p.addError("expected function after '/>'")
		return nil
	}

	// If right is a call, prepend left to its arguments.
	if call, ok := right.(*ast.CallExpression); ok {
		call.Arguments = append([]ast.Expression{left}, call.Arguments...)
		return call
	}

	// Otherwise, call right with only the chained left value.
	return &ast.CallExpression{
		Token:     p.curToken,
		Function:  right,
		Arguments: []ast.Expression{left},
	}
}

func (p *Parser) parseDotIdentifierToIndexExpression(left ast.Expression) ast.Expression {
	if !p.expectPeek(token.IDENT) {
		p.addError("expected identifier after '.', got %s instead", p.peekToken.Type)
		return nil
	}

	mapKey := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return &ast.IndexExpression{
		Token: mapKey.Token,
		Left:  left,
		Index: &ast.StringLiteral{Token: mapKey.Token, Value: mapKey.Value},
	}
}

func (p *Parser) generateSignature(params []*ast.FunctionParameter) ast.FSig {

	minP := len(params)
	maxP := len(params)
	variadic := false
	var tags bytes.Buffer

	if maxP > 0 && params[maxP-1].IsVariadic {
		maxP = math.MaxInt
		minP -= 1
		variadic = true
	}

	for i := minP - 1; i >= 0; i-- {
		param := params[i]
		if param.Default != nil {
			minP = i
		} else {
			break
		}
	}

	for _, param := range params {
		for _, tag := range param.Tags {
			tags.WriteString(tag.String())
		}
		tags.WriteString("|")
	}

	sig := ast.FSig{
		Tags:       tags.String(),
		Min:        minP,
		Max:        maxP,
		IsVariadic: variadic,
	}

	return sig
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseExpressionList(token.RPAREN)
	return exp
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()

	// Handle spread syntax here
	if p.curTokenIs(token.ELLIPSIS) {
		p.nextToken()
		spreadExpr := &ast.SpreadExpression{
			Token: p.curToken,
			Value: p.parseExpression(LOWEST),
		}
		list = append(list, spreadExpr)
	} else {
		list = append(list, p.parseExpression(LOWEST))
	}

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()

		// Handle spread syntax on subsequent arguments
		if p.curTokenIs(token.ELLIPSIS) {
			p.nextToken()
			spreadExpr := &ast.SpreadExpression{
				Token: p.curToken,
				Value: p.parseExpression(LOWEST),
			}
			list = append(list, spreadExpr)
		} else {
			list = append(list, p.parseExpression(LOWEST))
		}
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseListLiteral() ast.Expression {
	list := &ast.ListLiteral{Token: p.curToken}

	list.Elements = p.parseExpressionList(token.RBRACKET)

	return list
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	// Construct the IndexExpression node
	expr := &ast.IndexExpression{
		Token: p.curToken, // The '[' token
		Left:  left,
	}

	sliceParams := p.parseIndexExpressionList()
	if len(sliceParams) == 0 {
		return nil
	} else if len(sliceParams) == 1 && sliceParams[0] != nil {
		expr.Index = sliceParams[0]
	} else {
		slice := &ast.SliceExpression{
			Token: p.curToken,
		}
		if len(sliceParams) == 3 && sliceParams[2] != nil {
			slice.Step = sliceParams[2]
		}
		if len(sliceParams) >= 2 && sliceParams[1] != nil {
			slice.End = sliceParams[1]
		}
		if len(sliceParams) >= 1 && sliceParams[0] != nil {
			slice.Start = sliceParams[0]
		}
		expr.Index = slice
	}

	return expr
}

func (p *Parser) parseIndexExpressionList() []ast.Expression {
	var list []ast.Expression

	// Advance past '['
	p.nextToken()

	// Parse individual components of the slice (up to 3 parts)
	slice := false
	i := 0
	for i < 3 {
		if p.curTokenIs(token.COLON) { // Handle ':'
			// Append nil for an omitted part
			slice = true
			list = append(list, nil)
			if p.peekTokenIs(token.RBRACKET) {
				break
			}
		} else if p.curTokenIs(token.RBRACKET) { // End of slice
			break
		} else {
			// Parse an expression for a part
			list = append(list, p.parseExpression(LOWEST))

			// Check for the next delimiter or end
			if p.peekTokenIs(token.COLON) && i <= 1 {
				slice = true
				p.nextToken()
			}
			if p.peekTokenIs(token.RBRACKET) {
				break
			}
		}
		p.nextToken()
		i++
	}

	if slice {
		for len(list) < 3 {
			list = append(list, nil)
		}
	}

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return list
}

func (p *Parser) parseMapLiteral() ast.Expression {
	mapLit := &ast.MapLiteral{Token: p.curToken}
	mapLit.Pairs = make(map[ast.Expression]ast.Expression)

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

		mapLit.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return mapLit
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
	var ef *ast.MapLiteral = nil
	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // consume RPAREN
		ef = p.parseMapLiteral().(*ast.MapLiteral)
	}
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	if ef == nil {
		pairs := make(map[ast.Expression]ast.Expression)
		pairs[&ast.StringLiteral{Token: p.curToken, Value: "type"}] = &ast.StringLiteral{Token: p.curToken, Value: ident.String()}

		throw.Value = &ast.MapLiteral{
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

	throw.Value = &ast.MapLiteral{
		Token: p.curToken,
		Pairs: pairs,
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return throw
}

func (p *Parser) parseForeignFunctionDeclaration() *ast.ForeignFunctionDeclaration {
	foreignFunction := &ast.ForeignFunctionDeclaration{
		Token: p.curToken,
	}

	if p.pendingTags != nil {
		foreignFunction.Tags = p.pendingTags
		p.pendingTags = nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	foreignFunction.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	if !p.expectPeek(token.FUNCTION) {
		return nil
	}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	foreignFunction.Parameters = p.parseFunctionParameters()
	foreignFunction.Signature = p.generateSignature(foreignFunction.Parameters)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return foreignFunction
}

func (p *Parser) parseDeferStatement() *ast.DeferStatement {
	stmt := &ast.DeferStatement{Token: p.curToken} // Current token is 'defer'
	p.nextToken()

	if p.curTokenIs(token.LBRACE) {
		stmt.Call = p.parseBlockStatement()
	} else {
		stmt.Call = p.parseExpressionStatement()
	}

	return stmt
}

func (p *Parser) parseTag() *ast.Tag {
	annotation := &ast.Tag{Token: p.curToken}
	p.nextToken() // Consume '@'

	// Expect identifier for the annotation name
	if !p.curTokenIs(token.IDENT) {
		p.addError("expected annotation name after '@', got %s", p.curToken.Literal)
		return nil
	}
	annotation.Name = "@" + p.curToken.Literal

	// Parse optional argument list
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
		args := p.parseExpressionList(token.RPAREN)
		annotation.Args = args
	}

	return annotation
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
