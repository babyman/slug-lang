package foreign

import (
	"slug/internal/object"
)

func fnMetaHasTag() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 2 {
				return ctx.NewError("hasTag expects exactly 2 arguments: object and tag name")
			}

			tag, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to hasTag must be a string")
			}

			var tags map[string]object.List
			switch o := args[0].(type) {
			case *object.Function:
				tags = o.Tags
			case *object.Foreign:
				tags = o.Tags
			}

			if tags == nil {
				return ctx.NativeBoolToBooleanObject(false)
			}

			_, exists := tags[tag.Value]
			return ctx.NativeBoolToBooleanObject(exists)
		},
	}
}

func fnMetaGetTag() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("getTag expects exactly 2 arguments: object and tag name")
			}

			tag, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to getTag must be a string")
			}

			var tags map[string]object.List
			switch o := args[0].(type) {
			case *object.Function:
				tags = o.Tags
			case *object.Foreign:
				tags = o.Tags
			}

			if tags == nil {
				return ctx.Nil()
			}

			if args, exists := tags[tag.Value]; exists {
				return &args
			}

			return ctx.Nil()
		},
	}
}
