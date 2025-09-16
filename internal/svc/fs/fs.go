package fs

import (
	"os"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
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

var Operations = kernel.OpRights{
	reflect.TypeOf(FsRead{}):  kernel.RightRead,
	reflect.TypeOf(FsWrite{}): kernel.RightWrite,
}

type Fs struct {
}

func (fs *Fs) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case FsWrite:
		path := payload.Path
		data := payload.Data
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			svc.Reply(ctx, msg, FsWriteResp{Err: err})
			return kernel.Continue{}
		}
		svc.Reply(ctx, msg, FsWriteResp{Bytes: len(data)})
	case FsRead:
		workedId, _ := ctx.SpawnChild("file-reader", fs.ReadHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.Reply(ctx, msg, FsReadResp{Err: err})
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func (fs *Fs) ReadHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := msg.Payload.(kernel.Message)
	switch payload := fwdMsg.Payload.(type) {
	case FsRead:
		path := payload.Path
		data, err := os.ReadFile(path)
		if err != nil {
			svc.Reply(ctx, fwdMsg, FsReadResp{Err: err})
		} else {
			svc.Reply(ctx, fwdMsg, FsReadResp{Data: string(data)})
		}
	}
	return kernel.Terminate{}
}
