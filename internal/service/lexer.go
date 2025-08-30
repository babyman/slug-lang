package service

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/lexer"
	"slug/internal/token"
)

type LexString struct {
	Sourcecode string
}

type LexedTokens struct {
	Tokens []token.Token
}

var LexerOperations = kernel.OpRights{
	reflect.TypeOf(LexString{}): kernel.RightExec,
}

type LexingService struct {
}

func (m *LexingService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case LexString:
		l := lexer.New(payload.Sourcecode)
		tokens := make([]token.Token, 0)
		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			tokens = append(tokens, tok)
		}
		Reply(ctx, msg, LexedTokens{Tokens: tokens})
	}
}
