package foreign

import (
	"crypto/sha256"
	"slug/internal/object"
)

func fnCryptoSha256() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			inputObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("argument to `sha256` must be BYTES, got=%s", args[0].Type())
			}

			h := sha256.New()
			h.Write(inputObj.Value)
			return &object.Bytes{Value: h.Sum(nil)}
		},
	}
}
