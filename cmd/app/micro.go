package main

import (
	"log"
	"slug/internal/kernel"
	"slug/internal/service"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	k := kernel.NewKernel()

	// Register services
	timeID := k.RegisterService("time", kernel.OpRights{"now": kernel.RightRead, "sleep": kernel.RightExec}, service.TimeServiceBehavior)

	// CLI Service
	cli := &service.Cli{}
	cliID := k.RegisterService("cli", kernel.OpRights{"boot": kernel.RightExec}, cli.Behavior)

	// File system service
	fs := &service.Fs{}
	fsID := k.RegisterService("fs", kernel.OpRights{"read": kernel.RightRead, "write": kernel.RightWrite}, fs.Behavior)

	// Evaluator service (stub)
	eval := &service.Evaluator{}
	evalID := k.RegisterService("eval", kernel.OpRights{"eval": kernel.RightExec}, eval.Behavior)

	// REPL service
	repl := &service.ReplService{EvalID: evalID}
	replID := k.RegisterService("repl", kernel.OpRights{"eval": kernel.RightExec}, repl.Behavior)

	// Demo actor shows FS/TIME usage
	demoID := k.RegisterActor("demo", service.DemoBehavior)

	// Cap grants
	_ = k.GrantCap(cliID, fsID, kernel.RightRead|kernel.RightWrite, nil)
	_ = k.GrantCap(demoID, fsID, kernel.RightRead|kernel.RightWrite, nil)
	_ = k.GrantCap(demoID, timeID, kernel.RightRead|kernel.RightExec, nil)
	_ = k.GrantCap(replID, evalID, kernel.RightExec, nil)                  // REPL can call EVAL
	_ = k.GrantCap(replID, replID, kernel.RightExec, nil)                  // REPL can call EVAL
	_ = k.GrantCap(evalID, fsID, kernel.RightRead|kernel.RightWrite, nil)  // EVAL can touch FS
	_ = k.GrantCap(evalID, timeID, kernel.RightRead|kernel.RightExec, nil) // EVAL can call TIME

	k.Start()
}
