package evaluator

import (
	"slug/internal/object"
)

type Process struct {
	PID       int
	Mailbox   chan object.Message // FIFO queue for messages
	Evaluator *Evaluator
	Function  *object.Function //func(args ...object.Object) // Function to be executed
	Args      []object.Object  // Arguments passed to the process
}

func (p *Process) self() int {
	return p.PID
}

func (p *Process) run() int {
	if runtime.processes[p.PID] != nil {
		println("process running", p.PID)
		p.Evaluator.applyFunction(p.Function, p.Args)
	} else {
		println("skipping removed or invalid process", p.PID)
	}
	return p.PID
}
