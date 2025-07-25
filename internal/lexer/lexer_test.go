package lexer

import (
	"testing"

	"slug/internal/token"
)

func TestNextToken(t *testing.T) {
	input := `var five = 5;
var ten = 10;

var add = fn(x, y) {
 x + y;
};

var result = add(five, ten);
!-/*%~5;
5 < 10 > 5;
5 <= 10 >= 5;

if (5 < 10) {
	return true;
} else {
	return false;
}
// comment
# alt comment
10 == 10; // comment
10 != 9; # alt comment
true && false;
true || false;
10 & 9;
10 | 9;
10 ^ 9;
""
"foobar"
"foo bar"
[1, 2];
{"foo": "bar"}
...
..
f2=fn
???
+:
:+
(_hello)
// comment at eof
//`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.VAR, "var"},
		{token.IDENT, "five"},
		{token.ASSIGN, "="},
		{token.NUMBER, "5"},
		{token.SEMICOLON, ";"},
		{token.VAR, "var"},
		{token.IDENT, "ten"},
		{token.ASSIGN, "="},
		{token.NUMBER, "10"},
		{token.SEMICOLON, ";"},
		{token.VAR, "var"},
		{token.IDENT, "add"},
		{token.ASSIGN, "="},
		{token.FUNCTION, "fn"},
		{token.LPAREN, "("},
		{token.IDENT, "x"},
		{token.COMMA, ","},
		{token.IDENT, "y"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.IDENT, "x"},
		{token.PLUS, "+"},
		{token.IDENT, "y"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.VAR, "var"},
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "five"},
		{token.COMMA, ","},
		{token.IDENT, "ten"},
		{token.RPAREN, ")"},
		{token.SEMICOLON, ";"},
		{token.BANG, "!"},
		{token.MINUS, "-"},
		{token.SLASH, "/"},
		{token.ASTERISK, "*"},
		{token.PERCENT, "%"},
		{token.COMPLEMENT, "~"},
		{token.NUMBER, "5"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "5"},
		{token.LT, "<"},
		{token.NUMBER, "10"},
		{token.GT, ">"},
		{token.NUMBER, "5"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "5"},
		{token.LT_EQ, "<="},
		{token.NUMBER, "10"},
		{token.GT_EQ, ">="},
		{token.NUMBER, "5"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.LPAREN, "("},
		{token.NUMBER, "5"},
		{token.LT, "<"},
		{token.NUMBER, "10"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.TRUE, "true"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.NUMBER, "10"},
		{token.EQ, "=="},
		{token.NUMBER, "10"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "10"},
		{token.NOT_EQ, "!="},
		{token.NUMBER, "9"},
		{token.SEMICOLON, ";"},
		{token.TRUE, "true"},
		{token.LOGICAL_AND, "&&"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.TRUE, "true"},
		{token.LOGICAL_OR, "||"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "10"},
		{token.BITWISE_AND, "&"},
		{token.NUMBER, "9"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "10"},
		{token.BITWISE_OR, "|"},
		{token.NUMBER, "9"},
		{token.SEMICOLON, ";"},
		{token.NUMBER, "10"},
		{token.BITWISE_XOR, "^"},
		{token.NUMBER, "9"},
		{token.SEMICOLON, ";"},
		{token.STRING, ""},
		{token.STRING, "foobar"},
		{token.STRING, "foo bar"},
		{token.LBRACKET, "["},
		{token.NUMBER, "1"},
		{token.COMMA, ","},
		{token.NUMBER, "2"},
		{token.RBRACKET, "]"},
		{token.SEMICOLON, ";"},
		{token.LBRACE, "{"},
		{token.STRING, "foo"},
		{token.COLON, ":"},
		{token.STRING, "bar"},
		{token.RBRACE, "}"},
		{token.ELLIPSIS, "..."},
		{token.PERIOD, "."},
		{token.PERIOD, "."},
		{token.IDENT, "f2"},
		{token.ASSIGN, "="},
		{token.FUNCTION, "fn"},
		{token.NOT_IMPLEMENTED, "???"},
		{token.PREPEND_ITEM, "+:"},
		{token.APPEND_ITEM, ":+"},
		{token.LPAREN, "("},
		{token.IDENT, "_hello"},
		{token.RPAREN, ")"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q: '%q'",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
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

		{token.STRING, "start"},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},
		{token.STRING, "end"},

		{token.STRING, ""},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},

		{token.STRING, "{{slug"},

		{token.STRING, "A\nmulti-line\n"},
		{token.INTERPOLATION_START, "{{"},
		{token.IDENT, "slug"},
		{token.INTERPOLATION_END, "}}"},
		{token.STRING, "\nstring"},

		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q: '%q'",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
