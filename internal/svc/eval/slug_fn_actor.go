package eval

import (
	"log/slog"
	"slug/internal/dec64"
	"slug/internal/foreign"
	"slug/internal/kernel"
	"slug/internal/object"
)

func fnActorSelf() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			return &object.Number{Value: dec64.FromInt64(int64(ctx.ActCtx().Self))}
		},
	}
}

func fnActorSpawn() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want>=1", len(args))
			}

			objects := args[1:]
			fun, ok := foreign.ToFunctionArgument(args[0], objects)
			if !ok {
				return ctx.NewError("first argument to `spawn` must be FUNCTION, got=%s", args[0].Type())
			}

			actor := NewSlugFunctionActor(ctx.GetConfiguration(), fun)
			pid, err := ctx.ActCtx().SpawnChild("slug-actor", Operations, actor.Run)
			if err != nil {
				return ctx.NewError("failed to spawn actor: %v", err)
			}

			ctx.ActCtx().SendAsync(pid, SlugStart{
				Args: objects,
			})

			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}

func fnActorSend() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			pidObj, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to `send` must be NUMBER, got=%s", args[0].Type())
			}

			pid := kernel.ActorID(pidObj.Value.ToInt64())
			msg := SlugActorMessage{
				Msg: args[1],
			}
			//println("sending: ", msg.Msg.Inspect(), " to: ", pid)
			err := ctx.ActCtx().SendAsync(pid, msg)
			//println("sent: ", msg.Msg.Inspect(), " to: ", pid)
			if err != nil {
				return ctx.NewError("failed to send message: %v", err)
			}

			return pidObj
		},
	}
}

func fnActorReceive() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) > 1 {
				return ctx.NewError("receive expects zero or one timeout argument")
			}
			var timeout int64 // No timeout by default
			if len(args) == 1 {
				if to, ok := args[0].(*object.Number); ok {
					timeout = to.Value.ToInt64()
				} else {
					return ctx.NewError("timeout argument must be an integer")
				}
			}

			slog.Warn("ACT: Waiting for message",
				slog.Any("actor-pid", ctx.ActCtx().Self))

			message, b := ctx.WaitForMessage(timeout)
			if !b {
				return ctx.Nil()
			}

			slog.Warn("ACT: message received",
				slog.Any("actor-pid", ctx.ActCtx().Self))

			switch m := message.(type) {
			case SlugActorMessage:
				return m.Msg
			}

			return ctx.Nil()
		},
	}
}

func fnActorTerminate() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			slog.Error("ACT: Terminating actor NOT IMPLEMENTED YET")

			return ctx.Nil()
		},
	}
}
