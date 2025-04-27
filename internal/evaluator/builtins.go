package evaluator

import (
	"fmt"
	"slug/internal/object"
	"strings"
)

var builtins = map[string]*object.Builtin{
	"len":     funcLen(),
	"println": funcPrintLn(),

	// string functions
	"trim":       funcTrim(),
	"contains":   funcContains(),
	"startsWith": funcStartsWith(),
	"endsWith":   funcEndsWith(),
	"indexOf":    funcIndexOf(),

	// testing functions
	"assert":      funcAssert(),
	"assertEqual": funcAssertEqual(),

	// list functions
	"head": funcHead(),
	"tail": funcTail(),
	"peek": funcPeek(),
	"pop":  funcPop(),
	"push": funcPush(),
}

// funcPeek retrieves the last element of an array without modifying it.
// Returns NIL if the array is empty or an error for invalid input.
func funcPeek() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `peek` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				return arr.Elements[length-1]
			}

			return NIL
		},
	}
}

func funcPush() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return newError("wrong number of arguments. got=%d, want=2+",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `push` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			items := args[1:]
			length := len(arr.Elements)

			newElements := make([]object.Object, length+len(items))
			copy(newElements, arr.Elements)
			for i, item := range items {
				newElements[length+i] = item
			}

			return &object.Array{Elements: newElements}
		},
	}
}

func funcPop() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `pop` must be LIST, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				popped := arr.Elements[length-1]
				arr.Elements = arr.Elements[:length-1]
				return popped
			}

			return NIL
		},
	}
}

func funcHead() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `head` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			if len(arr.Elements) > 0 {
				return arr.Elements[0]
			}

			return NIL
		},
	}
}

func funcTail() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("argument to `tail` must be ARRAY, got %s",
					args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)
			if length > 0 {
				newElements := make([]object.Object, length-1)
				copy(newElements, arr.Elements[1:length])
				return &object.Array{Elements: newElements}
			}

			return NIL
		},
	}
}

func funcPrintLn() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				fmt.Print(arg.Inspect())
				if i < len(args)-1 {
					fmt.Print(" ")
				}
			}
			if len(args) > 0 {
				fmt.Println()
			}

			return NIL
		},
	}
}

func funcAssert() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return newError("wrong number of arguments. got=%d, want=1",
					len(args))
			}
			if args[0].Type() != object.BOOLEAN_OBJ {
				return newError("first argument to `assert` must be BOOLEAN, got %s",
					args[0].Type())
			}

			test := args[0].(*object.Boolean)
			if !test.Value {
				if len(args) == 2 {
					return newError(fmt.Sprintf("Assertion failed: %s", args[1].Inspect()))
				}
				return newError("Assertion failed")
			}

			return NIL
		},
	}
}

func funcAssertEqual() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 || len(args) > 3 {
				return newError("wrong number of arguments. got=%d, want=2",
					len(args))
			}
			if args[0].Type() != args[1].Type() {
				return newError("arguments to `assertEqual` must be the same type, got %s and %s",
					args[0].Type(), args[1].Type())
			}

			test := args[0].Inspect() == args[1].Inspect()
			if !test {
				if len(args) == 3 {
					return newError(fmt.Sprintf("Assertion failed: %s != %s, %s",
						args[0].Inspect(), args[1].Inspect(), args[2].Inspect()))
				}
				return newError(fmt.Sprintf("Assertion failed: %s != %s",
					args[0].Inspect(), args[1].Inspect()))
			}

			return NIL
		},
	}
}

func funcLen() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		switch arg := args[0].(type) {
		case *object.Array:
			return &object.Integer{Value: int64(len(arg.Elements))}
		case *object.String:
			return &object.Integer{Value: int64(len(arg.Value))}
		default:
			return newError("argument to `len` not supported, got %s",
				args[0].Type())
		}
	},
	}
}

func funcTrim() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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

func funcContains() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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

func funcStartsWith() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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

func funcEndsWith() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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

func funcIndexOf() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
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
