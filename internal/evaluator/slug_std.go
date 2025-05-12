package evaluator

import (
	"fmt"
	"slug/internal/object"
)

func fnStdPrintLn() *object.Foreign {
	return &object.Foreign{
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

func fnStdLen() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
		if len(args) != 1 {
			return newError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		switch arg := args[0].(type) {
		case *object.List:
			return &object.Integer{Value: int64(len(arg.Elements))}
		case *object.Map:
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

func fnStdType() *object.Foreign {
	return &object.Foreign{Fn: func(args ...object.Object) object.Object {
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

func fnStdKeys() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			// Check the number of arguments
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}

			// Ensure the argument is of type MAP_OBJ
			if args[0].Type() != object.MAP_OBJ {
				return newError("argument to `keys` must be a MAP, got=%s", args[0].Type())
			}

			// Extract the map
			mapObj := args[0].(*object.Map)

			// Collect keys
			keys := make([]object.Object, 0, len(mapObj.Pairs))
			for _, pair := range mapObj.Pairs {
				keys = append(keys, pair.Key)
			}

			// Return the keys as an List object
			return &object.List{Elements: keys}
		},
	}
}

// map functions
// -------------

func fnStdGet() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return newError("argument to `get` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as map key: %s", args[1].Type())
			}

			mapKey := key.MapKey()
			if pair, ok := mapObj.Pairs[mapKey]; ok {
				return pair.Value
			}

			return NIL
		},
	}
}

func fnStdPut() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return newError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return newError("argument to `put` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as map key: %s", args[1].Type())
			}

			newPairs := make(map[object.MapKey]object.MapPair)
			for k, v := range mapObj.Pairs {
				newPairs[k] = v
			}

			mapKey := key.MapKey()
			newPairs[mapKey] = object.MapPair{Key: args[1], Value: args[2]}

			return &object.Map{Pairs: newPairs}
		},
	}
}

func fnStdRemove() *object.Foreign {
	return &object.Foreign{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return newError("argument to `remove` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return newError("unusable as map key: %s", args[1].Type())
			}

			newPairs := make(map[object.MapKey]object.MapPair)
			for k, v := range mapObj.Pairs {
				newPairs[k] = v
			}

			mapKey := key.MapKey()
			delete(newPairs, mapKey)

			return &object.Map{Pairs: newPairs}
		},
	}
}
