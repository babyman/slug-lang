package foreign

import (
	"slug/internal/dec64"
	"slug/internal/object"
	"strings"
)

func fnStringTrim() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.TrimSpace(arg.Value)}
		default:
			return ctx.NewError("argument to `trim` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIndexOf() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) < 2 {
			return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return ctx.NewError("arguments to `indexOf` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			start := 0
			if len(args) > 2 && args[2].Type() == object.NUMBER_OBJ {
				start = args[2].(*object.Number).Value.ToInt()
			}
			index := strings.Index(arg.Value[start:], args[1].(*object.String).Value)
			return &object.Number{Value: dec64.FromInt(index)}
		default:
			return ctx.NewError("argument to `indexOf` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToUpper() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToUpper(arg.Value)}
		default:
			return ctx.NewError("argument to `toUpper` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToLower() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToLower(arg.Value)}
		default:
			return ctx.NewError("argument to `toLower` not supported, got %s", args[0].Type())
		}
	},
	}
}
