package modules

import (
	"bytes"
	"fmt"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
	"slug/internal/svc/resolver"
)

type ModuleLoader struct {
	DebugAST bool
	Module   *object.Module
	Error    error
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

	return lexAndParseModule(ctx, modData, ml.DebugAST)
}

func lexAndParseModule(
	ctx *kernel.ActCtx,
	modData resolver.ResolvedResult,
	debugAst bool,
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
		svc.SendInfof(ctx, "Failed to lex file: %s", err)
		return nil, err
	}

	tokens := lex.Payload.(lexer.LexedTokens).Tokens
	svc.SendDebugf(ctx, "Lexed %s, got %v", module.Path, tokens)

	parse, err := ctx.SendSync(parseId, parser.ParseTokens{Sourcecode: module.Src, Tokens: tokens})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to parse file: %s", err)
		return nil, err
	}

	ast := parse.Payload.(parser.ParsedAst).Program
	errors := parse.Payload.(parser.ParsedAst).Errors
	//fmt.Printf("Compiled %s, got %v", module.Path, ast)
	svc.SendDebugf(ctx, "Compiled %s, got %v", module.Path, ast)

	module.Program = ast

	// ==============================

	if debugAst {
		// todo use file service to write these
		if err := parser.WriteASTToJSON(module.Program, module.Path+".ast.json"); err != nil {
			return nil, fmt.Errorf("failed to write AST to JSON: %v", err)
		}
	}

	// Report any parsing errors
	if len(errors) > 0 {
		var out bytes.Buffer
		out.WriteString("Woops! Looks like we slid into some slimy slug trouble here!\n")
		out.WriteString("Parser errors:\n")
		for _, msg := range errors {
			out.WriteString(fmt.Sprintf("\t%s\n", msg))
		}
		return nil, fmt.Errorf(out.String())
	}

	return module, nil
}
