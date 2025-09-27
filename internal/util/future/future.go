package future

import (
	"sync"
	"time"
)

type result[T any] struct {
	v   T
	err error
}

// Future is a single-shot result that completes exactly once.
type Future[T any] struct {
	doneChannel chan struct{}
	res         result[T]
	once        sync.Once
}

// New runs fn in a goroutine and completes the Future when fn returns.
func New[T any](fn func() (T, error)) *Future[T] {
	f := &Future[T]{doneChannel: make(chan struct{})}
	go func() {
		v, err := fn()
		f.complete(v, err)
	}()
	return f
}

// FromValue creates an already-completed Future with a value.
func FromValue[T any](v T) *Future[T] {
	f := &Future[T]{doneChannel: make(chan struct{})}
	f.complete(v, nil)
	return f
}

// FromError creates an already-completed Future with an error.
func FromError[T any](err error) *Future[T] {
	f := &Future[T]{doneChannel: make(chan struct{})}
	var zero T
	f.complete(zero, err)
	return f
}

// Await blocks until completion and returns the result.
func (f *Future[T]) Await() (T, error) {
	<-f.doneChannel
	return f.res.v, f.res.err
}

// AwaitTimeout waits up to d for completion.
// Returns (value, err, ok). ok=false if timed out.
func (f *Future[T]) AwaitTimeout(d time.Duration) (T, error, bool) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-f.doneChannel:
		return f.res.v, f.res.err, true
	case <-timer.C:
		var zero T
		return zero, nil, false
	}
}

// Done returns a channel closed when the Future completes.
func (f *Future[T]) Done() <-chan struct{} { return f.doneChannel }

// Map applies f to the successful value, producing a new Future.
// If the original future failed, the error is propagated.
func Map[T, U any](in *Future[T], f func(T) (U, error)) *Future[U] {
	return New(func() (U, error) {
		v, err := in.Await()
		if err != nil {
			var zero U
			return zero, err
		}
		return f(v)
	})
}

// Then is an alias for Map; often used for sequencing.
func Then[T, U any](in *Future[T], f func(T) (U, error)) *Future[U] {
	return Map(in, f)
}

// All waits for all futures and returns their values in order.
// If any future fails, it returns the first error encountered.
func All[T any](futures ...*Future[T]) *Future[[]T] {
	return New(func() ([]T, error) {
		out := make([]T, len(futures))
		for i, fut := range futures {
			v, err := fut.Await()
			if err != nil {
				return nil, err
			}
			out[i] = v
		}
		return out, nil
	})
}

// First completes with the first future to finish (value or error).
func First[T any](futures ...*Future[T]) *Future[T] {
	type r struct {
		v   T
		err error
	}
	return New(func() (T, error) {
		ch := make(chan r, len(futures))
		for _, f := range futures {
			f := f
			go func() {
				v, err := f.Await()
				ch <- r{v: v, err: err}
			}()
		}
		ir := <-ch
		return ir.v, ir.err
	})
}

// complete sets the result exactly once and closes doneChannel.
func (f *Future[T]) complete(v T, err error) {
	f.once.Do(func() {
		f.res = result[T]{v: v, err: err}
		close(f.doneChannel)
	})
}
