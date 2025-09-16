package modules

import (
	"bytes"
	"fmt"
	"os"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/fs"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
	"strings"
)

func (m *Modules) onEvaluateFile(ctx *kernel.ActCtx, msg kernel.Message, payload EvaluateFile) kernel.HandlerSignal {

	svc.SendDebugf(ctx, "Evaluating file %s", payload.Path)

	fsId, _ := ctx.K.ActorByName(svc.FsService)
	lexId, _ := ctx.K.ActorByName(svc.LexerService)
	parseId, _ := ctx.K.ActorByName(svc.ParserService)
	evalId, _ := ctx.K.ActorByName(svc.EvalService)

	src, err := ctx.SendSync(fsId, fs.FsRead{Path: payload.Path})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to read file: %s", err)
		return kernel.Continue{}
	}

	file := src.Payload.(fs.FsReadResp).Data
	//service.SendInfof(ctx, "Evaluating %s, got %s", payload.Path, file)

	lex, err := ctx.SendSync(lexId, lexer.LexString{Sourcecode: file})
	if err != nil {
		svc.SendInfof(ctx, "Failed to lex file: %s", err)
		return kernel.Continue{}
	}

	tokens := lex.Payload.(lexer.LexedTokens).Tokens
	svc.SendDebugf(ctx, "Lexed %s, got %v", payload.Path, tokens)

	parse, err := ctx.SendSync(parseId, parser.ParseTokens{Sourcecode: file, Tokens: tokens})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to parse file: %s", err)
		return kernel.Continue{}
	}

	ast := parse.Payload.(parser.ParsedAst).Program
	svc.SendDebugf(ctx, "Compiled %s, got %v", payload.Path, ast)

	result, err := ctx.SendSync(evalId, svc.EvaluateProgram{
		Source:  file,
		Program: ast,
		Args:    payload.Args,
	})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to execute file: %s", err)
		return kernel.Continue{}
	}

	p := result.Payload
	svc.SendInfof(ctx, "Compiled %s, got %v", payload.Path, p)

	svc.Reply(ctx, msg, p)

	return kernel.Continue{}
}

func (m *Modules) onLoadModule(ctx *kernel.ActCtx, msg kernel.Message, payload LoadModule) kernel.HandlerSignal {
	// Generate the module moduleName from path parts
	moduleName := strings.Join(payload.PathParts, ".")

	// Check if the module already exists in the registry
	if module, exists := m.moduleRegistry[moduleName]; exists {
		svc.Reply(ctx, msg, LoadModuleResult{
			Module: module,
		})
		return nil
	}

	svc.SendInfof(ctx, "Loading module '%s' from path parts: %v  Root path: %s\n",
		moduleName, payload.PathParts, payload.RootPath)

	// Create a new environment and module object
	module := &object.Module{Name: moduleName, Env: nil}
	m.moduleRegistry[moduleName] = module // Cache the module

	// Complete the module path
	moduleRelativePath := strings.Join(payload.PathParts, "/")
	modulePath := fmt.Sprintf("%s/%s.slug", payload.RootPath, moduleRelativePath)

	// Attempt to load the module's source
	// todo use file service
	fsId, _ := ctx.K.ActorByName(svc.FsService)
	//moduleSrc, err := ioutil.ReadFile(modulePath)
	moduleSrc, err := ctx.SendSync(fsId, fs.FsRead{Path: modulePath})
	if err != nil {
		// Fallback to SLUG_HOME if the file doesn't exist
		slugHome := os.Getenv("SLUG_HOME")
		if slugHome == "" {
			svc.Reply(ctx, msg, LoadModuleResult{
				Error: fmt.Errorf("error reading module '%s': SLUG_HOME environment variable is not set", moduleName),
			})
			return nil
		}
		libPath := fmt.Sprintf("%s/lib/%s.slug", slugHome, moduleRelativePath)
		// todo use file service
		//moduleSrc, err = ioutil.ReadFile(libPath)
		moduleSrc, err = ctx.SendSync(fsId, fs.FsRead{Path: libPath})
		if err != nil {
			svc.Reply(ctx, msg, LoadModuleResult{
				Error: fmt.Errorf("error reading module (%s / %s) '%s': %s", modulePath, libPath, moduleName, err),
			})
			return nil
		} else {
			modulePath = libPath
		}
	}

	// Parse the source into an AST
	src := moduleSrc.Payload.(fs.FsReadResp).Data
	module.Src = src
	module.Path = modulePath

	// ==============================

	//fsId, _ := ctx.K.ActorByName(svc.FsService)
	lexId, _ := ctx.K.ActorByName(svc.LexerService)
	parseId, _ := ctx.K.ActorByName(svc.ParserService)
	//evalId, _ := ctx.K.ActorByName(svc.EvalService)

	//src, err := ctx.SendSync(fsId, fs.FsRead{Path: modulePath})
	//if err != nil {
	//	svc.SendWarnf(ctx, "Failed to read file: %s", err)
	//	return kernel.Continue{}
	//}

	//file := src.Payload.(fs.FsReadResp).Data
	//service.SendInfof(ctx, "Evaluating %s, got %s", payload.Path, file)

	lex, err := ctx.SendSync(lexId, lexer.LexString{Sourcecode: module.Src})
	if err != nil {
		svc.SendInfof(ctx, "Failed to lex file: %s", err)
		return kernel.Continue{}
	}

	tokens := lex.Payload.(lexer.LexedTokens).Tokens
	svc.SendDebugf(ctx, "Lexed %s, got %v", module.Path, tokens)

	parse, err := ctx.SendSync(parseId, parser.ParseTokens{Sourcecode: module.Src, Tokens: tokens})
	if err != nil {
		svc.SendWarnf(ctx, "Failed to parse file: %s", err)
		return kernel.Continue{}
	}

	ast := parse.Payload.(parser.ParsedAst).Program
	errors := parse.Payload.(parser.ParsedAst).Errors
	svc.SendDebugf(ctx, "Compiled %s, got %v", module.Path, ast)

	module.Program = ast

	// ==============================

	///// todo
	//l := lexer.New(src)
	//// todo
	//p := parser.New(l, src)
	//module.Program = p.ParseProgram()

	if payload.DebugAST {
		if err := parser.WriteASTToJSON(module.Program, module.Path+".ast.json"); err != nil {
			svc.Reply(ctx, msg, LoadModuleResult{
				Error: fmt.Errorf("failed to write AST to JSON: %v", err),
			})
			return nil

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
		svc.Reply(ctx, msg, LoadModuleResult{
			Error: fmt.Errorf(out.String()),
		})
		return nil
	}

	svc.Reply(ctx, msg, LoadModuleResult{
		Module: module,
	})
	return nil
}
