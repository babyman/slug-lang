package repl

import (
	"log/slog"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
)

type Repl struct {
}

func (r *Repl) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case RsEval:

		lexId, _ := ctx.K.ActorByName(svc.LexerService)
		parseId, _ := ctx.K.ActorByName(svc.ParserService)
		evalId, _ := ctx.K.ActorByName(svc.EvalService)

		lex, err := ctx.SendSync(lexId, lexer.LexString{Sourcecode: payload.Src})
		if err != nil {
			slog.Info("Failed to lex file", slog.Any("error", err))
			svc.Reply(ctx, fwdMsg, RsEvalResp{Error: err})
			return kernel.Continue{}
		}

		tokens := lex.Payload.(lexer.LexedTokens).Tokens

		parse, err := ctx.SendSync(parseId, parser.ParseTokens{Sourcecode: payload.Src, Tokens: tokens})
		if err != nil {
			slog.Warn("Failed to parse file", slog.Any("error", err))
			svc.Reply(ctx, fwdMsg, RsEvalResp{Error: err})
			return kernel.Continue{}
		}

		ast := parse.Payload.(parser.ParsedAst)

		exec, err := ctx.SendSync(evalId, svc.EvaluateProgram{
			Source:  payload.Src,
			Program: ast.Program,
		})

		p := exec.Payload.(svc.EvaluateResult)

		if p.Error != nil {
			svc.Reply(ctx, fwdMsg, RsEvalResp{Error: p.Error})
		} else {
			svc.Reply(ctx, fwdMsg, RsEvalResp{Result: p.Result})
		}

	}
	return kernel.Continue{}
}
