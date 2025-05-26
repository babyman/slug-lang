package evaluator

import (
	"slug/internal/object"
)

func fnActorSpawn() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("spawn expects a function literal and optional arguments")
			}
			fn, ok := args[0].(*object.Function)
			if !ok {
				return ctx.NewError("first argument to spawn must be a function")
			}
			processArgs := args[1:] // Remaining args to pass to function
			pid := runtime.spawn(fn, processArgs...)
			return &object.Integer{Value: int64(pid)}
		},
	}
}

func fnActorSelf() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) > 0 {
				return ctx.NewError("self takes no arguments")
			}
			if ctx.PID() == 0 {
				return ctx.NewError("self called outside of process context")
			}
			return &object.Integer{Value: int64(ctx.PID())}
		},
	}
}

func fnActorSend() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("send expects a PID and a message")
			}
			pid, ok := args[0].(*object.Integer)
			if !ok {
				return ctx.NewError("first argument to send must be integer PID")
			}
			msg := &object.Message{
				From: ctx.PID(),
				Data: args[1],
			}
			runtime.Send(int(pid.Value), *msg)
			return pid
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
				if to, ok := args[0].(*object.Integer); ok {
					timeout = to.Value
				} else {
					return ctx.NewError("timeout argument must be an integer")
				}
			}
			msg, ok := runtime.Receive(ctx.PID(), timeout)
			if !ok {
				return ctx.Nil() // Indicate timeout or no messages
			}
			return msg.Data
		},
	}
}
