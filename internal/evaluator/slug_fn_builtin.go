package evaluator

import (
	"bytes"
	"fmt"
	"slug/internal/dec64"
	"slug/internal/object"
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
