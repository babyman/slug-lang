package token

type TokenType string

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	NUMBER = "NUMBER" // 1343456
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
	AT         = "@"

	APPEND_ITEM  = ":+"
	PREPEND_ITEM = "+:"

	INTERPOLATION_START = "{{"
	INTERPOLATION_END   = "}}"

	MATCH_KEYS_EXACT = "{|"
	MATCH_KEYS_CLOSE = "|}"

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
	ELLIPSIS        = "..."
	NOT_IMPLEMENTED = "???"
	CALL_CHAIN      = "/>"

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
	FOREIGN  = "FOREIGN"
	FUNCTION = "FUNCTION"
	VAL      = "VAL"
	VAR      = "VAR"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	NIL      = "NIL"
	TRY      = "TRY"
	CATCH    = "CATCH"
	THROW    = "THROW"
	DEFER    = "DEFER"
	MATCH    = "MATCH"
)

type Token struct {
	Type     TokenType
	Literal  string
	Position int // the src index of the token
}

var keywords = map[string]TokenType{
	// constants
	"nil":   NIL,
	"true":  TRUE,
	"false": FALSE,

	// declarations
	"fn":      FUNCTION,
	"foreign": FOREIGN,
	"val":     VAL,
	"var":     VAR,

	// flow control
	"if":     IF,
	"else":   ELSE,
	"match":  MATCH,
	"return": RETURN,

	// error handling
	"try":   TRY,
	"catch": CATCH,
	"throw": THROW,
	"defer": DEFER,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
