package runtime

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"slug/internal/foreign"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/util"
	"strings"
	"sync/atomic"
)

type Runtime struct {
	Config           util.Configuration
	Modules          map[string]*object.Module
	Builtins         map[string]*object.Foreign
	ForeignFunctions map[string]*object.Foreign
	nextID           atomic.Int64
}

func NewRuntime(config util.Configuration) *Runtime {
	builtinFunctions := map[string]*object.Foreign{
		"argv":       fnBuiltinArgv(),
		"argm":       fnBuiltinArgm(),
		"import":     fnBuiltinImport(),
		"len":        fnBuiltinLen(),
		"print":      fnBuiltinPrint(),
		"println":    fnBuiltinPrintLn(),
		"stacktrace": fnBuiltinStacktrace(),
	}

	return &Runtime{
		Config:           config,
		Modules:          nil,
		Builtins:         builtinFunctions,
		ForeignFunctions: getForeignFunctions(),
	}
}

func (r *Runtime) NextHandleID() int64 {
	return r.nextID.Add(1)<<16 | int64(rand.Intn(0xFFFF))
}

func (r *Runtime) LookupForeign(name string) (*object.Foreign, bool) {
	if fn, ok := r.ForeignFunctions[name]; ok {
		return fn, true
	}
	return nil, false
}

func (r *Runtime) LoadModule(modName string) (*object.Module, error) {

	if r.Modules == nil {
		r.Modules = make(map[string]*object.Module)
	}

	if mod, ok := r.Modules[modName]; ok {
		slog.Info("Module loaded from cache",
			slog.String("name", modName))
		return mod, nil
	}

	// 1. Resolve module name to relative file path (r.g., "slug.std" -> "slug/std.slug")
	pathParts := strings.Split(modName, ".")
	relPath := filepath.Join(pathParts...) + ".slug"

	// 2. Search Paths: Check local RootPath, then SLUG_HOME/lib
	var fullPath string
	var source []byte
	var err error

	// Try local RootPath (directory of the entry script)
	fullPath = filepath.Join(r.Config.RootPath, relPath)
	source, errFirst := os.ReadFile(fullPath)

	if errFirst != nil {
		//// Fallback to $SLUG_HOME/lib
		if r.Config.SlugHome != "" {
			fullPath = filepath.Join(r.Config.SlugHome, "lib", relPath)
			source, err = os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("could not load module %s: local error: %v, lib error: %v", modName, errFirst, err)
			}
		} else {
			return nil, fmt.Errorf("could not load module %s: %v (SLUG_HOME not set)", modName, errFirst)
		}
	}

	// 3. Tokenize and Parse
	l := lexer.New(string(source))
	p := parser.New(l, fullPath, string(source))
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		slog.Warn("Error loading module",
			slog.String("name", modName),
			slog.String("fullPath", fullPath),
			slog.String("errors", strings.Join(p.Errors(), "\n")),
		)
		return nil, fmt.Errorf("parse errors in module %s:\n%s", modName, strings.Join(p.Errors(), "\n"))
	}

	if r.Config.DebugJsonAST {
		json, err := parser.RenderASTAsJSON(program)
		if err != nil {
			slog.Error("Failed to render AST as JSON",
				slog.Any("error", err))
		} else {
			jsonPath := fullPath + ".ast.json"
			err = os.WriteFile(jsonPath, []byte(json), 0644)
			if err != nil {
				slog.Error("Failed to write AST as JSON")
			}
		}
	}
	if r.Config.DebugTxtAST {
		txtPath := fullPath + ".ast.txt"
		text := parser.RenderASTAsText(program, 0)
		err = os.WriteFile(txtPath, []byte(text), 0644)
		if err != nil {
			slog.Error("Failed to write AST as JSON")
		}
	}

	// 4. Setup Module Object and Environment
	moduleEnv := object.NewEnvironment()
	moduleEnv.Path = fullPath
	moduleEnv.ModuleFqn = modName
	moduleEnv.Src = string(source)

	module := &object.Module{
		Name:    modName,
		Path:    fullPath,
		Src:     string(source),
		Program: program,
		Env:     moduleEnv,
	}

	r.Modules[modName] = module
	slog.Info("Module loaded, added to cache",
		slog.String("name", modName),
		slog.String("fullPath", fullPath),
	)

	// 5. Evaluate the module in its own environment
	slog.Debug("loading module", slog.String("name", modName), slog.String("path", fullPath))

	e := &Task{
		Runtime: r,
	}
	e.PushNurseryScope(&NurseryScope{
		Limit: make(chan struct{}, r.Config.DefaultLimit),
	})
	e.PushEnv(moduleEnv)
	out := e.Eval(program)
	// We pop the env, but the moduleEnv now contains all the defined bindings
	e.PopEnv(out)

	if e.isError(out) {
		return nil, fmt.Errorf("runtime error while loading module %s: %s", modName, out.Inspect())
	}

	return module, nil
}

func getForeignFunctions() map[string]*object.Foreign {
	foreignFunctions := map[string]*object.Foreign{}
	for k, v := range foreign.GetForeignFunctions() {
		foreignFunctions[k] = v
	}
	return foreignFunctions
}
