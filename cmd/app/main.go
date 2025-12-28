package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slug/internal/evaluator"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/util"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
	Commit    = "unknown"
	help      bool
	version   bool
	logLevel  string
	logFile   string
	logSource bool
	// config vars
	rootPath     string
	debugJsonAST bool
	debugTxtAST  bool
)

func init() {
	flag.BoolVar(&help, "help", false, "Display help information and exit")
	flag.BoolVar(&help, "h", false, "Display help information and exit")
	flag.BoolVar(&version, "version", false, "Display version information and exit")
	flag.BoolVar(&version, "v", false, "Display version information and exit")
	// evaluator config
	flag.StringVar(&rootPath, "root", "", "Set the root context for the program (used for imports)")
	// parser config
	flag.BoolVar(&debugJsonAST, "debug-json-ast", false, "Render the AST as a JSON file")
	flag.BoolVar(&debugTxtAST, "debug-txt-ast", false, "Render the AST as a TXT file")
	// log config
	flag.StringVar(&logLevel, "log-level", "NONE", "Log level: trace, debug, info, warn, error, none")
	flag.StringVar(&logFile, "log-file", "", "Log file path (if not set, logs to stderr)")
	flag.BoolVar(&logSource, "log-source", false, "Include the source file name in log messages")
}

func main() {
	flag.Parse()

	if version {
		fmt.Printf("slug version 'v%s' %s %s\n", Version, BuildDate, Commit)
		return
	}

	if help || flag.NArg() == 0 {
		printHelp()
		return
	}

	setupLogging()

	// 1. Resolve Script Path
	targetName := flag.Arg(0)
	scriptPath, source, err := resolveScript(targetName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 2. Prepare Configuration
	// RootPath is the directory of the resolved script
	resolvedRootPath := filepath.Dir(scriptPath)
	if rootPath != "" {
		resolvedRootPath, _ = filepath.Abs(rootPath)
	}
	config := util.Configuration{
		Version:      Version,
		RootPath:     resolvedRootPath,
		SlugHome:     os.Getenv("SLUG_HOME"),
		DebugJsonAST: debugJsonAST,
		DebugTxtAST:  debugTxtAST,
	}

	// 3. Tokenize & Parse
	l := lexer.New(string(source))
	p := parser.New(l, scriptPath, string(source))
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Fprintf(os.Stderr, "Parse errors:\n")
		for _, msg := range p.Errors() {
			fmt.Fprintf(os.Stderr, "\t%s\n", msg)
		}
		os.Exit(1)
	}

	// 5. Initialize Evaluator & Environment
	env := object.NewEnvironment()

	// Inject command line arguments into the environment as args[]
	programArgs := []object.Object{}
	for _, arg := range flag.Args()[1:] {
		programArgs = append(programArgs, &object.String{Value: arg})
	}
	env.Define("args", &object.List{Elements: programArgs}, false, false)

	eval := &evaluator.Evaluator{
		Config: config,
	}
	eval.PushEnv(env)

	// 6. Execute
	result := eval.Eval(program)

	// 7. Handle Result/Errors
	if result != nil {
		if result.Type() == object.ERROR_OBJ {
			fmt.Fprintf(os.Stderr, "Runtime Error: %s\n", result.Inspect())
			os.Exit(1)
		}
		// In non-REPL mode, we usually don't print the final expression result
		// unless it's an error, but you can if you want to.
	}
}

func resolveScript(target string) (string, []byte, error) {
	slugHome := os.Getenv("SLUG_HOME")

	// Search order:
	// 1. Exact local path
	// 2. Local path + .slug
	// 3. $SLUG_HOME/lib + .slug

	searchPaths := []string{
		target,
		target + ".slug",
	}

	if slugHome != "" {
		searchPaths = append(searchPaths, filepath.Join(slugHome, "lib", target+".slug"))
	}

	for _, path := range searchPaths {
		source, err := os.ReadFile(path)
		if err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath, source, nil
		}
	}

	return "", nil, fmt.Errorf("could not find script '%s' locally or in $SLUG_HOME/lib", target)
}

func setupLogging() {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}

	// Creates a new Logger that uses a JSONHandler to write to standard output
	loggerOptions := &slog.HandlerOptions{
		AddSource: logSource,
		Level:     level,
	}
	logWriter := configureLogWriter()
	defaultLogger := slog.New(slog.NewTextHandler(logWriter, loggerOptions))
	slog.SetDefault(defaultLogger)
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

func printHelp() {
	fmt.Printf(`Usage: slug [options] <filename> <args>

Options:
  -root <path>       Set the root context
  -version, -v       Show version
  -help, -h          Show this help
  -log-source        Include the source file name in log messages.
  -log-level <level> Set log level (debug, info, warn, error)
  -log-file <path>   Specify a log file to write logs. Default is stderr.
  -debug-json-ast    Render the AST as a JSON file.
  -debug-txt-ast     Render the AST as a TXT file.
`)
}
