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

type WriteBytes struct {
	Data []byte
	Path string
}

type WriteResp struct {
	Bytes int
	Err   error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(Read{}):       kernel.RightRead,
	reflect.TypeOf(WriteBytes{}): kernel.RightWrite,
}

type Fs struct {
}

func (fs *Fs) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case Read:
		worker := FsWorker{}
		workedId, _ := ctx.SpawnChild("file-reader", Operations, worker.readHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.Reply(ctx, msg, ReadResp{Err: err})
		}
	case WriteBytes:
		worker := FsWorker{}
		workedId, _ := ctx.SpawnChild("file-write-bytes", Operations, worker.writeHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.Reply(ctx, msg, WriteResp{Err: err})
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

type FsWorker struct {
}

func (fs *FsWorker) readHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
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

func (fs *FsWorker) writeHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	switch payload := fwdMsg.Payload.(type) {
	case WriteBytes:
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
