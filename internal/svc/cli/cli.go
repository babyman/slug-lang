package cli

import (
	"flag"
	"fmt"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc"
	"slug/internal/svc/modules"
	"slug/internal/svc/resolver"
	"time"
)

var (
	rootPath string
	debugAST bool
	logLevel string
	logFile  string
	help     bool
	version  bool
)

const TenYears = time.Hour * 24 * 365 * 10

func init() {
	flag.BoolVar(&help, "help", false, "Display help information and exit")
	flag.BoolVar(&help, "h", false, "Display help information and exit")
	flag.BoolVar(&version, "version", false, "Display version information and exit")
	flag.BoolVar(&version, "v", false, "Display version information and exit")
	// evaluator config
	flag.StringVar(&rootPath, "root", resolver.DefaultRootPath, "Set the root context for the program (used for imports)")
	// parser config
	flag.BoolVar(&debugAST, "debug-ast", false, "Render the AST as a JSON file")
	// log config
	flag.StringVar(&logLevel, "log-level", "NONE", "Log level: trace, debug, info, warn, error, none")
	flag.StringVar(&logFile, "log-file", "", "Log file path (if not set, logs to stderr)")
}

func (cli *Cli) onBoot(ctx *kernel.ActCtx) any {

	flag.Parse()

	kernelID, _ := ctx.K.ActorByName(kernel.KernelService)

	if version {
		cli.handleVersionRequest(ctx, kernelID)
		return nil
	}

	if help {
		cli.handleHelpRequest(ctx, kernelID)
		return nil
	}

	if len(flag.Args()) > 0 {
		cli.configureSystem(ctx)
		return cli.handleCommandlineArguments(ctx, kernelID)
	}

	return nil
}

func (cli *Cli) configureSystem(ctx *kernel.ActCtx) (kernel.Message, error) {

	kernelID, _ := ctx.K.ActorByName(kernel.KernelService)
	level := logger.ParseLevel(logLevel)
	return ctx.SendSync(kernelID, kernel.Broadcast{
		Payload: kernel.ConfigureSystem{
			LogLevel:       level,
			LogPath:        logFile,
			SystemRootPath: rootPath,
			DebugAST:       debugAST,
		}})
}

func (cli *Cli) handleCommandlineArguments(ctx *kernel.ActCtx, kernelID kernel.ActorID) any {

	filename := flag.Args()[0]
	args := flag.Args()[1:]

	svc.SendInfof(ctx, "Executing %s with args %v", filename, args)

	modsID, _ := ctx.K.ActorByName(svc.ModuleService)
	evalId, _ := ctx.K.ActorByName(svc.EvalService)

	modReply, err := ctx.SendSync(modsID, modules.LoadFile{ // todo
		Path: filename,
		Args: args,
	})
	if err != nil {
		svc.SendErrorf(ctx, "err: %v", err)
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
		svc.SendWarnf(ctx, "Failed to execute file: %s", err)
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

func (cli *Cli) handleVersionRequest(ctx *kernel.ActCtx, kernelID kernel.ActorID) {

	svc.SendStdOut(ctx, fmt.Sprintf("slug version 'v%s' %s %s\n", cli.Version, cli.BuildDate, cli.Commit))
	ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 0})
}

func (cli *Cli) handleHelpRequest(ctx *kernel.ActCtx, kernelID kernel.ActorID) {

	svc.SendStdOut(ctx, cli.helpMessage())
	ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 0})
}

func (cli *Cli) helpMessage() string {
	return `Usage: slug [options] [filename [args...]]

Options:
  -root <path>       Set the root context for the program (used for imports). Default is '.'
  -debug-ast         Render the AST as a JSON file.
  -help              Display this help information and exit.
  -version           Display version information and exit.
  -log-level <level> Set the log level: trace, debug, info, warn, error, none. Default is 'none'.
  -log-file <path>   Specify a log file to write logs. Default is stderr.

Details:
This is the Slug programming language. 

Examples:
  slug -log-level=debug         Start with debug logging enabled
  slug myfile.slug              Execute the provided Slug file
  slug myfile.slug arg1 arg2    Execute the file with command-line arguments

Version Information:
  Version:    ` + cli.Version + `
  Build Date: ` + cli.BuildDate + `
  Commit:     ` + cli.Commit + `
`
}
