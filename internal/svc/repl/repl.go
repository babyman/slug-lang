package repl

import (
	"bufio"
	"fmt"
	"io"
	"slug/internal/object"
	"slug/internal/svc/eval"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprintf(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l, "repl", line)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		e := eval.Evaluator{}
		e.PushEnv(object.NewEnvironment())
		evaluated := e.Eval(program)
		e.PopEnv(evaluated)

		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Woops! Looks like we slid into some slimy slug trouble here!\n")
	io.WriteString(out, " parser errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
