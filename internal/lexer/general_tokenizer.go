package lexer

import (
	"slug/internal/token"
)

type GeneralTokenizer struct {
	lexer *Lexer
}

func NewGeneralTokenizer(lexer *Lexer) *GeneralTokenizer {
	return &GeneralTokenizer{lexer: lexer}
}

func (g *GeneralTokenizer) NextToken() token.Token {
	var tok token.Token

	g.lexer.skipWhitespace()

	startPosition := g.lexer.position // Record the current position as the start of the token

	switch g.lexer.ch {
	case '=':
		tok = g.lexer.handleCompoundToken2(token.ASSIGN, '=', token.EQ, '>', token.ROCKET)
	case '+':
		tok = g.lexer.handleCompoundToken(token.PLUS, ':', token.PREPEND_ITEM)
	case '-':
		tok = newToken(token.MINUS, g.lexer.ch, startPosition)
	case '!':
		tok = g.lexer.handleCompoundToken(token.BANG, '=', token.NOT_EQ)
	case '/':
		tok = newToken(token.SLASH, g.lexer.ch, startPosition)
	case '*':
		tok = newToken(token.ASTERISK, g.lexer.ch, startPosition)
	case '%':
		tok = newToken(token.PERCENT, g.lexer.ch, startPosition)
	case '~':
		tok = newToken(token.COMPLEMENT, g.lexer.ch, startPosition)
	case '&':
		tok = g.lexer.handleCompoundToken(token.BITWISE_AND, '&', token.LOGICAL_AND)
	case '|':
		tok = g.lexer.handleCompoundToken2(token.BITWISE_OR, '|', token.LOGICAL_OR, '}', token.MATCH_KEYS_CLOSE)
	case '_':
		if isLetter(g.lexer.peekChar()) {
			tok.Literal = g.lexer.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Position = startPosition
			return tok
		} else {
			tok = newToken(token.UNDERSCORE, g.lexer.ch, startPosition)
		}
	case '^':
		tok = newToken(token.BITWISE_XOR, g.lexer.ch, startPosition)
	case '<':
		tok = g.lexer.handleCompoundToken2(token.LT, '=', token.LT_EQ, '<', token.SHIFT_LEFT)
	case '>':
		tok = g.lexer.handleCompoundToken2(token.GT, '=', token.GT_EQ, '>', token.SHIFT_RIGHT)
	case ';':
		tok = newToken(token.SEMICOLON, g.lexer.ch, startPosition)
	case ':':
		tok = g.lexer.handleCompoundToken(token.COLON, '+', token.APPEND_ITEM)
	case ',':
		tok = newToken(token.COMMA, g.lexer.ch, startPosition)
	case '.':
		if g.lexer.peekChar() == '.' && g.lexer.peekTwoChars() == '.' {
			tok = token.Token{Type: token.ELLIPSIS, Literal: "...", Position: startPosition}
			g.lexer.readChar()
			g.lexer.readChar()
		} else {
			tok = newToken(token.PERIOD, g.lexer.ch, startPosition)
		}
	case '?':
		if g.lexer.peekChar() == '?' && g.lexer.peekTwoChars() == '?' {
			tok = token.Token{Type: token.NOT_IMPLEMENTED, Literal: "???", Position: startPosition}
			g.lexer.readChar()
			g.lexer.readChar()
		} else {
			tok = newToken(token.ILLEGAL, g.lexer.ch, startPosition)
		}
	case '{':
		tok = g.lexer.handleCompoundToken2(token.LBRACE, '{', token.INTERPOLATION_START, '|', token.MATCH_KEYS_EXACT)
	case '}':
		if g.lexer.prevMode != nil && g.lexer.peekChar() == '}' && g.lexer.peekTwoChars() == '"' {
			g.lexer.readChar() // consume the }
			g.lexer.readChar() // Consume the closing "
			tok = token.Token{Type: token.INTERPOLATION_END, Literal: "}}", Position: startPosition}
		} else if g.lexer.prevMode != nil && g.lexer.peekChar() == '}' {
			g.lexer.readChar() // consume the }
			tok = token.Token{Type: token.INTERPOLATION_END, Literal: "}}", Position: startPosition}
			g.lexer.switchMode(g.lexer.prevMode) // Return to the previous string tokenizer
		} else {
			tok = newToken(token.RBRACE, g.lexer.ch, g.lexer.position)
		}
	case '(':
		tok = newToken(token.LPAREN, g.lexer.ch, startPosition)
	case ')':
		tok = newToken(token.RPAREN, g.lexer.ch, startPosition)
	case '"':
		if g.lexer.peekChar() == '"' && g.lexer.peekTwoChars() == '"' {
			g.lexer.readChar() // Consume the first ""
			g.lexer.readChar() // Consume the second ""
			g.lexer.readChar() // Consume the third ""
			g.lexer.readChar() // Consume the \n
			g.lexer.switchMode(NewMultiLineStringTokenizer(g.lexer))
		} else {
			g.lexer.readChar() // consume the opening "
			g.lexer.switchMode(NewSingleLineStringTokenizer(g.lexer))
		}
		return g.lexer.currentMode.NextToken()
	case '[':
		tok = newToken(token.LBRACKET, g.lexer.ch, startPosition)
	case ']':
		tok = newToken(token.RBRACKET, g.lexer.ch, startPosition)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Position = startPosition
	default:
		if isLetter(g.lexer.ch) {
			tok.Literal = g.lexer.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Position = startPosition
			return tok
		} else if isDigit(g.lexer.ch) {
			tok.Type = token.INT
			tok.Literal = g.lexer.readNumber()
			tok.Position = startPosition
			return tok
		} else {
			tok = newToken(token.ILLEGAL, g.lexer.ch, startPosition)
		}
	}

	g.lexer.readChar()
	return tok
}
