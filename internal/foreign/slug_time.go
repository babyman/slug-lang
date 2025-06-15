package foreign

import (
	"slug/internal/dec64"
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

			return &object.Number{Value: dec64.FromInt64(time.Now().UnixMilli())}
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

			return &object.Number{Value: dec64.FromInt64(time.Now().UnixNano())}
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
			intArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("argument to `sleep` must be an NUMBER, got=%s", args[0].Type())
			}

			millis := intArg.Value.ToInt64()
			if millis < 0 {
				return ctx.NewError("argument to `sleep` must be non-negative, got=%v", intArg.Value)
			}

			// Pause execution for the specified duration
			time.Sleep(time.Duration(millis) * time.Millisecond)

			return ctx.Nil()
		},
	}
}

func fnTimeFmtClock() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Validate the number of arguments
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			// First argument: milliseconds (integer)
			millisArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to `formatMillis` must be INTEGER, got=%s", args[0].Type())
			}

			// Second argument: format string
			formatArg, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to `formatMillis` must be STRING, got=%s", args[1].Type())
			}

			// Convert milliseconds to time.Time
			t := time.UnixMilli(millisArg.Value.ToInt64())

			// Format the time using the provided format string
			formattedTime := t.Format(formatArg.Value)

			// Return the formatted string
			return &object.String{Value: formattedTime}
		},
	}
}
