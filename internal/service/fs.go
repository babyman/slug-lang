package service

import (
	"os"
	"slug/internal/kernel"
)

type FsWrite struct {
	Data []byte
	Path string
}

type FsWriteResp struct {
	Bytes int
	Err   error
}

type FsRead struct {
	Path string
}

type FsReadResp struct {
	Data string
	Err  error
}

type Fs struct{}

func (fs *Fs) Behavior(ctx *kernel.ActCtx, msg kernel.Message) {
	switch p := msg.Payload.(type) {
	case FsWrite:
		path := p.Path
		data := p.Data
		err := os.WriteFile(path, data, 0644)
		if err != nil {
			reply(ctx, msg, FsWriteResp{Err: err})
			return
		}
		reply(ctx, msg, FsWriteResp{Bytes: len(data)})
	case FsRead:
		path := p.Path
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
