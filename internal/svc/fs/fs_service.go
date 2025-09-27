package fs

import (
	"os"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

type Read struct {
	Path string
}

type ReadResp struct {
	Data string
	Err  error
}

type Write struct {
	Data []byte
	Path string
}

type WriteResp struct {
	Bytes int
	Err   error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(Read{}):  kernel.RightRead,
	reflect.TypeOf(Write{}): kernel.RightWrite,
}

type Fs struct {
}

func (fs *Fs) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case Read:
		workedId, _ := ctx.SpawnChild("file-reader", fs.readHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.Reply(ctx, msg, ReadResp{Err: err})
		}
	case Write:
		workedId, _ := ctx.SpawnChild("file-writer", fs.writeHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.Reply(ctx, msg, WriteResp{Err: err})
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func (fs *Fs) readHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case Read:
		path := payload.Path
		data, err := os.ReadFile(path)
		if err != nil {
			svc.Reply(ctx, fwdMsg, ReadResp{Err: err})
		} else {
			svc.Reply(ctx, fwdMsg, ReadResp{Data: string(data)})
		}
	}
	return kernel.Terminate{}
}

func (fs *Fs) writeHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case Write:
		path := payload.Path
		data := payload.Data
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			svc.Reply(ctx, msg, WriteResp{Err: err})
		} else {
			svc.Reply(ctx, msg, WriteResp{Bytes: len(data)})
		}
	}
	return kernel.Terminate{}
}
