package evaluator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"slug/internal/object"
	"slug/internal/svc/lexer"
	"slug/internal/svc/parser"
	"strings"
)

// moduleRegistry is a global cache for loaded modules
var moduleRegistry = make(map[string]*object.Module)

// LoadModule loads a module by its path parts into the module registry and evaluates it.
func LoadModule(pathParts []string) (*object.Module, error) {
	// Generate the module moduleName from path parts
	moduleName := strings.Join(pathParts, ".")

	// Check if the module already exists in the registry
	if module, exists := moduleRegistry[moduleName]; exists {
		return module, nil // Return the cached module
	}

	//fmt.Printf("Loading module '%s' from path parts: %v  Root path: %s\n", moduleName, pathParts, RootPath)

	// Create a new environment and module object
	module := &object.Module{Name: moduleName, Env: nil}
	moduleRegistry[moduleName] = module // Cache the module

	// Complete the module path
	moduleRelativePath := strings.Join(pathParts, "/")
	modulePath := fmt.Sprintf("%s/%s.slug", RootPath, moduleRelativePath)

	// Attempt to load the module's source
	moduleSrc, err := ioutil.ReadFile(modulePath)
	if err != nil {
		// Fallback to SLUG_HOME if the file doesn't exist
		slugHome := os.Getenv("SLUG_HOME")
		if slugHome == "" {
			return nil, fmt.Errorf("error reading module '%s': SLUG_HOME environment variable is not set", moduleName)
		}
		libPath := fmt.Sprintf("%s/lib/%s.slug", slugHome, moduleRelativePath)
		moduleSrc, err = ioutil.ReadFile(libPath)
		if err != nil {
			return nil, fmt.Errorf("error reading module (%s / %s) '%s': %s", modulePath, libPath, moduleName, err)
		} else {
			modulePath = libPath
		}
	}

	// Parse the source into an AST
	src := string(moduleSrc)
	module.Src = src
	module.Path = modulePath

	l := lexer.New(src)
	p := parser.New(l, src)
	module.Program = p.ParseProgram()

	if DebugAST {
		if err := parser.WriteASTToJSON(module.Program, module.Path+".ast.json"); err != nil {
			return nil, fmt.Errorf("Failed to write AST to JSON: %v", err)
		}
	}

	// Report any parsing errors
	if len(p.Errors()) > 0 {
		var out bytes.Buffer
		out.WriteString("Woops! Looks like we slid into some slimy slug trouble here!\n")
		out.WriteString("Parser errors:\n")
		for _, msg := range p.Errors() {
			out.WriteString(fmt.Sprintf("\t%s\n", msg))
		}
		return nil, fmt.Errorf(out.String())
	}

	return module, nil
}
