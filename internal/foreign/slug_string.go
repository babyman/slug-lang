package foreign

import (
	"slug/internal/dec64"
	"slug/internal/object"
	"strings"
	"unicode"
	"unicode/utf8"
)

func fnStringTrim() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			trimmed := strings.TrimFunc(arg.Value, unicode.IsSpace)
			return &object.String{Value: trimmed}
		default:
			return ctx.NewError("argument to `trim` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringIndexOf() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) < 2 {
			return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
		}
		if args[0].Type() != args[1].Type() {
			return ctx.NewError("arguments to `indexOf` must be the same type, got %s and %s", args[0].Type(), args[1].Type())
		}

		switch arg := args[0].(type) {
		case *object.String:
			hay := arg.Value
			needle := args[1].(*object.String).Value

			// Optional start index is in runes, not bytes
			startRunes := 0
			if len(args) > 2 && args[2].Type() == object.NUMBER_OBJ {
				startRunes = args[2].(*object.Number).Value.ToInt()
				if startRunes < 0 {
					startRunes = 0
				}
			}

			// Convert rune start to byte offset
			byteStart := 0
			if startRunes > 0 {
				for i := 0; i < startRunes && byteStart < len(hay); i++ {
					_, sz := utf8.DecodeRuneInString(hay[byteStart:])
					byteStart += sz
				}
			}

			byteIdx := strings.Index(hay[byteStart:], needle)
			if byteIdx < 0 {
				return &object.Number{Value: dec64.FromInt(-1)}
			}

			// Convert byte index back to rune index
			totalByteIdx := byteStart + byteIdx
			runeIdx := utf8.RuneCountInString(hay[:totalByteIdx])
			return &object.Number{Value: dec64.FromInt(runeIdx)}
		default:
			return ctx.NewError("argument to `indexOf` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToUpper() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToUpper(arg.Value)}
		default:
			return ctx.NewError("argument to `toUpper` not supported, got %s", args[0].Type())
		}
	},
	}
}

func fnStringToLower() *object.Foreign {
	return &object.Foreign{Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
		if len(args) != 1 {
			return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
		}

		switch arg := args[0].(type) {
		case *object.String:
			return &object.String{Value: strings.ToLower(arg.Value)}
		default:
			return ctx.NewError("argument to `toLower` not supported, got %s", args[0].Type())
		}
	},
	}
}
