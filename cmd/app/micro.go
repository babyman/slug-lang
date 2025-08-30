package main

import (
	"log"
	"slug/internal/kernel"
	"slug/internal/privileged"
	"slug/internal/service"
)

const (
	r  = kernel.RightRead
	w  = kernel.RightWrite
	rw = kernel.RightRead | kernel.RightWrite
	rx = kernel.RightRead | kernel.RightExec
	wx = kernel.RightWrite | kernel.RightExec
	x  = kernel.RightExec
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	k := kernel.NewKernel()

	controlPlane := &privileged.ControlPlane{}
	k.RegisterPrivilegedService("control-plane", controlPlane)

	// system out Service
	sout := &service.SOut{}
	soutID := k.RegisterService("sout", service.SOutOperations, sout.Handler)

	// system out Service
	logSvc := &service.Log{}
	logID := k.RegisterService("log", service.LogOperations, logSvc.Handler)

	// system out Service
	mods := &service.Modules{}
	modsID := k.RegisterService("mods", service.ModulesOperations, mods.Handler)

	// CLI Service
	cli := &service.Cli{}
	cliID := k.RegisterService("cli", service.CliOperations, cli.Handler)

	// File system service
	fs := &service.Fs{}
	fsID := k.RegisterService("fs", service.FsOperations, fs.Handler)

	// Lexer service
	lexer := &service.LexingService{}
	lexerID := k.RegisterService("lexer", service.LexerOperations, lexer.Handler)

	// Parser service
	parser := &service.ParserService{}
	parserID := k.RegisterService("parser", service.ParserOperations, parser.Handler)

	// Evaluator service
	eval := &service.EvaluatorService{}
	evalID := k.RegisterService("eval", service.EvaluatorOperations, eval.Handler)

	// REPL service
	repl := &service.ReplService{EvalID: evalID}
	replID := k.RegisterService("repl", service.RsOperations, repl.Handler)

	// Cap grants
	_ = k.GrantCap(cliID, cliID, x, nil) // call self required for boot message
	_ = k.GrantCap(cliID, soutID, w, nil)
	_ = k.GrantCap(cliID, modsID, x, nil)
	_ = k.GrantCap(cliID, logID, wx, nil)

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
