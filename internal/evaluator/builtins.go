package evaluator

import (
	"fmt"
	"slug/internal/object"
)

var builtins = map[string]*object.Builtin{
	"assert":      funcAssert(),
	"assertEqual": funcAssertEqual(),
	"head":        funcHead(),
	"len":         funcLen(),
	"peek":        funcPeek(),
	"pop":         funcPop(),
	"push":        funcPush(),
	"println":     funcPrintLn(),
	"tail":        funcTail(),
}

// funcPeek retrieves the last element of an array without modifying it.
// Returns NULL if the array is empty or an error for invalid input.
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

			return NULL
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

			return NULL
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

			return NULL
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

			return NULL
		},
	}
}

func funcPrintLn() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
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

			return NULL
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
					args[0].Type(), args[0].Type())
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

			return NULL
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
