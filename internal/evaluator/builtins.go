package evaluator

import (
	"fmt"
	"slug/internal/object"
	"strings"
)

var builtins = map[string]*object.Builtin{
	"type":    funcType(),
	"len":     funcLen(),
	"println": funcPrintLn(),

	// string functions
	"trim":       funcTrim(),
	"contains":   funcContains(),
	"startsWith": funcStartsWith(),
	"endsWith":   funcEndsWith(),
	"indexOf":    funcIndexOf(),

	// map functions
	"keys":   funcKeys(),
	"get":    funcGet(),
	"put":    funcPut(),
	"remove": funcRemove(),

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

func funcLen() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		switch arg := args[0].(type) {
		case *object.Array:
			return &object.Integer{Value: int64(len(arg.Elements))}
		case *object.Hash:
			return &object.Integer{Value: int64(len(arg.Pairs))}
		case *object.String:
			return &object.Integer{Value: int64(len(arg.Value))}
		default:
			return newError("argument to `len` not supported, got %s",
				args[0].Type())
		}
	},
	}
}

func funcType() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		return &object.String{
			Value: string(args[0].Type()),
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

func funcKeys() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			// Check the number of arguments
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			// Ensure the argument is of type HASH_OBJ
			if args[0].Type() != object.HASH_OBJ {
				return newError("argument to `keys` must be a MAP, got=%s", args[0].Type())
			}

			// Extract the hash map
			hash := args[0].(*object.Hash)

			// Collect keys
			keys := make([]object.Object, 0, len(hash.Pairs))
			for _, pair := range hash.Pairs {
				keys = append(keys, pair.Key)
			}

			// Return the keys as an Array object
			return &object.Array{Elements: keys}
		},
	}
}

func funcGet() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.HASH_OBJ {
				return newError("argument to `get` must be HASH, got %s", args[0].Type())
			}

			hash := args[0].(*object.Hash)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", args[1].Type())
			}

			hashKey := key.HashKey()
			if pair, ok := hash.Pairs[hashKey]; ok {
				return pair.Value
			}

			return NIL
		},
	}
}

func funcPut() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.HASH_OBJ {
				return newError("argument to `put` must be HASH, got %s", args[0].Type())
			}

			hash := args[0].(*object.Hash)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", args[1].Type())
			}

			newPairs := make(map[object.HashKey]object.HashPair)
			for k, v := range hash.Pairs {
				newPairs[k] = v
			}

			hashKey := key.HashKey()
			newPairs[hashKey] = object.HashPair{Key: args[1], Value: args[2]}

			return &object.Hash{Pairs: newPairs}
		},
	}
}

func funcRemove() *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.HASH_OBJ {
				return newError("argument to `remove` must be HASH, got %s", args[0].Type())
			}

			hash := args[0].(*object.Hash)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as hash key: %s", args[1].Type())
			}

			newPairs := make(map[object.HashKey]object.HashPair)
			for k, v := range hash.Pairs {
				newPairs[k] = v
			}

			hashKey := key.HashKey()
			delete(newPairs, hashKey)

			return &object.Hash{Pairs: newPairs}
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
