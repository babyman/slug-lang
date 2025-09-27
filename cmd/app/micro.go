package main

import (
	"slug/internal/kernel"
	"slug/internal/privileged"
	"slug/internal/svc"
	"slug/internal/svc/cli"
	"slug/internal/svc/eval"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/log"
	"slug/internal/svc/modules"
	"slug/internal/svc/parser"
	"slug/internal/svc/repl"
	"slug/internal/svc/resolver"
	"slug/internal/svc/sout"
)

const (
	r   = kernel.RightRead
	rw  = kernel.RightRead | kernel.RightWrite
	rwx = kernel.RightRead | kernel.RightWrite | kernel.RightExec
	rx  = kernel.RightRead | kernel.RightExec
	w   = kernel.RightWrite
	wx  = kernel.RightWrite | kernel.RightExec
	x   = kernel.RightExec
)

func main() {

	k := kernel.NewKernel()

	kernelID, _ := k.ActorByName(kernel.KernelService)

	controlPlane := &privileged.ControlPlane{}
	k.RegisterPrivilegedService(privileged.ControlPlaneService, controlPlane)

	// system out Service
	out := &sout.SOut{}
	soutID := k.RegisterService(svc.SOutService, sout.Operations, out.Handler)

	// system out Service
	logSvc := &log.LogService{}
	logID := k.RegisterService(svc.LogService, log.Operations, logSvc.Handler)

	// system out Service
	mods := modules.NewModules()
	modsID := k.RegisterService(svc.ModuleService, modules.Operations, mods.Handler)

	// system out Service
	res := resolver.NewResolver()
	resID := k.RegisterService(svc.ResolverService, resolver.Operations, res.Handler)

	// CLI Service
	cliSvc := &cli.Cli{}
	cliID := k.RegisterService(svc.CliService, cli.Operations, cliSvc.Handler)

	// File system service
	fsSvc := &fs.Fs{}
	fsID := k.RegisterService(svc.FsService, fs.Operations, fsSvc.Handler)

	// Lexer service
	lexerSvc := &lexer.LexingService{}
	lexerID := k.RegisterService(svc.LexerService, lexer.Operations, lexerSvc.Handler)

	// Parser service
	parserSvc := &parser.Service{}
	parserID := k.RegisterService(svc.ParserService, parser.Operations, parserSvc.Handler)

	// Evaluator service
	evalSvc := &eval.EvaluatorService{}
	evalID := k.RegisterService(svc.EvalService, eval.Operations, evalSvc.Handler)

	// REPL service
	replSvc := &repl.ReplService{EvalID: evalID}
	replID := k.RegisterService(svc.ReplService, repl.Operations, replSvc.Handler)

	// Cap grants
	_ = k.GrantCap(kernelID, cliID, x, nil)

	_ = k.GrantCap(cliID, resID, x, nil)   // x for kernel.ConfigureSystem
	_ = k.GrantCap(cliID, modsID, rx, nil) // x for kernel.ConfigureSystem
	_ = k.GrantCap(cliID, logID, wx, nil)  // x for kernel.ConfigureSystem
	_ = k.GrantCap(cliID, soutID, w, nil)
	_ = k.GrantCap(cliID, evalID, x, nil)
	_ = k.GrantCap(cliID, kernelID, x, nil)

	_ = k.GrantCap(modsID, resID, r, nil)
	_ = k.GrantCap(modsID, lexerID, x, nil)
	_ = k.GrantCap(modsID, parserID, x, nil)
	_ = k.GrantCap(modsID, logID, w, nil)

	_ = k.GrantCap(resID, fsID, r, nil)
	_ = k.GrantCap(resID, logID, w, nil)

	_ = k.GrantCap(fsID, logID, w, nil)

	_ = k.GrantCap(lexerID, logID, w, nil)

	_ = k.GrantCap(parserID, logID, w, nil)

	_ = k.GrantCap(evalID, modsID, r, nil)
	_ = k.GrantCap(evalID, soutID, w, nil)
	_ = k.GrantCap(evalID, logID, w, nil)

	_ = k.GrantCap(replID, evalID, x, nil) // REPL can call EVAL
	_ = k.GrantCap(replID, logID, w, nil)

	k.Start()
}
