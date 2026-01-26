package foreign

import (
	"log/slog"
	"slug/internal/object"
)

func fnMetaHasTag() *object.Foreign {
	return &object.Foreign{
		Name: "hasTag",
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
		Name: "getTag",
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
		Name: "searchModuleTags",
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
			module, err := ctx.LoadModule(moduleName.Value)
			if err != nil {
				return ctx.NewError("failed to load module '%s': %s", moduleName.Value, err.Error())
			}

			slog.Debug("module loaded",
				slog.Any("module-name", module.Name),
				slog.Any("path", module.Path),
				slog.Any("binding-count", len(module.Env.Bindings)))

			m := &object.Map{}

			for name, binding := range module.Env.Bindings {

				slog.Debug("binding module value",
					slog.Any("module-name", module.Name),
					slog.Any("binding-name", name),
					slog.Any("bound-value", binding.Value.Type()),
				)

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
		Name: "searchScopeTags",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("searchScopeTags expects 1 argument: tag name")
			}

			tagName, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument must be a string tag name")
			}

			var tuples []object.Object

			env := ctx.CurrentEnv()

			for env != nil {
				for name, binding := range env.Bindings {
					if hasTag(binding, tagName.Value) {
						taggable, ok := binding.Value.(object.Taggable)
						if ok {
							opts, _ := taggable.GetTagParams(tagName.Value)
							var tuple = make([]object.Object, 3)
							tuple[0] = &object.String{Value: name}
							tuple[1] = binding.Value
							tuple[2] = &opts
							tuples = append(tuples, &object.List{
								Elements: tuple,
							})
						} else {
							slog.Warn("this should not happen")
						}
					}
				}
				env = env.Outer
			}

			return &object.List{
				Elements: tuples,
			}
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
