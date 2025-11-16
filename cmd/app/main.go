package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slug/internal/kernel"
	"slug/internal/privileged"
	"slug/internal/svc"
	"slug/internal/svc/cli"
	"slug/internal/svc/eval"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/modules"
	"slug/internal/svc/parser"
	"slug/internal/svc/repl"
	"slug/internal/svc/resolver"
	"slug/internal/svc/sout"
	"slug/internal/util"
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

const (
	DefaultRootPath = "."
)

var (
	// Version is the current version of the slug binary loaded from the VERSION file in the root of the project.
	Version   = "dev"
	BuildDate = "unknown"
	Commit    = "unknown"
	help      bool
	version   bool
	// logging
	logLevel string
	logFile  string
	// config vars
	rootPath string
	debugAST bool
)

func init() {
	flag.BoolVar(&help, "help", false, "Display help information and exit")
	flag.BoolVar(&help, "h", false, "Display help information and exit")
	flag.BoolVar(&version, "version", false, "Display version information and exit")
	flag.BoolVar(&version, "v", false, "Display version information and exit")
	// evaluator config
	flag.StringVar(&rootPath, "root", DefaultRootPath, "Set the root context for the program (used for imports)")
	// parser config
	flag.BoolVar(&debugAST, "debug-ast", false, "Render the AST as a JSON file")
	// log config
	flag.StringVar(&logLevel, "log-level", "NONE", "Log level: trace, debug, info, warn, error, none")
	flag.StringVar(&logFile, "log-file", "", "Log file path (if not set, logs to stderr)")
}

func main() {

	flag.Parse()

	// Creates a new Logger that uses a JSONHandler to write to standard output
	loggerOptions := &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevelFromString(logLevel),
	}
	logWriter := configureLogWriter()
	defaultLogger := slog.New(slog.NewJSONHandler(logWriter, loggerOptions))
	slog.SetDefault(defaultLogger)

	if version {
		printVersion()
		return
	}

	if help {
		printHelp()
		return
	}

	config := util.Configuration{
		Version:   Version,
		BuildDate: BuildDate,
		Commit:    Commit,
		RootPath:  rootPath,
		DebugAST:  debugAST,
		SlugHome:  os.Getenv("SLUG_HOME"),
	}

	k := kernel.NewKernel()

	kernelID, _ := k.ActorByName(kernel.KernelService)

	controlPlane := &privileged.ControlPlane{}
	k.RegisterPrivilegedService(privileged.ControlPlaneService, controlPlane)

	// system out Service
	out := &sout.SOut{}
	soutID := k.RegisterService(svc.SOutService, sout.Operations, out.Handler)

	// system out Service
	mods := modules.NewModules(config)
	modsID := k.RegisterService(svc.ModuleService, modules.Operations, mods.Handler)

	// system out Service
	res := &resolver.Resolver{
		Config: config,
	}
	resID := k.RegisterService(svc.ResolverService, resolver.Operations, res.Handler)

	// CLI Service
	cliSvc := &cli.Cli{
		Config:   config,
		FileName: flag.Arg(0),
		Args:     flag.Args()[1:],
	}
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
	evalSvc := &eval.EvaluatorService{
		Config: config,
	}
	evalID := k.RegisterService(svc.EvalService, eval.Operations, evalSvc.Handler)

	// REPL service
	replSvc := repl.NewReplService()
	replID := k.RegisterService(svc.ReplService, repl.Operations, replSvc.Handler)

	// Cap grants
	_ = k.GrantCap(kernelID, cliID, x, nil)

	_ = k.GrantCap(cliID, evalID, x, nil)
	_ = k.GrantCap(cliID, kernelID, x, nil)
	_ = k.GrantCap(cliID, modsID, rx, nil)
	_ = k.GrantCap(cliID, soutID, w, nil)

	_ = k.GrantCap(modsID, fsID, w, nil)
	_ = k.GrantCap(modsID, lexerID, x, nil)
	_ = k.GrantCap(modsID, parserID, x, nil)
	_ = k.GrantCap(modsID, resID, r, nil)

	_ = k.GrantCap(resID, fsID, r, nil)

	_ = k.GrantCap(evalID, evalID, rwx, nil)
	_ = k.GrantCap(evalID, modsID, r, nil)
	_ = k.GrantCap(evalID, soutID, w, nil)

	_ = k.GrantCap(replID, evalID, x, nil)
	_ = k.GrantCap(replID, lexerID, x, nil)
	_ = k.GrantCap(replID, parserID, x, nil)

	k.Start()
}

func configureLogWriter() *os.File {
	var logWriter *os.File
	var err error
	if logFile != "" {
		// Create parent directories if they don't exist
		if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create log directory for '%s': %v; falling back to stderr\n", logFile, err)
			return os.Stderr
		}
		logWriter, err = os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open log file '%s': %v; falling back to stderr\n", logFile, err)
			logWriter = os.Stderr
		}
	} else {
		logWriter = os.Stderr
	}
	return logWriter
}

func printVersion() {

	fmt.Printf("slug version 'v%s' %s %s\n", Version, BuildDate, Commit)
}

func printHelp() {
	fmt.Printf(`Usage: slug [options] [filename [args...]]

Options:
  -root <path>       Set the root context for the program (used for imports). Default is '.'
  -debug-ast         Render the AST as a JSON file.
  -help              Display this help information and exit.
  -version           Display version information and exit.
  -log-level <level> Set the log level: debug, info, warn, error. Default is 'error'.
  -log-file <path>   Specify a log file to write logs. Default is stderr.

Details:
This is the Slug programming language. 

Examples:
  slug -log-level=debug         Start with debug logging enabled
  slug myfile.slug              Execute the provided Slug file
  slug myfile.slug arg1 arg2    Execute the file with command-line arguments

Version Information:
  Version:    %s
  Build Date: %s
  Commit:     %s
`, Version, BuildDate, Commit)
}

func logLevelFromString(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
