package service

import (
	"errors"
	"slug/internal/kernel"
	"slug/internal/logger"
)

func Reply(ctx *kernel.ActCtx, req kernel.Message, payload any) {
	if req.Resp != nil {
		req.Resp <- kernel.Message{From: ctx.Self, To: req.From, Payload: payload}
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
	BlockingSend(ctx, "sout", SOutPrintln{
		Str:  str,
		Args: args,
	})
}

func sendLogf(ctx *kernel.ActCtx, level logger.Level, str string, args ...any) {
	Send(ctx, "log", LogfMessage{
		level:   level,
		Message: str,
		Args:    args,
	})
}

func sendLog(ctx *kernel.ActCtx, level logger.Level, str string) {
	Send(ctx, "log", LogMessage{
		level:   level,
		Message: str,
	})
}

func SendDebugf(ctx *kernel.ActCtx, str string, args ...any) {
	sendLogf(ctx, logger.DEBUG, str, args...)
}

func SendInfof(ctx *kernel.ActCtx, str string, args ...any) {
	sendLogf(ctx, logger.INFO, str, args...)
}

func SendWarnf(ctx *kernel.ActCtx, str string, args ...any) {
	sendLogf(ctx, logger.WARN, str, args...)
}

func SendErrorf(ctx *kernel.ActCtx, str string, args ...any) {
	sendLogf(ctx, logger.ERROR, str, args...)
}

func SendFatalf(ctx *kernel.ActCtx, str string, args ...any) {
	sendLogf(ctx, logger.FATAL, str, args...)
}

func SendDebug(ctx *kernel.ActCtx, str string) {
	sendLog(ctx, logger.DEBUG, str)
}

func SendInfo(ctx *kernel.ActCtx, str string) {
	sendLog(ctx, logger.INFO, str)
}

func SendWarn(ctx *kernel.ActCtx, str string) {
	sendLog(ctx, logger.WARN, str)
}

func SendError(ctx *kernel.ActCtx, str string) {
	sendLog(ctx, logger.ERROR, str)
}

func SendFatal(ctx *kernel.ActCtx, str string) {
	sendLog(ctx, logger.FATAL, str)
}
