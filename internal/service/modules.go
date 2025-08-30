package service

import (
	"reflect"
	"slug/internal/kernel"
)

type ModuleEvaluateFile struct {
	Path string
	Args []string
}

var ModulesOperations = kernel.OpRights{
	reflect.TypeOf(ModuleEvaluateFile{}): kernel.RightExec,
}

type Modules struct {
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case ModuleEvaluateFile:

		SendInfof(ctx, "Evaluating file %s", payload.Path)

		fsId, _ := ctx.K.ActorByName("fs")
		lexId, _ := ctx.K.ActorByName("lexer")
		parseId, _ := ctx.K.ActorByName("parser")

		src, err := ctx.SendSync(fsId, FsRead{Path: payload.Path})
		if err != nil {
			SendInfof(ctx, "Failed to read file: %s", err)
			return
		}

		file := src.Payload.(FsReadResp).Data
		//SendInfof(ctx, "Evaluating %s, got %s", payload.Path, file)

		lex, err := ctx.SendSync(lexId, LexString{Sourcecode: file})
		if err != nil {
			SendInfof(ctx, "Failed to lex file: %s", err)
			return
		}

		tokens := lex.Payload.(LexedTokens).Tokens
		SendInfof(ctx, "Lexed %s, got %v", payload.Path, tokens)

		parse, err := ctx.SendSync(parseId, ParseTokens{Sourcecode: file, Tokens: tokens})
		if err != nil {
			SendInfof(ctx, "Failed to lex file: %s", err)
			return
		}

		SendInfof(ctx, "Lexed %s, got %v", payload.Path, parse)

	}
}
