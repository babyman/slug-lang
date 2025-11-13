package eval

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/pprof"
	"slug/internal/evaluator"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
)

func Run(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	payload, _ := fwdMsg.Payload.(svc.EvaluateProgram)

	evaluated := evaluateMessagePayload(ctx, payload)
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

func evaluateMessagePayload(ctx *kernel.ActCtx, payload svc.EvaluateProgram) object.Object {

	// Optional profiling via env var: SLUG_CPU_PROFILE=<path>
	profPath := os.Getenv("SLUG_CPU_PROFILE")
	var profFile *os.File
	if profPath != "" {
		var err error
		profFile, err = os.Create(profPath)
		if err != nil {
			fmt.Printf("could not create CPU profile %q: %v\n", profPath, err)
		} else if err := pprof.StartCPUProfile(profFile); err != nil {
			fmt.Printf("could not start CPU profile: %v\n", err)
			_ = profFile.Close()
			profFile = nil
		} else {
			defer func() {
				pprof.StopCPUProfile()
				_ = profFile.Close()
			}()
		}
	}

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

	slog.Info(" ---- begin ----")
	defer slog.Info(" ---- done ----")

	// Evaluate the program within the provided environment
	return e.Eval(module.Program)
}
