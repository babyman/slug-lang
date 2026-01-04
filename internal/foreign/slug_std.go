package foreign

import (
	"fmt"
	"slug/internal/dec64"
	"slug/internal/object"
)

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

func fnStdPrintf() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want>=1", len(args))
			}

			format, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument must be a format STRING, got %s", args[0].Type())
			}

			fmtArgs := make([]interface{}, len(args)-1)
			for i := 1; i < len(args); i++ {
				fmtArgs[i-1] = ToNative(args[i])
			}
			fmt.Printf(format.Value, fmtArgs...)
			return ctx.Nil()
		},
	}
}

func fnStdSprintf() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want>=1", len(args))
			}

			format, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument must be a format STRING, got %s", args[0].Type())
			}

			fmtArgs := make([]interface{}, len(args)-1)
			for i := 1; i < len(args); i++ {
				fmtArgs[i-1] = ToNative(args[i])
			}

			return &object.String{Value: fmt.Sprintf(format.Value, fmtArgs...)}
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

func fnStdUpdate() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 3 {
				return ctx.NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.LIST_OBJ {
				return ctx.NewError("argument to `update` must be list, got %s", args[0].Type())
			}

			list := args[0].(*object.List)
			index, ok := args[1].(*object.Number)
			if !ok {
				return ctx.NewError("index must be NUMBER, got %s", args[1].Type())
			}

			i := index.Value.ToInt()

			if i < 0 || i >= len(list.Elements) {
				return ctx.NewError("index out of range: %v", index.Value)
			}

			newElements := make([]object.Object, len(list.Elements))
			copy(newElements, list.Elements)
			newElements[i] = args[2]

			return &object.List{Elements: newElements}
		},
	}
}

func fnStdSwap() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 3 {
				return ctx.NewError("wrong number of arguments. got=%d, want=3", len(args))
			}

			if args[0].Type() != object.LIST_OBJ {
				return ctx.NewError("argument to `swap` must be list, got %s", args[0].Type())
			}

			list := args[0].(*object.List)
			index1, ok1 := args[1].(*object.Number)
			index2, ok2 := args[2].(*object.Number)
			if !ok1 || !ok2 {
				return ctx.NewError("indices must be NUMBERS, got %s and %s", args[1].Type(), args[2].Type())
			}

			i1 := index1.Value.ToInt()
			i2 := index2.Value.ToInt()

			if i1 < 0 || i1 >= len(list.Elements) ||
				i2 < 0 || i2 >= len(list.Elements) {
				return ctx.NewError("index out of range")
			}

			newElements := make([]object.Object, len(list.Elements))
			copy(newElements, list.Elements)
			newElements[i1], newElements[i2] = newElements[i2], newElements[i1]

			return &object.List{Elements: newElements}
		},
	}
}

func fnStdParseNumber() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `parseNumber` must be STRING, got %s", args[0].Type())
			}

			n, err := dec64.FromString(str.Value)
			if err != nil {
				return ctx.NewError("could not convert string to number: %s", err)
			}

			return &object.Number{Value: n}
		},
	}
}
