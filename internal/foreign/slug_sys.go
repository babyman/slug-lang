package foreign

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"slug/internal/object"
	"time"
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
