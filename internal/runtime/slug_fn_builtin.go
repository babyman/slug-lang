package runtime

import (
	"bytes"
	"fmt"
	"slug/internal/dec64"
	"slug/internal/foreign"
	"slug/internal/object"
	"strings"
	"unicode/utf8"
)

func fnBuiltinImport() *object.Foreign {
	return &object.Foreign{
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
					if binding.Meta.IsExport {
						// Handle function groups (polymorphism)
						if fg, ok := binding.Value.(*object.FunctionGroup); ok {
							if existing, exists := tempMap[name]; exists {
								if existingFg, ok := existing.(*object.FunctionGroup); ok {
									for sig, fn := range fg.Functions {
										existingFg.Functions[sig] = fn
									}
									continue
								}
							}
						}
						tempMap[name] = binding.Value
					}
				}
			}

			// Return a map containing the imported members
			m := &object.Map{
				Tags: map[string]object.List{
					object.IMPORT_TAG: {},
				},
			}
			for k, v := range tempMap {
				m.Put(&object.String{Value: k}, v)
			}
			return m
		},
	}
}

func fnBuiltinLen() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			var out bytes.Buffer
			for i, arg := range args {
				out.WriteString(arg.Inspect())
				if i < len(args)-1 {
					out.WriteString(" ")
				}
			}
			fmt.Print(out.String())
			if len(args) > 0 {
				return args[0]
			}
			return ctx.Nil()
		},
	}
}

func fnBuiltinPrintLn() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			var out bytes.Buffer
			for i, arg := range args {
				out.WriteString(arg.Inspect())
				if i < len(args)-1 {
					out.WriteString(" ")
				} else {
					out.WriteString("\n")
				}
			}
			fmt.Print(out.String())
			if len(args) > 0 {
				return args[0]
			}
			return ctx.Nil()
		},
	}
}

func fnBuiltinStacktrace() *object.Foreign {
	return &object.Foreign{
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
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			argv := ctx.GetConfiguration().Argv
			positionals := &object.List{Elements: []object.Object{}}
			options := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}

			parsingOptions := true
			i := 0
			for i < len(argv) {
				arg := argv[i]

				if !parsingOptions {
					positionals.Elements = append(positionals.Elements, &object.String{Value: arg})
					i++
					continue
				}

				if arg == "--" {
					parsingOptions = false
					i++
					continue
				}

				if strings.HasPrefix(arg, "--") {
					// Long option
					name := arg[2:]

					if idx := strings.IndexByte(name, '='); idx != -1 {
						foreign.PutString(options, name, name[idx+1:])
					} else {
						resolvedName := name
						// If next arg doesn't start with '-', it's a value
						if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
							foreign.PutString(options, resolvedName, argv[i+1])
							i++
						} else {
							foreign.PutBool(options, resolvedName, true)
						}
					}
					i++
				} else if len(arg) > 1 && arg[0] == '-' {
					// Short options
					key := arg[1:]
					if len(key) == 1 {
						resolved := key
						// If exactly one char and next arg isn't an option, it's a value
						if i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
							foreign.PutString(options, resolved, argv[i+1])
							i += 2
						} else {
							foreign.PutBool(options, resolved, true)
							i++
						}
					} else {
						// Multiple chars: treat all as boolean flags
						for _, char := range key {
							foreign.PutBool(options, string(char), true)
						}
						i++
					}
				} else {
					positionals.Elements = append(positionals.Elements, &object.String{Value: arg})
					i++
				}
			}

			res := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
			res.Put(&object.String{Value: "options"}, options)
			res.Put(&object.String{Value: "positional"}, positionals)
			return res
		},
	}
}

func fnBuiltinCfg() *object.Foreign {
	return &object.Foreign{
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
				if moduleName != "" && moduleName != "<main>" {
					key = moduleName + "." + key
				}
			}

			store := ctx.GetConfiguration().Store
			val, found := store.Get(key)

			if !found {
				// Key not in TOML, use the required default
				return defaultValue
			}

			// Convert Go type from TOML to Slug Object
			return nativeToSlugObject(val)
		},
	}
}

func nativeToSlugObject(val interface{}) object.Object {
	switch v := val.(type) {
	case string:
		return &object.String{Value: v}
	case int64:
		return &object.Number{Value: dec64.FromInt64(v)}
	case float64:
		return &object.Number{Value: dec64.FromFloat64(v)}
	case bool:
		if v {
			return object.TRUE
		}
		return object.FALSE
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, item := range v {
			elements[i] = nativeToSlugObject(item)
		}
		return &object.List{Elements: elements}
	default:
		return object.NIL
	}
}
