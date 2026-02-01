package parser

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/lexer"
	"slug/internal/token"
	"slug/internal/util"
	"strings"
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
	LIST_CONCAT // +: and :+
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
	token.APPEND_ITEM:         LIST_CONCAT,
	token.PREPEND_ITEM:        LIST_CONCAT,
	token.CALL_CHAIN:          CALL_CHAIN,
	token.COPY:                CALL,
	token.PERIOD:              CALL,
	token.LPAREN:              CALL,
	token.INTERPOLATION_START: CALL,
	token.LBRACKET:            INDEX,
	token.LBRACE:              CALL,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	tokenizer       lexer.Tokenizer
	Path            string
	src             string // source code here
	errors          []string
	pendingTags     []*ast.Tag
	pendingDoc      string
	hasPendingDoc   bool
	moduleDoc       string
	hasModuleDoc    bool
	seenMeaningful  bool
	scopeDepth      int
	allowStructInit bool

	curToken   token.Token
	peekToken  token.Token
	peek2Token token.Token // NEW: 2nd lookahead

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l lexer.Tokenizer, path, source string) *Parser {
	p := &Parser{
		tokenizer:       l,
		Path:            path,
		src:             source,
		errors:          []string{},
		allowStructInit: true,
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.NIL, p.parseNil)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.SYMBOL, p.parseSymbolLiteral)
	p.registerPrefix(token.COLON, p.parseSymbolLiteralFromColon)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.BYTES, p.parseBytesLiteral)
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
	p.registerPrefix(token.SELECT, p.parseSelectExpression)
	p.registerPrefix(token.VAR, p.parseVarStatement)
	p.registerPrefix(token.VAL, p.parseValStatement)
	p.registerPrefix(token.RECUR, p.parseRecurExpression)
	p.registerPrefix(token.NURSERY, p.parseNurseryExpression)
	p.registerPrefix(token.SPAWN, p.parseSpawnExpression)
	p.registerPrefix(token.STRUCT, p.parseStructSchemaExpression)

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
	p.registerInfix(token.COPY, p.parseStructCopyExpression)
	p.registerInfix(token.PERIOD, p.parseDotIdentifierToIndexExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.LBRACE, p.parseStructInitExpression)
	p.registerInfix(token.INTERPOLATION_START, p.parseInterpolationExpression)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.peek2Token
	p.peek2Token = p.tokenizer.NextToken()

	// check if the token we just grabbed is illegal
	if p.curToken.Type == token.ILLEGAL {
		p.addErrorAt(p.curToken.Position, "lexer error at position %d: %s", p.curToken.Position, p.curToken.Literal)
	}
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	// Line and column are extracted using the position of the current token.
	p.addErrorAt(p.curToken.Position, "no prefix parse function for '%s' found", t)
}

func (p *Parser) peekError(t token.TokenType) {
	// Line and column are extracted using the position of the peek token.
	p.addErrorAt(p.peekToken.Position, "expected next token to be %s, got %s instead", t, p.peekToken.Type)
}

// addErrorAt reports an error at the given absolute position in the source.
func (p *Parser) addErrorAt(pos int, message string, args ...interface{}) {
	line, col := util.GetLineAndColumn(p.src, pos)
	m := fmt.Sprintf(message, args...)

	// Build the error message in the new format
	var errorMsg bytes.Buffer

	errorMsg.WriteString(fmt.Sprintf("\nParseError: %s\n", m))
	errorMsg.WriteString(fmt.Sprintf("    --> %s:%d:%d\n", p.Path, line, col))

	// Get context lines (2 lines before, the error line, and potentially lines after)
	lines := util.GetContextLines(p.src, line, col)
	errorMsg.WriteString(lines)

	p.errors = append(p.errors, errorMsg.String())
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
	program := &ast.Program{Statements: []ast.Statement{}}

	p.skipStatementSeparators()

	for !p.curTokenIs(token.EOF) && !p.curTokenIs(token.ILLEGAL) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
			if p.scopeDepth == 0 {
				p.seenMeaningful = true
				if p.hasPendingDoc && !isDocAttachStatement(stmt) {
					p.clearPendingDoc()
				}
			}
		}

		// Move forward at least once, then skip separators.
		p.nextToken()
		p.skipStatementSeparators()
	}

	program.ModuleDoc = p.moduleDoc
	program.HasModuleDoc = p.hasModuleDoc

	p.validateStructSchemaUsage(program)

	return program
}

func (p *Parser) skipStatementSeparators() {
	for {
		if p.curTokenIs(token.SEMICOLON) {
			if p.scopeDepth == 0 && p.hasPendingDoc {
				p.clearPendingDoc()
			}
			p.nextToken()
			continue
		}
		if p.curTokenIs(token.NEWLINE) {
			// In statement position, NEWLINE is always a separator.
			// If a newline was a continuation, parseExpression would have eaten it.
			p.nextToken()
			continue
		}
		break
	}
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.NEWLINE, token.SEMICOLON:
		// separators, not statements
		return nil
	case token.DOC_COMMENT:
		p.handleDocComment()
		return nil
	case token.FOREIGN:
		stmt := p.parseForeignFunctionDeclaration()
		if p.scopeDepth == 0 {
			p.seenMeaningful = true
			if p.hasPendingDoc && !isDocAttachStatement(stmt) {
				p.clearPendingDoc()
			}
		}
		return stmt
	case token.RETURN:
		return p.parseReturnStatement()
	case token.NOT_IMPLEMENTED:
		return p.parseNotImplemented()
	case token.DEFER:
		return p.parseDeferStatement()
	case token.THROW:
		return p.parseThrowStatement()
	case token.AT:
		tag := p.parseTag()
		p.pendingTags = append(p.pendingTags, tag)
		if p.scopeDepth == 0 {
			p.seenMeaningful = true
		}
		return nil
	default:
		stmt := p.parseExpressionStatement()
		if p.scopeDepth == 0 {
			p.seenMeaningful = true
			if p.hasPendingDoc && !isDocAttachStatement(stmt) {
				p.clearPendingDoc()
			}
		}
		return stmt
	}
}

func (p *Parser) handleDocComment() {
	if p.scopeDepth != 0 {
		return
	}

	doc := p.curToken.Literal
	if !p.seenMeaningful && !p.hasModuleDoc && p.isModuleDocCandidate() {
		p.moduleDoc = doc
		p.hasModuleDoc = true
		p.seenMeaningful = true
		return
	}

	p.pendingDoc = doc
	p.hasPendingDoc = true
	if !p.seenMeaningful {
		p.seenMeaningful = true
	}
}

