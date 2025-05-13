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

	APPEND_ITEM  = ":+"
	PREPEND_ITEM = "+:"

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
	IMPORT   = "IMPORT"
	AS       = "AS"
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
	// declarations
	"nil":     NIL,
	"true":    TRUE,
	"false":   FALSE,
	"foreign": FOREIGN,
	"fn":      FUNCTION,
	"val":     VAL,
	"var":     VAR,

	// import etc
	"import": IMPORT,
	"as":     AS,
	//"export": EXPORT

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

	// Thread related
	//"spawn": SPAWN
	//"send": SEND
	//"receive": RECEIVE
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
