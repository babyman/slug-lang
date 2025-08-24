package main

import (
	"log"
	"slug/internal/kernel"
	kernel_service "slug/internal/kernel-service"
	"slug/internal/service"
)

func main() {

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	k := kernel.NewKernel()

	controlPlane := &kernel_service.ControlPlane{}
	k.RegisterKernelService("control-plane", controlPlane)

	// Register services
	timeID := k.RegisterService("time", service.TsOperations, service.TimeServiceBehavior)

	// CLI Service
	cli := &service.Cli{}
	cliID := k.RegisterService("cli", service.CliOperations, cli.Behavior)

	// File system service
	fs := &service.Fs{}
	fsID := k.RegisterService("fs", service.FsOperations, fs.Behavior)

	// Evaluator service (stub)
	eval := &service.Evaluator{}
	evalID := k.RegisterService("eval", service.EvaluatorOperations, eval.Behavior)

	// REPL service
	repl := &service.ReplService{EvalID: evalID}
	replID := k.RegisterService("repl", service.RsOperations, repl.Behavior)

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