func (p *Parser) isModuleDocCandidate() bool {
	if !p.peekTokenIs(token.NEWLINE) {
		return false
	}
	return strings.Count(p.peekToken.Literal, "\n") >= 2
}

func (p *Parser) clearPendingDoc() {
	p.pendingDoc = ""
	p.hasPendingDoc = false
}

func isDocAttachStatement(stmt ast.Statement) bool {
	if _, ok := stmt.(*ast.ForeignFunctionDeclaration); ok {
		return true
	}
	exprStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok || exprStmt.Expression == nil {
		return false
	}
	switch exprStmt.Expression.(type) {
	case *ast.ValExpression, *ast.VarExpression:
		return true
	default:
		return false
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
	if p.scopeDepth == 0 && p.hasPendingDoc {
		varExp.Doc = p.pendingDoc
		varExp.HasDoc = true
		p.clearPendingDoc()
	}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) ||
		p.peekTokenIs(token.LBRACE) || p.peekTokenIs(token.MATCH_KEYS_EXACT)) {
		p.addErrorAt(p.curToken.Position, "expected identifier, list, or map literal after 'var'")
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
	if p.scopeDepth == 0 && p.hasPendingDoc {
		valExp.Doc = p.pendingDoc
		valExp.HasDoc = true
		p.clearPendingDoc()
	}

	if !(p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.LBRACE)) {
		p.addErrorAt(p.curToken.Position, "expected identifier, list, or map literal after 'val'")
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
	p.skipLeadingNewlines()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) peekIsExpressionTerminator(precedence int) bool {
	if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.EOF) || p.peekTokenIs(token.RBRACE) {
		return true
	}
	if p.peekTokenIs(token.NEWLINE) && p.newlineTerminates() {
		return true
	}
	return false
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	p.skipLeadingNewlines()

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for {
		// 1) Stop on real terminators
		if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.EOF) || p.peekTokenIs(token.RBRACE) {
			return leftExp
		}

		// 2) NEWLINE: either terminator or whitespace
		if p.peekTokenIs(token.NEWLINE) {
			if p.newlineTerminates() {
				return leftExp
			}
			// continuation newline: consume it and restart loop
			p.nextToken() // cur becomes NEWLINE
			continue
		}

		if !p.allowStructInit && p.peekTokenIs(token.LBRACE) {
			return leftExp
		}

		// 3) Normal Pratt precedence gate
		if precedence >= p.peekPrecedence() {
			return leftExp
		}

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken() // advance onto operator (e.g. '/>')
		leftExp = infix(leftExp)
	}
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

func (p *Parser) parseSymbolLiteral() ast.Expression {
	name := strings.TrimPrefix(p.curToken.Literal, ":")
	return &ast.SymbolLiteral{Token: p.curToken, Value: name}
}

func (p *Parser) parseSymbolLiteralFromColon() ast.Expression {
	if !p.peekTokenIsSymbolName() && !p.peekTokenIs(token.STRING) {
		p.peekError(token.IDENT)
		return nil
	}
	p.nextToken()
	if p.curToken.Type == token.STRING {
		return &ast.SymbolLiteral{Token: p.curToken, Value: p.curToken.Literal}
	}
	return &ast.SymbolLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func tokenIsSymbolName(t token.TokenType) bool {
	switch t {
	case token.IDENT,
		token.NIL,
		token.TRUE,
		token.FALSE,
		token.FOREIGN,
		token.FUNCTION,
		token.VAL,
		token.VAR,
		token.IF,
		token.ELSE,
		token.MATCH,
		token.RETURN,
		token.RECUR,
		token.THROW,
		token.DEFER,
		token.ONSUCCESS,
		token.ONERROR,
		token.STRUCT,
		token.COPY,
		token.NURSERY,
		token.SPAWN,
		token.LIMIT:
		return true
	default:
		return false
	}
}

func (p *Parser) peekTokenIsSymbolName() bool {
	return tokenIsSymbolName(p.peekToken.Type)
}

func (p *Parser) curTokenIsSymbolName() bool {
	return tokenIsSymbolName(p.curToken.Type)
}

// Modify parseNumberLiteral to include line and column
func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Token: p.curToken}

	// Support hexadecimal integer literals starting with 0x or 0X
	if len(p.curToken.Literal) > 2 && (p.curToken.Literal[0] == '0') && (p.curToken.Literal[1] == 'x') {
		// Decode hex digits (after the 0x/0X prefix)
		bytesVal, err := hex.DecodeString(p.curToken.Literal[2:])
		if err != nil {
			p.addErrorAt(p.curToken.Position, "could not parse %q as hex number", p.curToken.Literal)
			return nil
		}
		// Convert the resulting bytes into an unsigned integer (big-endian)
		var u uint64
		for _, b := range bytesVal {
			u = (u << 8) | uint64(b)
		}
		// Store as a decimal value
		lit.Value = dec64.FromUint(u)
		return lit
	}

	value, err := dec64.FromString(p.curToken.Literal)
	if err != nil {
		p.addErrorAt(p.curToken.Position, "could not parse %q as number", p.curToken.Literal)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBytesLiteral() ast.Expression {
	lit := &ast.BytesLiteral{Token: p.curToken}
	if p.curToken.Literal == "" {
		lit.Value = []byte{}
	} else {
		value, err := hex.DecodeString(p.curToken.Literal)
		if err != nil {
			p.addErrorAt(p.curToken.Position, "could not parse %q as bytes", p.curToken.Literal)
			return nil
		}
		lit.Value = value
	}
	return lit
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
	// If it's a prefix expression (match value { ... }), we expect a value.
	// If it's part of a pipeline (val /> match { ... }), the value will be injected later.
	if !p.peekTokenIs(token.LBRACE) {
		p.nextToken()
		prevAllowStructInit := p.allowStructInit
		p.allowStructInit = false
		match.Value = p.parseExpression(LOWEST)
		p.allowStructInit = prevAllowStructInit
	}

	if !p.expectPeek(token.LBRACE) {
		p.addErrorAt(match.Token.Position, "'{' expected after match expression")
		return nil
	}

	match.Cases = []*ast.MatchCase{}

	// Skip the opening brace
	p.nextToken()
	p.skipCaseSeparators() // allow blank lines after '{'

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		matchCase := p.parseMatchCase()
		if matchCase != nil {
			match.Cases = append(match.Cases, matchCase)
		}

		// Move forward and skip any NEWLINE/; between cases
		p.nextToken()
		p.skipCaseSeparators()
	}

	return match
}

