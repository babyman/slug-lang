package foreign

import (
	"os"
	"slug/internal/object"
)

func fnSysEnv() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			arg := args[0]

			// Ensure the argument is of type MAP_OBJ
			if arg.Type() != object.STRING_OBJ {
				return ctx.NewError("argument to STRING, got=%s", arg.Type())
			}

			value, ok := os.LookupEnv(arg.(*object.String).Value)

			if ok {
				return &object.String{Value: value}
			}
			return ctx.Nil()
		},
	}
}
