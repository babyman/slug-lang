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

		src, _ := ctx.SendSync(fsId, FsRead{Path: payload.Path})

		SendInfof(ctx, "Evaluating %s, got %s", payload.Path, src.Payload.(FsReadResp).Data)
	}
}
