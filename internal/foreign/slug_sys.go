package foreign

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"slug/internal/dec64"
	"slug/internal/object"
	"sync"
	"syscall"
	"time"
)

var (
	sysProcsMu sync.RWMutex
	sysProcs   = map[int64]*exec.Cmd{}
)

func fnSysExit() *object.Foreign {
	return &object.Foreign{
		Name: "exit",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			code := args[0]
			if code.Type() != object.NUMBER_OBJ {
				return ctx.NewError("argument must be NUMBER, got=%s", code.Type())
			}

			os.Exit(code.(*object.Number).Value.ToInt())
			return ctx.Nil()
		},
	}
}

func fnSysEnv() *object.Foreign {
	return &object.Foreign{
		Name: "env",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			arg := args[0]

			// Ensure the argument is of type MAP_OBJ
			if arg.Type() != object.STRING_OBJ {
				return ctx.NewError("argument to STRING, got=%s", arg.Type())
			}

			value, ok := os.LookupEnv(arg.(*object.String).Value)

			if ok {
				return &object.String{Value: value}
			}
			return ctx.Nil()
		},
	}
}

func fnSysSetEnv() *object.Foreign {
	return &object.Foreign{
		Name: "setEnv",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2",
					len(args))
			}

			key := args[0]
			value := args[1]

			if key.Type() != object.STRING_OBJ {
				return ctx.NewError("first argument must be STRING, got=%s", key.Type())
			}
			if value.Type() != object.STRING_OBJ {
				return ctx.NewError("second argument must be STRING, got=%s", value.Type())
			}

			err := os.Setenv(key.(*object.String).Value, value.(*object.String).Value)
			if err != nil {
				return ctx.NewError("failed to set environment variable: %v", err)
			}

			return ctx.Nil()
		},
	}
}

func fnSysExec() *object.Foreign {
	return &object.Foreign{
		Name: "exec",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1 or 2",
					len(args))
			}

			cmdArg := args[0]
			if cmdArg.Type() != object.STRING_OBJ {
				return ctx.NewError("first argument must be STRING, got=%s", cmdArg.Type())
			}
			cmdStr := cmdArg.(*object.String).Value

			// Optional timeout in milliseconds (second argument)
			var (
				goCtx  context.Context
				cancel context.CancelFunc
			)

			if len(args) == 2 && args[1] != ctx.Nil() {
				timeoutArg := args[1]
				if timeoutArg.Type() != object.NUMBER_OBJ {
					return ctx.NewError("second argument must be NUMBER (timeout ms) or nil, got=%s", timeoutArg.Type())
				}
				timeoutMs := timeoutArg.(*object.Number).Value.ToInt()
				if timeoutMs > 0 {
					goCtx, cancel = context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
				} else {
					goCtx, cancel = context.WithCancel(context.Background())
				}
			} else {
				// No timeout provided: no deadline (but still have a context)
				goCtx, cancel = context.WithCancel(context.Background())
			}
			defer cancel()

			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				cmd = exec.CommandContext(goCtx, "cmd", "/C", cmdStr)
			} else {
				cmd = exec.CommandContext(goCtx, "sh", "-c", cmdStr)
			}

			var stdoutBuf, stderrBuf bytes.Buffer
			cmd.Stdout = &stdoutBuf
			cmd.Stderr = &stderrBuf

			if err := cmd.Run(); err != nil {
				// If the context timed out, make that explicit in stderr
				if errors.Is(goCtx.Err(), context.DeadlineExceeded) {
					if stderrBuf.Len() > 0 {
						stderrBuf.WriteString("\n")
					}
					stderrBuf.WriteString("command timed out")
				} else {
					// Append the Go error message so caller can inspect it
					if stderrBuf.Len() > 0 {
						stderrBuf.WriteString("\n")
					}
					stderrBuf.WriteString(err.Error())
				}
			}

			stdoutObj := &object.String{Value: stdoutBuf.String()}
			stderrObj := &object.String{Value: stderrBuf.String()}

			return &object.List{Elements: []object.Object{stdoutObj, stderrObj}}
		},
	}
}

