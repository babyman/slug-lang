package foreign

import (
	"encoding/base64"
	"encoding/hex"
	"slug/internal/object"
)

func fnBytesStrToBytes() *object.Foreign {
	return &object.Foreign{
		Name: "strToBytes",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("strToBytes: expected 1 argument")
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("strToBytes: expected string")
			}
			return &object.Bytes{Value: []byte(strObj.Value)}
		},
	}
}

func fnBytesBytesToStr() *object.Foreign {
	return &object.Foreign{
		Name: "bytesToStr",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("bytesToStr: expected 1 argument")
			}
			bytesObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("bytesToStr: expected Bytes")
			}
			return &object.String{Value: string(bytesObj.Value)}
		},
	}
}

func fnBytesHexStrToBytes() *object.Foreign {
	return &object.Foreign{
		Name: "hexStrToBytes",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("hexStrToBytes: expected 1 argument")
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("hexStrToBytes: expected string")
			}
			bytes, err := hex.DecodeString(strObj.Value)
			if err != nil {
				return ctx.NewError("hexStrToBytes: invalid hex string: %v", err)
			}
			return &object.Bytes{Value: bytes}
		},
	}
}

func fnBytesBytesToHexStr() *object.Foreign {
	return &object.Foreign{
		Name: "bytesToHexStr",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("bytesToHexStr: expected 1 argument")
			}
			bytesObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("bytesToHexStr: expected Bytes")
			}
			return &object.String{Value: hex.EncodeToString(bytesObj.Value)}
		},
	}
}

func fnBytesBase64Encode() *object.Foreign {
	return &object.Foreign{
		Name: "base64Encode",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("base64Encode: expected 1 argument")
			}
			bytesObj, ok := args[0].(*object.Bytes)
			if !ok {
				return ctx.NewError("base64Encode: expected Bytes")
			}
			return &object.String{Value: base64.StdEncoding.EncodeToString(bytesObj.Value)}
		},
	}
}

func fnBytesBase64Decode() *object.Foreign {
	return &object.Foreign{
		Name: "base64Decode",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("base64Decode: expected 1 argument")
			}
			strObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("base64Decode: expected string")
			}
			bytes, err := base64.StdEncoding.DecodeString(strObj.Value)
			if err != nil {
				return ctx.NewError("base64Decode: invalid base64 string: %v", err)
			}
			return &object.Bytes{Value: bytes}
		},
	}
}
