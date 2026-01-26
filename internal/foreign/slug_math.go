package foreign

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"slug/internal/dec64"
	"slug/internal/object"
)

func pow10i64(n int8) int64 {
	r := int64(1)
	for i := int8(0); i < n; i++ {
		r *= 10
	}
	return r
}

// floor returns the greatest integer <= n, as a NUMBER with exponent 0.
func fnMathFloor() *object.Foreign {
	return &object.Foreign{
		Name: "floor",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			nArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("argument 1 to `floor` must be a NUMBER, got=%s", args[0].Type())
			}

			v := nArg.Value
			if v.IsNaN() {
				return &object.Number{Value: dec64.NAN}
			}

			exp := v.Exponent()
			if exp >= 0 {
				// Already an integer in dec64's representation.
				return &object.Number{Value: v}
			}

			coef := v.Coefficient()
			div := pow10i64(int8(-exp))

			q := coef / div
			r := coef % div

			// For negatives with a fractional part, floor goes "more negative".
			if r != 0 && coef < 0 {
				q--
			}

			return &object.Number{Value: dec64.New(q, 0)}
		},
	}
}

// ceil returns the least integer >= n, as a NUMBER with exponent 0.
func fnMathCeil() *object.Foreign {
	return &object.Foreign{
		Name: "ceil",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			nArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("argument 1 to `ceil` must be a NUMBER, got=%s", args[0].Type())
			}

			v := nArg.Value
			if v.IsNaN() {
				return &object.Number{Value: dec64.NAN}
			}

			exp := v.Exponent()
			if exp >= 0 {
				return &object.Number{Value: v}
			}

			coef := v.Coefficient()
			div := pow10i64(int8(-exp))

			q := coef / div
			r := coef % div

			// For positives with a fractional part, ceil goes "more positive".
			if r != 0 && coef > 0 {
				q++
			}

			return &object.Number{Value: dec64.New(q, 0)}
		},
	}
}

// sqrt returns the square root of n.
// If n < 0, returns NaN.
func fnMathSqrt() *object.Foreign {
	return &object.Foreign{
		Name: "sqrt",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			nArg, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("argument 1 to `sqrt` must be a NUMBER, got=%s", args[0].Type())
			}

			v := nArg.Value
			if v.IsNaN() {
				return &object.Number{Value: dec64.NAN}
			}

			// Negative check (dec64 has no Inf; treat negatives as invalid for sqrt)
			if v.Coefficient() < 0 {
				return &object.Number{Value: dec64.NAN}
			}

			f := v.ToFloat64()
			if f < 0 {
				return &object.Number{Value: dec64.NAN}
			}

			return &object.Number{Value: dec64.FromFloat64(math.Sqrt(f))}
		},
	}
}

// random_range generates a random integer between min and max (inclusive).
func fnMathRndRange() *object.Foreign {
	return &object.Foreign{
		Name: "rndRange",
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
