package eval

import (
	"errors"
	"fmt"
	"slug/internal/evaluator"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
)

func Run(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(svc.EvaluateProgram)

	// Start the environment
	env := object.NewEnvironment()

	// Prepare args list
	objects := make([]object.Object, len(payload.Args))
	for i, arg := range payload.Args {
		objects[i] = &object.String{Value: arg}
	}
	env.Define("args", &object.List{Elements: objects}, false, false)

	env.Path = payload.Path
	env.ModuleFqn = payload.Name
	env.Src = payload.Source

	module := &object.Module{
		Name:    payload.Name,
		Path:    payload.Path,
		Src:     payload.Source,
		Program: payload.Program,
		Env:     env,
	}

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
			svc.Reply(ctx, fwdMsg, svc.EvaluateResult{
				Error: errors.New(evaluated.Inspect()),
			})
		} else {
			svc.Reply(ctx, fwdMsg, svc.EvaluateResult{
				Result: fmt.Sprint(evaluated.Inspect()),
			})
		}
	} else {
		svc.Reply(ctx, fwdMsg, svc.EvaluateResult{})
	}

	// terminate when complete
	return kernel.Terminate{}
}
