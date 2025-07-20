package foreign

import (
	"slug/internal/log"
	"slug/internal/object"
	"strings"
)

func fnMetaHasTag() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 2 {
				return ctx.NewError("hasTag expects exactly 2 arguments: object and tagName name")
			}

			tagName, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to hasTag must be a string")
			}

			switch o := args[0].(type) {
			case object.Taggable:
				return ctx.NativeBoolToBooleanObject(o.HasTag(tagName.Value))
			}
			return ctx.NativeBoolToBooleanObject(false)
		},
	}
}

func fnMetaGetTag() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("getTag expects exactly 2 arguments: object and tagName name")
			}

			tagName, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to getTag must be a string")
			}

			switch o := args[0].(type) {
			case object.Taggable:
				if args, exists := o.GetTagParams(tagName.Value); exists {
					return &args
				}
			}
			return ctx.Nil()
		},
	}
}

func fnMetaSearchModuleTags() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 || len(args) > 3 {
				return ctx.NewError("searchModuleTags expects 1-3 arguments: module name, tag name, and optional includePrivate flag")
			}

			// Check module name
			moduleName, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument must be the module name as a string")
			}

			// Check tag name
			tagName, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument must be the tag name as a string")
			}

			// Check optional includePrivate flag
			includePrivate := false
			if len(args) == 3 {
				flag, ok := args[2].(*object.Boolean)
				if !ok {
					return ctx.NewError("third argument must be a boolean for includePrivate")
				}
				includePrivate = flag.Value
			}

			// Load the targeted module
			module, err := ctx.LoadModule(strings.Split(moduleName.Value, "."))
			if err != nil {
				return ctx.NewError("failed to load module '%s': %s", moduleName.Value, err.Error())
			}

			log.Debug("module: %s (%s), len %d\n", module.Name, module.Path, len(module.Env.Bindings))

			m := &object.Map{}

			for name, binding := range module.Env.Bindings {

				log.Debug("name: %s, binding: %s\n", name, binding.Value.Type())

				if binding.Meta.IsImport {
					continue
				}

				if (includePrivate || binding.Meta.IsExport) &&
					hasTag(binding, tagName.Value) {

					m.Put(&object.String{Value: name}, binding.Value)
				}
			}

			return m
		},
	}
}

func fnMetaSearchScopeTags() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("searchScopeTags expects 1 argument: tag name")
			}

			tagName, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument must be a string tag name")
			}

			m := &object.Map{}
			env := ctx.CurrentEnv()

			for env != nil {
				for name, binding := range env.Bindings {
					if hasTag(binding, tagName.Value) {
						m.Put(&object.String{Value: name}, binding.Value)
					}
				}
				env = env.Outer
			}

			return m
		},
	}
}

func fnMetaRebindScopeTags() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("rebindScopeTags expects 2 arguments: tag name and supplier function")
			}

			tagName, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument must be a string tag name")
			}

			supplier := func(name string, value object.Object) object.Object { return value }
			if fn, ok := args[1].(*object.Function); !ok {
				return ctx.NewError("second argument must be a supplier function")
			} else {
				supplier = func(name string, value object.Object) object.Object {
					return ctx.ApplyFunction("<annon>", fn, []object.Object{
						&object.String{Value: name},
						value,
					})
				}
			}

			env := ctx.CurrentEnv()
			for env != nil {
				for name, binding := range env.Bindings {
					if hasTag(binding, tagName.Value) {
						if binding.IsMutable {
							newValue := supplier(name, binding.Value)
							if newValue.Type() == object.ERROR_OBJ {
								return newValue
							}
							if _, err := env.Assign(name, newValue); err != nil {
								return ctx.NewError(err.Error())
							}
						} else {
							log.Debug("rebind not supported for %s, %s is not mutable", name, binding.Value.Inspect())
						}
					}
				}
				env = env.Outer
			}
			return ctx.Nil()
		},
	}
}

func hasTag(binding *object.Binding, tagName string) bool {
	if binding == nil {
		return false
	}

	// Check if the binding contains a group of functions
	fg, ok := binding.Value.(object.Taggable)
	return ok && fg.HasTag(tagName)
}
