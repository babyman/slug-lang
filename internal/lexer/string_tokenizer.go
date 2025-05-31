package lexer

import (
	"slug/internal/token"
	"strings"
)

type SingleLineStringTokenizer struct {
	lexer *Lexer
}

func NewSingleLineStringTokenizer(lexer *Lexer) *SingleLineStringTokenizer {
	return &SingleLineStringTokenizer{lexer: lexer}
}

func (s *SingleLineStringTokenizer) NextToken() token.Token {
	var result strings.Builder
	startPosition := s.lexer.position

	// start reading the string right away, assume the opening `"` has already been read

	for {
		if s.lexer.ch == 0 {
			return newToken(token.ILLEGAL, s.lexer.ch, startPosition)
		}

		if s.lexer.ch == '{' && s.lexer.peekChar() == '{' {
			// Switch to interpolation mode
			s.lexer.switchMode(NewGeneralTokenizer(s.lexer))
			break
		}

		if s.lexer.ch == '"' {
			// End of the single-line string
			s.lexer.readChar() // Consume the closing `"`
			s.lexer.setMode(NewGeneralTokenizer(s.lexer))
			break
		}

		if s.lexer.ch == '\\' {
			// Handle escape sequences
			s.lexer.readChar() // Move to the escaped character
			switch s.lexer.ch {
			case 'n':
				result.WriteRune('\n')
			case 'r':
				result.WriteRune('\r')
			case 't':
				result.WriteRune('\t')
			case '\\':
				result.WriteRune('\\')
			case '"':
				result.WriteRune('"')
			case '{':
				result.WriteRune('{')
			default:
				result.WriteRune('\\')
				result.WriteByte(s.lexer.ch)
			}
		} else {
			result.WriteByte(s.lexer.ch)
		}

		s.lexer.readChar()
	}

	return token.Token{
		Type:     token.STRING,
		Literal:  result.String(),
		Position: startPosition,
	}
}
