package eval

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/pprof"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/util"
	"time"
)

const (
	ProgramArgs = "args"
)

type SlugProgramActor struct {
	Config  util.Configuration
	Mailbox chan svc.SlugActorMessage
	// internal state
	started bool
}

func (r *SlugProgramActor) WaitForMessage(timeout int64) (any, bool) {
	// Semantics:
	//  - timeout < 0 => block forever
	//  - timeout == 0 => poll (non-blocking)
	//  - timeout > 0 => timeout milliseconds
	if timeout < 0 {
		msg := <-r.Mailbox
		return msg, true
	}

	if timeout == 0 {
		select {
		case msg := <-r.Mailbox:
			return msg, true
		default:
			return svc.SlugActorMessage{}, false
		}
	}

	select {
	case msg := <-r.Mailbox:
		return msg, true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return svc.SlugActorMessage{}, false
	}
}

func (r *SlugProgramActor) Run(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)

	switch payload := fwdMsg.Payload.(type) {
	case svc.EvaluateProgram:
		// avoid starting the same code multiple times
		if r.started {
			return kernel.Continue{}
		}
		r.started = true

		// Run the code in its own goroutine so this actor can keep
		// handling SlugActorMessage inputs used by WaitForMessage.
		go func() {

			evaluated := r.evaluateMessagePayload(ctx, payload)
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
		}()

		return kernel.Continue{}

	case svc.SlugActorMessage:
		slog.Warn("received actor message", slog.Any("payload", payload.Msg))
		r.Mailbox <- payload
		return kernel.Continue{}

	default:
		return kernel.Continue{}
	}
}

func (r *SlugProgramActor) evaluateMessagePayload(ctx *kernel.ActCtx, payload svc.EvaluateProgram) object.Object {

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
	env.Define(ProgramArgs, &object.List{Elements: objects}, false, false)

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

	e := Evaluator{
		Config:       r.Config,
		SlugReceiver: r,
		Ctx:          ctx,
	}

	slog.Info(" ---- begin ----")
	defer slog.Info(" ---- done ----")

	// Evaluate the program within the provided environment
	e.PushEnv(env)
	result := e.Eval(module.Program)
	e.PopEnv(result)

	return result
}
