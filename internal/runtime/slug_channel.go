package runtime

import (
	"slug/internal/object"
)

func fnChannelChan() *object.Foreign {
	return &object.Foreign{
		Name: "chan",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) > 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=0 or 1", len(args))
			}
			capacity := int64(0)
			if len(args) == 1 {
				num, ok := args[0].(*object.Number)
				if !ok {
					return ctx.NewError("channel capacity must be a number, got %s", args[0].Type())
				}
				capacity = num.Value.ToInt64()
				if capacity < 0 {
					return ctx.NewError("channel capacity must be >= 0")
				}
				maxInt := int64(^uint(0) >> 1)
				if capacity > maxInt {
					return ctx.NewError("channel capacity exceeds maximum size")
				}
			}
			return object.NewChannel(int(capacity))
		},
	}
}

func fnChannelClose() *object.Foreign {
	return &object.Foreign{
		Name: "close",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}
			ch, ok := args[0].(*object.Channel)
			if !ok {
				return ctx.NewError("argument to close must be a channel, got %s", args[0].Type())
			}
			ch.Close()
			return object.NIL
		},
	}
}
