package foreign

import (
	"bytes"
	"fmt"
	"slug/internal/object"
)

func fnStdPrint() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			var out bytes.Buffer
			for i, arg := range args {
				out.WriteString(arg.Inspect())
				if i < len(args)-1 {
					out.WriteString(" ")
				}
			}
			fmt.Print(out.String())
			//return &object.String{Value: out.String()}
			return ctx.Nil()
		},
	}
}

func fnStdLen() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1",
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
			return ctx.NewError("argument to `len` not supported, got %s",
				args[0].Type())
		}
	},
	}
}

func fnStdType() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		return &object.String{
			Value: string(args[0].Type()),
		}
	},
	}
}

func fnStdIsDefined() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1",
				len(args))
		}
		v, ok := args[0].(*object.String)
		if !ok {
			return ctx.NewError("argument to `defined` must be a string, got %s",
				args[0].Type())
		}

		_, ok = ctx.CurrentEnv().GetBinding(v.Value)

		return ctx.NativeBoolToBooleanObject(ok)
	},
	}
}

func fnStdKeys() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Check the number of arguments
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			// Ensure the argument is of type MAP_OBJ
			if args[0].Type() != object.MAP_OBJ {
				return ctx.NewError("argument to `keys` must be a MAP, got=%s", args[0].Type())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return ctx.NewError("argument to `get` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return ctx.NewError("unusable as map key: %s", args[1].Type())
			}

			mapKey := key.MapKey()
			if pair, ok := mapObj.Pairs[mapKey]; ok {
				return pair.Value
			}

			return ctx.Nil()
		},
	}
}

func fnStdPut() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 3 {
				return ctx.NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return ctx.NewError("argument to `put` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return ctx.NewError("unusable as map key: %s", args[1].Type())
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			if args[0].Type() != object.MAP_OBJ {
				return ctx.NewError("argument to `remove` must be map, got %s", args[0].Type())
			}

			mapObj := args[0].(*object.Map)
			key, ok := args[1].(object.Hashable)
			if !ok {
				return ctx.NewError("unusable as map key: %s", args[1].Type())
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
