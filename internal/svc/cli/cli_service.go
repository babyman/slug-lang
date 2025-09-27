package cli

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(kernel.Boot{}): kernel.RightExec,
}

type Cli struct {
}

func (cli *Cli) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch msg.Payload.(type) {
	case kernel.Boot:
		svc.Reply(ctx, msg, cli.onBoot(ctx))
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}

}
