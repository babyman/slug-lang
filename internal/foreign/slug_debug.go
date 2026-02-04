package foreign

import (
	"fmt"
	"slug/internal/object"
)

func fnDebugIdent() *object.Foreign {
	return &object.Foreign{
		Name: "ident",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("id() expects exactly 1 argument")
			}

			// Use the pointer address as a unique identifier
			obj := args[0]

			addr := fmt.Sprintf("%s@%p", obj.Type()[0:3], obj)
			return &object.String{Value: addr}
		},
	}
}
