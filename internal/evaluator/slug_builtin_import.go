package evaluator

import (
	"slug/internal/object"
	"strings"
)

func fnBuiltinImport() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 {
				return ctx.NewError("import expects at least one argument")
			}

			m := &object.Map{}

			for i, arg := range args {
				if arg.Type() != object.STRING_OBJ {
					return ctx.NewError("argument %d to import must be a string", i)
				}
				module, err := ctx.LoadModule(strings.Split(arg.Inspect(), "."))
				if err != nil {
					return newError(err.Error())
				}

				for name, val := range module.Env.Store {
					m.Put(&object.String{Value: name}, val.Value)
				}
			}
			return m
		},
	}
}
