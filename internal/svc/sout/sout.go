package sout

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

type SOutResp struct {
	BytesWritten int
}

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SOutPrintf{}): kernel.RightWrite,
}

type SOut struct {
}

func (s *SOut) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case svc.SOutPrintf:
		str := payload.Str
		if payload.Args != nil && len(payload.Args) > 0 {
			bytesWritten, _ := fmt.Printf(str, payload.Args...)
			svc.Reply(ctx, msg, SOutResp{BytesWritten: bytesWritten})
		} else {
			bytesWritten, _ := fmt.Print(str)
			svc.Reply(ctx, msg, SOutResp{BytesWritten: bytesWritten})
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
