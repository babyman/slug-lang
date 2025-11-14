package modules

import (
	"log/slog"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"strings"
)

type LoadFile struct {
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
	reflect.TypeOf(kernel.ConfigureSystem{}): kernel.RightExec,
	reflect.TypeOf(LoadFile{}):               kernel.RightRead,
	reflect.TypeOf(LoadModule{}):             kernel.RightRead,
}

type Modules struct {
	debugAST       bool
	moduleRegistry map[string]kernel.ActorID
}

func NewModules() *Modules {
	return &Modules{
		debugAST:       false,
		moduleRegistry: make(map[string]kernel.ActorID),
	}
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case kernel.ConfigureSystem:
		m.debugAST = payload.DebugAST
		svc.Reply(ctx, msg, nil)

	case LoadFile:
		worker := FileLoader{
			DebugAST: m.debugAST,
		}
		workedId, _ := ctx.SpawnChild("mods-fl-wrk", worker.loadFileHandler)
		err := ctx.SendAsync(workedId, msg)
		if err != nil {
			slog.Error("error sending message to file loader",
				slog.Any("error", err.Error()))
		}

	case LoadModule:
		moduleName := strings.Join(payload.PathParts, ".")
		if id, ok := m.moduleRegistry[moduleName]; ok {
			err := ctx.SendAsync(id, msg)
			if err != nil {
				slog.Error("error sending message to existing module loader",
					slog.Any("error", err.Error()))
			}
		} else {
			worker := ModuleLoader{
				DebugAST: m.debugAST,
			}
			workedId, _ := ctx.SpawnChild("mods-load-wrk:"+moduleName, worker.loadModuleHandler)
			m.moduleRegistry[moduleName] = workedId
			err := ctx.SendAsync(workedId, msg)
			if err != nil {
				slog.Error("error sending message to new module worker",
					slog.Any("error", err.Error()))
			}
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
