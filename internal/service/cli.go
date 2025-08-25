package service

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"slug/internal/kernel"
)

var CliOperations = kernel.OpRights{
	reflect.TypeOf(kernel.Boot{}): kernel.RightExec,
}

type Cli struct {
}

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
	flag.BoolVar(&color, "log-color", true, "Enable color output in terminal")
}

func (cli *Cli) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch msg.Payload.(type) {
	case kernel.Boot:

		flag.Parse()

		if help {
			printHelp()
			os.Exit(0)
		}

		if len(flag.Args()) > 0 {
			filename := flag.Args()[0]
			args := flag.Args()[1:]

			// todo - execute file
			sendStdOut(ctx, filename, args)
		}
	}
}

func printHelp() {
	fmt.Println(`Usage: slug [options] [filename [args...]]

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
  slug myfile.slug arg1 arg2    Execute the file with command-line arguments`)
}
