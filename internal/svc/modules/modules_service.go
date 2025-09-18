package modules

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"strings"
)

type EvaluateFile struct {
	Path string
	Args []string
}

type LoadModule struct {
	RootPath  string
	PathParts []string
	DebugAST  bool
}

type LoadModuleResult struct {
	Module *object.Module
	Error  error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(EvaluateFile{}): kernel.RightExec,
	reflect.TypeOf(LoadModule{}):   kernel.RightRead,
}

type Modules struct {
	moduleRegistry map[string]kernel.ActorID
}

func NewModules() *Modules {
	return &Modules{
		moduleRegistry: make(map[string]kernel.ActorID),
	}
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case EvaluateFile:
		workedId, _ := ctx.SpawnChild("mods-eval-wrk", m.evaluateFileHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			svc.SendError(ctx, err.Error())
		}
	case LoadModule:
		moduleName := strings.Join(payload.PathParts, ".")
		if id, ok := m.moduleRegistry[moduleName]; ok {
			err := ctx.SendAsync(id, msg)
			if err != nil {
				svc.SendError(ctx, err.Error())
			}
		} else {
			loader := ModuleLoader{}
			workedId, _ := ctx.SpawnChild("mods-load-wrk:"+moduleName, loader.loadModuleHandler)
			m.moduleRegistry[moduleName] = workedId
			err := ctx.SendAsync(workedId, msg)
			if err != nil {
				svc.SendError(ctx, err.Error())
			}
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func (m *Modules) evaluateFileHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(EvaluateFile)
	onEvaluateFile(ctx, fwdMsg, payload)
	return kernel.Terminate{}
}