func (p *Parser) parseSelectExpression() ast.Expression {
	selectExpr := &ast.SelectExpression{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		p.addErrorAt(selectExpr.Token.Position, "'{' expected after select expression")
		return nil
	}

	selectExpr.Cases = []*ast.SelectCase{}

	p.nextToken()
	p.skipCaseSeparators()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		selectCase := p.parseSelectCase()
		if selectCase != nil {
			selectExpr.Cases = append(selectExpr.Cases, selectCase)
		}

		p.nextToken()
		p.skipCaseSeparators()
	}

	return selectExpr
}

func (p *Parser) skipCaseSeparators() {
	for p.curTokenIs(token.NEWLINE) || p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
}

func (p *Parser) parseSelectCase() *ast.SelectCase {
	selectCase := &ast.SelectCase{Token: p.curToken}

	switch {
	case p.curToken.Type == token.IDENT && p.curToken.Literal == "recv":
		selectCase.Kind = ast.SelectRecv
		p.nextToken()
		selectCase.Channel = p.parseExpression(CALL_CHAIN)
		if selectCase.Channel == nil {
			return nil
		}
	case p.curToken.Type == token.IDENT && p.curToken.Literal == "send":
		selectCase.Kind = ast.SelectSend
		p.nextToken()
		selectCase.Channel = p.parseExpression(CALL_CHAIN)
		if selectCase.Channel == nil {
			return nil
		}
		if !p.expectPeek(token.COMMA) {
			return nil
		}
		p.nextToken()
		selectCase.Value = p.parseExpression(CALL_CHAIN)
		if selectCase.Value == nil {
			return nil
		}
	case p.curToken.Type == token.IDENT && p.curToken.Literal == "after":
		selectCase.Kind = ast.SelectAfter
		p.nextToken()
		selectCase.After = p.parseExpression(CALL_CHAIN)
		if selectCase.After == nil {
			return nil
		}
	case p.curToken.Type == token.IDENT && p.curToken.Literal == "await":
		selectCase.Kind = ast.SelectAwait
		p.nextToken()
		selectCase.Await = p.parseExpression(CALL_CHAIN)
		if selectCase.Await == nil {
			return nil
		}
	case p.curToken.Type == token.UNDERSCORE:
		selectCase.Kind = ast.SelectDefault
	default:
		p.addErrorAt(p.curToken.Position, "expected select case, got %s", p.curToken.Type)
		return nil
	}

	if p.peekTokenIs(token.CALL_CHAIN) {
		p.nextToken()
		p.nextToken()
		p.skipLeadingNewlines()
		selectCase.Handler = p.parseExpression(LOWEST)
		if selectCase.Handler == nil {
			p.addErrorAt(p.curToken.Position, "select case handler expected after '/>'")
			return nil
		}
	}

	return selectCase
}

func (p *Parser) parseMatchCase() *ast.MatchCase {
	matchCase := &ast.MatchCase{Token: p.curToken}

	// Parse the pattern
	var pattern ast.MatchPattern
	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.AT) {
		name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		if !p.expectPeek(token.AT) {
			return nil
		}
		p.nextToken()
		inner := p.parseMatchPattern()
		if inner == nil {
			return nil
		}
		pattern = &ast.BindingPattern{
			Token:   name.Token,
			Name:    name,
			Pattern: inner,
		}
	} else if p.peekTokenIs(token.COMMA) {
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
		// For single-statement cases (expression statement OR throw/return/etc)
		p.nextToken() // move to first token of body (may be NEWLINE)
		p.skipLeadingNewlines()

		stmt := p.parseStatement()
		if stmt == nil {
			// If the body line is empty, it's a real syntax error (match arms require a body)
			p.addErrorAt(p.curToken.Position, "match case body expected after '=>', got %s", p.curToken.Type)
			return nil
		}

		matchCase.Body = &ast.BlockStatement{
			Token: matchCase.Token,
			Statements: []ast.Statement{
				stmt,
			},
		}

		// case terminator: ; OR NEWLINE OR } (outer loop handles })
		if p.peekTokenIs(token.SEMICOLON) || p.peekTokenIs(token.NEWLINE) {
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

	case token.BITWISE_XOR:
		// Pinned identifier pattern: ^name
		// Restriction: atomic only (must be IDENT, no expressions)
		if !p.peekTokenIs(token.IDENT) {
			p.addErrorAt(p.curToken.Position, "pinned identifier must be followed by an identifier (e.g. ^name)")
			return nil
		}
		p.nextToken() // consume IDENT
		return &ast.PinnedIdentifierPattern{
			Token: p.curToken,
			Value: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
		}

	case token.IDENT:
		if p.peekTokenIs(token.LBRACE) {
			schema := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			return p.parseStructPattern(schema)
		}
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
		p.addErrorAt(p.curToken.Position, "unexpected token in match pattern: %s", p.curToken.Type)
		return nil
	}
}

