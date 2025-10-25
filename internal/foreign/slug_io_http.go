package foreign

import (
	"io"
	"net/http"
	"slug/internal/dec64"
	"slug/internal/object"
	"strings"
)

func fnIoHttpRequest() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 4 {
				return ctx.NewError("wrong number of arguments to `request`, got=%d, want=4", len(args))
			}

			method, err := unpackString(args[0], "method")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			url, err := unpackString(args[1], "url")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			body, err := unpackString(args[2], "body")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			mapObj, ok := args[3].(*object.Map)
			if !ok {
				return ctx.NewError("argument to `headers` must be a MAP, got=%s", args[3].Type())
			}

			client := &http.Client{}
			req, err := http.NewRequest(method, url, strings.NewReader(body))
			if err != nil {
				return ctx.NewError(err.Error())
			}

			for _, v := range mapObj.Pairs {
				req.Header.Set(v.Key.Inspect(), v.Value.Inspect())
			}

			resp, err := client.Do(req)
			defer resp.Body.Close()
			if err != nil {
				return ctx.NewError(err.Error())
			}

			bytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return ctx.NewError(err.Error())
			}

			return &object.List{
				Elements: []object.Object{
					&object.Number{Value: dec64.FromInt64(int64(resp.StatusCode))},
					&object.String{Value: string(bytes)},
				},
			}
		},
	}
}
