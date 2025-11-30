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

	modReply, err := ctx.SendSync(modsID, modules.LoadFile{
		Path: filename,
		Args: args,
	})

	if err != nil {
		slog.Error("Error sending load file message",
			slog.Any("filename", filename),
			slog.Any("error", err))
		svc.SendStdOut(ctx, err.Error())
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 1})
		return nil
	}

	loadResult, _ := modReply.Payload.(modules.LoadModuleResult)
	if loadResult.Error != nil {
		slog.Error("Failed to load file",
			slog.Any("filename", filename),
			slog.Any("error", loadResult.Error))
		svc.SendStdOut(ctx, loadResult.Error.Error())
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 1})
		return nil
	}

	module := loadResult.Module

	future, err := ctx.SendFuture(evalId, svc.EvaluateProgram{
		Name:    module.Name,
		Path:    filename,
		Source:  module.Src,
		Program: module.Program,
		Args:    args,
	})
	if err != nil {
		slog.Warn("Failed to execute file",
			slog.Any("filename", filename),
			slog.Any("error", err))
		return nil
	}

	result, err, ok := future.AwaitTimeout(TenYears) // 10 years
	if !ok {
		slog.Error("Timeout waiting for evaluation, should never happen!")
	}
	p := result.Payload.(svc.EvaluateResult)

	if p.Error != nil {
		svc.SendStdOut(ctx, p.Error.Error()+"\n")
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 1})
	} else {
		svc.SendStdOut(ctx, p.Result+"\n")
		ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 0})
	}

	return nil
}
