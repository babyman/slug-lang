package ast

import (
	"bytes"
	"encoding/hex"
	"slug/internal/dec64"
	"slug/internal/token"
	"strings"
)

type FSig struct {
	Tags       string
	Min        int
	Max        int
	IsVariadic bool
}

// The base Node interface
type Node interface {
	TokenLiteral() string
	String() string
}

type Statement interface {
	Node
	statementNode()
}

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

type VarExpression struct {
	Tags    []*Tag
	Token   token.Token // the token.VAR token
	Pattern MatchPattern
	Value   Expression
}

func (ls *VarExpression) expressionNode()      {}
func (ls *VarExpression) TokenLiteral() string { return ls.Token.Literal }
func (ls *VarExpression) String() string {
	var out bytes.Buffer

	for _, a := range ls.Tags {
		out.WriteString(a.String())
		out.WriteString(" ")
	}

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Pattern.String())

	if ls.Value != nil {
		out.WriteString(" = ")
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

type ValExpression struct {
	Tags    []*Tag
	Token   token.Token  // The token.VAL token
	Pattern MatchPattern // Constant name
	Value   Expression   // The assigned value
}

func (vs *ValExpression) expressionNode()      {}
func (vs *ValExpression) TokenLiteral() string { return vs.Token.Literal }
func (vs *ValExpression) String() string {
	var out bytes.Buffer

	for _, a := range vs.Tags {
		out.WriteString(a.String())
		out.WriteString(" ")
	}

	out.WriteString(vs.TokenLiteral() + " ")
	out.WriteString(vs.Pattern.String())
	out.WriteString(" = ")
	if vs.Value != nil {
		out.WriteString(vs.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type ForeignFunctionDeclaration struct {
	Tags       []*Tag
	Token      token.Token // The `FOREIGN` token
	Name       *Identifier // Name of the foreign function
	Parameters []*FunctionParameter
	Signature  FSig
}

func (ffd *ForeignFunctionDeclaration) statementNode()       {}
func (ffd *ForeignFunctionDeclaration) TokenLiteral() string { return ffd.Token.Literal }
func (ffd *ForeignFunctionDeclaration) String() string {
	var out bytes.Buffer

	for _, a := range ffd.Tags {
		out.WriteString(a.String())
		out.WriteString(" ")
	}

	out.WriteString("foreign ")
	out.WriteString(ffd.Name.String())
	out.WriteString(" = ")

	params := []string{}
	for _, p := range ffd.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(ffd.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")

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
	IsAsync    bool       // handles async { ... }
	Limit      Expression // handles limit N { ... }
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) expressionNode()      {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	if bs.IsAsync {
		out.WriteString("async ")
		if bs.Limit != nil {
			out.WriteString("limit ")
			out.WriteString(bs.Limit.String())
			out.WriteString(" ")
		}
	}
	out.WriteString("{")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	out.WriteString("}")

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

type NumberLiteral struct {
	Token token.Token
	Value dec64.Dec64
}

func (n *NumberLiteral) expressionNode()      {}
func (n *NumberLiteral) TokenLiteral() string { return n.Token.Literal }
func (n *NumberLiteral) String() string       { return n.Token.Literal }

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
		out.WriteString(" else ")
		out.WriteString(ie.ElseBranch.String())
	}

	return out.String()
}

type FunctionLiteral struct {
	Token       token.Token // The 'fn' token
	Signature   FSig
	Parameters  []*FunctionParameter
	Body        *BlockStatement
	HasTailCall bool // Whether this function has tail calls
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

type RecurExpression struct {
	Token     token.Token  // the 'recur' token
	Arguments []Expression // recur(arg1, arg2, ...)
}

func (r *RecurExpression) expressionNode()      {}
func (r *RecurExpression) TokenLiteral() string { return r.Token.Literal }
func (r *RecurExpression) String() string {
	out := bytes.Buffer{}
	out.WriteString("recur(")
	for i, a := range r.Arguments {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(a.String())
	}
	out.WriteString(")")
	return out.String()
}

type SpawnExpression struct {
	Token token.Token // The 'spawn' token
	Body  Expression  // Usually a BlockStatement or FunctionLiteral
}

func (se *SpawnExpression) expressionNode()      {}
func (se *SpawnExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpawnExpression) String() string {
	var out bytes.Buffer
	out.WriteString("spawn ")
	out.WriteString(se.Body.String())
	return out.String()
}

type AwaitExpression struct {
	Token   token.Token // The 'await' token
	Value   Expression  // The task handle
	Timeout Expression  // The 'within' expression (optional)
}

func (ae *AwaitExpression) expressionNode()      {}
func (ae *AwaitExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AwaitExpression) String() string {
	var out bytes.Buffer
	out.WriteString("await ")
	out.WriteString(ae.Value.String())
	if ae.Timeout != nil {
		out.WriteString(" within ")
		out.WriteString(ae.Timeout.String())
	}
	return out.String()
}

type CallExpression struct {
	Token      token.Token // The '(' token
	Function   Expression  // Identifier or FunctionLiteral
	Arguments  []Expression
	IsTailCall bool // Whether this is a tail call
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
	Tags       []*Tag
	Name       *Identifier // Parameter name
	Default    Expression  // Default value (optional)
	IsVariadic bool        // Whether this is a variadic argument
}

func (p *FunctionParameter) expressionNode()      {}
func (p *FunctionParameter) TokenLiteral() string { return p.Name.Token.Literal }
func (p *FunctionParameter) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	for _, tag := range p.Tags {
		out.WriteString(tag.String() + " ")
	}
	if p.IsVariadic {
		out.WriteString("...")
	}
	out.WriteString(p.Name.String())
	if p.Default != nil {
		out.WriteString("=")
		out.WriteString(p.Default.String())
	}
	out.WriteString(")")

	return out.String()
}

type BytesLiteral struct {
	Token token.Token
	Value []byte
}

func (bl *BytesLiteral) expressionNode()      {}
func (bl *BytesLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BytesLiteral) String() string {
	return `0x"` + hex.EncodeToString(bl.Value) + `"`
}

type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string       { return sl.Token.Literal }

type ListLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (al *ListLiteral) expressionNode()      {}
func (al *ListLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ListLiteral) String() string {
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

type SliceExpression struct {
	Token token.Token // The '[' token
	Start Expression  // Start index
	End   Expression  // End index
	Step  Expression  // Step value (optional)
}

func (se *SliceExpression) expressionNode()      {}
func (se *SliceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SliceExpression) String() string {
	var out bytes.Buffer
	if se.Start != nil {
		out.WriteString(se.Start.String())
	}
	out.WriteString(":")
	if se.End != nil {
		out.WriteString(se.End.String())
	}
	if se.Step != nil {
		out.WriteString(":")
		out.WriteString(se.Step.String())
	}
	return out.String()
}

type MapLiteral struct {
	Token token.Token // the '{' token
	Pairs map[Expression]Expression
}

func (hl *MapLiteral) expressionNode()      {}
func (hl *MapLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *MapLiteral) String() string {
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

type MatchExpression struct {
	Token token.Token // The 'match' token
	Value Expression  // The value to match against
	Cases []*MatchCase
}

func (m *MatchExpression) expressionNode()      {}
func (m *MatchExpression) TokenLiteral() string { return m.Token.Literal }
func (m *MatchExpression) String() string {
	var out bytes.Buffer

	out.WriteString("match")
	out.WriteString(" ")
	out.WriteString(m.Value.String())
	out.WriteString(" {")

	for _, c := range m.Cases {
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

func (m *MatchCase) expressionNode()      {}
func (m *MatchCase) TokenLiteral() string { return m.Token.Literal }
func (m *MatchCase) String() string {
	var out bytes.Buffer

	out.WriteString(m.Pattern.String())

	if m.Guard != nil {
		out.WriteString(" if ")
		out.WriteString(m.Guard.String())
	}

	out.WriteString(" => ")
	out.WriteString(m.Body.String())

	return out.String()
}

// MatchPattern interface for different pattern types
type MatchPattern interface {
	Node
	patternNode()
	String() string
}

// AllPattern  (_)
type AllPattern struct {
	Token token.Token // The '*' token
}

func (wp *AllPattern) expressionNode()      {}
func (wp *AllPattern) patternNode()         {}
func (wp *AllPattern) TokenLiteral() string { return wp.Token.Literal }
func (wp *AllPattern) String() string       { return "*" }

// WildcardPattern  (_)
type WildcardPattern struct {
	Token token.Token // The '_' token
}

func (wp *WildcardPattern) expressionNode()      {}
func (wp *WildcardPattern) patternNode()         {}
func (wp *WildcardPattern) TokenLiteral() string { return wp.Token.Literal }
func (wp *WildcardPattern) String() string       { return "_" }

// SpreadPattern  (_)
type SpreadPattern struct {
	Token token.Token // The '...' token
	Value *Identifier // identifier for the spread if bound
}

func (wp *SpreadPattern) expressionNode()      {}
func (wp *SpreadPattern) patternNode()         {}
func (wp *SpreadPattern) TokenLiteral() string { return wp.Token.Literal }
func (wp *SpreadPattern) String() string {
	var out bytes.Buffer

	out.WriteString(wp.Token.Literal)

	if wp.Value != nil {
		out.WriteString(wp.Value.String())
	}

	return out.String()
}

// LiteralPattern for matching constants
type LiteralPattern struct {
	Token token.Token
	Value Expression // NumberLiteral, StringLiteral, Boolean, etc.
}

func (lp *LiteralPattern) expressionNode()      {}
func (lp *LiteralPattern) patternNode()         {}
func (lp *LiteralPattern) TokenLiteral() string { return lp.Token.Literal }
func (lp *LiteralPattern) String() string       { return lp.Value.String() }

// IdentifierPattern for binding values to variables
type IdentifierPattern struct {
	Token token.Token
	Value *Identifier
}

func (ip *IdentifierPattern) expressionNode()      {}
func (ip *IdentifierPattern) patternNode()         {}
func (ip *IdentifierPattern) TokenLiteral() string { return ip.Token.Literal }
func (ip *IdentifierPattern) String() string       { return ip.Value.String() }

// PinnedIdentifierPattern for matching against an existing identifier from an enclosing scope
// Syntax: ^name
// Semantics:
//   - name must already exist in the enclosing lexical scope (outside pattern bindings)
//   - the pattern matches iff matchedValue == env[name]
//   - it does NOT bind
type PinnedIdentifierPattern struct {
	Token token.Token // the '^' token
	Value *Identifier // the pinned identifier name
}

func (pp *PinnedIdentifierPattern) expressionNode()      {}
func (pp *PinnedIdentifierPattern) patternNode()         {}
func (pp *PinnedIdentifierPattern) TokenLiteral() string { return pp.Token.Literal }
func (pp *PinnedIdentifierPattern) String() string {
	var out bytes.Buffer
	out.WriteString("^")
	if pp.Value != nil {
		out.WriteString(pp.Value.String())
	}
	return out.String()
}

// MultiPattern for matching against multiple patterns
type MultiPattern struct {
	Token    token.Token
	Patterns []MatchPattern
}

func (mp *MultiPattern) expressionNode()      {}
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

// ListPattern for matching list structure
type ListPattern struct {
	Token    token.Token
	Elements []MatchPattern
}

func (ap *ListPattern) expressionNode()      {}
func (ap *ListPattern) patternNode()         {}
func (ap *ListPattern) TokenLiteral() string { return ap.Token.Literal }
func (ap *ListPattern) String() string {
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

// MapPattern for matching map structure
type MapPattern struct {
	Token     token.Token             // The token representing the map pattern.
	Pairs     map[string]MatchPattern // List of keys for matching (for `{key}`).
	Spread    bool                    // Whether ... is present to match additional fields
	Exact     bool                    // True for exact match patterns `{|k1, k2|}`.
	SelectAll bool                    // True for wildcard match `{*}`.
}

func (mp *MapPattern) expressionNode()      {}
func (mp *MapPattern) patternNode()         {}
func (mp *MapPattern) TokenLiteral() string { return mp.Token.Literal }
func (mp *MapPattern) String() string {
	var out bytes.Buffer
	pairs := []string{}

	for key, pattern := range mp.Pairs {
		if key == "_" {
			pairs = append(pairs, "_")
		} else {
			pairs = append(pairs, key+": "+pattern.String())
		}
	}

	out.WriteString("{")
	if mp.Exact {
		out.WriteString("|")
	}
	if mp.SelectAll {
		out.WriteString("*")
	}
	out.WriteString(strings.Join(pairs, ", "))
	if mp.Exact {
		out.WriteString("|")
	}
	out.WriteString("}")

	return out.String()
}

type ThrowStatement struct {
	Token token.Token // The 'throw' token
	Value Expression  // The expression to be thrown
}

func (ts *ThrowStatement) statementNode()       {}
func (ts *ThrowStatement) TokenLiteral() string { return ts.Token.Literal }
func (ts *ThrowStatement) String() string {
	var out bytes.Buffer
	out.WriteString(ts.TokenLiteral() + " ")
	if ts.Value != nil {
		out.WriteString(ts.Value.String())
	}
	out.WriteString(";")
	return out.String()
}

type NotImplemented struct {
	Token token.Token // The ??? token
}

func (ni *NotImplemented) expressionNode()      {}
func (ni *NotImplemented) TokenLiteral() string { return ni.Token.Literal }
func (ni *NotImplemented) String() string       { return ni.Token.Literal }

type DeferMode int

const (
	DeferAlways DeferMode = iota
	DeferOnSuccess
	DeferOnError
)

type DeferStatement struct {
	Token     token.Token // The 'defer' token
	Call      Statement   // Expression or block to execute later
	Mode      DeferMode
	ErrorName *Identifier // Only set if Mode == DeferOnError
}

func (ds *DeferStatement) statementNode()       {}
func (ds *DeferStatement) TokenLiteral() string { return ds.Token.Literal }
func (ds *DeferStatement) String() string {
	var out bytes.Buffer
	out.WriteString("defer ")
	switch ds.Mode {
	case DeferOnSuccess:
		out.WriteString("onsuccess ")
	case DeferOnError:
		out.WriteString("onerror")
		if ds.ErrorName != nil {
			out.WriteString("(")
			out.WriteString(ds.ErrorName.String())
			out.WriteString(")")
		}
		out.WriteString(" ")
	}
	out.WriteString(ds.Call.String())
	return out.String()
}

type SpreadExpression struct {
	Token token.Token // The `...` token
	Value Expression  // The expression to spread
}

func (se *SpreadExpression) expressionNode()      {}
func (se *SpreadExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SpreadExpression) String() string {
	var out bytes.Buffer
	out.WriteString("...")
	out.WriteString(se.Value.String())
	return out.String()
}

type Tag struct {
	Token token.Token  // The '@' token
	Name  string       // The tag name (e.g., "tag")
	Args  []Expression // Arguments as expressions, e.g., ["42", "x + 1"]
}

func (a *Tag) expressionNode()      {}
func (a *Tag) TokenLiteral() string { return a.Token.Literal }
func (a *Tag) String() string {
	var out bytes.Buffer
	out.WriteString(a.Name)
	if len(a.Args) > 0 {
		args := []string{}
		for _, arg := range a.Args {
			args = append(args, arg.String())
		}
		out.WriteString("(")
		out.WriteString(strings.Join(args, ", "))
		out.WriteString(")")
	}
	return out.String()
}
