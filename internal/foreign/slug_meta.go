package foreign

import (
	"log/slog"
	"slug/internal/ast"
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

			supplier := func(name string, value object.Object, args object.Object) object.Object { return value }
			if fn, ok := args[1].(*object.Function); !ok {
				return ctx.NewError("second argument must be a supplier function")
			} else {
				supplier = func(name string, value object.Object, args object.Object) object.Object {
					return ctx.ApplyFunction("<annon>", fn, []object.Object{
						&object.String{Value: name},
						value,
						args,
					})
				}
			}

			env := ctx.CurrentEnv()
			for env != nil {
				for name, binding := range env.Bindings {
					taggable, isTaggable := binding.Value.(object.Taggable)
					if isTaggable && hasTag(binding, tagName.Value) {
						if binding.IsMutable {
							tagParamList, _ := taggable.GetTagParams(tagName.Value)
							newValue := supplier(name, binding.Value, &tagParamList)
							if taggable == newValue {
								continue
							}
							if newValue.Type() == object.ERROR_OBJ {
								return newValue
							}
							// Clone existing tags into the new value (if Taggable)
							if newTaggable, ok := newValue.(object.Taggable); ok {
								for tag, params := range taggable.GetTags() {
									newTaggable.SetTag(tag, params)
								}
							}
							if _, err := env.Assign(name, newValue); err != nil {
								return ctx.NewError(err.Error())
							}
						} else {
							slog.Debug("rebind not supported for immutable value",
								slog.Any("name", name),
								slog.Any("value", binding.Value.Inspect()))
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

func fnMetaWithEnv() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("withEnv expects exactly 1 argument: a FunctionGroup")
			}

			origFG, ok := args[0].(*object.FunctionGroup)
			if !ok {
				return ctx.NewError("withEnv expects a FunctionGroup, but got: %s", args[0].Type())
			}

			wrapped := &object.FunctionGroup{
				Functions: make(map[ast.FSig]object.Object, len(origFG.Functions)),
			}

			for sig, fnObj := range origFG.Functions {
				switch fn := fnObj.(type) {
				case *object.Function:
					// Wrap each user-defined function so that at call time we compose:
					// inner = module function's env, outer = caller's env.
					adapter := &object.Foreign{
						Signature:  fn.Signature,
						Parameters: fn.Parameters,
						Name:       "withEnv",
						Fn: func(callCtx object.EvaluatorContext, callArgs ...object.Object) object.Object {
							callerEnv := callCtx.CurrentEnv()

							// Compose an environment that uses the module's bindings as inner and
							// the caller as outer. Reuse bindings map; preserve metadata for diagnostics.
							composed := object.NewEnvironment()
							composed.Bindings = fn.Env.Bindings
							composed.Outer = callerEnv
							composed.Path = fn.Env.Path
							composed.ModuleFqn = fn.Env.ModuleFqn
							composed.Src = fn.Env.Src

							// Clone the function with the composed env and dispatch via ApplyFunction
							cloned := &object.Function{
								Signature:   fn.Signature,
								Tags:        fn.Tags,
								Parameters:  fn.Parameters,
								Body:        fn.Body,
								Env:         composed,
								HasTailCall: fn.HasTailCall,
							}
							return callCtx.ApplyFunction("<withEnv>", cloned, callArgs)
						},
					}
					wrapped.Functions[sig] = adapter

				case *object.Foreign:
					// Foreign functions have no module Env to extend. Leave as-is.
					// Documented limitation: withEnv currently doesn’t extend env for foreign calls.
					wrapped.Functions[sig] = fn

				default:
					return ctx.NewError("withEnv only supports wrapping FunctionGroup entries that are functions or foreign functions, got: %s", fnObj.Type())
				}
			}
			return wrapped
		},
	}
}
