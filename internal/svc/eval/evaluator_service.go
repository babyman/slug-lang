package eval

import (
	"fmt"
	"reflect"
	"slug/internal/ast"
	"slug/internal/evaluator"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
)

type EvaluateProgram struct {
	Source  string
	Args    []string
	Program *ast.Program
}

var Operations = kernel.OpRights{
	reflect.TypeOf(EvaluateProgram{}): kernel.RightExec,
}

type EvaluatorService struct {
	DebugAST       bool
	SystemRootPath string
}

func (m *EvaluatorService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case kernel.ConfigureSystem:
		m.DebugAST = payload.DebugAST
		m.SystemRootPath = payload.SystemRootPath
		svc.Reply(ctx, msg, nil)

	case EvaluateProgram:

		evaluator.DebugAST = m.DebugAST
		evaluator.RootPath = m.SystemRootPath

		module := &object.Module{Name: "main.slug", Env: nil}
		module.Path = "main.slug"
		module.Src = payload.Source
		module.Program = payload.Program

		// Start the environment
		env := object.NewEnvironment()

		// Prepare args list
		objects := make([]object.Object, len(payload.Args))
		for i, arg := range payload.Args {
			objects[i] = &object.String{Value: arg}
		}
		env.Define("args", &object.List{Elements: objects}, false, false)

		env.Path = module.Path
		env.ModuleFqn = module.Name
		env.Src = module.Src
		module.Env = env

		e := evaluator.Evaluator{
			Actor: evaluator.CreateMainThreadMailbox(),
			Ctx:   ctx,
		}
		e.PushEnv(env)
		defer e.PopEnv()

		svc.SendInfo(ctx, " ---- begin ----")
		defer svc.SendInfo(ctx, " ---- done ----")
		// Evaluate the program within the provided environment
		evaluated := e.Eval(module.Program)
		if evaluated != nil && evaluated.Type() != object.NIL_OBJ {
			if evaluated.Type() == object.ERROR_OBJ {
				//return fmt.Errorf(evaluated.Inspect())
				svc.Reply(ctx, msg, fmt.Sprint(evaluated.Inspect()))
			} else {
				svc.Reply(ctx, msg, fmt.Sprint(evaluated.Inspect()))
				//fmt.Println(evaluated.Inspect())
			}
		} else {
			svc.Reply(ctx, msg, nil)
		}
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
