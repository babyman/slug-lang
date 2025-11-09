package foreign

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"slug/internal/object"
)

func fnCryptoMd5() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			inputObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("argument to `md5` must be BYTES, got=%s", args[0].Type())
			}

			h := md5.New()
			h.Write(inputObj.Value)
			return &object.Bytes{Value: h.Sum(nil)}
		},
	}
}

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

func fnCryptoSha512() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			inputObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("argument to `sha512` must be BYTES, got=%s", args[0].Type())
			}

			h := sha512.New()
			h.Write(inputObj.Value)
			return &object.Bytes{Value: h.Sum(nil)}
		},
	}
}
