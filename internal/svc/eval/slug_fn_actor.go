package eval

import (
	"bytes"
	"fmt"
	"log/slog"
	"slug/internal/dec64"
	"slug/internal/foreign"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
)

func fnActorSelf() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			return &object.Number{Value: dec64.FromInt64(int64(ctx.ActCtx().Self))}
		},
	}
}

func fnActorSpawn() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want>=1", len(args))
			}

			objects := args[1:]
			fun, ok := foreign.ToFunctionArgument(args[0], objects)
			if !ok {
				return ctx.NewError("first argument to `spawn` must be FUNCTION, got=%s", args[0].Type())
			}

			actor := NewSlugFunctionActor(ctx.GetConfiguration(), fun)
			pid, err := ctx.ActCtx().SpawnChild("<anon>", Operations, actor.Run)
			if err != nil {
				return ctx.NewError("failed to spawn actor: %v", err)
			}

			ctx.ActCtx().SendAsync(pid, SlugStart{
				Args: objects,
			})

			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}

func fnActorSpawnSrc() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 1 && len(args) > 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1 or 2", len(args))
			}

			// Expect a slug string value as per `foreign spawnSrc = fn(@str src)`
			srcObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `spawnSrc` must be STRING, got=%s", args[0].Type())
			}

			var allowedImports []string
			if len(args) == 2 {
				importList, ok := args[1].(*object.List)
				if !ok {
					return ctx.NewError("second argument to `spawnSrc` must be ARRAY, got=%s", args[1].Type())
				}
				for _, imp := range importList.Elements {
					if strObj, ok := imp.(*object.String); ok {
						allowedImports = append(allowedImports, strObj.Value)
					} else {
						return ctx.NewError("import list must contain only strings, got=%s", imp.Type())
					}
				}
			}

			src := srcObj.Value

			lexId, _ := ctx.ActCtx().K.ActorByName(svc.LexerService)
			parseId, _ := ctx.ActCtx().K.ActorByName(svc.ParserService)

			lex, err := ctx.ActCtx().SendSync(lexId, lexer.LexString{Sourcecode: src})
			if err != nil {
				slog.Error("Failed to lex src",
					slog.Any("error", err))
				return ctx.NewError("failed to lex source: %s", err.Error())
			}

			tokens := lex.Payload.(lexer.LexedTokens).Tokens
			slog.Debug("Lexed module",
				slog.Any("tokens", tokens))

			parse, err := ctx.ActCtx().SendSync(parseId, parser.ParseTokens{
				Sourcecode: src,
				Path:       "src-actor",
				Tokens:     tokens,
			})
			if err != nil {
				slog.Error("Failed to parse source",
					slog.Any("error", err))
				return ctx.NewError("failed to parse source: %s", err.Error())
			}

			ast := parse.Payload.(parser.ParsedAst).Program
			errors := parse.Payload.(parser.ParsedAst).Errors
			if len(errors) > 0 {
				var out bytes.Buffer
				for _, msg := range errors {
					out.WriteString(fmt.Sprintf("\t%s\n", msg))
				}
				return ctx.NewError("parser errors: %s", out.String())
			}

			actor := NewSlugSandboxActor(ctx.GetConfiguration(), src, ast, allowedImports)
			pid, err := ctx.ActCtx().SpawnChild("<src-anon>", Operations, actor.Run)
			if err != nil {
				return ctx.NewError("failed to spawn actor: %v", err)
			}

			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}

func fnActorSend() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			pidObj, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to `send` must be NUMBER, got=%s", args[0].Type())
			}

			pid := kernel.ActorID(pidObj.Value.ToInt64())
			msg := SlugActorMessage{
				Msg: args[1],
			}

			err := ctx.ActCtx().SendAsync(pid, msg)

			if err != nil {
				return ctx.NewError("failed to send message: %v", err)
			}

			return pidObj
		},
	}
}

func fnActorReceive() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) > 1 {
				return ctx.NewError("receive expects zero or one timeout argument")
			}
			var timeout int64 // No timeout by default
			if len(args) == 1 {
				if to, ok := args[0].(*object.Number); ok {
					timeout = to.Value.ToInt64()
				} else {
					return ctx.NewError("timeout argument must be an integer")
				}
			}

			slog.Warn("ACT: Waiting for message",
				slog.Any("actor-pid", ctx.ActCtx().Self))

			message, b := ctx.WaitForMessage(timeout)
			if !b {
				return ctx.Nil()
			}

			slog.Warn("ACT: message received",
				slog.Any("actor-pid", ctx.ActCtx().Self))

			switch m := message.(type) {
			case SlugActorMessage:
				return m.Msg
			}

			return ctx.Nil()
		},
	}
}

func fnActorTerminate() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) < 1 || len(args) > 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1..2", len(args))
			}

			pidObj, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to `terminate` must be NUMBER, got=%s", args[0].Type())
			}

			ctx.ActCtx().SendAsync(
				kernel.ActorID(pidObj.Value.ToInt64()),
				kernel.Exit{Reason: "actor terminated by user request"},
			)

			return ctx.Nil()
		},
	}
}

func fnActorRegister() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			pidObj, ok := args[0].(*object.Number)
			if !ok {
				return ctx.NewError("first argument to `register` must be NUMBER, got=%s", args[0].Type())
			}

			nameObj, ok := args[1].(*object.String)
			if !ok {
				return ctx.NewError("second argument to `register` must be STRING, got=%s", args[1].Type())
			}

			pid := kernel.ActorID(pidObj.Value.ToInt64())

			actorName := slugNamespace + nameObj.Value

			_, exists := ctx.ActCtx().K.ActorByName(actorName)
			if exists {
				return ctx.NewError("actor already registered: %s", actorName)
			}

			ctx.ActCtx().K.Register(actorName, pid)
			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}

func fnActorUnregister() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			nameObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `unregister` must be STRING, got=%s", args[0].Type())
			}

			pid := ctx.ActCtx().K.Unregister(slugNamespace + nameObj.Value)
			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}

func fnActorRegistered() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			names := ctx.ActCtx().K.Registered()
			elements := make([]object.Object, 0)
			for _, name := range names {
				if len(name) > 5 && name[:5] == slugNamespace {
					elements = append(elements, &object.String{Value: name[5:]})
				}
			}
			return &object.List{Elements: elements}
		},
	}
}

func fnActorLookup() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("wrong number of arguments. got=%d, want=1", len(args))
			}

			nameObj, ok := args[0].(*object.String)
			if !ok {
				return ctx.NewError("argument to `lookup` must be STRING, got=%s", args[0].Type())
			}

			pid := ctx.ActCtx().K.Lookup(slugNamespace + nameObj.Value)
			return &object.Number{Value: dec64.FromInt64(int64(pid))}
		},
	}
}
