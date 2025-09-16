package modules

import (
	"reflect"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
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
	moduleRegistry map[string]*object.Module
}

func NewModules() *Modules {
	return &Modules{
		moduleRegistry: make(map[string]*object.Module),
	}
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case EvaluateFile:
		return m.onEvaluateFile(ctx, msg, payload)
	case LoadModule:
		println(">>>>>>>>>> LoadModule")

		return m.onLoadModule(ctx, msg, payload)
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}

func (m *Modules) ModuleHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case EvaluateFile:
		return m.onEvaluateFile(ctx, msg, payload)
	case LoadModule:
		println(">>>>>>>>>> LoadModule")
		return m.onLoadModule(ctx, msg, payload)
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
