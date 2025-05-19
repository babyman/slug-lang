package lexer

import (
	"slug/internal/token"
	"strings"
)

type MultiLineStringTokenizer struct {
	lexer *Lexer
}

func NewMultiLineStringTokenizer(lexer *Lexer) *MultiLineStringTokenizer {
	return &MultiLineStringTokenizer{lexer: lexer}
}

func (m *MultiLineStringTokenizer) NextToken() token.Token {
	var result strings.Builder
	startPosition := m.lexer.position

	for {
		if m.lexer.ch == 0 {
			return newToken(token.ILLEGAL, m.lexer.ch, startPosition)
		}

		if m.lexer.ch == '{' && m.lexer.peekChar() == '{' {
			break
		}

		if m.lexer.ch == '"' && m.lexer.peekChar() == '"' && m.lexer.peekTwoChars() == '"' {
			// Handle the closing `\n"""`
			for i := 0; i < 3; i++ {
				m.lexer.readChar()
			}
			original := result.String()          // Get the current string
			trimmed := original[:result.Len()-1] // Trim the last character
			result.Reset()                       // Reset the builder
			result.WriteString(trimmed)
			break
		}

		result.WriteByte(m.lexer.ch)
		m.lexer.readChar()
	}

	m.lexer.switchMode(NewGeneralTokenizer(m.lexer)) // Fall back after multi-line string

	return token.Token{
		Type:     token.STRING,
		Literal:  result.String(),
		Position: startPosition,
	}
}
