package foreign

import (
	"math/rand"
	"slug/internal/dec64"
	"slug/internal/object"
)

var (
	mathRnd *rand.Rand
)

// seed sets the seed for the pseudo-random number generator.
func fnMathRndSeed() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) == 0 {

				mathRnd = nil

			} else {

				// Validate the seed (must be an integer)
				seedArg, ok := args[0].(*object.Number)
				if !ok {
					return ctx.NewError("argument to `seed` must be a NUMBER, got=%s", args[0].Type())
				}

				// Set the random seed
				mathRnd = rand.New(rand.NewSource(seedArg.Value.ToInt64()))
			}

			return ctx.Nil()
		},
	}
}

// random_range generates a random integer between min and max (inclusive).
func fnMathRndRange() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			// Validate min (first argument)
			minArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("argument 1 to `random_range` must be a NUMBER, got=%s", args[0].Type())
			}

			// Validate max (second argument)
			maxArg, ok := args[1].(*object.Number)
			if !ok {
				return ctx.NewError("argument 2 to `random_range` must be a NUMBER, got=%s", args[1].Type())
			}

			// Ensure min <= max
			if minArg.Value.Gte(maxArg.Value) {
				return ctx.NewError("invalid range: min (%v) cannot be greater than max (%v)", minArg.Value, maxArg.Value)
			}

			result := minArg.Value.ToInt64()

			rangeSize := maxArg.Value.ToInt64() - result

			if mathRnd == nil {
				result += rand.Int63n(rangeSize)
			} else {
				result += mathRnd.Int63n(rangeSize)
			}

			return &object.Number{Value: dec64.FromInt64(result)}
		},
	}
}
