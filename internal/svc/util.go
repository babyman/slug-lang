package svc

import (
	"errors"
	"log/slog"
	"slug/internal/kernel"
)

func Reply(ctx *kernel.ActCtx, req kernel.Message, payload any) {
	if req.Resp != nil {
		req.Resp <- kernel.Message{From: ctx.Self, To: req.From, Payload: payload}
	} else {
		slog.Debug("no response channel found",
			slog.Any("request", req))
	}
}

func BlockingSend(ctx *kernel.ActCtx, actorName string, message any) (kernel.Message, error) {
	id, ok := ctx.K.ActorByName(actorName)
	if ok {
		return ctx.SendSync(id, message)
	} else {
		return kernel.Message{}, errors.New("Actor not found: " + actorName + "")
	}
}

func Send(ctx *kernel.ActCtx, actorName string, message any) {
	id, ok := ctx.K.ActorByName(actorName)
	if ok {
		ctx.SendAsync(id, message)
	}
}

func SendStdOut(ctx *kernel.ActCtx, str string, args ...any) {
	BlockingSend(ctx, SOutService, SOutPrintf{
		Str:  str,
		Args: args,
	})
}

func UnpackFwd(msg kernel.Message) kernel.Message {
	fwdMsg, ok := msg.Payload.(kernel.Message)
	if ok {
		return UnpackFwd(fwdMsg)
	} else {
		return msg
	}
}
