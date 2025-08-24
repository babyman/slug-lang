package service

import (
	"reflect"
	"slug/internal/kernel"
)

var CliOperations = kernel.OpRights{
	reflect.TypeOf(kernel.Boot{}): kernel.RightExec,
}

type Cli struct {
}

func (cli *Cli) Behavior(ctx *kernel.ActCtx, msg kernel.Message) {

}
