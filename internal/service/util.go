package service

import "slug/internal/kernel"

func reply(ctx *kernel.ActCtx, req kernel.Message, payload any) {
	if req.Resp != nil {
		req.Resp <- kernel.Message{From: ctx.Self, To: req.From, Payload: payload}
	}
}

func sendStdOut(ctx *kernel.ActCtx, str string, args ...any) {
	stdioID, ok := ctx.K.ActorByName("sout")
	if ok {
		ctx.SendAsync(stdioID, SOutPrintln{
			Str:  str,
			Args: args,
		})
	}
}
