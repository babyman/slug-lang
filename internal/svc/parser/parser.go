package parser

import (
	"reflect"
	"slug/internal/ast"
	"slug/internal/kernel"
	"slug/internal/parser"
	"slug/internal/svc"
	"slug/internal/token"
)

type ParseTokens struct {
	Sourcecode string
	Tokens     []token.Token
}

type ParsedAst struct {
	Program *ast.Program
}

var ParserOperations = kernel.OpRights{
	reflect.TypeOf(ParseTokens{}): kernel.RightExec,
}

type TokenSliceProvider struct {
	tokens []token.Token
	pos    int
}

func NewTokenSliceProvider(tokens []token.Token) *TokenSliceProvider {
	return &TokenSliceProvider{
		tokens: tokens,
		pos:    0,
	}
}

func (tsp *TokenSliceProvider) NextToken() token.Token {
	if tsp.pos >= len(tsp.tokens) {
		return token.Token{Type: token.EOF}
	}
	tok := tsp.tokens[tsp.pos]
	tsp.pos++
	return tok
}

type ParserService struct {
}

func (m *ParserService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case ParseTokens:
		p := parser.New(NewTokenSliceProvider(payload.Tokens), payload.Sourcecode)
		program := p.ParseProgram()
		svc.SendInfof(ctx, "Parsed program: %v", program)
		svc.Reply(ctx, msg, ParsedAst{Program: program})
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
}
