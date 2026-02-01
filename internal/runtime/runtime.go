package runtime

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"slug/internal/ast"
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
	FullSchema       *object.StructSchema
	EmptySchema      *object.StructSchema
	nextID           atomic.Int64
}

func NewRuntime(config util.Configuration) *Runtime {

	config.Store = util.NewConfigStore(config.RootPath, config.SlugHome, config.MainModule, config.Argv)

	builtinFunctions := map[string]*object.Foreign{
		"argv":       fnBuiltinArgv(),
		"argm":       fnBuiltinArgm(),
		"cfg":        fnBuiltinCfg(),
		"import":     fnBuiltinImport(),
		"len":        fnBuiltinLen(),
		"print":      fnBuiltinPrint(),
		"println":    fnBuiltinPrintLn(),
		"stacktrace": fnBuiltinStacktrace(),
	}

	functions := getForeignFunctions()
	functions["slug.channel.chan"] = fnChannelChan()
	functions["slug.channel.close"] = fnChannelClose()
	functions["slug.channel.recv"] = fnChannelRecv()
	functions["slug.channel.send"] = fnChannelSend()

	return &Runtime{
		Config:           config,
		Modules:          nil,
		Builtins:         builtinFunctions,
		ForeignFunctions: functions,
		FullSchema: &object.StructSchema{
			Name:       "Full",
			Fields:     []object.StructSchemaField{{Name: "value"}},
			FieldIndex: map[string]int{"value": 0},
		},
		EmptySchema: &object.StructSchema{
			Name:       "Empty",
			Fields:     []object.StructSchemaField{},
			FieldIndex: map[string]int{},
		},
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
	if modName == "slug.channel" {
		if _, err := moduleEnv.DefineConstant("Full", r.FullSchema, true, false); err != nil {
			return nil, fmt.Errorf("failed to install channel schema for module %s: %w", modName, err)
		}
		if _, err := moduleEnv.DefineConstant("Empty", r.EmptySchema, true, false); err != nil {
			return nil, fmt.Errorf("failed to install channel schema for module %s: %w", modName, err)
		}
	}

	module := &object.Module{
		Name:    modName,
		Path:    fullPath,
		Src:     string(source),
		Program: program,
		Env:     moduleEnv,
		Doc:     program.ModuleDoc,
		HasDoc:  program.HasModuleDoc,
	}

	r.Modules[modName] = module

	// Declare pass: prebind top-level names to support circular imports.
	predeclareTopLevel(program, moduleEnv)
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

func predeclareTopLevel(program *ast.Program, env *object.Environment) error {
	for _, stmt := range program.Statements {
		// Statements may be wrapped in ExpressionStatement
		var expr ast.Expression
		if exprStmt, ok := stmt.(*ast.ExpressionStatement); ok {
			expr = exprStmt.Expression
		}

		switch s := expr.(type) {
		case *ast.ValExpression:
			if err := predeclarePattern(s.Pattern, true, hasExportTag(s.Tags), env); err != nil {
				return err
			}
		case *ast.VarExpression:
			if err := predeclarePattern(s.Pattern, false, hasExportTag(s.Tags), env); err != nil {
				return err
			}
		}
	}
	return nil
}

func predeclarePattern(pat ast.MatchPattern, isConst bool, isExport bool, env *object.Environment) error {
	switch p := pat.(type) {

	case *ast.IdentifierPattern:
		name := p.Value.Value
		if isConst {
			_, err := env.DefineConstant(name, object.BINDING_UNINITIALIZED, isExport, false)
			return err
		}
		_, err := env.Define(name, object.BINDING_UNINITIALIZED, isExport, false)
		return err

	case *ast.SpreadPattern:
		// spread can bind a name: var {..rest} = ...
		if p.Value == nil {
			return nil
		}
		name := p.Value.Value
		if isConst {
			_, err := env.DefineConstant(name, object.BINDING_UNINITIALIZED, isExport, false)
			return err
		}
		_, err := env.Define(name, object.BINDING_UNINITIALIZED, isExport, false)
		return err

	case *ast.ListPattern:
		for _, elem := range p.Elements {
			if err := predeclarePattern(elem, isConst, isExport, env); err != nil {
				return err
			}
		}
		return nil

	case *ast.MapPattern:
		// var {*} = ... : cannot predeclare (no names!)
		if p.SelectAll {
			return nil
		}
		// declare any identifiers inside explicit subpatterns
		for _, entry := range p.Pairs {
			if entry.Pattern == nil {
				continue
			}
			if err := predeclarePattern(entry.Pattern, isConst, isExport, env); err != nil {
				return err
			}
		}
		if p.Spread != nil {
			if err := predeclarePattern(p.Spread, isConst, isExport, env); err != nil {
				return err
			}
		}
		return nil

	case *ast.MultiPattern:
		// no new bindings are *guaranteed* across alternatives; safest is to predeclare none
		return nil

	default:
		// LiteralPattern, WildcardPattern, etc: no bindings
		return nil
	}
}

func hasExportTag(tags []*ast.Tag) bool {
	for _, tag := range tags {
		if tag.Name == object.EXPORT_TAG {
			return true
		}
	}
	return false
}

func getForeignFunctions() map[string]*object.Foreign {
	foreignFunctions := map[string]*object.Foreign{}
	for k, v := range foreign.GetForeignFunctions() {
		foreignFunctions[k] = v
	}
	return foreignFunctions
}
