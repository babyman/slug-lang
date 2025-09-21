package eval

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.EvaluateProgram{}): kernel.RightExec,
}

type EvaluatorService struct {
}

func (m *EvaluatorService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case svc.EvaluateProgram:
		workedId, _ := ctx.SpawnChild("run:"+payload.Name, Run)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.SendError(ctx, err.Error())
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
