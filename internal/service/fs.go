package service

import (
	"errors"
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
			if msg.Resp != nil {
				msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: FsWriteResp{Err: err}}
			}
			return
		}
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "write.ok", Payload: FsWriteResp{Bytes: len(data)}}
		}
	case FsRead:
		path := p.Path
		data, err := os.ReadFile(path)
		if err != nil {
			if msg.Resp != nil {
				msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: FsReadResp{Err: err}}
			}
			return
		}
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "read.ok", Payload: FsReadResp{Data: string(data)}}
		}
	default:
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: FsReadResp{Err: errors.New("unknown op")}}
		}
	}
}
