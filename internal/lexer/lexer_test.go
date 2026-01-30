package lexer

import (
	"slug/internal/token"
	"testing"
)

//func TestNextToken(t *testing.T) {
//	input := `var five = 5;
//var ten = 10;
//
//var add = fn(x, y) {
//x + y;
//};
//
//var result = add(five, ten);
//!- / * %~5;
//5 < 10 > 5;
//5 <= 10 >= 5;
//
//if (5 < 10) {
//	return true;
//} else {
//	return false;
//}
//// comment
//# alt comment
//:foo
//10 == 10; // comment
//10 != 9; # alt comment
//true && false;
//true || false;
//10 & 9;
//10 | 9;
//10 ^ 9;
//""
//"foobar"
//"foo bar"
//[1, 2];
//{"foo": "bar"}
//...
//..
//f2=fn
//???
//+:
//:+
//(_hello)
///>
//0x"414243"
//0x""
//recur
//// comment at eof
////`
//
//	tests := []struct {
//		expectedType    token.TokenType
//		expectedLiteral string
//	}{
//		{token.VAR, "var"},
//		{token.IDENT, "five"},
//		{token.ASSIGN, "="},
//		{token.NUMBER, "5"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n\n"},
//		{token.VAR, "var"},
//		{token.IDENT, "ten"},
//		{token.ASSIGN, "="},
//		{token.NUMBER, "10"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n\n"},
//		{token.VAR, "var"},
//		{token.IDENT, "add"},
//		{token.ASSIGN, "="},
//		{token.FUNCTION, "fn"},
//		{token.LPAREN, "("},
//		{token.IDENT, "x"},
//		{token.COMMA, ","},
//		{token.IDENT, "y"},
//		{token.RPAREN, ")"},
//		{token.LBRACE, "{"},
//		{token.NEWLINE, "\n"},
//		{token.IDENT, "x"},
//		{token.PLUS, "+"},
//		{token.IDENT, "y"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.RBRACE, "}"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n\n"},
//		{token.VAR, "var"},
//		{token.IDENT, "result"},
//		{token.ASSIGN, "="},
//		{token.IDENT, "add"},
//		{token.LPAREN, "("},
//		{token.IDENT, "five"},
//		{token.COMMA, ","},
//		{token.IDENT, "ten"},
//		{token.RPAREN, ")"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.BANG, "!"},
//		{token.MINUS, "-"},
//		{token.SLASH, "/"},
//		{token.ASTERISK, "*"},
//		{token.PERCENT, "%"},
//		{token.COMPLEMENT, "~"},
//		{token.NUMBER, "5"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "5"},
//		{token.LT, "<"},
//		{token.NUMBER, "10"},
//		{token.GT, ">"},
//		{token.NUMBER, "5"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "5"},
//		{token.LT_EQ, "<="},
//		{token.NUMBER, "10"},
//		{token.GT_EQ, ">="},
//		{token.NUMBER, "5"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.IF, "if"},
//		{token.LPAREN, "("},
//		{token.NUMBER, "5"},
//		{token.LT, "<"},
//		{token.NUMBER, "10"},
//		{token.RPAREN, ")"},
//		{token.LBRACE, "{"},
//		{token.NEWLINE, "\n"},
//		{token.RETURN, "return"},
//		{token.TRUE, "true"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.RBRACE, "}"},
//		{token.ELSE, "else"},
//		{token.LBRACE, "{"},
//		{token.NEWLINE, "\n"},
//		{token.RETURN, "return"},
//		{token.FALSE, "false"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.RBRACE, "}"},
//		{token.NEWLINE, "\n"},
//		{token.COLON, ":"},
//		{token.IDENT, "foo"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "10"},
//		{token.EQ, "=="},
//		{token.NUMBER, "10"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "10"},
//		{token.NOT_EQ, "!="},
//		{token.NUMBER, "9"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.TRUE, "true"},
//		{token.LOGICAL_AND, "&&"},
//		{token.FALSE, "false"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.TRUE, "true"},
//		{token.LOGICAL_OR, "||"},
//		{token.FALSE, "false"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "10"},
//		{token.BITWISE_AND, "&"},
//		{token.NUMBER, "9"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "10"},
//		{token.BITWISE_OR, "|"},
//		{token.NUMBER, "9"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.NUMBER, "10"},
//		{token.BITWISE_XOR, "^"},
//		{token.NUMBER, "9"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.STRING, ""},
//		{token.NEWLINE, "\n"},
//		{token.STRING, "foobar"},
//		{token.NEWLINE, "\n"},
//		{token.STRING, "foo bar"},
//		{token.NEWLINE, "\n"},
//		{token.LBRACKET, "["},
//		{token.NUMBER, "1"},
//		{token.COMMA, ","},
//		{token.NUMBER, "2"},
//		{token.RBRACKET, "]"},
//		{token.SEMICOLON, ";"},
//		{token.NEWLINE, "\n"},
//		{token.LBRACE, "{"},
//		{token.STRING, "foo"},
//		{token.COLON, ":"},
//		{token.STRING, "bar"},
//		{token.RBRACE, "}"},
//		{token.NEWLINE, "\n"},
//		{token.ELLIPSIS, "..."},
//		{token.NEWLINE, "\n"},
//		{token.PERIOD, "."},
//		{token.PERIOD, "."},
//		{token.NEWLINE, "\n"},
//		{token.IDENT, "f2"},
//		{token.ASSIGN, "="},
//		{token.FUNCTION, "fn"},
//		{token.NEWLINE, "\n"},
//		{token.NOT_IMPLEMENTED, "???"},
//		{token.NEWLINE, "\n"},
//		{token.PREPEND_ITEM, "+:"},
//		{token.NEWLINE, "\n"},
//		{token.APPEND_ITEM, ":+"},
//		{token.NEWLINE, "\n"},
//		{token.LPAREN, "("},
//		{token.IDENT, "_hello"},
//		{token.RPAREN, ")"},
//		{token.NEWLINE, "\n"},
//		{token.CALL_CHAIN, "/>"},
//		{token.NEWLINE, "\n"},
//		{token.BYTES, "414243"},
//		{token.NEWLINE, "\n"},
//		{token.BYTES, ""},
//		{token.NEWLINE, "\n"},
//		{token.RECUR, "recur"},
//		{token.NEWLINE, "\n"},
//		{token.EOF, ""},
//	}
//
//	l := New(input)
//
//	for i, tt := range tests {
//		tok := l.NextToken()
//
//		if tok.Type != tt.expectedType {
//			t.Fatalf("tests[%d] - tokentype wrong. expected=%q '%q', got=%q: '%q'",
//				i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)
//		}
//
//		if tok.Literal != tt.expectedLiteral {
//			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
//				i, tt.expectedLiteral, tok.Literal)
//		}
//	}
//}

