package evaluator

type Scheduler struct {
	runQueue chan *Process
}

func (s *Scheduler) Add(process *Process) {
	s.runQueue <- process
}

func (s *Scheduler) Run() {
	for process := range s.runQueue {
		go process.run()
	}
}
