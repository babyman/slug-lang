package main

//
//import (
//	"flag"
//	"fmt"
//	"os"
//	"os/user"
//	"path/filepath"
//	"slug/internal/evaluator"
//	"slug/internal/log"
//	"slug/internal/object"
//	"slug/internal/svc/repl"
//	"strings"
//)
//
//var (
//	rootPath string
//	debugAST bool // New debug-ast flag
//	logLevel string
//	logFile  string
//	color    bool
//)
//
//func init() {
//	flag.StringVar(&rootPath, "root", ".", "Set the root context for the program (used for imports)")
//	flag.BoolVar(&debugAST, "debug-ast", false, "Render the AST as a JSON file")
//	flag.StringVar(&logLevel, "log-level", "NONE", "Log level: trace, debug, info, warn, error, none")
//	flag.StringVar(&logFile, "log-file", "", "Log file path (if not set, logs to stderr)")
//	flag.BoolVar(&color, "log-color", true, "Enable color output in terminal")
//}
//
//func mainX() {
//	// Define a help flag
//	help := flag.Bool("help", false, "Display help information and exit")
//
//	flag.Parse() // Parse the command-line flags
//
//	if *help {
//		printHelp()
//		os.Exit(0) // Exit after printing help
//	}
//
//	log.InitLogger(logLevel, logFile, color)
//	defer log.Close()
//
//	if len(flag.Args()) > 0 { // Remaining arguments after flags
//		// If an argument is passed, treat it as a filename to execute
//		filename := flag.Args()[0]
//		args := flag.Args()[1:]
//
//		if err := executeFile(filename, rootPath, args); err != nil {
//			fmt.Fprintf(os.Stderr, "Unhandled error: %v\n", err)
//			os.Exit(1)
//		}
//	} else {
//		// No arguments, launch the REPL
//		usr, err := user.Current()
//		if err != nil {
//			panic(err)
//		}
//
//		evaluator.rootPath = rootPath
//
//		fmt.Printf("Hello %s! This is the Slug programming language!\n", usr.Username)
//		fmt.Printf("Feel free to type in commands\n")
//		repl.Start(os.Stdin, os.Stdout)
//	}
//}
//
//func printHelp() {
//	fmt.Println(`Usage: slug [options] [filename [args...]]
//
//Options:
//  -root <path>       Set the root context for the program (used for imports). Default is '.'
//  -debug-ast         Render the AST as a JSON file.
//  -help              Display this help information and exit.
//  -log-level <level> Set the log level: trace, debug, info, warn, error, none. Default is 'none'.
//  -log-file <path>   Specify a log file to write logs. Default is stderr.
//  -log-color         Enable (default) or disable colored log output in the terminal.
//
//Details:
//This is the Slug programming language.
//You can provide a filename to execute a Slug program, or run without arguments to start the interactive REPL.
//
//Examples:
//  slug                          Start the interactive REPL
//  slug -root=/path/to/root      Start the REPL with a specific root path
//  slug -log-level=debug         Start with debug logging enabled
//  slug myfile.slug              Execute the provided Slug file
//  slug myfile.slug arg1 arg2    Execute the file with command-line arguments`)
//}
//
//func executeFile(filename, rootPath string, args []string) error {
//
//	// Ensure the file has the correct extension
//	if !strings.HasSuffix(filename, ".slug") {
//		filename += ".slug"
//	}
//
//	systemRootPath, modulePath, err2 := calculateModulePath(filename, rootPath)
//	if err2 != nil {
//		return err2
//	}
//
//	evaluator.debugAST = debugAST
//	evaluator.rootPath = systemRootPath
//	module, err := evaluator.LoadModule(modulePath)
//	if err != nil {
//		return fmt.Errorf("failed to load main module '%s':\n%v", filename, err)
//	}
//
//	// Start the environment
//	env := setupEnvironment(args)
//	env.Path = module.Path
//	env.ModuleFqn = module.Name
//	env.Src = module.Src
//	module.Env = env
//
//	// Parse and evaluate the content
//	err = evaluateModule(module, env)
//	if err != nil {
//		return fmt.Errorf("'%s': %v", filename, err)
//	}
//
//	return nil
//}
//
//func setupEnvironment(args []string) *object.Environment {
//	env := object.NewEnvironment()
//
//	// Prepare args list
//	objects := make([]object.Object, len(args))
//	for i, arg := range args {
//		objects[i] = &object.String{Value: arg}
//	}
//	env.Define("args", &object.List{Elements: objects}, false, false)
//
//	return env
//}
//
//func evaluateModule(module *object.Module, env *object.Environment) error {
//
//	// Parse src into a Program AST
//	program := module.Program
//
//	e := evaluator.Evaluator{
//		Actor: evaluator.CreateMainThreadMailbox(),
//	}
//	e.PushEnv(env)
//	defer e.PopEnv()
//
//	log.Info(" ---- begin ----")
//	defer log.Info(" ---- done ----")
//
//	// Evaluate the program within the provided environment
//	evaluated := e.Eval(program)
//	if evaluated != nil && evaluated.Type() != object.NIL_OBJ {
//		if evaluated.Type() == object.ERROR_OBJ {
//			return fmt.Errorf(evaluated.Inspect())
//		} else {
//			fmt.Println(evaluated.Inspect())
//		}
//	}
//
//	return nil
//}
//
//func isSourceFile(filename string) (bool, error) {
//
//	fileInfo, err := os.Stat(filename)
//
//	if err != nil {
//		if os.IsNotExist(err) {
//			return false, nil
//		}
//		return false, fmt.Errorf("error accessing file '%s': %v", filename, err)
//	}
//
//	if fileInfo.IsDir() {
//		return false, fmt.Errorf("'%s' is a directory, not a file", filename)
//	}
//
//	return true, nil
//}
//
//func calculateModulePath(filename string, rootPath string) (string, []string, error) {
//
//	// Check if file exists and is not a directory
//	isSource, err := isSourceFile(filename)
//	if err != nil {
//		return "", nil, err
//	}
//
//	if rootPath == "." && isSource {
//		rootPath = filepath.Dir(filename)
//	}
//
//	// Calculate the module path relative to root path
//	absFilePath, err := filepath.Abs(filename)
//	if err != nil {
//		return "", nil, fmt.Errorf("failed to get absolute path for '%s': %v", filename, err)
//	}
//
//	absRootPath, err := filepath.Abs(rootPath)
//	if err != nil {
//		return "", nil, fmt.Errorf("failed to get absolute path for root '%s': %v", rootPath, err)
//	}
//
//	if !isSource {
//		absFilePath = absRootPath
//	}
//
//	modulePath, err := filepath.Rel(absRootPath, absFilePath)
//	if err != nil {
//		return "", nil, fmt.Errorf("failed to calculate relative path: %v", err)
//	}
//	if !isSource {
//		modulePath = filename
//	}
//
//	// Remove .slug extension if present
//	modulePath = strings.TrimSuffix(modulePath, ".slug")
//
//	modulePathParts := strings.Split(modulePath, string(filepath.Separator))
//	return absRootPath, modulePathParts, nil
//}
