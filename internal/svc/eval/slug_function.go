package eval

import (
	"log/slog"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/util"
	"time"
)

type SlugActorMessage struct {
	Msg object.Object
}

type SlugStart struct {
	Args []object.Object
}

type SlugFunctionDone struct{}

type SlugFunctionActor struct {
	Config   util.Configuration
	Function *object.Function
	Mailbox  chan SlugActorMessage

	// internal state
	started bool
}

func NewSlugFunctionActor(c util.Configuration, function *object.Function) *SlugFunctionActor {
	return &SlugFunctionActor{
		Function: function,
		Config:   c,
		Mailbox:  make(chan SlugActorMessage),
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
			out := e.ApplyFunction("<anon>", s.Function, args)

			slog.Debug("function result",
				slog.Any("result", out.Inspect()))

			// Tell this actor to terminate.
			ctx.SendAsync(ctx.Self, SlugFunctionDone{})
		}(payload.Args)

		return kernel.Continue{}

	case SlugActorMessage:
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
	if timeout <= 0 {
		msg := <-s.Mailbox
		return msg, true
	}

	select {
	case msg := <-s.Mailbox:
		return msg, true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return SlugActorMessage{}, false
	}
}
