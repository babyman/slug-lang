package evaluator

import (
	"math/rand"
	"slug/internal/object"
	"sync"
	"time"
)

type Runtime struct {
	processes map[int64]*Process
	scheduler *Scheduler
}

var (
	actorNextID int64 = 1
	actorMutex  sync.Mutex
	runtime     = &Runtime{
		scheduler: &Scheduler{
			runQueue: make(chan *Process, 100),
		},
		processes: make(map[int64]*Process),
	}
)

func NewPID() int64 {
	actorMutex.Lock()
	defer actorMutex.Unlock()
	id := actorNextID<<16 | int64(rand.Intn(0xFFFF))
	actorNextID++
	return id
}

func AddProcess(process *Process) {
	runtime.processes[process.PID] = process
	runtime.scheduler.Add(process)
}

func (rt *Runtime) RemoveProcess(pid int64) {
	if process, exists := rt.processes[pid]; exists {
		delete(rt.processes, pid)
		close(process.Mailbox)
	}
}

func (rt *Runtime) spawn(fn *object.Function, args ...object.Object) int64 {
	evaluator := Evaluator{
		envStack: []*object.Environment{fn.Env},
		Process: &Process{
			PID:      NewPID(),
			Mailbox:  make(chan object.Message, 10), // Buffered mailbox
			Function: fn,
			Args:     args,
		},
	}
	evaluator.Process.Evaluator = &evaluator
	process := evaluator.Process

	runtime.processes[process.PID] = process
	runtime.scheduler.Add(process)

	go func() {
		defer func() {
			rt.RemoveProcess(process.PID)
		}()
		/*out := */ process.Evaluator.applyFunction(process.Function, process.Args)
		// todo: could broadcast an exit message with the last response message in it?
	}()
	return process.PID
}

func (rt *Runtime) Send(toPID int64, msg object.Message) {
	if process, exists := rt.processes[toPID]; exists {
		process.Mailbox <- msg
	}
}

func (rt *Runtime) Receive(pid int64, timeout int64) (object.Message, bool) {
	if process, exists := rt.processes[pid]; exists {
		if timeout <= 0 {
			msg := <-process.Mailbox
			return msg, true
		}

		select {
		case msg := <-process.Mailbox:
			return msg, true
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			return object.Message{}, false
		}
	} else {
		newError("Invalid PID not found %d", pid)
	}
	return object.Message{}, false
}
