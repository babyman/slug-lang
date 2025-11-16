package parser

import (
	"log/slog"
	"reflect"
	"slug/internal/ast"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/token"
)

type ParseTokens struct {
	Sourcecode string
	Tokens     []token.Token
}

type ParsedAst struct {
	Program *ast.Program
	Errors  []string
}

var Operations = kernel.OpRights{
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

type Service struct {
}

func (s *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case ParseTokens:
		workedId, _ := ctx.SpawnChild("parse-wrk", Operations, parseHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error sending message to new parser", slog.Any("error", err.Error()))
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func parseHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case ParseTokens:
		p := New(NewTokenSliceProvider(payload.Tokens), payload.Sourcecode)
		program := p.ParseProgram()

		slog.Debug("Parsed program", slog.Any("program-ast", program))
		svc.Reply(ctx, fwdMsg, ParsedAst{
			Program: program,
			Errors:  p.errors,
		})
	}
	return kernel.Terminate{}
}
