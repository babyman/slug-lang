package modules

import (
	"bytes"
	"fmt"
	"log/slog"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
	"slug/internal/svc/resolver"
)

const (
	MaxErrorsToShow = 5
)

type ModuleLoader struct {
	DebugJsonAST bool
	DebugTxtAST  bool
	Module       *object.Module
	Error        error
}

func (ml *ModuleLoader) loadModuleHandler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	fwdMsg := svc.UnpackFwd(msg)
	if ml.Module != nil {
		svc.Reply(ctx, fwdMsg, LoadModuleResult{
			Module: ml.Module,
			Error:  ml.Error,
		})
	} else {
		payload, _ := fwdMsg.Payload.(LoadModule)
		mod, err := ml.loadModule(ctx, payload.PathParts)
		ml.Module = mod
		ml.Error = err
		svc.Reply(ctx, fwdMsg, LoadModuleResult{
			Module: mod,
			Error:  err,
		})
	}
	return kernel.Continue{}
}

func (ml *ModuleLoader) loadModule(ctx *kernel.ActCtx, pathParts []string) (*object.Module, error) {

	resId, _ := ctx.K.ActorByName(svc.ResolverService)

	resResult, err := ctx.SendSync(resId, resolver.ResolveModule{
		PathParts: pathParts,
	})
	if err != nil {
		return nil, err
	}

	modData, _ := resResult.Payload.(resolver.ResolvedResult)

	if modData.Error != nil {
		slog.Error("Failed to resolve module",
			slog.Any("path", pathParts),
			slog.Any("error", err))
		return nil, modData.Error
	}

	return lexAndParseModule(ctx, modData, ml.DebugJsonAST, ml.DebugTxtAST)
}

func lexAndParseModule(
	ctx *kernel.ActCtx,
	modData resolver.ResolvedResult,
	debugJsonAst bool,
	debugTxtAst bool,
) (*object.Module, error) {

	// Create a new environment and module object
	module := &object.Module{Name: modData.ModuleName, Env: nil}

	module.Src = modData.Data
	module.Path = modData.ModulePath

	// ==============================

	lexId, _ := ctx.K.ActorByName(svc.LexerService)
	parseId, _ := ctx.K.ActorByName(svc.ParserService)

	lex, err := ctx.SendSync(lexId, lexer.LexString{Sourcecode: module.Src})
	if err != nil {
		slog.Info("Failed to lex module source",
			slog.Any("error", err))
		return nil, err
	}

	tokens := lex.Payload.(lexer.LexedTokens).Tokens
	slog.Debug("Lexed module",
		slog.Any("path", module.Path),
		slog.Any("tokens", tokens))

	parse, err := ctx.SendSync(parseId, parser.ParseTokens{
		Path:       module.Path,
		Sourcecode: module.Src,
		Tokens:     tokens,
	})
	if err != nil {
		slog.Warn("Failed to parse source",
			slog.Any("error", err))
		return nil, err
	}

	ast := parse.Payload.(parser.ParsedAst).Program
	errors := parse.Payload.(parser.ParsedAst).Errors
	slog.Debug("Compiled",
		slog.Any("path", module.Path),
		slog.Any("ast", ast))

	module.Program = ast

	// ==============================

	if debugJsonAst || debugTxtAst {
		json, err := parser.RenderASTAsJSON(module.Program)
		text := parser.RenderASTAsText(module.Program, 0)
		if err != nil {
			slog.Error("Failed to render AST as JSON",
				slog.Any("error", err))
		} else {
			fsID, ok := ctx.K.ActorByName(svc.FsService)
			if ok && debugJsonAst {
				ctx.SendAsync(fsID, fs.WriteBytes{
					Data: []byte(json),
					Path: module.Path + ".ast.json",
				})
			}
			if ok && debugTxtAst {
				ctx.SendAsync(fsID, fs.WriteBytes{
					Data: []byte(text),
					Path: module.Path + ".ast.txt",
				})
			}
		}
	}

	// Report any parsing errors
	if len(errors) > 0 {
		var out bytes.Buffer
		out.WriteString("Woops! Looks like we slid into some slimy slug trouble here!\n")
		out.WriteString("Parser errors:\n")
		displayCount := len(errors)
		if displayCount > MaxErrorsToShow {
			displayCount = MaxErrorsToShow
		}
		for _, msg := range errors[:displayCount] {
			out.WriteString(fmt.Sprintf("\t%s\n", msg))
		}
		if displayCount < len(errors) {
			out.WriteString(fmt.Sprintf("\n...and %d more!\n", len(errors)-displayCount))
		}
		return nil, fmt.Errorf("%s\n", out.String())
	}

	return module, nil
}
