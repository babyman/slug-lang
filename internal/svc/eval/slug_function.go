package eval

import (
	"log/slog"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/util"
	"time"
)

type SlugStart struct {
	Args []object.Object
}

type SlugFunctionDone struct{}

type SlugFunctionActor struct {
	Config   util.Configuration
	Function *object.Function
	Mailbox  chan svc.SlugActorMessage

	// internal state
	started bool
}

func NewSlugFunctionActor(c util.Configuration, function *object.Function) *SlugFunctionActor {
	return &SlugFunctionActor{
		Function: function,
		Config:   c,
		Mailbox:  make(chan svc.SlugActorMessage),
	}
}

func (s *SlugFunctionActor) Run(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case SlugStart:
		// avoid starting the same function multiple times
		if s.started {
			return kernel.Continue{}
		}
		s.started = true

		// Run the function in its own goroutine so this actor can keep
		// handling SlugActorMessage inputs used by WaitForMessage.
		go func(args []object.Object) {
			e := Evaluator{
				envStack:     []*object.Environment{s.Function.Env},
				SlugReceiver: s,
				Config:       s.Config,
				Ctx:          ctx,
			}
			out := e.ApplyFunction(0, "<anon>", s.Function, args)

			slog.Debug("function result",
				slog.Any("result", out.Inspect()))

			// Tell this actor to terminate.
			ctx.SendAsync(ctx.Self, SlugFunctionDone{})
		}(payload.Args)

		return kernel.Continue{}

	case svc.SlugActorMessage:
		s.Mailbox <- payload
		return kernel.Continue{}

	case SlugFunctionDone:
		slog.Info("actor terminated",
			slog.Any("actor-id", ctx.Self))
		return kernel.Terminate{Reason: "function-complete"}

	default:
		return kernel.Continue{}
	}
}

func (s *SlugFunctionActor) WaitForMessage(timeout int64) (any, bool) {
	// Semantics:
	//  - timeout < 0 => block forever
	//  - timeout == 0 => poll (non-blocking)
	//  - timeout > 0 => timeout milliseconds
	if timeout < 0 {
		msg := <-s.Mailbox
		return msg, true
	}

	if timeout == 0 {
		select {
		case msg := <-s.Mailbox:
			return msg, true
		default:
			return svc.SlugActorMessage{}, false
		}
	}

	select {
	case msg := <-s.Mailbox:
		return msg, true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return svc.SlugActorMessage{}, false
	}
}
