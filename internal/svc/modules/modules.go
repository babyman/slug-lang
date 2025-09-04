package modules

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/svc/eval"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
)

type ModuleEvaluateFile struct {
	Path string
	Args []string
}

var Operations = kernel.OpRights{
	reflect.TypeOf(ModuleEvaluateFile{}): kernel.RightExec,
}

type Modules struct {
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case ModuleEvaluateFile:

		svc.SendDebugf(ctx, "Evaluating file %s", payload.Path)

		fsId, _ := ctx.K.ActorByName(svc.FsService)
		lexId, _ := ctx.K.ActorByName(svc.LexerService)
		parseId, _ := ctx.K.ActorByName(svc.ParserService)
		evalId, _ := ctx.K.ActorByName(svc.EvalService)

		src, err := ctx.SendSync(fsId, fs.FsRead{Path: payload.Path})
		if err != nil {
			svc.SendWarnf(ctx, "Failed to read file: %s", err)
			return kernel.Continue{}
		}

		file := src.Payload.(fs.FsReadResp).Data
		//service.SendInfof(ctx, "Evaluating %s, got %s", payload.Path, file)

		lex, err := ctx.SendSync(lexId, lexer.LexString{Sourcecode: file})
		if err != nil {
			svc.SendInfof(ctx, "Failed to lex file: %s", err)
			return kernel.Continue{}
		}

		tokens := lex.Payload.(lexer.LexedTokens).Tokens
		svc.SendDebugf(ctx, "Lexed %s, got %v", payload.Path, tokens)

		parse, err := ctx.SendSync(parseId, parser.ParseTokens{Sourcecode: file, Tokens: tokens})
		if err != nil {
			svc.SendWarnf(ctx, "Failed to parse file: %s", err)
			return kernel.Continue{}
		}

		ast := parse.Payload.(parser.ParsedAst).Program
		svc.SendDebugf(ctx, "Compiled %s, got %v", payload.Path, ast)

		result, err := ctx.SendSync(evalId, eval.EvaluateProgram{
			Source:  file,
			Program: ast,
			Args:    payload.Args,
		})
		if err != nil {
			svc.SendWarnf(ctx, "Failed to execute file: %s", err)
			return kernel.Continue{}
		}

		p := result.Payload
		svc.SendInfof(ctx, "Compiled %s, got %v", payload.Path, p)

		svc.Reply(ctx, msg, p)

	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
