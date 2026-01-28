package foreign

import (
	"fmt"
	"slug/internal/dec64"
	"slug/internal/object"
)

func fnStdType() *object.Foreign {
	return &object.Foreign{Name: "type", Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1",
				len(args))
		}

		if structVal, ok := args[0].(*object.StructValue); ok {
			if structVal.Schema == nil {
				return ctx.NewError("struct has no schema")
			}
			return structVal.Schema
		}

		if _, ok := args[0].(*object.StructSchema); ok {
			return object.InternSymbol("struct")
		}

		tag, ok := typeTagForObject(args[0])
		if !ok {
			return ctx.NewError("unknown type: %s", args[0].Type())
		}

		return object.InternSymbol(tag)
	}}
}

func typeTagForObject(obj object.Object) (string, bool) {
	switch obj.Type() {
	case object.NIL_OBJ:
		return "nil", true
	case object.BOOLEAN_OBJ:
		return "bool", true
	case object.NUMBER_OBJ:
		return "number", true
	case object.STRING_OBJ:
		return "string", true
	case object.LIST_OBJ:
		return "list", true
	case object.MAP_OBJ:
		return "map", true
	case object.BYTE_OBJ:
		return "bytes", true
	case object.SYMBOL_OBJ:
		return "symbol", true
	case object.ERROR_OBJ:
		return "error", true
	case object.FUNCTION_OBJ:
		return "function", true
	case object.TASK_HANDLE_OBJ:
		return "task", true
	case object.STRUCT_SCHEMA_OBJ:
		return "struct", true
	default:
		return "", false
	}
}

func fnStdIsDefined() *object.Foreign {
	return &object.Foreign{Name: "isDefined", Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
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
		Name: "printf",
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
		Name: "sprintf",
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
		Name: "keys",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Check the number of arguments
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch obj := args[0].(type) {
			case *object.Map:
				keys := make([]object.Object, 0, len(obj.Pairs))
				for _, pair := range obj.Pairs {
					keys = append(keys, pair.Key)
				}
				return &object.List{Elements: keys}
			case *object.StructValue:
				if obj.Schema == nil {
					return ctx.NewError("struct has no schema")
				}
				keys := make([]object.Object, len(obj.Schema.Fields))
				for i, field := range obj.Schema.Fields {
					keys[i] = object.InternSymbol(field.Name)
				}
				return &object.List{Elements: keys}
			case *object.StructSchema:
				keys := make([]object.Object, len(obj.Fields))
				for i, field := range obj.Fields {
					keys[i] = object.InternSymbol(field.Name)
				}
				return &object.List{Elements: keys}
			default:
				return ctx.NewError("argument to `keys` must be a map or struct, got=%s", args[0].Type())
			}
		},
	}
}

func fnStdSym() *object.Foreign {
	return &object.Foreign{
		Name: "sym",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}
			str, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `sym` must be a string, got=%s", args[0].Type())
			}
			return object.InternSymbol(str.Value)
		},
	}
}

func fnStdLabel() *object.Foreign {
	return &object.Foreign{
		Name: "label",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}
			sym, ok := args[0].(*object.Symbol)
			if !ok {
				return ctx.NewError("argument to `label` must be a symbol, got=%s", args[0].Type())
			}
			return &object.String{Value: sym.Name}
		},
	}
}

// map functions
// -------------

func fnStdGet() *object.Foreign {
	return &object.Foreign{
		Name: "get",
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
		Name: "put",
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
		Name: "remove",
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
		Name: "update",
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
		Name: "swap",
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
		Name: "parseNumber",
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
