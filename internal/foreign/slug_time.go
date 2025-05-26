package foreign

import (
	"slug/internal/object"
	"time"
)

func fnTimeClock() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 0 {
				return ctx.NewError("wrong number of arguments. got=%d, want=0",
					len(args))
			}

			return &object.Integer{Value: time.Now().UnixMilli()}
		},
	}
}

func fnTimeClockNanos() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 0 {
				return ctx.NewError("wrong number of arguments. got=%d, want=0",
					len(args))
			}

			return &object.Integer{Value: time.Now().UnixNano()}
		},
	}
}

func fnTimeSleep() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Ensure a single integer argument is provided
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			// Check if the argument is an integer
			intArg, ok := args[0].(*object.Integer)
			if !ok {
				return ctx.NewError("argument to `sleep` must be an INTEGER, got=%s", args[0].Type())
			}

			// Validate non-negative milliseconds
			if intArg.Value < 0 {
				return ctx.NewError("argument to `sleep` must be non-negative, got=%d", intArg.Value)
			}

			// Pause execution for the specified duration
			time.Sleep(time.Duration(intArg.Value) * time.Millisecond)

			// Return ctx.Nil() as there is no meaningful response
			return ctx.Nil()
		},
	}
}
