package evaluator

import (
	"slug/internal/object"
	"strings"
)

func fnStringTrim() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.TrimSpace(arg.Value)}
		default:
			return newError("argument to `trim` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringContains() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return newError("arguments to `contains` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return nativeBoolToBooleanObject(strings.Contains(arg.Value, args[1].(*object.String).Value))
			// todo support for lists contains()
		default:
			return newError("argument to `contains` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringStartsWith() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return newError("arguments to `startsWith` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return nativeBoolToBooleanObject(strings.HasPrefix(arg.Value, args[1].(*object.String).Value))
		default:
			return newError("argument to `startsWith` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringEndsWith() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return newError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return newError("arguments to `endsWith` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			return nativeBoolToBooleanObject(strings.HasSuffix(arg.Value, args[1].(*object.String).Value))
		default:
			return newError("argument to `endsWith` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIndexOf() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) < 2 {
			return newError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return newError("arguments to `indexOf` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			start := 0
			if len(args) > 2 && args[2].Type() == object.INTEGER_OBJ {
				start = int(args[2].(*object.Integer).Value)
			}
			return &object.Integer{Value: int64(strings.Index(arg.Value[start:], args[1].(*object.String).Value))}
		default:
			return newError("argument to `indexOf` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToUpper() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToUpper(arg.Value)}
		default:
			return newError("argument to `toUpper` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToLower() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToLower(arg.Value)}
		default:
			return newError("argument to `toLower` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIsUpper() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return nativeBoolToBooleanObject(arg.Value == strings.ToUpper(arg.Value))
		default:
			return newError("argument to `isUpper` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIsLower() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return nativeBoolToBooleanObject(arg.Value == strings.ToLower(arg.Value))
		default:
			return newError("argument to `isLower` not supported, got %s", args[0].Type())
		}
	},
	}
}
