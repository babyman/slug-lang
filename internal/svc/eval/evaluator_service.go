package eval

import (
	"log/slog"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
	"slug/internal/util"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.EvaluateProgram{}):  kernel.RightExec,
	reflect.TypeOf(svc.SlugActorMessage{}): kernel.RightExec,
	reflect.TypeOf(SlugFunctionDone{}):     kernel.RightExec,
}

type EvaluatorService struct {
	Config util.Configuration
}

func (m *EvaluatorService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case svc.EvaluateProgram:
		worker := SlugProgramActor{
			Config:  m.Config,
			Mailbox: make(chan svc.SlugActorMessage),
		}
		workedId, _ := ctx.SpawnChild("program: "+payload.Name, Operations, worker.Run)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error sending message",
				slog.Any("pid", workedId),
				slog.Any("error", err.Error()))
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
