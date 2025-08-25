package service

import "slug/internal/kernel"

func reply(ctx *kernel.ActCtx, req kernel.Message, payload any) {
	if req.Resp != nil {
		req.Resp <- kernel.Message{From: ctx.Self, To: req.From, Payload: payload}
	}
}

func sendStdOut(ctx *kernel.ActCtx, args ...any) {
	stdioID, ok := ctx.K.ActorByName("sout")
	if ok {
		ctx.SendSync(stdioID, SOutPrintln{
			Str:  "Executing %s with args %v",
			Args: args,
		})
	}
}
