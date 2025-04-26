package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	INT    = "INT"    // 1343456
	STRING = "STRING" // "foobar"

	// Operators
	ASSIGN     = "="
	PLUS       = "+"
	MINUS      = "-"
	BANG       = "!"
	ASTERISK   = "*"
	SLASH      = "/"
	PERCENT    = "%"
	UNDERSCORE = "_"

	LT    = "<"
	LT_EQ = "<="
	GT    = ">"
	GT_EQ = ">="

	COMPLEMENT  = "~"
	BITWISE_AND = "&"
	BITWISE_OR  = "|"
	BITWISE_XOR = "^"
	SHIFT_LEFT  = "<<"
	SHIFT_RIGHT = ">>"

	LOGICAL_AND = "&&"
	LOGICAL_OR  = "||"

	EQ     = "=="
	NOT_EQ = "!="

	ROCKET = "=>"
	//RANGE    = ".."
	ELLIPSIS = "..."

	// Delimiters
	PERIOD    = "."
	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// Keywords
	FUNCTION = "FUNCTION"
	VAR      = "VAR"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	NIL      = "NIL"
	IMPORT   = "IMPORT"
	AS       = "AS"
	TRY      = "TRY"
	CATCH    = "CATCH"
	MATCH    = "MATCH"
)

type Token struct {
	Type    TokenType
	Literal string
}

var keywords = map[string]TokenType{
	"fn": FUNCTION,
	//"val"
	"var":    VAR,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"nil":    NIL,
	"import": IMPORT,
	"as":     AS,
	//"export"
	"try":   TRY,
	"catch": CATCH,
	"match": MATCH,
	//"defer"
	//"native"
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
