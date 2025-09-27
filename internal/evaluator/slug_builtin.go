package evaluator

import (
	"bytes"
	"slug/internal/dec64"
	"slug/internal/object"
	"slug/internal/svc"
	"strings"
)

func fnBuiltinImport() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 {
				return ctx.NewError("import expects at least one argument")
			}

			tempMap := make(map[string]object.Object)

			for i, arg := range args {
				if arg.Type() != object.STRING_OBJ {
					return ctx.NewError("argument %d to import must be a string", i)
				}
				module, err := ctx.LoadModule(strings.Split(arg.Inspect(), "."))
				if err != nil {
					return ctx.NewError(err.Error())
				}
				for name, binding := range module.Env.Bindings {
					if binding.Meta.IsExport {
						// check if this is a FunctionGroup, if it is combine it, do not overwrite
						if fg, ok := binding.Value.(*object.FunctionGroup); ok {
							if existing, exists := tempMap[name]; exists {
								if existingFg, ok := existing.(*object.FunctionGroup); ok {
									for k, v := range fg.Functions {
										existingFg.Functions[k] = v
									}
									continue
								}
							}
						}
						tempMap[name] = binding.Value
					}
				}
			}

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
			svc.SendStdOut(ctx.ActCtx(), out.String())
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
			svc.SendStdOut(ctx.ActCtx(), out.String())
			return ctx.Nil()
		},
	}
}