func (p *Parser) parseMultiPattern() ast.MatchPattern {
	// Multi-pattern: comma-separated list of patterns.
	//
	// Important restriction:
	// Alternatives must be NON-BINDING, otherwise this becomes ambiguous (which alternative bound what?).
	// So we allow literals, pinned identifiers, wildcards, and destructuring patterns that themselves don't bind.
	//
	// This enables: `nil, [] => ...` and `0, ^p1, ^p2, 3 => ...`

	startTok := p.curToken

	var isNonBinding func(mp ast.MatchPattern) bool
	isNonBinding = func(mp ast.MatchPattern) bool {
		switch pt := mp.(type) {
		case *ast.BindingPattern:
			return false
		case *ast.WildcardPattern:
			return true
		case *ast.LiteralPattern:
			return true
		case *ast.PinnedIdentifierPattern:
			return true

		case *ast.IdentifierPattern:
			// `x` would bind; disallow in multi-pattern alternatives
			return false

		case *ast.SpreadPattern:
			// `...t` would bind; disallow
			// (If you later want to allow bare `...` without binding, this is where you’d permit pt.Value == nil.)
			return false

		case *ast.ListPattern:
			for _, el := range pt.Elements {
				if !isNonBinding(el) {
					return false
				}
			}
			return true

		case *ast.MapPattern:
			for _, entry := range pt.Pairs {
				if !isNonBinding(entry.Pattern) {
					return false
				}
			}
			if pt.Spread != nil && !isNonBinding(pt.Spread) {
				return false
			}
			return true

		case *ast.MultiPattern:
			// Nested multi-pattern is not expected here; treat as invalid to keep grammar simple.
			return false

		default:
			return false
		}
	}

	parseAlt := func() ast.MatchPattern {
		alt := p.parseMatchPattern()
		if alt == nil {
			return nil
		}
		if !isNonBinding(alt) {
			p.addErrorAt(p.curToken.Position, "multi-pattern alternatives must not introduce bindings (use literals, ^pinned identifiers, or non-binding destructuring like [])")
			return nil
		}
		return alt
	}

	first := parseAlt()
	if first == nil {
		return nil
	}

	multi := &ast.MultiPattern{
		Token:    startTok,
		Patterns: []ast.MatchPattern{first},
	}

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume comma
		p.nextToken() // move to next alternative

		next := parseAlt()
		if next == nil {
			return nil
		}
		multi.Patterns = append(multi.Patterns, next)
	}

	if len(multi.Patterns) == 1 {
		return multi.Patterns[0]
	}
	return multi
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

		// Enforce: `...` must remain the final element in list patterns
		if _, isSpread := element.(*ast.SpreadPattern); isSpread && !p.peekTokenIs(token.RBRACKET) {
			p.addErrorAt(p.curToken.Position, "spread (...) must be the final element in a list pattern")
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
		Pairs:     []ast.MapPatternEntry{},
		Spread:    nil,
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
				p.addErrorAt(p.curToken.Position, "spread not allowed in exact match")
				return nil
			} else {
				value := p.parseMatchPattern()
				mapPattern.Spread = value
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

		ident, isIdent := key.(*ast.Identifier)
		if isIdent && !readIdent {
			key = &ast.SymbolLiteral{
				Token: ident.Token,
				Value: ident.Value,
			}
		}

		if p.peekTokenIs(token.COLON) {
			p.nextToken()
			p.nextToken()
			value := p.parseMatchPattern()
			mapPattern.Pairs = append(mapPattern.Pairs, ast.MapPatternEntry{
				Key:     key,
				Pattern: value,
			})
		} else {
			name := key.String()
			switch literal := key.(type) {
			case *ast.Identifier:
				name = literal.Value
			case *ast.StringLiteral:
				name = literal.Value
			case *ast.SymbolLiteral:
				name = literal.Value
			}
			mapPattern.Pairs = append(mapPattern.Pairs, ast.MapPatternEntry{
				Key: key,
				Pattern: &ast.IdentifierPattern{
					Token: p.curToken,
					Value: &ast.Identifier{Token: p.curToken, Value: name},
				},
			})
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

func (p *Parser) parseStructPattern(schema *ast.Identifier) ast.MatchPattern {
	pattern := &ast.StructPattern{
		Token:  schema.Token,
		Schema: schema,
		Fields: []*ast.StructPatternField{},
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return pattern
	}

	seen := make(map[string]struct{})

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		p.skipLeadingNewlines()

		if p.curTokenIs(token.RBRACE) {
			return pattern
		}

		if !p.curTokenIs(token.IDENT) {
			p.addErrorAt(p.curToken.Position, "expected identifier in struct pattern, got %s", p.curToken.Type)
			return nil
		}

		name := p.curToken.Literal
		if _, ok := seen[name]; ok {
			p.addErrorAt(p.curToken.Position, "duplicate field in struct pattern: %s", name)
			return nil
		}
		seen[name] = struct{}{}

		var fieldPattern ast.MatchPattern
		if p.peekTokenIs(token.COLON) {
			p.nextToken()
			p.nextToken()
			fieldPattern = p.parseMatchPattern()
			if fieldPattern == nil {
				return nil
			}
		} else {
			fieldPattern = &ast.IdentifierPattern{
				Token: p.curToken,
				Value: &ast.Identifier{Token: p.curToken, Value: name},
			}
		}

		pattern.Fields = append(pattern.Fields, &ast.StructPatternField{
			Name:    name,
			Pattern: fieldPattern,
		})

		if p.peekTokenIs(token.RBRACE) {
			break
		}

		if !p.expectPeek(token.COMMA) {
			return nil
		}

		for p.peekTokenIs(token.NEWLINE) {
			p.nextToken()
		}

		if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			return pattern
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return pattern
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	prevAllowStructInit := p.allowStructInit
	p.allowStructInit = true
	exp := p.parseExpression(LOWEST)
	p.allowStructInit = prevAllowStructInit

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
	block := &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}

	p.nextToken() // move to first token inside block
	p.skipStatementSeparators()

	p.scopeDepth++
	defer func() { p.scopeDepth-- }()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		p.nextToken()
		p.skipStatementSeparators()
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

	if p.peekTokenIs(token.MATCH) {
		p.nextToken() // consume 'match'
		match := p.parseMatchExpression().(*ast.MatchExpression)

		// Inject the parameter(s) as the value to match against
		if len(lit.Parameters) == 1 {
			match.Value = lit.Parameters[0].Name
		} else {
			// For multiple parameters, match against a list of them
			list := &ast.ListLiteral{Token: lit.Token}
			for _, param := range lit.Parameters {
				list.Elements = append(list.Elements, param.Name)
			}
			match.Value = list
		}

		// Wrap the match in a block statement to serve as the function body
		lit.Body = &ast.BlockStatement{
			Token: match.Token,
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Token:      match.Token,
					Expression: match,
				},
			},
		}
	} else {
		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		lit.Body = p.parseBlockStatement()
	}

	// Analyze function body for tail calls
	p.setTailCallFlags(lit)

	// Validate that all `recur` occurrences are in tail position
	p.validateRecurUsage(lit)

	return lit
}

// setTailCallFlags analyzes a function literal and marks call expressions in tail position
func (p *Parser) setTailCallFlags(fn *ast.FunctionLiteral) {
	if fn.Body == nil || len(fn.Body.Statements) == 0 {
		return
	}

	// Check for tail calls in the function body
	hasTailCall := p.checkTailCallsInBlock(fn.Body)
	fn.HasTailCall = hasTailCall
}

