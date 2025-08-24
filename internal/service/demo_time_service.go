package service

import (
	"reflect"
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

var TsOperations = kernel.OpRights{
	reflect.TypeOf(TsNow{}):   kernel.RightRead,
	reflect.TypeOf(TsSleep{}): kernel.RightExec,
}

func TimeServiceBehavior(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case TsNow:
		ns := time.Now().UnixNano()
		reply(ctx, msg, TsNowResp{Nanos: ns})
	case TsSleep:
		ms := payload.Ms
		t := time.Duration(ms) * time.Millisecond
		time.Sleep(t)
		reply(ctx, msg, nil)
	default:
		reply(ctx, msg, kernel.UnknownOperation{})
	}
}
