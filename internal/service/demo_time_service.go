package service

import (
	"slug/internal/kernel"
	"time"
)

// ===== Built-in Services =====

type TsNow struct {
}

type TsNowResp struct {
	Nanos int64
}

type TsSleep struct {
	Ms int
}

func TimeServiceBehavior(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case TsNow:
		ns := time.Now().UnixNano()
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "now.ok", Payload: TsNowResp{Nanos: ns}}
		}
	case TsSleep:
		ms := payload.Ms
		t := time.Duration(ms) * time.Millisecond
		time.Sleep(t)
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "sleep.ok"}
		}
	default:
		if msg.Resp != nil {
			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: map[string]any{"error": "unknown op"}}
		}
	}
}
