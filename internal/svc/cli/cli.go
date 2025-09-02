package cli

import (
	"flag"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc"
	"slug/internal/svc/modules"
)

var (
	rootPath string
	debugAST bool // New debug-ast flag
	logLevel string
	logFile  string
	color    bool
	help     bool
)

func init() {
	flag.BoolVar(&help, "help", false, "Display help information and exit")
	// evaluator config
	flag.StringVar(&rootPath, "root", ".", "Set the root context for the program (used for imports)")
	// parser config
	flag.BoolVar(&debugAST, "debug-ast", false, "Render the AST as a JSON file")
	// log config
	flag.StringVar(&logLevel, "log-level", "NONE", "Log level: trace, debug, info, warn, error, none")
	flag.StringVar(&logFile, "log-file", "", "Log file path (if not set, logs to stderr)")
}

func (cli *Cli) onBoot(ctx *kernel.ActCtx) any {

	flag.Parse()

	kernelID, _ := ctx.K.ActorByName(kernel.KernelService)

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

	output, err := ctx.SendSync(modsID, modules.ModuleEvaluateFile{
		Path: filename,
		Args: args,
	})
	if err != nil {
		svc.SendErrorf(ctx, "err: %v", err)
	} else {
		r, ok := output.Payload.(string)
		if ok {
			svc.SendStdOut(ctx, r)
		}
	}

	r, _ := ctx.SendSync(kernelID, kernel.RequestShutdown{ExitCode: 0})
	return r.Payload
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
  -log-level <level> Set the log level: trace, debug, info, warn, error, none. Default is 'none'.
  -log-file <path>   Specify a log file to write logs. Default is stderr.
  -log-color         Enable (default) or disable colored log output in the terminal.

Details:
This is the Slug programming language. 
You can provide a filename to execute a Slug program, or run without arguments to start the interactive REPL.

Examples:
  slug                          Start the interactive REPL
  slug -root=/path/to/root      Start the REPL with a specific root path
  slug -log-level=debug         Start with debug logging enabled
  slug myfile.slug              Execute the provided Slug file
  slug myfile.slug arg1 arg2    Execute the file with command-line arguments`
}
