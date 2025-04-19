package main

import (
	"fmt"
	"os"
	"os/user"
	"slug/internal/evaluator"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/repl"
)

func main() {
	if len(os.Args) > 1 {
		// If an argument is passed, treat it as a filename to execute
		filename := os.Args[1]
		if err := executeFile(filename); err != nil {
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
		repl.Start(os.Stdin, os.Stdout)
	}
}

func executeFile(filename string) error {
	// Read the file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read file %s: %v", filename, err)
	}

	// Set up lexer, parser, and environment
	source := string(content)
	l := lexer.New(source)
	p := parser.New(l)

	program := p.ParseProgram()
	if len(p.Errors()) != 0 {
		fmt.Println("Woops! Looks like we slid into some slimy slug trouble here!")
		fmt.Println("Parser errors:")
		for _, msg := range p.Errors() {
			fmt.Printf("\t%s\n", msg)
		}
		return fmt.Errorf("parsing errors encountered")
	}

	env := object.NewEnvironment()
	evaluated := evaluator.Eval(program, env)
	if evaluated != nil && evaluated.Type() != object.NULL_OBJ {
		if evaluated.Type() == object.ERROR_OBJ {
			return fmt.Errorf(evaluated.Inspect())
		} else {
			fmt.Println(evaluated.Inspect())
		}
	}

	return nil
}
