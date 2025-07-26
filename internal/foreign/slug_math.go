package foreign

import (
	"crypto/rand"
	"encoding/binary"
	"slug/internal/dec64"
	"slug/internal/object"
)

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

			var b [8]byte
			_, err := rand.Read(b[:])
			if err != nil {
				return ctx.NewError("failed to generate random number: %v", err)
			}
			randInt := binary.BigEndian.Uint64(b[:]) % uint64(rangeSize)
			result += int64(randInt)

			return &object.Number{Value: dec64.FromInt64(result)}
		},
	}
}
