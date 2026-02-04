package runtime

import (
	"bytes"
	"fmt"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/foreign"
	"slug/internal/object"
	"slug/internal/util"
	"strconv"
	"strings"
	"unicode/utf8"
)

func fnBuiltinImport() *object.Foreign {
	return &object.Foreign{
		Name: "import",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 {
				return ctx.NewError("import expects at least one argument")
			}

			tempMap := make(map[string]object.Object)

			for i, arg := range args {
				strArg, ok := arg.(*object.String)
				if !ok {
					return ctx.NewError("argument %d to import must be a string, got %s", i, arg.Type())
				}

				module, err := ctx.LoadModule(strArg.Value)
				if err != nil {
					return ctx.NewError("failed to import '%s': %v", strArg.Value, err)
				}

				// Import exported bindings into the temp map
				for name, binding := range module.Env.Bindings {
					if !binding.Meta.IsExport {
						continue
					}
					//println(ctx.CurrentEnv().Path, "importing "+name)

					// Functions are always stored as a FunctionGroup in the binding layer.
					if fg, ok := binding.Value.(*object.FunctionGroup); ok {
						// Merge function groups across imported modules.
						if existing, exists := tempMap[name]; exists {
							existingFg, ok := existing.(*object.FunctionGroup)
							if !ok {
								// Short-term: warn and keep the first value.
								fmt.Printf("WARNING: import name collision for '%s' (non-function vs function) while importing '%s' (keeping first)", name, strArg.Value)
								continue
							}

							// Build/extend a composite group that delegates to both groups.
							// We keep this *live* by delegating to the original groups, not copying functions.
							// Detect duplicate signatures to keep things explicit.
							sigSeen := map[ast.FSig]bool{}
							// existing group's own functions
							for sig := range existingFg.Functions {
								sigSeen[sig] = true
							}
							// plus any delegated groups
							for _, dg := range existingFg.Delegates {
								for sig := range dg.Functions {
									sigSeen[sig] = true
								}
							}
							for sig := range fg.Functions {
								if sigSeen[sig] {
									//return ctx.NewError("import collision for function '%s' with duplicate signature %v while importing '%s'", name, sig, strArg.Value)
									fmt.Printf("WARNING: import collision for function '%s' with duplicate signature %v while importing '%s'\n", name, sig, strArg.Value)
								}
							}

							// If existingFg is not already composite, make it composite by delegating to itself.
							if existingFg.Delegates == nil {
								existingFg.Delegates = []*object.FunctionGroup{}
							}
							existingFg.Delegates = append(existingFg.Delegates, fg)
							continue
						}

						// First occurrence wins the slot; if later modules export same name,
						// we will merge by delegating.
						// Note: we store the group itself (not a BindingRef) so extensions to the group
						// are visible to the importer.
						tempMap[name] = fg
						continue
					}

					// Non-function exports: warn on collisions and keep the first value.
					if _, exists := tempMap[name]; exists {
						fmt.Printf("WARNING: import name collision for '%s' while importing '%s' (keeping first)", name, strArg.Value)
						continue
					}

					// Use live binding references for non-function exports.
					tempMap[name] = &object.BindingRef{Env: module.Env, Name: name}
				}
			}

			// Return a map containing the imported members
			m := &object.Map{
				Tags: map[string]object.List{
					object.IMPORT_TAG: {},
				},
			}
			for k, v := range tempMap {
				m.Put(object.InternSymbol(k), v)
			}
			return m
		},
	}
}

func fnBuiltinLen() *object.Foreign {
	return &object.Foreign{
		Name: "len",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			switch arg := args[0].(type) {
			case *object.List:
				return &object.Number{Value: dec64.FromInt(len(arg.Elements))}
			case *object.Map:
				return &object.Number{Value: dec64.FromInt(len(arg.Pairs))}
			case *object.String:
				return &object.Number{Value: dec64.FromInt(utf8.RuneCountInString(arg.Value))}
			case *object.Bytes:
				return &object.Number{Value: dec64.FromInt(len(arg.Value))}
			default:
				return ctx.NewError("argument to `len` not supported, got %s",
					args[0].Type())
			}
		},
	}
}

func fnBuiltinPrint() *object.Foreign {
	return &object.Foreign{
		Name: "print",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			var out bytes.Buffer
			for i, arg := range args {
				out.WriteString(arg.Inspect())
				if i < len(args)-1 {
					out.WriteString(" ")
				}
			}
			print(out.String())
			if len(args) > 0 {
				return args[0]
			}
			return ctx.Nil()
		},
	}
}

