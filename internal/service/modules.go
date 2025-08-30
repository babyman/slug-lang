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

		src, err := ctx.SendSync(fsId, FsRead{Path: payload.Path})
		if err != nil {
			SendInfof(ctx, "Failed to read file: %s", err)
			return
		}

		file := src.Payload.(FsReadResp).Data
		SendInfof(ctx, "Evaluating %s, got %s", payload.Path, file)

		lex, err := ctx.SendSync(lexId, LexString{Sourcecode: file})
		SendInfof(ctx, "Lexed %s, got %v", payload.Path, lex)

	}
}
