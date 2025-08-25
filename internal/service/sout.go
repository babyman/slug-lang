package service

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
)

type SOutPrintln struct {
	Str  string
	Args []any
}

type SOutResp struct {
	BytesWritten int
	Err          error
}

var SOutOperations = kernel.OpRights{
	reflect.TypeOf(SOutPrintln{}): kernel.RightWrite,
}

type SOut struct {
}

func (s *SOut) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case SOutPrintln:
		str := payload.Str + "\n"
		bytesWritten, err := fmt.Printf(str, payload.Args...)
		reply(ctx, msg, SOutResp{BytesWritten: bytesWritten, Err: err})
	}
}
