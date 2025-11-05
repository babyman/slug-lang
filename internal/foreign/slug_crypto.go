package foreign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"slug/internal/object"
)

func fnCryptoSha256() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			strObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `sha256` must be STRING, got=%s", args[0].Type())
			}

			hash := sha256.Sum256([]byte(strObj.Value))
			return &object.String{Value: hex.EncodeToString(hash[:])}
		},
	}
}

func fnCryptoHmacSha256() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			messageObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("first argument to `hmacSha256` must be STRING, got=%s", args[1].Type())
			}

			secretObj, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to `hmacSha256` must be STRING, got=%s", args[0].Type())
			}

			h := hmac.New(sha256.New, []byte(secretObj.Value))
			h.Write([]byte(messageObj.Value))
			return &object.String{Value: hex.EncodeToString(h.Sum(nil))}
		},
	}
}
