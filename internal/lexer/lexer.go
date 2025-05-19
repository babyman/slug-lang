package lexer

import (
	"slug/internal/token"
)

type Lexer struct {
	input        string
	position     int       // current position in input (points to current char)
	readPosition int       // current reading position in input (after current char)
	ch           byte      // current char under examination
	prevMode     Tokenizer // Current tokenizer strategy
	currentMode  Tokenizer // Current tokenizer strategy
}

type Tokenizer interface {
	NextToken() token.Token
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.switchMode(NewGeneralTokenizer(l))
	l.readChar()
	return l
}

func (l *Lexer) switchMode(mode Tokenizer) {
	l.prevMode = l.currentMode
	l.currentMode = mode
}

func (l *Lexer) NextToken() token.Token {
	return l.currentMode.NextToken()
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

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch byte, position int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Position: position}
}