func fnSysSpawnProc() *object.Foreign {
	return &object.Foreign{
		Name: "spawnProc",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1",
					len(args))
			}

			cmdArg := args[0]
			if cmdArg.Type() != object.LIST_OBJ {
				return ctx.NewError("argument must be LIST, got=%s", cmdArg.Type())
			}

			cmdList := cmdArg.(*object.List)
			if len(cmdList.Elements) == 0 {
				return ctx.NewError("command list must not be empty")
			}

			cmdArgs := make([]string, len(cmdList.Elements))
			for i, elem := range cmdList.Elements {
				if elem.Type() != object.STRING_OBJ {
					return ctx.NewError("command argument %d must be STRING, got=%s", i, elem.Type())
				}
				cmdArgs[i] = elem.(*object.String).Value
			}

			cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin

			if err := cmd.Start(); err != nil {
				return ctx.NewError("failed to start command: %v", err)
			}

			id := ctx.NextHandleID()
			sysProcsMu.Lock()
			sysProcs[id] = cmd
			sysProcsMu.Unlock()

			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnSysWaitProc() *object.Foreign {
	return &object.Foreign{
		Name: "waitProc",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			timeoutMs, err := unpackNumber(args[1], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			sysProcsMu.RLock()
			e, ok := sysProcs[id]
			sysProcsMu.RUnlock()
			if !ok {
				return ctx.NewError("invalid process ID: %d", id)
			}

			done := make(chan error, 1)
			go func() {
				done <- e.Wait()
			}()

			if timeoutMs > 0 {
				select {
				case err := <-done:
					// finished
					info, _ := exitInfoFromErr(err, e)
					sysProcsMu.Lock()
					delete(sysProcs, id)
					sysProcsMu.Unlock()
					return &object.Number{Value: dec64.FromInt(info)}
				case <-time.After(time.Duration(timeoutMs) * time.Millisecond):
					return &object.Number{Value: dec64.FromInt(-1)}
				}
			}

			// wait forever
			err = <-done
			info, _ := exitInfoFromErr(err, e)
			sysProcsMu.Lock()
			delete(sysProcs, id)
			sysProcsMu.Unlock()
			return &object.Number{Value: dec64.FromInt(info)}
		},
	}
}

func fnSysKillProc() *object.Foreign {
	return &object.Foreign{
		Name: "killProc",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			sysProcsMu.RLock()
			e, ok := sysProcs[id]
			sysProcsMu.RUnlock()
			if !ok {
				return ctx.NewError("invalid process ID: %d", id)
			}

			if e.Process == nil {
				println("process already exited")
				return ctx.NativeBoolToBooleanObject(false)
			}

			// 1. Try to terminate gracefully (SIGTERM/Interrupt)
			_ = e.Process.Signal(os.Interrupt)

			// 2. Poll for a short time to see if it exited
			// We don't call e.Wait() here because the Slug thread is likely already calling it.
			// Instead, we check if the process is still responsive to signals.
			success := false
			for i := 0; i < 10; i++ {
				time.Sleep(50 * time.Millisecond)
				// Signal(0) returns nil if the process is still alive,
				// or an error if it has finished.
				if err := e.Process.Signal(syscall.Signal(0)); err != nil {
					success = true
					break
				}
			}

			if success {
				println("terminated process")
				return ctx.NativeBoolToBooleanObject(true)
			}

			// 3. Still running? Force kill.
			if err := e.Process.Kill(); err != nil {
				println("failed to kill process:", err)
				return ctx.NativeBoolToBooleanObject(false)
			}
			println("killed process")
			return ctx.NativeBoolToBooleanObject(true)
		},
	}
}

func exitInfoFromErr(err error, cmd *exec.Cmd) (int, bool) {
	// Go sets ProcessState even if exit error occurs.
	code := 0
	if cmd.ProcessState != nil {
		code = cmd.ProcessState.ExitCode()
	}
	// If err == nil, ExitCode() is still meaningful.
	_ = err
	return code, false
}
