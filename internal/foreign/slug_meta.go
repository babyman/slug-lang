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
func fnMetaSearchTags() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 || len(args) > 3 {
				return ctx.NewError("searchTags expects 1-3 arguments: module name, tag name, and optional ignoreExports flag")
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

			// Check optional ignoreExports flag
			ignoreExports := false
			if len(args) == 3 {
				flag, ok := args[2].(*object.Boolean)
				if !ok {
					return ctx.NewError("third argument must be a boolean for ignoreExports")
				}
				ignoreExports = flag.Value
			}

			// Load the targeted module
			module, err := ctx.LoadModule(strings.Split(moduleName.Value, "."))
			if err != nil {
				return ctx.NewError("failed to load module '%s': %s", moduleName.Value, err.Error())
			}

			// Search through module bindings
			searchTarget := module.Env.Exports
			if ignoreExports {
				searchTarget = module.Env.Store
			}

			m := &object.Map{}

			for name, binding := range searchTarget {

				// ignore imported values
				if module.Env.Imports != nil && module.Env.Imports[name] != nil {
					continue
				}

				if hasTag(binding, tagName.Value) {
					log.Debug("%s.%s tagged with '%s'", module.Name, name, tagName.Value)
					m.Put(&object.String{Value: name}, binding.Value)
				}
			}

			return m
		},
	}
}

func hasTag(binding *object.Binding, tagName string) bool {
	if binding == nil || binding.Value == nil {
		return false
	}

	var tags map[string]object.List
	switch v := binding.Value.(type) {
	case *object.Function:
		tags = v.Tags
	case *object.Foreign:
		tags = v.Tags
		// Add more cases if needed
	}

	if tags == nil {
		return false
	}

	_, exists := tags[tagName]
	return exists
}