func TestDocCommentTokenization(t *testing.T) {
	input := `/**
 * Hello
 *
 * World
 */
val x = 1`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.DOC_COMMENT, "Hello\n\nWorld"},
		{token.NEWLINE, "\n"},
		{token.VAL, "val"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.NUMBER, "1"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q '%q', got=%q: '%q'",
				i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestDocCommentFormatError(t *testing.T) {
	input := `/**
not ok
*/`

	l := New(input)
	tok := l.NextToken()

	if tok.Type != token.ILLEGAL {
		t.Fatalf("expected ILLEGAL token, got %q: %q", tok.Type, tok.Literal)
	}
}

func TestNextStringToken(t *testing.T) {
	input := `"\n\t\\\""
"start{{slug}}end"
"{{slug}}"
"\{\{slug"
"""
A
multi-line
{{slug}}
string
"""
// comment at eof
//`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.STRING, "\n\t\\\""},
		{token.NEWLINE, "\n"},

		{token.STRING, "start"},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},
		{token.STRING, "end"},
		{token.NEWLINE, "\n"},

		{token.STRING, ""},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},
		{token.NEWLINE, "\n"},

		{token.STRING, "{{slug"},
		{token.NEWLINE, "\n"},

		{token.STRING, "A\nmulti-line\n"},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},
		{token.STRING, "\nstring"},
		{token.NEWLINE, "\n\n"},
		//{token.NEWLINE, "\n"},

		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q '%q', got=%q: '%q'",
				i, tt.expectedType, tt.expectedLiteral, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