// checkTailCallsInBlock checks for tail calls in a block statement
func (p *Parser) checkTailCallsInBlock(block *ast.BlockStatement) bool {
	if block == nil || len(block.Statements) == 0 {
		return false
	}

	// Check if this block introduces any defers
	for _, stmt := range block.Statements {
		if _, ok := stmt.(*ast.DeferStatement); ok {
			// If the block has a defer, we cannot perform tail call optimization
			// because the defer must run AFTER the call returns.
			return false
		}
	}

	// Only the last statement in a block can contain a tail call
	lastStmt := block.Statements[len(block.Statements)-1]
	return p.checkTailCallsInStatement(lastStmt)
}

// checkTailCallsInStatement checks for tail calls in a statement
func (p *Parser) checkTailCallsInStatement(stmt ast.Statement) bool {
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		// In a return statement, the returned expression might be a tail call
		return p.markTailCall(s.ReturnValue)

	case *ast.ExpressionStatement:
		// In an expression statement at the end of a block, the expression might be a tail call
		return p.markTailCall(s.Expression)

	default:
		return false
	}
}

// markTailCall marks call expressions as tail calls and returns whether a tail call was found
func (p *Parser) markTailCall(expr ast.Expression) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		// This is a direct tail call
		e.IsTailCall = true
		return true

	case *ast.RecurExpression:
		// recur is always a tail call if we see it in a tail position
		// (enforced by how this function is called)
		return true

	case *ast.IfExpression:
		// An if expression has tail calls if both branches have tail calls in their final statements
		thenHasTail := p.checkTailCallsInBlock(e.ThenBranch)
		elseHasTail := false
		if e.ElseBranch != nil {
			elseHasTail = p.checkTailCallsInBlock(e.ElseBranch)
		}
		return thenHasTail || elseHasTail

	case *ast.MatchExpression:
		// A match expression has tail calls if any of its cases have tail calls
		hasAnyTailCall := false
		for _, matchCase := range e.Cases {
			if matchCase.Body != nil && p.checkTailCallsInBlock(matchCase.Body) {
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
		p.addErrorAt(p.curToken.Position, "expected function or match after '/>'")
		return nil
	}

	// Support for value /> match { ... }
	if match, ok := right.(*ast.MatchExpression); ok {
		if match.Value != nil {
			p.addErrorAt(match.Token.Position, "match in pipeline cannot have an explicit value")
			return nil
		}
		match.Value = left
		return match
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
		p.addErrorAt(p.curToken.Position, "expected identifier after '.', got %s instead", p.peekToken.Type)
		return nil
	}

	mapKey := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return &ast.IndexExpression{
		Token: mapKey.Token,
		Left:  left,
		Index: &ast.SymbolLiteral{Token: mapKey.Token, Value: mapKey.Value},
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

func (p *Parser) parseRecurExpression() ast.Expression {
	// Current token is 'recur'
	expr := &ast.RecurExpression{
		Token:     p.curToken,
		Arguments: nil,
	}

	// Expect '(' after 'recur'
	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	// Reuse generic expression list parsing until ')'
	args := p.parseCallArguments(token.RPAREN)
	expr.Arguments = args

	return expr
}

func (p *Parser) parseNurseryExpression() ast.Expression {
	tok := p.curToken
	var limit ast.Expression

	p.nextToken()

	if p.curTokenIs(token.LIMIT) {
		p.nextToken()
		limit = p.parseExpression(LOWEST)
		p.nextToken()
	}

	// nursery can prefix a function or a block
	expr := p.parseExpression(LOWEST)

	switch node := expr.(type) {
	case *ast.FunctionLiteral:
		// Mark the function's body as the nursery scope
		if node.Body != nil {
			node.Body.IsNursery = true
			node.Body.Limit = limit
		}
		return node
	case *ast.BlockStatement:
		node.IsNursery = true
		node.Limit = limit
		return node
	default:
		p.addErrorAt(tok.Position, "nursery must be followed by a function or block, got %v", expr)
		return nil
	}
}

func (p *Parser) parseSpawnExpression() ast.Expression {
	expr := &ast.SpawnExpression{Token: p.curToken}
	p.nextToken()

	// If it's a block, wrap it in a function
	if p.curTokenIs(token.LBRACE) {
		block := p.parseBlockStatement()
		expr.Body = &ast.FunctionLiteral{
			Token: token.Token{Type: token.FUNCTION, Literal: "fn"},
			Body:  block,
		}
	} else {
		expr.Body = p.parseExpression(PREFIX)
	}

	return expr
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseCallArguments(token.RPAREN)
	return exp
}

func (p *Parser) parseCallArguments(end token.TokenType) []ast.Expression {
	var list []ast.Expression

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseCallArgument())

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // consume comma

		// Allow a dangling/trailing comma before the closing token
		if p.peekTokenIs(end) {
			p.nextToken() // consume end
			return list
		}

		p.nextToken() // move to next element
		list = append(list, p.parseCallArgument())
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseCallArgument() ast.Expression {
	if p.curTokenIs(token.ELLIPSIS) {
		p.nextToken()
		return &ast.SpreadExpression{
			Token: p.curToken,
			Value: p.parseExpression(LOWEST),
		}
	}

	if p.curTokenIs(token.IDENT) && p.peekTokenIs(token.ASSIGN) {
		name := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken() // consume '='
		p.nextToken() // move to value
		return &ast.NamedArgument{
			Token: name.Token,
			Name:  name,
			Value: p.parseExpression(LOWEST),
		}
	}

	return p.parseExpression(LOWEST)
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
		p.nextToken() // consume comma

		// Allow a dangling/trailing comma before the closing token
		if p.peekTokenIs(end) {
			p.nextToken() // consume end
			return list
		}

		p.nextToken() // move to next element

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
			if i == 0 {
				list = append(list, p.parseExpression(LOWEST))
				if p.peekTokenIs(token.RBRACKET) {
					break
				}
			} else {
				// Append nil for an omitted part
				slice = true
				list = append(list, nil)
				if p.peekTokenIs(token.RBRACKET) {
					break
				}
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

	for {
		// If next is '}', we're done
		if p.peekTokenIs(token.RBRACE) {
			break
		}

		// Advance to the next token (key start), then skip newlines/blank lines
		p.nextToken()
		p.skipLeadingNewlines()

		// NEW: allow newline(s) before closing brace (e.g. after trailing comma)
		if p.curTokenIs(token.RBRACE) {
			return mapLit
		}

		readIdent := p.curTokenIs(token.LBRACKET)
		if readIdent {
			p.nextToken() // consume '['
		}

		key := p.parseExpression(LOWEST)

		if readIdent {
			p.expectPeek(token.RBRACKET)
		}

		if ident, ok := key.(*ast.Identifier); ok && !readIdent {
			key = &ast.SymbolLiteral{Token: ident.Token, Value: ident.Value}
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		p.skipLeadingNewlines() // NEW: allow value on next line
		value := p.parseExpression(LOWEST)

		mapLit.Pairs[key] = value

		// If next is '}', we're done (no comma)
		if p.peekTokenIs(token.RBRACE) {
			break
		}

		// Require comma between entries
		if !p.expectPeek(token.COMMA) {
			return nil
		}

		// NEW: allow trailing comma followed by NEWLINE(s) and then '}'
		for p.peekTokenIs(token.NEWLINE) {
			p.nextToken() // consume NEWLINE
		}
		if p.peekTokenIs(token.RBRACE) {
			p.nextToken() // consume '}'
			return mapLit
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return mapLit
}

func (p *Parser) parseStructSchemaExpression() ast.Expression {
	schema := &ast.StructSchemaExpression{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	schema.Fields = []*ast.StructField{}

	for p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		p.skipLeadingNewlines()

		if p.curTokenIs(token.RBRACE) {
			return schema
		}

		field := p.parseStructSchemaField()
		if field != nil {
			schema.Fields = append(schema.Fields, field)
		}

		if p.peekTokenIs(token.RBRACE) {
			break
		}

		if !p.expectPeek(token.COMMA) {
			return nil
		}

		for p.peekTokenIs(token.NEWLINE) {
			p.nextToken()
		}

		if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			return schema
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return schema
}

func (p *Parser) parseStructSchemaField() *ast.StructField {
	field := &ast.StructField{Token: p.curToken}

	if p.curTokenIs(token.AT) {
		tag := p.parseTag()
		field.Hint = tag.Name
		if !p.expectPeek(token.IDENT) {
			return nil
		}
		field.Token = p.curToken
	} else if !p.curTokenIs(token.IDENT) {
		p.addErrorAt(p.curToken.Position, "expected identifier for struct field, got %s", p.curToken.Type)
		return nil
	}

	field.Name = p.curToken.Literal

	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken()
		p.nextToken()
		field.Default = p.parseExpression(LOWEST)
	}

	return field
}

func (p *Parser) parseStructInitExpression(left ast.Expression) ast.Expression {
	startToken := p.curToken
	fields := p.parseStructInitFields()
	if fields == nil {
		return nil
	}

	return &ast.StructInitExpression{
		Token:  startToken,
		Schema: left,
		Fields: fields,
	}
}

func (p *Parser) parseStructCopyExpression(left ast.Expression) ast.Expression {
	copyExpr := &ast.StructCopyExpression{Token: p.curToken, Source: left}
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	fields := p.parseStructInitFields()
	if fields == nil {
		return nil
	}
	copyExpr.Fields = fields
	return copyExpr
}

func (p *Parser) parseStructInitFields() []*ast.StructInitField {
	fields := []*ast.StructInitField{}

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return fields
	}

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		p.skipLeadingNewlines()

		if p.curTokenIs(token.RBRACE) {
			return fields
		}

		if !p.curTokenIs(token.IDENT) {
			p.addErrorAt(p.curToken.Position, "expected identifier for struct field, got %s", p.curToken.Type)
			return nil
		}

		field := &ast.StructInitField{
			Token: p.curToken,
			Name:  p.curToken.Literal,
		}

		if !p.expectPeek(token.COLON) {
			return nil
		}

		p.nextToken()
		p.skipLeadingNewlines()
		field.Value = p.parseExpression(LOWEST)
		fields = append(fields, field)

		if p.peekTokenIs(token.RBRACE) {
			break
		}

		if !p.expectPeek(token.COMMA) {
			return nil
		}

		for p.peekTokenIs(token.NEWLINE) {
			p.nextToken()
		}

		if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			return fields
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return fields
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	throw := &ast.ThrowStatement{Token: p.curToken}

	// Advance to the expression after `throw`
	p.nextToken()
	p.skipLeadingNewlines()

	ident := p.parseExpression(LOWEST)

	throw.Value = ident

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
	if p.scopeDepth == 0 && p.hasPendingDoc {
		foreignFunction.Doc = p.pendingDoc
		foreignFunction.HasDoc = true
		p.clearPendingDoc()
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
	stmt := &ast.DeferStatement{Token: p.curToken, Mode: ast.DeferAlways} // Current token is 'defer'
	p.nextToken()

	if p.curTokenIs(token.ONSUCCESS) {
		stmt.Mode = ast.DeferOnSuccess
		p.nextToken()
	} else if p.curTokenIs(token.ONERROR) {
		stmt.Mode = ast.DeferOnError
		p.nextToken()

		if p.curTokenIs(token.LPAREN) {
			p.nextToken() // Consume '('

			if !p.curTokenIs(token.IDENT) {
				p.addErrorAt(p.curToken.Position, "expected identifier for error variable")
				return nil
			}
			stmt.ErrorName = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken() // Consume identifier

			if !p.curTokenIs(token.RPAREN) {
				p.addErrorAt(p.curToken.Position, "expected closing parenthesis")
				return nil
			}
			p.nextToken() // Consume ')'
		} else {
			p.addErrorAt(p.curToken.Position, "expected '(' after 'onerror'")
			return nil
		}
	}

	if stmt.Mode == ast.DeferOnError && p.curTokenIs(token.MATCH) {

		match := p.parseMatchExpression().(*ast.MatchExpression)

		// Inject the error variable as the value to match against
		match.Value = stmt.ErrorName

		// Wrap the match in a block statement to serve as the defer body
		stmt.Call = &ast.BlockStatement{
			Token: match.Token,
			Statements: []ast.Statement{
				&ast.ExpressionStatement{
					Token:      match.Token,
					Expression: match,
				},
			},
		}
	} else if p.curTokenIs(token.LBRACE) {
		stmt.Call = p.parseBlockStatement()
	} else {
		stmt.Call = p.parseExpressionStatement()
	}

	return stmt
}

func (p *Parser) parseTag() *ast.Tag {
	annotation := &ast.Tag{Token: p.curToken}
	p.nextToken() // Consume '@'

	// Expect identifier or keyword for the annotation name
	if !p.curTokenIsSymbolName() {
		p.addErrorAt(p.curToken.Position, "expected annotation name after '@', got %s", p.curToken.Literal)
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

func (p *Parser) skipLeadingNewlines() {
	for p.curTokenIs(token.NEWLINE) {
		p.nextToken()
	}
}

func (p *Parser) curTokenIsAny(ts ...token.TokenType) bool {
	for _, t := range ts {
		if p.curTokenIs(t) {
			return true
		}
	}
	return false
}

func (p *Parser) peekTokenIsAny(ts ...token.TokenType) bool {
	for _, t := range ts {
		if p.peekTokenIs(t) {
			return true
		}
	}
	return false
}

func isContinuationToken(t token.TokenType) bool {
	switch t {
	// binary/infix operators
	case token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.PERCENT,
		token.EQ, token.NOT_EQ, token.LT, token.LT_EQ, token.GT, token.GT_EQ,
		token.LOGICAL_AND, token.LOGICAL_OR,
		token.BITWISE_AND, token.BITWISE_OR,
		token.SHIFT_LEFT, token.SHIFT_RIGHT,
		token.APPEND_ITEM, token.PREPEND_ITEM,
		token.CALL_CHAIN, // '/>'
		token.PERIOD:
		return true
	default:
		return false
	}
}

// Tokens that mean "the expression is incomplete; newline must not terminate"
func isRhsRequiredToken(t token.TokenType) bool {
	// If the current token is one of these, we're expecting a RHS next.
	// (This is mostly a safety net; your Pratt parse often enforces it naturally.)
	switch t {
	case token.ASSIGN,
		token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.PERCENT,
		token.EQ, token.NOT_EQ, token.LT, token.LT_EQ, token.GT, token.GT_EQ,
		token.LOGICAL_AND, token.LOGICAL_OR,
		token.BITWISE_AND, token.BITWISE_OR, token.BITWISE_XOR,
		token.SHIFT_LEFT, token.SHIFT_RIGHT,
		token.APPEND_ITEM, token.PREPEND_ITEM,
		token.CALL_CHAIN,
		token.PERIOD,
		token.COLON,  // if you ever parse "key: value" inside expressions
		token.ROCKET: // in match arms, if relevant to your parse flow
		return true
	default:
		return false
	}
}

func (p *Parser) newlineTerminates() bool {
	//fmt.Printf("NL? cur=%s peek=%s peek2=%s\n", p.curToken.Type, p.peekToken.Type, p.peek2Token.Type)

	// We only call this when peekToken is NEWLINE.
	// Rule 4: newline-call/index is forbidden:
	if p.peek2Token.Type == token.LPAREN || p.peek2Token.Type == token.LBRACKET {
		return true
	}
	// Continuation tokens keep the statement going (e.g. +, />, .)
	if isContinuationToken(p.peek2Token.Type) {
		return false
	}
	// If we're in a context that requires a RHS, newline is whitespace.
	if isRhsRequiredToken(p.curToken.Type) {
		return false
	}
	// Default: newline ends the statement
	return true
}

// validateRecurUsage ensures that all `recur` expressions inside a function
// appear only in tail position. Violations are reported as parser errors.
func (p *Parser) validateRecurUsage(fn *ast.FunctionLiteral) {
	if fn.Body == nil {
		return
	}
	// The function body as a whole is in tail position.
	p.validateRecurInBlock(fn.Body, true)
}

// validateRecurInBlock walks a block and validates `recur` usage.
// `inTail` indicates whether the *result* of this block is in tail position.
func (p *Parser) validateRecurInBlock(block *ast.BlockStatement, inTail bool) {
	if block == nil || len(block.Statements) == 0 {
		return
	}

	for i, stmt := range block.Statements {
		// Only the last statement can be in tail position relative to this block.
		stmtInTail := inTail && (i == len(block.Statements)-1)
		p.validateRecurInStatement(stmt, stmtInTail)
	}
}

// validateRecurInStatement validates recur usage in a single statement,
// propagating tail-position information appropriately.
func (p *Parser) validateRecurInStatement(stmt ast.Statement, inTail bool) {
	switch s := stmt.(type) {
	case *ast.ReturnStatement:
		// The returned expression is in tail position.
		p.validateRecurInExpr(s.ReturnValue, true)

	case *ast.ExpressionStatement:
		// The expression is tail-position only if this statement is.
		p.validateRecurInExpr(s.Expression, inTail)

	default:
		// Other statement types cannot be in tail position (their inner
		// expressions are not used as the function result), so any `recur`
		// inside them is always invalid.
		// We still traverse their sub-expressions with inTail=false if needed later.
	}
}

// validateRecurInExpr walks an expression tree and reports non-tail `recur` usage.
// `inTail` is true only when this expression as a whole is in tail position.
func (p *Parser) validateRecurInExpr(expr ast.Expression, inTail bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.RecurExpression:
		// `recur` is only allowed when the entire expression is in tail position.
		if !inTail {
			p.addErrorAt(e.Token.Position, "'recur' is only allowed in tail position")
		}
		// No need to descend further; `recur` has only arguments which are not expressions themselves here.

	case *ast.IfExpression:
		// Condition is never tail position.
		p.validateRecurInExpr(e.Condition, false)

		// The result of the then/else block is the result of the if-expression.
		p.validateRecurInBlock(e.ThenBranch, inTail)
		if e.ElseBranch != nil {
			p.validateRecurInBlock(e.ElseBranch, inTail)
		}

	case *ast.MatchExpression:
		// The matched value is not tail-position.
		if e.Value != nil {
			p.validateRecurInExpr(e.Value, false)
		}

		// Each case body contributes to the result of the whole match.
		for _, c := range e.Cases {
			if c == nil {
				continue
			}
			// Pattern and guard are never tail-position.
			// (Patterns are not expressions; guard is a condition.)
			if c.Guard != nil {
				p.validateRecurInExpr(c.Guard, false)
			}
			if c.Body != nil {
				p.validateRecurInBlock(c.Body, inTail)
			}
		}

	case *ast.SelectExpression:
		for _, c := range e.Cases {
			if c == nil {
				continue
			}
			switch c.Kind {
			case ast.SelectRecv:
				p.validateRecurInExpr(c.Channel, false)
			case ast.SelectSend:
				p.validateRecurInExpr(c.Channel, false)
				p.validateRecurInExpr(c.Value, false)
			case ast.SelectAfter:
				p.validateRecurInExpr(c.After, false)
			case ast.SelectAwait:
				p.validateRecurInExpr(c.Await, false)
			case ast.SelectDefault:
				// no header expression
			}
			if c.Handler != nil {
				p.validateRecurInExpr(c.Handler, inTail)
			}
		}

	case *ast.CallExpression:
		// Even if the call itself is in tail position, its callee/args are not.
		p.validateRecurInExpr(e.Function, false)
		for _, arg := range e.Arguments {
			p.validateRecurInExpr(arg, false)
		}

	case *ast.PrefixExpression:
		p.validateRecurInExpr(e.Right, false)

	case *ast.InfixExpression:
		p.validateRecurInExpr(e.Left, false)
		p.validateRecurInExpr(e.Right, false)

	case *ast.ListLiteral:
		for _, el := range e.Elements {
			p.validateRecurInExpr(el, false)
		}

	case *ast.MapLiteral:
		for k, v := range e.Pairs {
			p.validateRecurInExpr(k, false)
			p.validateRecurInExpr(v, false)
		}

	case *ast.IndexExpression:
		p.validateRecurInExpr(e.Left, false)
		p.validateRecurInExpr(e.Index, false)

	case *ast.SliceExpression:
		p.validateRecurInExpr(e.Start, false)
		p.validateRecurInExpr(e.End, false)
		p.validateRecurInExpr(e.Step, false)

	case *ast.SpreadExpression:
		p.validateRecurInExpr(e.Value, false)

	case *ast.FunctionLiteral:
		// Nested function literals have their own tail-position semantics.
		// Don't propagate outer tailness into inner functions.
		p.validateRecurUsage(e)

	default:
		// For literals, identifiers, etc., there is nothing to check.
	}
}

func (p *Parser) validateStructSchemaUsage(program *ast.Program) {
	for _, stmt := range program.Statements {
		p.validateStructSchemaInStatement(stmt)
	}
}

func (p *Parser) validateStructSchemaInStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		switch expr := s.Expression.(type) {
		case *ast.VarExpression:
			if _, ok := expr.Value.(*ast.StructSchemaExpression); ok {
				return
			}
			if p.containsStructSchema(expr.Value) {
				p.addErrorAt(expr.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
			}
		case *ast.ValExpression:
			if _, ok := expr.Value.(*ast.StructSchemaExpression); ok {
				return
			}
			if p.containsStructSchema(expr.Value) {
				p.addErrorAt(expr.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
			}
		default:
			if p.containsStructSchema(s.Expression) {
				p.addErrorAt(s.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
			}
		}
	case *ast.ReturnStatement:
		if p.containsStructSchema(s.ReturnValue) {
			p.addErrorAt(s.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
		}
	case *ast.ThrowStatement:
		if p.containsStructSchema(s.Value) {
			p.addErrorAt(s.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
		}
	case *ast.BlockStatement:
		for _, child := range s.Statements {
			p.validateStructSchemaInStatement(child)
		}
	case *ast.DeferStatement:
		if p.containsStructSchemaInStatement(s.Call) {
			p.addErrorAt(s.Token.Position, "struct schemas are only allowed on the right-hand side of val/var bindings")
		}
	default:
		// No-op for other statements.
	}
}

func (p *Parser) containsStructSchema(expr ast.Expression) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.StructSchemaExpression:
		return true
	case *ast.StructInitExpression:
		if p.containsStructSchema(e.Schema) {
			return true
		}
		for _, field := range e.Fields {
			if p.containsStructSchema(field.Value) {
				return true
			}
		}
		return false
	case *ast.StructCopyExpression:
		if p.containsStructSchema(e.Source) {
			return true
		}
		for _, field := range e.Fields {
			if p.containsStructSchema(field.Value) {
				return true
			}
		}
		return false
	case *ast.FunctionLiteral:
		if e.Body != nil {
			for _, stmt := range e.Body.Statements {
				if p.containsStructSchemaInStatement(stmt) {
					return true
				}
			}
		}
		return false
	case *ast.BlockStatement:
		for _, stmt := range e.Statements {
			if p.containsStructSchemaInStatement(stmt) {
				return true
			}
		}
		return false
	case *ast.CallExpression:
		if p.containsStructSchema(e.Function) {
			return true
		}
		for _, arg := range e.Arguments {
			if p.containsStructSchema(arg) {
				return true
			}
		}
		return false
	case *ast.InfixExpression:
		return p.containsStructSchema(e.Left) || p.containsStructSchema(e.Right)
	case *ast.PrefixExpression:
		return p.containsStructSchema(e.Right)
	case *ast.IfExpression:
		if p.containsStructSchema(e.Condition) {
			return true
		}
		if e.ThenBranch != nil && p.containsStructSchema(e.ThenBranch) {
			return true
		}
		if e.ElseBranch != nil && p.containsStructSchema(e.ElseBranch) {
			return true
		}
		return false
	case *ast.MatchExpression:
		if p.containsStructSchema(e.Value) {
			return true
		}
		for _, c := range e.Cases {
			if c == nil {
				continue
			}
			if p.containsStructSchema(c.Guard) {
				return true
			}
			if c.Body != nil && p.containsStructSchema(c.Body) {
				return true
			}
		}
		return false
	case *ast.ListLiteral:
		for _, el := range e.Elements {
			if p.containsStructSchema(el) {
				return true
			}
		}
		return false
	case *ast.MapLiteral:
		for k, v := range e.Pairs {
			if p.containsStructSchema(k) || p.containsStructSchema(v) {
				return true
			}
		}
		return false
	case *ast.IndexExpression:
		return p.containsStructSchema(e.Left) || p.containsStructSchema(e.Index)
	case *ast.SliceExpression:
		return p.containsStructSchema(e.Start) || p.containsStructSchema(e.End) || p.containsStructSchema(e.Step)
	case *ast.SpreadExpression:
		return p.containsStructSchema(e.Value)
	default:
		return false
	}
}

func (p *Parser) containsStructSchemaInStatement(stmt ast.Statement) bool {
	if stmt == nil {
		return false
	}
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		return p.containsStructSchema(s.Expression)
	case *ast.ReturnStatement:
		return p.containsStructSchema(s.ReturnValue)
	case *ast.ThrowStatement:
		return p.containsStructSchema(s.Value)
	case *ast.BlockStatement:
		for _, child := range s.Statements {
			if p.containsStructSchemaInStatement(child) {
				return true
			}
		}
		return false
	case *ast.DeferStatement:
		return p.containsStructSchemaInStatement(s.Call)
	default:
		return false
	}
}
