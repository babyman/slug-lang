package foreign

import (
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

func fnStringContains() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 2 {
			return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return ctx.NewError("arguments to `contains` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return ctx.NativeBoolToBooleanObject(strings.Contains(arg.Value, args[1].(*object.String).Value))
			// todo support for lists contains()
		default:
			return ctx.NewError("argument to `contains` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringStartsWith() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 2 {
			return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return ctx.NewError("arguments to `startsWith` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return ctx.NativeBoolToBooleanObject(strings.HasPrefix(arg.Value, args[1].(*object.String).Value))
		default:
			return ctx.NewError("argument to `startsWith` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringEndsWith() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 2 {
			return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return ctx.NewError("arguments to `endsWith` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return ctx.NativeBoolToBooleanObject(strings.HasSuffix(arg.Value, args[1].(*object.String).Value))
		default:
			return ctx.NewError("argument to `endsWith` not supported, got %s", args[0].Type())
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
			if len(args) > 2 && args[2].Type() == object.INTEGER_OBJ {
				start = int(args[2].(*object.Integer).Value)
			}
			return &object.Integer{Value: int64(strings.Index(arg.Value[start:], args[1].(*object.String).Value))}
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

func fnStringIsUpper() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return ctx.NativeBoolToBooleanObject(arg.Value == strings.ToUpper(arg.Value))
		default:
			return ctx.NewError("argument to `isUpper` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIsLower() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return ctx.NativeBoolToBooleanObject(arg.Value == strings.ToLower(arg.Value))
		default:
			return ctx.NewError("argument to `isLower` not supported, got %s", args[0].Type())
		}
	},
	}
}
