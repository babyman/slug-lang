package lexer

import (
	"errors"
	"slug/internal/token"
	"unicode"
	"unicode/utf8"
)

type Lexer struct {
	input        string
	position     int       // current byte position in input (points to start of current rune)
	readPosition int       // next byte position in input (start of next rune)
	ch           rune      // current rune under examination; 0 means EOF
	prevMode     Tokenizer // Prev tokenizer strategy if we are in interpolation mode
	currentMode  Tokenizer // Current tokenizer strategy

	parenDepth   int // Track nesting of ( )
	bracketDepth int // Track nesting of [ ]
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

// retain previous mode, this should be called when parsing a string
func (l *Lexer) switchMode(mode Tokenizer) {
	l.prevMode = l.currentMode
	l.currentMode = mode
}

// clear the previous mode, this should be called when we are exiting string interpolation mode
func (l *Lexer) setMode(mode Tokenizer) {
	l.prevMode = nil
	l.currentMode = mode
}

func (l *Lexer) NextToken() token.Token {
	return l.currentMode.NextToken()
}

func (l *Lexer) handleCompoundToken(
	t token.TokenType,
	ch1 rune,
	t1 token.TokenType,
) token.Token {
	startPosition := l.position
	if l.peekChar() == ch1 {
		first := l.ch
		l.readChar()
		literal := string(first) + string(l.ch)
		return token.Token{Type: t1, Literal: literal, Position: startPosition}
	} else {
		return newToken(t, l.ch, startPosition)
	}
}

func (l *Lexer) handleCompoundToken2(
	t token.TokenType,
	ch1 rune,
	t1 token.TokenType,
	ch2 rune,
	t2 token.TokenType,
) token.Token {
	startPosition := l.position
	peek := l.peekChar()
	if peek == ch1 {
		first := l.ch
		l.readChar()
		literal := string(first) + string(l.ch)
		return token.Token{Type: t1, Literal: literal, Position: startPosition}
	} else if peek == ch2 {
		first := l.ch
		l.readChar()
		literal := string(first) + string(l.ch)
		return token.Token{Type: t2, Literal: literal, Position: startPosition}
	} else {
		return newToken(t, l.ch, startPosition)
	}
}

func (l *Lexer) skipWhitespace() {
	for {
		switch l.ch {
		case ' ', '\t', '\r':
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

// readChar advances by one UTF-8 rune, updating byte positions
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition
		return
	}
	r, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
	l.ch = r
	l.position = l.readPosition
	l.readPosition += size
}

// peekChar returns the next rune without advancing; returns 0 at EOF
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

// peekTwoChars returns the rune after next without advancing; returns 0 if unavailable
func (l *Lexer) peekTwoChars() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	_, size := utf8.DecodeRuneInString(l.input[l.readPosition:])
	idx := l.readPosition + size
	if idx >= len(l.input) {
		return 0
	}
	r2, _ := utf8.DecodeRuneInString(l.input[idx:])
	return r2
}

// readIdentifier returns the substring (bytes) covering the identifier runes
func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[start:l.position]
}

// readNumber keeps previous ASCII-based number rules; extends to Unicode digits for integer part
func (l *Lexer) readNumber() (string, error) {
	numStr := ""
	for isDigit(l.ch) || l.ch == '_' {
		if l.ch == '_' {
			peek := l.peekChar()
			prev := l.input[l.position-1]
			// Rule: _ must be between hex digits (or after prefix handled above)
			if !isDigit(rune(prev)) || !isDigit(peek) {
				return "", errors.New("underscore must be between digits in number literal")
			}
		} else {
			numStr += string(l.ch)
		}
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		numStr += string(l.ch)
		l.readChar()
		for isDigit(l.ch) || l.ch == '_' {
			if l.ch == '_' {
				peek := l.peekChar()
				prev := l.input[l.position-1]
				// Rule: _ must be between hex digits (or after prefix handled above)
				if !isDigit(rune(prev)) || !isDigit(peek) {
					return "", errors.New("underscore must be between digits in number literal")
				}
			} else {
				numStr += string(l.ch)
			}
			l.readChar()
		}
	}
	if l.ch == 'e' || l.ch == 'E' {
		numStr += string(l.ch)
		l.readChar()
		if l.ch == '+' || l.ch == '-' {
			numStr += string(l.ch)
			l.readChar()
		}
		for isDigit(l.ch) {
			numStr += string(l.ch)
			l.readChar()
		}
	}
	return numStr, nil
}

func (l *Lexer) readHexLiteral() (string, error) {
	hexStr := ""
	hexStr += string(l.ch)
	l.readChar() // consume '0'
	if l.ch != 'x' {
		return "", errors.New("expected 'x' after '0'")
	}
	hexStr += string(l.ch)
	l.readChar() // consume 'x'

	// Rule: _ allowed immediately after 0x if followed by a hex digit
	if l.ch == '_' {
		if isHexDigit(l.peekChar()) {
			l.readChar()
		} else {
			return "", errors.New("expected hex digit after '0x'")
		}
	}

	if !isHexDigit(l.ch) {
		return "", errors.New("expected hex digit after '0x'")
	}

	for isHexDigit(l.ch) || l.ch == '_' {
		if l.ch == '_' {
			peek := l.peekChar()
			prev := l.input[l.position-1]
			// Rule: _ must be between hex digits (or after prefix handled above)
			if !isHexDigit(rune(prev)) || !isHexDigit(peek) {
				break
			}
		} else {
			hexStr += string(l.ch)
		}
		l.readChar()
	}
	// check even length
	if len(hexStr)%2 != 0 {
		return "", errors.New("hex literal must have even length")
	}
	return hexStr, nil
}

func (l *Lexer) readByteArrayLiteral() (string, bool) {
	l.readChar() // consume 0
	l.readChar() // consume x
	l.readChar() // consume opening "
	start := l.position
	if l.ch != '"' {
		for {
			l.readChar()
			if l.ch == '"' || l.ch == 0 {
				break
			}
			if !((l.ch >= '0' && l.ch <= '9') ||
				(l.ch >= 'a' && l.ch <= 'f') ||
				(l.ch >= 'A' && l.ch <= 'F')) {
				return "", false
			}
		}
	}
	// stop at '"' without consuming beyond it
	if l.ch != '"' {
		return "", false
	}
	hexStr := l.input[start:l.position]
	// check even length
	if len(hexStr)%2 != 0 {
		return "", false
	}
	// after finishing, advance one char to move past closing quote
	l.readChar()
	// return the hex string (e.g., "414243") and true, or "", false on error
	return hexStr, true
}

// Unicode-aware helpers
func isLetter(ch rune) bool {
	// Letters, underscore, and categories like Letter and Mark to support identifiers like café,变量
	return ch == '_' || unicode.IsLetter(ch) || unicode.Is(unicode.Mn, ch) || unicode.Is(unicode.Mc, ch)
}

func isDigit(ch rune) bool {
	// Allow Unicode decimal digits
	return unicode.IsDigit(ch)
}

func isHexDigit(ch rune) bool {
	// Allow Unicode hex digits
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func newToken(tokenType token.TokenType, ch rune, position int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Position: position}
}
