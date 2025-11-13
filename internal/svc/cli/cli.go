package cli

import (
	"log/slog"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/svc/modules"
	"time"
)

const TenYears = time.Hour * 24 * 365 * 10

func (cli *Cli) onBoot(ctx *kernel.ActCtx) any {

	kernelID, _ := ctx.K.ActorByName(kernel.KernelService)

	ctx.SendSync(kernelID, kernel.Broadcast{
		Payload: kernel.ConfigureSystem{
			SystemRootPath: cli.RootPath,
			DebugAST:       cli.DebugAST,
		}})

	if cli.FileName != "" {
		return cli.handleCommandlineArguments(ctx, kernelID)
	}

	return nil
}

func (cli *Cli) handleCommandlineArguments(ctx *kernel.ActCtx, kernelID kernel.ActorID) any {

	filename := cli.FileName
	args := cli.Args

	slog.Info("Executing slug file",
		slog.Any("filename", filename),
		slog.Any("args", args))

	modsID, _ := ctx.K.ActorByName(svc.ModuleService)
	evalId, _ := ctx.K.ActorByName(svc.EvalService)

	modReply, err := ctx.SendSync(modsID, modules.LoadFile{ // todo
		Path: filename,
		Args: args,
	})
	if err != nil {
		slog.Error("module load error",
			slog.Any("error", err))
		return nil
	}
	loadResult, ok := modReply.Payload.(modules.LoadModuleResult)
	module := loadResult.Module
	if !ok {
		return nil
	}

	future, err := ctx.SendFuture(evalId, svc.EvaluateProgram{
		Name:    module.Name,
		Path:    filename,
		Source:  module.Src,
		Program: module.Program,
		Args:    args,
	})
	if err != nil {
		slog.Warn("Failed to execute file",
			slog.Any("error", err))
		return nil
	}

	result, err, ok := future.AwaitTimeout(TenYears) // 10 years
	p := result.Payload.(svc.EvaluateResult)

	if p.Error != nil {
		svc.SendStdOut(ctx, p.Error.Error()+"\n")
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: -100})
	} else {
		svc.SendStdOut(ctx, p.Result+"\n")
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 0})
	}

	return nil
}
