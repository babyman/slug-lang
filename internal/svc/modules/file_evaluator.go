package modules

import (
	"path/filepath"
	"slug/internal/kernel"
	"slug/internal/svc"
	"strings"
)

type FileEvaluator struct {
	DebugAST bool
	RootPath string
}

func (m *FileEvaluator) evaluateFileHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(EvaluateFile)
	m.onEvaluateFile(ctx, fwdMsg, payload)
	return kernel.Terminate{}
}

func (m *FileEvaluator) onEvaluateFile(ctx *kernel.ActCtx, msg kernel.Message, payload EvaluateFile) {

	svc.SendDebugf(ctx, "Evaluating file %s", payload.Path)

	evalId, _ := ctx.K.ActorByName(svc.EvalService)
	modsId, _ := ctx.K.ActorByName(svc.ModuleService)

	// todo fix me hard coded path with .slug removed
	modulePathParts := strings.Split("docs/examples/password-generator", string(filepath.Separator))
	//_, modulePathParts, _ := resolver.calculateModulePath(payload.Path, string(filepath.Separator))

	//modsId, _ := ctx.K.ActorByName(svc.ModuleService)
	reply, _ := ctx.SendSync(modsId, LoadModule{
		DebugAST:  m.DebugAST,
		RootPath:  m.RootPath,
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
		return
	}

	p := result.Payload
	svc.SendInfof(ctx, "Compiled %s, got %v", payload.Path, p)

	svc.Reply(ctx, msg, p)
}
