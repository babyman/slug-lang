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

func (fs *Fs) Behavior(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case FsWrite:
		path := payload.Path
		data := payload.Data
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			reply(ctx, msg, FsWriteResp{Err: err})
			return
		}
		reply(ctx, msg, FsWriteResp{Bytes: len(data)})
	case FsRead:
		path := payload.Path
		data, err := os.ReadFile(path)
		if err != nil {
			reply(ctx, msg, FsReadResp{Err: err})
			return
		}
		reply(ctx, msg, FsReadResp{Data: string(data)})
	default:
		reply(ctx, msg, kernel.UnknownOperation{})
	}
}
