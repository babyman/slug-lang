package service

import (
	"os"
	"reflect"
	"slug/internal/kernel"
)

type FsRead struct {
	Path string
}

type FsReadResp struct {
	Data string
	Err  error
}

type FsWrite struct {
	Data []byte
	Path string
}

type FsWriteResp struct {
	Bytes int
	Err   error
}

var FsOperations = kernel.OpRights{
	reflect.TypeOf(FsRead{}):  kernel.RightRead,
	reflect.TypeOf(FsWrite{}): kernel.RightWrite,
}

type Fs struct {
}

func (fs *Fs) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case FsWrite:
		path := payload.Path
		data := payload.Data
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			Reply(ctx, msg, FsWriteResp{Err: err})
			return
		}
		Reply(ctx, msg, FsWriteResp{Bytes: len(data)})
	case FsRead:
		path := payload.Path
		data, err := os.ReadFile(path)
		if err != nil {
			Reply(ctx, msg, FsReadResp{Err: err})
			return
		}
		Reply(ctx, msg, FsReadResp{Data: string(data)})
	default:
		Reply(ctx, msg, kernel.UnknownOperation{})
	}
}