func fnBuiltinPrintLn() *object.Foreign {
	return &object.Foreign{
		Name: "println",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			var out bytes.Buffer
			for i, arg := range args {
				out.WriteString(arg.Inspect())
				if i < len(args)-1 {
					out.WriteString(" ")
				}
			}
			println(out.String())
			if len(args) > 0 {
				return args[0]
			}
			return ctx.Nil()
		},
	}
}

func fnBuiltinStacktrace() *object.Foreign {
	return &object.Foreign{
		Name: "stacktrace",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// stacktrace(err) must take exactly one argument
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments to `stacktrace`, got=%d, want=1", len(args))
			}
			arg := args[0]

			// Case 1: The argument itself is a RuntimeError (from `throw`)
			if errObj, ok := arg.(*object.RuntimeError); ok {
				return &object.String{Value: object.RenderStacktrace(errObj)}
			}

			// Case 2: The argument is the *payload* bound in a defer-on-error handler.
			// We don't know the variable name, so we search bindings for a RuntimeError
			// whose Payload is the same object as `arg`.
			env := ctx.CurrentEnv()

			for e := env; e != nil; e = e.Outer {
				for _, binding := range e.Bindings {
					if binding != nil && binding.Err != nil {
						rtErr := binding.Err
						if rtErr != nil && rtErr.Payload == arg {
							return &object.String{Value: object.RenderStacktrace(rtErr)}
						}
					}
				}
			}

			// Case 3: The argument is some other error type or not associated with a RuntimeError.
			switch arg.(type) {
			case *object.Error:
				return ctx.NewError("no runtime stacktrace available for this error value; it is not a RuntimeError")
			default:
				return ctx.NewError("no runtime stacktrace associated with the provided value")
			}
		},
	}
}

func fnBuiltinArgv() *object.Foreign {
	return &object.Foreign{
		Name: "argv",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			argv := ctx.GetConfiguration().Argv
			elements := make([]object.Object, len(argv))
			for i, arg := range argv {
				elements[i] = &object.String{Value: arg}
			}

			return &object.List{Elements: elements}
		},
	}
}

func fnBuiltinArgm() *object.Foreign {
	return &object.Foreign{
		Name: "argm",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			argv := ctx.GetConfiguration().Argv
			options, positionals := util.ParseArgs(argv)

			slugOptions := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
			for k, v := range options {
				// We treat everything as strings/bools for raw args
				if len(v) == 1 {
					if v[0] == "true" {
						foreign.PutBool(slugOptions, k, true)
					} else if v[0] == "false" {
						foreign.PutBool(slugOptions, k, false)
					} else {
						foreign.PutString(slugOptions, k, v[0])
					}
				} else {
					var lst []object.Object
					for _, s := range v {
						if s == "true" {
							lst = append(lst, object.TRUE)
						} else if s == "false" {
							lst = append(lst, object.FALSE)
						} else {
							lst = append(lst, &object.String{Value: s})
						}
					}
					foreign.PutList(slugOptions, k, lst)
				}
			}

			slugPos := &object.List{Elements: make([]object.Object, len(positionals))}
			for i, p := range positionals {
				slugPos.Elements[i] = &object.String{Value: p}
			}

			res := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
			res.Put(object.InternSymbol("options"), slugOptions)
			res.Put(object.InternSymbol("positional"), slugPos)
			return res
		},
	}
}

func fnBuiltinCfg() *object.Foreign {
	return &object.Foreign{
		Name: "cfg",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Enforce exactly 2 arguments: key and default
			if len(args) != 2 {
				return ctx.NewError("cfg() requires 2 arguments: cfg(key, default). got=%d", len(args))
			}

			keyObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument to cfg() must be a string key")
			}
			key := keyObj.Value
			defaultValue := args[1]

			// Logic for local vs absolute keys
			if !strings.Contains(key, ".") {
				moduleName := ctx.CurrentEnv().ModuleFqn
				if moduleName != "" {
					key = moduleName + "." + key
				}
			}

			store := ctx.GetConfiguration().Store
			val, found := store.Get(key)

			if !found {
				return defaultValue
			}

			// Coercion Logic:
			// If the config value is a string, but the default is a different type, try to convert.
			if strVal, ok := val.(string); ok {
				switch defaultValue.(type) {
				case *object.Number:
					if d, err := dec64.FromString(strVal); err == nil {
						return &object.Number{Value: d}
					}
				case *object.Boolean:
					if b, err := strconv.ParseBool(strVal); err == nil {
						if b {
							return object.TRUE
						}
						return object.FALSE
					}
				}
			}

			return foreign.NativeToSlugObject(val)
		},
	}
}
