package evaluator

import (
	"slug/internal/object"
)

type Process struct {
	PID       int64
	Mailbox   chan object.Message // FIFO queue for messages
	Evaluator *Evaluator
	Function  *object.Function //func(args ...object.Object) // Function to be executed
	Args      []object.Object  // Arguments passed to the process
}

func (p *Process) self() int64 {
	return p.PID
}

func (p *Process) run() int64 {
	if runtime.processes[p.PID] != nil {
		p.Evaluator.applyFunction(p.Function, p.Args)
	}
	return p.PID
}
