package main

import (
	"slug/internal/kernel"
	"slug/internal/privileged"
	"slug/internal/svc/cli"
	"slug/internal/svc/eval"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/log"
	"slug/internal/svc/modules"
	"slug/internal/svc/parser"
	"slug/internal/svc/repl"
	"slug/internal/svc/sout"
)

const (
	r  = kernel.RightRead
	rw = kernel.RightRead | kernel.RightWrite
	rx = kernel.RightRead | kernel.RightExec
	w  = kernel.RightWrite
	wx = kernel.RightWrite | kernel.RightExec
	x  = kernel.RightExec
)

func main() {

	k := kernel.NewKernel()

	kernelID, _ := k.ActorByName("kernel")

	controlPlane := &privileged.ControlPlane{}
	k.RegisterPrivilegedService("control-plane", controlPlane)

	// system out Service
	out := &sout.SOut{}
	soutID := k.RegisterService("sout", sout.Operations, out.Handler)

	// system out Service
	logSvc := &log.Log{}
	logID := k.RegisterService("log", log.Operations, logSvc.Handler)

	// system out Service
	mods := &modules.Modules{}
	modsID := k.RegisterService("mods", modules.Operations, mods.Handler)

	// CLI Service
	cliSvc := &cli.Cli{}
	cliID := k.RegisterService("cli", cli.Operations, cliSvc.Handler)

	// File system service
	fsSvc := &fs.Fs{}
	fsID := k.RegisterService("fs", fs.Operations, fsSvc.Handler)

	// Lexer service
	lexerSvc := &lexer.LexingService{}
	lexerID := k.RegisterService("lexer", lexer.Operations, lexerSvc.Handler)

	// Parser service
	parserSvc := &parser.ParserService{}
	parserID := k.RegisterService("parser", parser.Operations, parserSvc.Handler)

	// Evaluator service
	evalSvc := &eval.EvaluatorService{}
	evalID := k.RegisterService("eval", eval.Operations, evalSvc.Handler)

	// REPL service
	replSvc := &repl.ReplService{EvalID: evalID}
	replID := k.RegisterService("repl", repl.Operations, replSvc.Handler)

	// Cap grants
	_ = k.GrantCap(kernelID, cliID, x, nil)

	_ = k.GrantCap(cliID, cliID, x, nil) // call self required for boot message

	_ = k.GrantCap(cliID, soutID, w, nil)
	_ = k.GrantCap(cliID, modsID, x, nil)
	_ = k.GrantCap(cliID, logID, wx, nil)
	_ = k.GrantCap(cliID, kernelID, x, nil)

	_ = k.GrantCap(modsID, cliID, w, nil)
	_ = k.GrantCap(modsID, fsID, r, nil)
	_ = k.GrantCap(modsID, lexerID, x, nil)
	_ = k.GrantCap(modsID, parserID, x, nil)
	_ = k.GrantCap(modsID, evalID, x, nil)
	_ = k.GrantCap(modsID, logID, w, nil)

	_ = k.GrantCap(fsID, logID, w, nil)

	_ = k.GrantCap(lexerID, logID, w, nil)

	_ = k.GrantCap(parserID, logID, w, nil)

	_ = k.GrantCap(evalID, logID, w, nil)

	_ = k.GrantCap(replID, replID, x, nil) // REPL can call itself
	_ = k.GrantCap(replID, evalID, x, nil) // REPL can call EVAL
	_ = k.GrantCap(replID, logID, w, nil)

	k.Start()
}
