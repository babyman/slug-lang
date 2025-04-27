package ast

import (
	"bytes"
	"slug/internal/token"
	"strings"
)

// The base Node interface
type Node interface {
	TokenLiteral() string
	String() string
}

// All statement nodes implement this
type Statement interface {
	Node
	statementNode()
}

// All expression nodes implement this
type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// Statements
type VarStatement struct {
	Token token.Token // the token.VAR token
	Name  *Identifier
	Value Expression
}

func (ls *VarStatement) statementNode()       {}
func (ls *VarStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *VarStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

type ReturnStatement struct {
	Token       token.Token // the 'return' token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")

	return out.String()
}

type ImportSymbol struct {
	Name  *Identifier // The symbol being imported (e.g., "sqr")
	Alias *Identifier // Optional alias for the symbol (e.g., "as a")
}

func (is *ImportSymbol) String() string {
	if is.Alias != nil {
		return is.Name.String() + " as " + is.Alias.String()
	}
	return is.Name.String()
}

type ImportStatement struct {
	Token     token.Token     // The 'import' token
	PathParts []*Identifier   // Dot-separated identifiers for module path (e.g., math.Arithmetic)
	Symbols   []*ImportSymbol // Symbols being imported, with optional aliases
	Wildcard  bool            // Whether the import uses a wildcard (*)
}

func (is *ImportStatement) statementNode()       {}
func (is *ImportStatement) TokenLiteral() string { return is.Token.Literal }
func (is *ImportStatement) String() string {
	var out bytes.Buffer

	out.WriteString("import ")
	out.WriteString(is.PathAsString())

	if is.Wildcard {
		out.WriteString(".*")
	} else if len(is.Symbols) > 0 {
		symbols := []string{}
		for _, sym := range is.Symbols {
			symbols = append(symbols, sym.String())
		}
		out.WriteString(".{" + strings.Join(symbols, ", ") + "}")
	}
	return out.String()
}

func (is *ImportStatement) PathAsString() string {
	parts := []string{}
	for _, part := range is.PathParts {
		parts = append(parts, part.Value)
	}
	return strings.Join(parts, ".")
}

type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type BlockStatement struct {
	Token      token.Token // the { token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// Expressions
type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

type Nil struct {
	Token token.Token
}

func (b *Nil) expressionNode()      {}
func (b *Nil) TokenLiteral() string { return b.Token.Literal }
func (b *Nil) String() string       { return b.Token.Literal }

type IntegerLiteral struct {
	Token token.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

type PrefixExpression struct {
	Token    token.Token // The prefix token, e.g. !
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

type InfixExpression struct {
	Token    token.Token // The operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

type IfExpression struct {
	Token      token.Token // The 'if' token
	Condition  Expression
	ThenBranch *BlockStatement
	ElseBranch *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.ThenBranch.String())

	if ie.ElseBranch != nil {
		out.WriteString("else ")
		out.WriteString(ie.ElseBranch.String())
	}

	return out.String()
}

type FunctionLiteral struct {
	Token      token.Token // The 'fn' token
	Parameters []*FunctionParameter
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

type CallExpression struct {
	Token     token.Token // The '(' token
	Function  Expression  // Identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

type FunctionParameter struct {
	Name        *Identifier         // Parameter name
	Default     Expression          // Default value (optional)
	IsVariadic  bool                // Whether this is a variadic argument
	Destructure *DestructureBinding // List destructuring binding (optional)
}

func (p *FunctionParameter) expressionNode()      {}
func (p *FunctionParameter) TokenLiteral() string { return p.Name.Token.Literal }
func (p *FunctionParameter) String() string {
	var out bytes.Buffer

	if p.Destructure != nil {
		out.WriteString(p.Destructure.String())
	} else {
		out.WriteString("(")
		if p.IsVariadic {
			out.WriteString("...")
		}
		out.WriteString(p.Name.String())
		if p.Default != nil {
			out.WriteString("=")
			out.WriteString(p.Default.String())
		}
		out.WriteString(")")
	}

	return out.String()
}

type DestructureBinding struct {
	Token token.Token // The ':', for example
	Head  *Identifier // The variable for the head (e.g., "h")
	Tail  *Identifier // The variable for the tail (e.g., "t")
}

func (b *DestructureBinding) expressionNode()      {}
func (b *DestructureBinding) TokenLiteral() string { return b.Token.Literal }
func (b *DestructureBinding) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(b.Head.String())
	out.WriteString(":")
	out.WriteString(b.Tail.String())
	out.WriteString(")")

	return out.String()
}

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

type ArrayLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type IndexExpression struct {
	Token token.Token // The [ token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")

	return out.String()
}

type HashLiteral struct {
	Token token.Token // the '{' token
	Pairs map[Expression]Expression
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	for key, value := range hl.Pairs {
		pairs = append(pairs, key.String()+":"+value.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// Pattern matching related AST nodes
type MatchExpression struct {
	Token token.Token // The 'match' token
	Value Expression  // The value to match against (nil for valueless match)
	Cases []*MatchCase
}

func (me *MatchExpression) expressionNode()      {}
func (me *MatchExpression) TokenLiteral() string { return me.Token.Literal }
func (me *MatchExpression) String() string {
	var out bytes.Buffer

	out.WriteString("match")
	if me.Value != nil {
		out.WriteString(" ")
		out.WriteString(me.Value.String())
	}
	out.WriteString(" {")

	for _, c := range me.Cases {
		out.WriteString("\n    ")
		out.WriteString(c.String())
	}
	out.WriteString("\n}")

	return out.String()
}

type MatchCase struct {
	Token   token.Token  // The token for this case
	Pattern MatchPattern // The pattern to match against
	Guard   Expression   // Optional guard condition (if pattern)
	Body    *BlockStatement
}

func (mc *MatchCase) String() string {
	var out bytes.Buffer

	out.WriteString(mc.Pattern.String())

	if mc.Guard != nil {
		out.WriteString(" if ")
		out.WriteString(mc.Guard.String())
	}

	out.WriteString(" => ")
	out.WriteString(mc.Body.String())

	return out.String()
}

// MatchPattern interface for different pattern types
type MatchPattern interface {
	Node
	patternNode()
	String() string
}

// Wildcard pattern (_)
type WildcardPattern struct {
	Token token.Token // The '_' token
}

func (wp *WildcardPattern) patternNode()         {}
func (wp *WildcardPattern) TokenLiteral() string { return wp.Token.Literal }
func (wp *WildcardPattern) String() string       { return "_" }

// LiteralPattern for matching constants
type LiteralPattern struct {
	Token token.Token
	Value Expression // IntegerLiteral, StringLiteral, Boolean, etc.
}

func (lp *LiteralPattern) patternNode()         {}
func (lp *LiteralPattern) TokenLiteral() string { return lp.Token.Literal }
func (lp *LiteralPattern) String() string       { return lp.Value.String() }

// IdentifierPattern for binding values to variables
type IdentifierPattern struct {
	Token token.Token
	Value *Identifier
}

func (ip *IdentifierPattern) patternNode()         {}
func (ip *IdentifierPattern) TokenLiteral() string { return ip.Token.Literal }
func (ip *IdentifierPattern) String() string       { return ip.Value.String() }

// MultiPattern for matching against multiple patterns
type MultiPattern struct {
	Token    token.Token
	Patterns []MatchPattern
}

func (mp *MultiPattern) patternNode()         {}
func (mp *MultiPattern) TokenLiteral() string { return mp.Token.Literal }
func (mp *MultiPattern) String() string {
	var out bytes.Buffer
	patterns := []string{}

	for _, p := range mp.Patterns {
		patterns = append(patterns, p.String())
	}

	out.WriteString(strings.Join(patterns, ", "))
	return out.String()
}

// ArrayPattern for matching array structure
type ArrayPattern struct {
	Token    token.Token
	Elements []MatchPattern
}

func (ap *ArrayPattern) patternNode()         {}
func (ap *ArrayPattern) TokenLiteral() string { return ap.Token.Literal }
func (ap *ArrayPattern) String() string {
	var out bytes.Buffer
	elements := []string{}

	for _, e := range ap.Elements {
		elements = append(elements, e.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// HashPattern for matching hash structure
type HashPattern struct {
	Token  token.Token
	Pairs  map[string]MatchPattern
	Spread bool // Whether _ is present to match additional fields
}

func (hp *HashPattern) patternNode()         {}
func (hp *HashPattern) TokenLiteral() string { return hp.Token.Literal }
func (hp *HashPattern) String() string {
	var out bytes.Buffer
	pairs := []string{}

	for key, pattern := range hp.Pairs {
		if key == "_" {
			pairs = append(pairs, "_")
		} else {
			pairs = append(pairs, key+": "+pattern.String())
		}
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// ConsPattern for list destructuring patterns like a:b:c:[]
type ConsPattern struct {
	Token token.Token
	Head  MatchPattern
	Tail  MatchPattern
}

func (cp *ConsPattern) patternNode()         {}
func (cp *ConsPattern) TokenLiteral() string { return cp.Token.Literal }
func (cp *ConsPattern) String() string {
	return cp.Head.String() + ":" + cp.Tail.String()
}
