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
			case '0', '1', '2', '3', '4', '5', '6', '7':
				// Handle octal escape sequences
				octalValue := s.consumeOctal(s.lexer.ch)
				result.WriteRune(rune(octalValue))
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

// consumeOctal interprets up to three octal digits to return their numeric value.
func (s *SingleLineStringTokenizer) consumeOctal(firstChar byte) int {
	value := int(firstChar - '0') // Convert the first octal digit
	for i := 0; i < 2; i++ {      // Consume up to two more octal digits
		next := s.lexer.peekChar()
		if next < '0' || next > '7' {
			break
		}
		s.lexer.readChar() // Consume the next octal digit
		value = value*8 + int(next-'0')
	}
	return value
}
