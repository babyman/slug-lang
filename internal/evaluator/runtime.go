package evaluator

import (
	"slug/internal/object"
	"time"
)

type Runtime struct {
	pidCounter int
	processes  map[int]*Process
	scheduler  *Scheduler
}

func (rt *Runtime) NewPID() int {
	rt.pidCounter += 1
	return rt.pidCounter
}

func (rt *Runtime) RemoveProcess(pid int) {
	if process, exists := rt.processes[pid]; exists {
		delete(rt.processes, pid)
		close(process.Mailbox)
		println("process removed from runtime", pid)
	}
}

func (rt *Runtime) spawn(fn *object.Function, args ...object.Object) int {
	pid := rt.NewPID()
	evaluator := Evaluator{
		envStack: []*object.Environment{fn.Env},
		Process: &Process{
			PID:      pid,
			Mailbox:  make(chan object.Message, 10), // Buffered mailbox
			Function: fn,
			Args:     args,
		},
	}
	evaluator.Process.Evaluator = &evaluator
	process := evaluator.Process

	rt.processes[pid] = process
	rt.scheduler.Add(process)

	go func() {
		println("process spawned", process.PID)
		defer func() {
			rt.RemoveProcess(process.PID)
		}()
		out := process.Evaluator.applyFunction(process.Function, process.Args)
		// todo: could broadcast an exit message with the last response message in it?
		println("out: ", out.Inspect())
	}()
	return pid
}

func (rt *Runtime) Send(toPID int, msg object.Message) {
	if process, exists := rt.processes[toPID]; exists {
		process.Mailbox <- msg
	}
}

func (rt *Runtime) Receive(pid int, timeout int64) (object.Message, bool) {
	if process, exists := rt.processes[pid]; exists {
		if timeout <= 0 {
			println("waiting for message")
			msg := <-process.Mailbox
			println("message received", msg.Inspect())
			return msg, true
		}

		select {
		case msg := <-process.Mailbox:
			return msg, true
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			return object.Message{}, false
		}
	} else {
		println("invalid pid", pid)
	}
	return object.Message{}, false
}
