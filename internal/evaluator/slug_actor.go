package evaluator

import (
	"slug/internal/dec64"
	"slug/internal/object"
)

func fnActorMailbox() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			mode := RoundRobin

			if len(args) > 0 {
				if modeArg, ok := args[0].(*object.String); ok {
					if modeArg.Value != "broadcast" {
						mode = Broadcast
					}
				} else {
					return ctx.NewError("mode argument must be a string 'broadcast' or 'round_robin'")
				}
			}

			pid := System.NewMailbox(mode)

			return &object.Number{Value: dec64.FromInt64(pid)}
		},
	}
}

func fnActorBindActor() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 2 {
				return ctx.NewError("bindActor expects a mailbox PID and function literal")
			}

			pid, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to bindActor must be integer PID")
			}

			fn, ok := args[1].(*object.FunctionGroup)
			if !ok {
				return ctx.NewError("second argument to bindActor must be a function, got %T", args[1])
			}

			processArgs := args[2:]
			function, ok := fn.DispatchToFunction("", processArgs)
			f, ok := function.(*object.Function)

			_, ok = System.BindNewActor(pid.Value.ToInt64(), ctx.ActCtx(), f, processArgs...)

			if ok {
				return pid
			}
			return ctx.Nil()
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
			return &object.Number{Value: dec64.FromInt64(ctx.PID())}
		},
	}
}

func fnActorSend() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 2 {
				return ctx.NewError("send expects a PID and a message")
			}
			pid, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to send must be integer PID")
			}

			for _, arg := range args[1:] {
				//println("sending: ", arg.Inspect(), " to: ", pid.Value, "<<<")
				System.SendData(pid.Value.ToInt64(), arg)
			}
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
				if to, ok := args[0].(*object.Number); ok {
					timeout = to.Value.ToInt64()
				} else {
					return ctx.NewError("timeout argument must be an integer")
				}
			}

			msg, ok := ctx.Receive(timeout)
			if !ok {
				return ctx.Nil()
			}
			return msg
		},
	}
}

func fnActorRegister() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 2 {
				return ctx.NewError("register expects PID and name")
			}
			pid, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to register must be a PID")
			}
			name, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to register must be a string name")
			}

			log.Debug("BOX: %d registering as '%s'", pid.Value.ToInt64(), name.Value)
			System.Register(pid.Value.ToInt64(), name.Value)
			return pid
		},
	}
}

func fnActorUnregister() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("unregister expects PID and name arguments")
			}

			name, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to unregister must be a string name")
			}

			System.Unregister(name.Value)

			return ctx.Nil()
		},
	}
}

func fnActorWhereIs() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("whereIs expects a name argument")
			}
			name, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to whereIs must be a string name")
			}

			mailbox, exists := System.WhereIs(name.Value)

			if !exists {
				return ctx.Nil()
			}
			return &object.Number{Value: dec64.FromInt64(mailbox)}
		},
	}
}

func fnActorSupervisor() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) > 0 {
				return ctx.NewError("supervisor takes no arguments")
			}

			supervisor, exists := System.Supervisor(ctx.PID())
			if exists {
				return &object.Number{Value: dec64.FromInt64(supervisor)}
			}
			return ctx.Nil()
		},
	}
}

func fnActorSupervise() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("supervise expects supervisor_pid and actor_pid arguments")
			}

			supPID, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument must be supervisor PID")
			}

			for i, arg := range args[1:] {
				actorPID, ok := arg.(*object.Number)
				if !ok {
					return ctx.NewError("%d argument must be actor PID", i)
				}

				System.Supervise(supPID.Value.ToInt64(), actorPID.Value.ToInt64())
			}

			return supPID
		},
	}
}

func fnActorChildren() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			//if len(args) != 1 {
			//	return ctx.NewError("children expects a supervisor PID argument")
			//}
			//
			//supPID, ok := args[0].(*object.Number)
			//if !ok {
			//	return ctx.NewError("argument must be supervisor PID")
			//}
			//
			//children := runtime.LookupChildren(supPID.Value)
			//
			//elements := make([]object.Object, len(children))
			//for i, pid := range children {
			//	elements[i] = &object.Number{Value: pid}
			//}
			//return &object.List{Elements: elements}
			return ctx.Nil()
		},
	}
}

func fnActorTerminate() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			//if len(args) != 1 {
			//	return ctx.NewError("terminate expects a PID argument")
			//}
			//
			//pid, ok := args[0].(*object.Number)
			//if !ok {
			//	return ctx.NewError("argument must be PID")
			//}
			//
			//runtime.RemoveProcess(pid.Value)
			//return pid
			return ctx.Nil()
		},
	}
}
