package modules

import (
	"log/slog"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/util"
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
	reflect.TypeOf(LoadFile{}):   kernel.RightRead,
	reflect.TypeOf(LoadModule{}): kernel.RightRead,
}

type Modules struct {
	Config         util.Configuration
	moduleRegistry map[string]kernel.ActorID
}

func NewModules(config util.Configuration) *Modules {
	return &Modules{
		Config:         config,
		moduleRegistry: make(map[string]kernel.ActorID),
	}
}

func (m *Modules) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case LoadFile:
		worker := FileLoader{
			DebugAST: m.Config.DebugAST,
		}
		workedId, _ := ctx.SpawnChild("mods-fl-wrk", Operations, worker.loadFileHandler)
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
				DebugAST: m.Config.DebugAST,
			}
			workedId, _ := ctx.SpawnChild("module: "+moduleName, Operations, worker.loadModuleHandler)
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
