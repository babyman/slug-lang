package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"slug/internal/evaluator"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/repl"
	"strings"
)

var (
	rootPath string
	debugAST bool // New debug-ast flag
)

func init() {
	flag.StringVar(&rootPath, "root", ".", "Set the root context for the program (used for imports)")
	flag.BoolVar(&debugAST, "debug-ast", false, "Render the AST as a JSON file")
}

func main() {
	// Define a help flag
	help := flag.Bool("help", false, "Display help information and exit")

	flag.Parse() // Parse the command-line flags

	if *help {
		printHelp()
		os.Exit(0) // Exit after printing help
	}

	if len(flag.Args()) > 0 { // Remaining arguments after flags
		// If an argument is passed, treat it as a filename to execute
		filename := flag.Args()[0]
		args := flag.Args()[1:]

		if err := executeFile(filename, rootPath, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// No arguments, launch the REPL
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}

		fmt.Printf("Hello %s! This is the Slug programming language!\n", usr.Username)
		fmt.Printf("Feel free to type in commands\n")
		repl.SetRootPath(rootPath) // Pass rootPath to REPL
		repl.Start(os.Stdin, os.Stdout)
	}
}

func printHelp() {
	fmt.Println(`Usage: slug [options] [filename [args...]]

Options:
  -root <path>       Set the root context for the program (used for imports). Default is '.'
  -debug-ast         Render the AST as a JSON file.
  -help              Display this help information and exit.

Details:
This is the Slug programming language. 
You can provide a filename to execute a Slug program, or run without arguments to start the interactive REPL.

Examples:
  slug                          Start the interactive REPL
  slug -root=/path/to/root      Start the REPL with a specific root path
  slug myfile.slug              Execute the provided Slug file
  slug myfile.slug arg1 arg2    Execute the file with command-line arguments`)
}

func executeFile(filename, rootPath string, args []string) error {

	// Step 1: Ensure the file has the correct extension
	if !strings.HasSuffix(filename, ".slug") {
		filename += ".slug"
	}

	// Step 2: Attempt to load the file content
	content, err := readFileWithFallback(filename)
	if err != nil {
		return fmt.Errorf("failed to load file '%s': %v", filename, err)
	}

	// Step 3: Initialize the environment
	env := setupEnvironment(rootPath, args)

	// Step 4: Parse and evaluate the content
	err = parseAndEvaluate(filename, content, env)
	if err != nil {
		return fmt.Errorf("execution error for file '%s': %v", filename, err)
	}

	return nil
}

func readFileWithFallback(filename string) (string, error) {
	content, err := os.ReadFile(filename) // Primary file path
	if err == nil {
		return string(content), nil
	}

	// Fallback logic to check SLUG_HOME/lib
	slugHome := os.Getenv("SLUG_HOME")
	if slugHome == "" {
		return "", fmt.Errorf("file not found at '%s' and SLUG_HOME is not set", filename)
	}

	fallbackPath := fmt.Sprintf("%s/lib/%s", slugHome, filename)
	content, err = os.ReadFile(fallbackPath)
	if err != nil {
		return "", fmt.Errorf("file not found at '%s' and fallback to '%s' also failed", filename, fallbackPath)
	}

	return string(content), nil
}

func setupEnvironment(rootPath string, args []string) *object.Environment {
	env := object.NewEnvironment()
	env.SetRootPath(rootPath)

	// Prepare args array
	objects := make([]object.Object, len(args))
	for i, arg := range args {
		objects[i] = &object.String{Value: arg}
	}
	env.Set("args", &object.Array{Elements: objects})

	return env
}

func parseAndEvaluate(filename string, src string, env *object.Environment) error {
	// Initialize Lexer and Parser
	l := lexer.New(src)
	p := parser.New(l, src)

	// Parse src into a Program AST
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Println("Woops! Looks like we slid into some slimy slug trouble here!")
		fmt.Println("Parser errors:")
		for _, msg := range p.Errors() {
			fmt.Printf("\t%s\n", msg)
		}
		return fmt.Errorf("parsing errors encountered")
	}

	if debugAST {
		if err := parser.WriteASTToJSON(program, filename+".ast.json"); err != nil {
			return fmt.Errorf("failed to write AST to JSON: %v", err)
		}
	}

	// Evaluate the program within the provided environment
	evaluated := evaluator.Eval(program, env)
	if evaluated != nil && evaluated.Type() != object.NIL_OBJ {
		if evaluated.Type() == object.ERROR_OBJ {
			return fmt.Errorf(evaluated.Inspect())
		} else {
			fmt.Println(evaluated.Inspect())
		}
	}

	return nil
}
