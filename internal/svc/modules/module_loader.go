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

type ModuleLoader struct {
	Module *object.Module
	Error  error
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
		mod, err := ml.loadModule(ctx, payload)
		ml.Module = mod
		ml.Error = err
		svc.Reply(ctx, fwdMsg, LoadModuleResult{
			Module: mod,
			Error:  err,
		})
	}
	return kernel.Continue{}
}

func (ml *ModuleLoader) loadModule(ctx *kernel.ActCtx, payload LoadModule) (*object.Module, error) {
	// Generate the module moduleName from path parts
	moduleName := strings.Join(payload.PathParts, ".")

	svc.SendInfof(ctx, "Loading module '%s' from path parts: %v  Root path: %s\n",
		moduleName, payload.PathParts, payload.RootPath)

	// Create a new environment and module object
	module := &object.Module{Name: moduleName, Env: nil}

	// Complete the module path
	moduleRelativePath := strings.Join(payload.PathParts, "/")
	modulePath := fmt.Sprintf("%s/%s.slug", payload.RootPath, moduleRelativePath)

	// Attempt to load the module's source
	// todo use file service
	fsId, _ := ctx.K.ActorByName(svc.FsService)
	//moduleSrc, err := ioutil.ReadFile(modulePath)
	moduleSrc, err := ctx.SendSync(fsId, fs.Read{Path: modulePath})
	if err != nil {
		svc.SendInfof(ctx, "Failed to read file: %s", err)
	}

	// Parse the source into an AST
	readResp := moduleSrc.Payload.(fs.ReadResp)

	if readResp.Err != nil {
		// Fallback to SLUG_HOME if the file doesn't exist
		slugHome := os.Getenv("SLUG_HOME")
		if slugHome == "" {
			return nil, fmt.Errorf("error reading module '%s': SLUG_HOME environment variable is not set", moduleName)
		}
		libPath := fmt.Sprintf("%s/lib/%s.slug", slugHome, moduleRelativePath)
		// todo use file service
		//moduleSrc, err = ioutil.ReadFile(libPath)
		moduleSrc, err = ctx.SendSync(fsId, fs.Read{Path: libPath})
		if err != nil {
			return nil, fmt.Errorf("error reading module (%s / %s) '%s': %s", modulePath, libPath, moduleName, err)
		} else {
			modulePath = libPath
			readResp = moduleSrc.Payload.(fs.ReadResp)
		}
	}

	src := readResp.Data
	//println("module_loader", "source", len(src))
	module.Src = src
	module.Path = modulePath

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

	if payload.DebugAST {
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
