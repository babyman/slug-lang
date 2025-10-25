package foreign

import (
	"regexp"
	"slug/internal/object"
)

func fnRegexMatches() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2",
					len(args))
			}

			str, err := unpackString(args[0], "str")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			pattern, err := unpackString(args[1], "pattern")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			matches, err := regexp.MatchString(pattern, str)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			return ctx.NativeBoolToBooleanObject(matches)
		},
	}
}

func fnRegexSplit() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2",
					len(args))
			}

			str, err := unpackString(args[0], "str")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			pattern, err := unpackString(args[1], "pattern")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			splits := re.Split(str, -1)

			elements := make([]object.Object, len(splits))
			for i, split := range splits {
				elements[i] = &object.String{Value: split}
			}

			return &object.List{Elements: elements}
		},
	}
}

func fnRegexFindAll() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2",
					len(args))
			}

			str, err := unpackString(args[0], "str")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			pattern, err := unpackString(args[1], "pattern")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			matches := re.FindAllString(str, -1)

			elements := make([]object.Object, len(matches))
			for i, match := range matches {
				elements[i] = &object.String{Value: match}
			}

			return &object.List{Elements: elements}
		},
	}
}

func fnRegexFindAllGroups() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2",
					len(args))
			}

			str, err := unpackString(args[0], "str")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			pattern, err := unpackString(args[1], "pattern")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			matches := re.FindAllStringSubmatch(str, -1)

			elements := make([]object.Object, len(matches))
			for i, match := range matches {
				subElements := make([]object.Object, len(match))
				for j, submatch := range match {
					subElements[j] = &object.String{Value: submatch}
				}
				elements[i] = &object.List{Elements: subElements}
			}

			return &object.List{Elements: elements}
		},
	}
}

func fnRegexReplaceAll() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 3 {
				return ctx.NewError("wrong number of arguments. got=%d, want=3",
					len(args))
			}

			str, err := unpackString(args[0], "str")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			pattern, err := unpackString(args[1], "pattern")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			replacement, err := unpackString(args[2], "repl")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			updatedString := re.ReplaceAllString(str, replacement)

			return &object.String{Value: updatedString}
		},
	}
}
