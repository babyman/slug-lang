package modules

import (
	"path/filepath"
	"slug/internal/kernel"
	"slug/internal/svc"
	"strings"
)

func onEvaluateFile(ctx *kernel.ActCtx, msg kernel.Message, payload EvaluateFile) kernel.HandlerSignal {

	svc.SendDebugf(ctx, "Evaluating file %s", payload.Path)

	evalId, _ := ctx.K.ActorByName(svc.EvalService)

	// todo fix me hard coded path with .slug removed
	modulePathParts := strings.Split("docs/examples/password-generator", string(filepath.Separator))

	modsId, _ := ctx.K.ActorByName(svc.ModuleService)
	reply, _ := ctx.SendSync(modsId, LoadModule{
		DebugAST:  false, // todo fix me
		RootPath:  ".",   // todo fix me
		PathParts: modulePathParts,
	})

	loadResult, _ := reply.Payload.(LoadModuleResult)
	module := loadResult.Module

	result, err := ctx.SendSync(evalId, svc.EvaluateProgram{
		Name:    module.Name,
		Path:    payload.Path,
		Source:  module.Src,
		Program: module.Program,
		Args:    payload.Args,
	})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to execute file: %s", err)
		return kernel.Continue{}
	}

	p := result.Payload
	svc.SendInfof(ctx, "Compiled %s, got %v", payload.Path, p)

	svc.Reply(ctx, msg, p)

	return kernel.Continue{}
}
