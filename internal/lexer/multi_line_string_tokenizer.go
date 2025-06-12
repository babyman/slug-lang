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
			m.lexer.switchMode(NewGeneralTokenizer(m.lexer))
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
			m.lexer.setMode(NewGeneralTokenizer(m.lexer))
			break
		}

		if m.lexer.ch == '\\' {
			// Handle escape sequences
			m.lexer.readChar() // Move to the escaped character
			switch m.lexer.ch {
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
				octalValue := m.consumeOctal(m.lexer.ch)
				result.WriteRune(rune(octalValue))
			default:
				result.WriteRune('\\')
				result.WriteByte(m.lexer.ch)
			}
		} else {
			result.WriteByte(m.lexer.ch)
		}

		m.lexer.readChar()
	}

	return token.Token{
		Type:     token.STRING,
		Literal:  result.String(),
		Position: startPosition,
	}
}

// consumeOctal interprets up to three octal digits to return their numeric value.
func (m *MultiLineStringTokenizer) consumeOctal(firstChar byte) int {
	value := int(firstChar - '0') // Convert the first octal digit
	for i := 0; i < 2; i++ {      // Consume up to two more octal digits
		next := m.lexer.peekChar()
		if next < '0' || next > '7' {
			break
		}
		m.lexer.readChar() // Consume the next octal digit
		value = value*8 + int(next-'0')
	}
	return value
}
