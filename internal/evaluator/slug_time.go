package evaluator

import (
	"slug/internal/object"
	"time"
)

func fnTimeClock() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return newError("wrong number of arguments. got=%d, want=0",
					len(args))
			}

			return &object.Integer{Value: time.Now().UnixMilli()}
		},
	}
}

func fnTimeClockNanos() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return newError("wrong number of arguments. got=%d, want=0",
					len(args))
			}

			return &object.Integer{Value: time.Now().UnixNano()}
		},
	}
}
