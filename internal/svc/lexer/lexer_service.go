package lexer

import (
	"log/slog"
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

func (m *LexingService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case LexString:
		workedId, _ := ctx.SpawnChild("lex-wrk", Operations, lexHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error sending message",
				slog.Any("pid", workedId),
				slog.Any("error", err.Error()))
		}

	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func lexHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case LexString:
		l := New(payload.Sourcecode)
		tokens := make([]token.Token, 0)
		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
			tokens = append(tokens, tok)
		}
		svc.Reply(ctx, fwdMsg, LexedTokens{Tokens: tokens})
	}
	return kernel.Terminate{}
}
