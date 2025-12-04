package eval

import (
	"log/slog"
	"slug/internal/ast"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/util"
)

type SlugSandboxActor struct {
	Config         util.Configuration
	AllowedImports []string
	Src            string
	Program        *ast.Program
}

func NewSlugSandboxActor(c util.Configuration, src string, program *ast.Program, imports []string) *SlugSandboxActor {
	return &SlugSandboxActor{
		Config:         c,
		AllowedImports: imports,
		Src:            src,
		Program:        program,
	}
}

func (s *SlugSandboxActor) Run(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {

	case SlugActorMessage:

		// Start the environment
		env := object.NewEnvironment()
		env.Src = s.Src
		env.Path = "<anon>"
		env.ModuleFqn = "<anon>"

		// Handle message binding based on type
		switch payloadMessage := payload.Msg.(type) {
		case *object.Map:
			// Bind map keys to variables
			for _, pair := range payloadMessage.Pairs {
				env.Define(pair.Key.Inspect(), pair.Value, false, true)
			}
		case *object.List:
			// Bind list to args
			env.Define(ProgramArgs, payloadMessage, false, true)
		default:
			// Wrap other types in a list and bind to args
			env.Define(ProgramArgs, &object.List{Elements: []object.Object{payloadMessage}}, false, true)
		}

		e := Evaluator{
			Config:         s.Config,
			Sandbox:        true,
			AllowedImports: s.AllowedImports,
			Ctx:            ctx,
		}

		slog.Info(" ---- begin ----")
		defer slog.Info(" ---- done ----")

		// Evaluate the program within the provided environment
		e.PushEnv(env)
		v := e.Eval(s.Program)
		e.PopEnv(v)

		ctx.SendAsync(msg.From, SlugActorMessage{Msg: v})
		return kernel.Continue{}

	default:
		return kernel.Continue{}
	}
}
