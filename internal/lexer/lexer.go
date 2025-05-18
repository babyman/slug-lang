package lexer

import (
	"slug/internal/token"
	"strings"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	startPosition := l.position // Record the current position as the start of the token

	switch l.ch {
	case '=':
		tok = l.handleCompoundToken2(token.ASSIGN, '=', token.EQ, '>', token.ROCKET)
	case '+':
		tok = l.handleCompoundToken(token.PLUS, ':', token.PREPEND_ITEM)
	case '-':
		tok = newToken(token.MINUS, l.ch, startPosition)
	case '!':
		tok = l.handleCompoundToken(token.BANG, '=', token.NOT_EQ)
	case '/':
		tok = newToken(token.SLASH, l.ch, startPosition)
	case '*':
		tok = newToken(token.ASTERISK, l.ch, startPosition)
	case '%':
		tok = newToken(token.PERCENT, l.ch, startPosition)
	case '~':
		tok = newToken(token.COMPLEMENT, l.ch, startPosition)
	case '&':
		tok = l.handleCompoundToken(token.BITWISE_AND, '&', token.LOGICAL_AND)
	case '|':
		tok = l.handleCompoundToken(token.BITWISE_OR, '|', token.LOGICAL_OR)
	case '_':
		tok = newToken(token.UNDERSCORE, l.ch, startPosition)
	case '^':
		tok = newToken(token.BITWISE_XOR, l.ch, startPosition)
	case '<':
		tok = l.handleCompoundToken2(token.LT, '=', token.LT_EQ, '<', token.SHIFT_LEFT)
	case '>':
		tok = l.handleCompoundToken2(token.GT, '=', token.GT_EQ, '>', token.SHIFT_RIGHT)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, startPosition)
	case ':':
		tok = l.handleCompoundToken(token.COLON, '+', token.APPEND_ITEM)
	case ',':
		tok = newToken(token.COMMA, l.ch, startPosition)
	case '.':
		if l.peekChar() == '.' && l.peekTwoChars() == '.' {
			tok = token.Token{Type: token.ELLIPSIS, Literal: "...", Position: startPosition}
			l.readChar()
			l.readChar()
			//} else if l.peekChar() == '.'{
			//	l.readChar()
			//	tok = token.Token{Type: token.RANGE, Literal: ".."}
		} else {
			tok = newToken(token.PERIOD, l.ch, startPosition)
		}
	case '?':
		if l.peekChar() == '?' && l.peekTwoChars() == '?' {
			tok = token.Token{Type: token.NOT_IMPLEMENTED, Literal: "???", Position: startPosition}
			l.readChar()
			l.readChar()
		} else {
			tok = newToken(token.ILLEGAL, l.ch, startPosition)
		}
	case '{':
		tok = newToken(token.LBRACE, l.ch, startPosition)
	case '}':
		tok = newToken(token.RBRACE, l.ch, startPosition)
	case '(':
		tok = newToken(token.LPAREN, l.ch, startPosition)
	case ')':
		tok = newToken(token.RPAREN, l.ch, startPosition)
	case '"':
		tok.Type = token.STRING
		tok.Position = startPosition
		tok.Literal = l.readString()
	case '[':
		tok = newToken(token.LBRACKET, l.ch, startPosition)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, startPosition)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
		tok.Position = startPosition
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			tok.Position = startPosition
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			tok.Position = startPosition
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch, startPosition)
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) handleCompoundToken(
	t token.TokenType,
	ch1 byte,
	t1 token.TokenType,
) token.Token {
	startPosition := l.position
	if l.peekChar() == ch1 {
		ch := l.ch
		l.readChar()
		literal := string(ch) + string(l.ch)
		return token.Token{Type: t1, Literal: literal, Position: startPosition}
	} else {
		return newToken(t, l.ch, startPosition)
	}
}

func (l *Lexer) handleCompoundToken2(
	t token.TokenType,
	ch1 byte,
	t1 token.TokenType,
	ch2 byte,
	t2 token.TokenType,
) token.Token {
	startPosition := l.position
	if l.peekChar() == ch1 {
		ch := l.ch
		l.readChar()
		literal := string(ch) + string(l.ch)
		return token.Token{Type: t1, Literal: literal, Position: startPosition}
	} else if l.peekChar() == ch2 {
		ch := l.ch
		l.readChar()
		literal := string(ch) + string(l.ch)
		return token.Token{Type: t2, Literal: literal, Position: startPosition}
	} else {
		return newToken(t, l.ch, startPosition)
	}
}

func (l *Lexer) skipWhitespace() {
	for {
		switch l.ch {
		case ' ', '\t', '\n', '\r':
			l.readChar()
		case '#':
			l.skipToLineEnd()
		case '/':
			if l.peekChar() == '/' {
				l.skipToLineEnd()
			} else {
				return
			}
		default:
			return
		}
	}
}

func (l *Lexer) skipToLineEnd() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func (l *Lexer) peekTwoChars() byte {
	if l.readPosition+1 >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition+1]
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	var result strings.Builder

	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}

		// Handle escape sequences
		if l.ch == '\\' {
			l.readChar() // Move to the next character
			switch l.ch {
			case 'n':
				result.WriteRune('\n')
			case 't':
				result.WriteRune('\t')
			case '\\':
				result.WriteRune('\\')
			case '"':
				result.WriteRune('"')
			}
		} else {
			result.WriteRune(rune(l.ch)) // Add normal character
		}
	}

	return result.String()
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch byte, position int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Position: position}
}
