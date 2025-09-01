package sout

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

type SOutResp struct {
	BytesWritten int
	Err          error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SOutPrintln{}): kernel.RightWrite,
}

type SOut struct {
}

func (s *SOut) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case svc.SOutPrintln:
		str := payload.Str + "\n"
		bytesWritten, err := fmt.Printf(str, payload.Args...)
		svc.Reply(ctx, msg, SOutResp{BytesWritten: bytesWritten, Err: err})
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
}
