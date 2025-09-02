package lexer

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/token"
)

type LexString struct {
	Sourcecode string
}

type LexedTokens struct {
	Tokens []token.Token
}

var Operations = kernel.OpRights{
	reflect.TypeOf(LexString{}): kernel.RightExec,
}

type LexingService struct {
}

func (m *LexingService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case LexString:
		l := New(payload.Sourcecode)
		tokens := make([]token.Token, 0)
		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			tokens = append(tokens, tok)
		}
		svc.Reply(ctx, msg, LexedTokens{Tokens: tokens})
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
}
