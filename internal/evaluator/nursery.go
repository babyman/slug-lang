package evaluator

import (
	"slug/internal/object"
	"sync"
)

type NurseryScope struct {
	// Concurrency tracking
	Children   []*Task       // Tasks owned by this scope
	Limit      chan struct{} // Semaphore for 'async limit N'
	NurseryErr object.Object // fail-fast state (first failure wins)
	mu         sync.RWMutex
}

// AddChild registers a task handle with this environment
func (n *NurseryScope) AddChild(th *Task) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Children = append(n.Children, th)
	th.OwnerNursery = n
}

func (n *NurseryScope) RemoveChild(th *Task) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i, child := range n.Children {
		if child == th {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			break
		}
	}
}

// CancelChildren cancels all children except `except` (if non-nil).
func (n *NurseryScope) CancelChildren(except *Task, cause *object.RuntimeError, reason string) {
	n.mu.Lock()
	children := make([]*Task, len(n.Children))
	copy(children, n.Children)
	n.mu.Unlock()

	for _, ch := range children {
		if except != nil && ch == except {
			continue
		}
		ch.Cancel(cause, reason)
	}
}

// NoteChildFailure records the first failure and cancels siblings (fail-fast).
func (n *NurseryScope) NoteChildFailure(failed *Task, err object.Object) {
	n.mu.Lock()
	alreadyFailed := n.NurseryErr != nil
	if !alreadyFailed {
		n.NurseryErr = err
	}
	n.mu.Unlock()

	// Only the first failure triggers sibling cancellation
	if !alreadyFailed {
		// If it's a RuntimeError, we can pass it as a cause.
		// If it's a plain Error, we just cancel without a specific RT cause.
		var rtCause *object.RuntimeError
		if rt, ok := err.(*object.RuntimeError); ok {
			rtCause = rt
		}
		n.CancelChildren(failed, rtCause, "sibling cancelled due to fail-fast")
	}
}

// WaitChildren blocks until all direct children of this scope have settled
func (n *NurseryScope) WaitChildren() {
	n.mu.RLock()
	children := make([]*Task, len(n.Children))
	copy(children, n.Children)
	n.mu.RUnlock()

	for _, child := range children {
		<-child.Done
	}
}
