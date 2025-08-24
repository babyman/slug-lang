package service

import "slug/internal/kernel"

func reply(ctx *kernel.ActCtx, req kernel.Message, payload any) {
	if req.Resp != nil {
		req.Resp <- kernel.Message{From: ctx.Self.Id, To: req.From, Payload: payload}
	}
}
