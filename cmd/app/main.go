package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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
	flag.Parse() // Parse the command-line flags

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

func executeFile(filename, rootPath string, args []string) error {
	// Read the file
	if !strings.HasSuffix(filename, ".slug") {
		filename = filename + ".slug"
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		// If not found, attempt fallback to ${SLUG_HOME}/lib
		slugHome := os.Getenv("SLUG_HOME")
		if slugHome == "" {
			return fmt.Errorf("environment variable SLUG_HOME is not set")
		}

		fallbackPath := fmt.Sprintf("%s/lib/%s", slugHome, filename)
		content, err = ioutil.ReadFile(fallbackPath)
		if err != nil {
			return fmt.Errorf("error reading module '%s': %s", fallbackPath, err)
		}
	}

	// Set up lexer, parser, and environment
	source := string(content)
	l := lexer.New(source)
	p := parser.New(l)

	program := p.ParseProgram()

	// Debug-ast flag: write AST to JSON file
	if debugAST {
		if err := parser.WriteASTToJSON(program, filename+".ast.json"); err != nil {
			return fmt.Errorf("failed to write AST to JSON: %v", err)
		}
	}

	if len(p.Errors()) != 0 {
		fmt.Println("Woops! Looks like we slid into some slimy slug trouble here!")
		fmt.Println("Parser errors:")
		for _, msg := range p.Errors() {
			fmt.Printf("\t%s\n", msg)
		}
		return fmt.Errorf("parsing errors encountered")
	}

	env := object.NewEnvironment()
	env.SetRootPath(rootPath) // Set the root path in the environment

	// Set up the args list
	objects := make([]object.Object, len(args))
	for i, arg := range args {
		objects[i] = &object.String{Value: arg}
	}
	env.Set("args", &object.Array{Elements: objects})

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
