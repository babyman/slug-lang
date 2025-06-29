package object

import (
	"fmt"
	"slug/internal/ast"
	"slug/internal/log"
	"slug/internal/parser"
)

// NewEnclosedEnvironment initializes an environment with a parent and optional stack frame.
func NewEnclosedEnvironment(outer *Environment, stackFrame *StackFrame) *Environment {
	log.Trace("------ new env ------\n")
	env := NewEnvironment()
	env.outer = outer
	env.Path = outer.Path
	env.ModuleFqn = outer.ModuleFqn
	env.Src = outer.Src
	env.StackInfo = stackFrame
	return env
}

func NewEnvironment() *Environment {
	s := make(map[string]*Binding)
	return &Environment{Bindings: s, outer: nil}
}

type Environment struct {
	Bindings   map[string]*Binding
	outer      *Environment
	Src        string
	Path       string
	ModuleFqn  string
	StackInfo  *StackFrame     // Optional stack frame information
	deferStack []ast.Statement // Stack for deferred statements
}

type Binding struct {
	Value Object // can be a FunctionGroup
	Meta  Meta
	//MetaIndex map[string]Meta // todo add metadata for function group functions
	IsMutable bool
}

type Meta struct {
	IsImport bool
	IsExport bool
}

func (e *Environment) GetBinding(name string) (*Binding, bool) {
	if binding, ok := e.Bindings[name]; ok {
		return binding, true
	}
	if e.outer != nil {
		return e.outer.GetBinding(name)
	}
	return nil, false
}

func (e *Environment) Get(name string) (Object, bool) {
	binding, ok := e.GetBinding(name)
	if !ok {
		return nil, false
	}
	log.Info("Found binding for '%s': %v", name, binding)
	return binding.Value, true
}

func (e *Environment) DefineConstant(name string, val Object, isExported bool, isImport bool) (Object, error) {
	return e.define(name, val, false, isExported, isImport)
}

// Define adds a new variable with the given name and value to the environment and returns the value
func (e *Environment) Define(name string, val Object, isExported bool, isImport bool) (Object, error) {
	return e.define(name, val, true, isExported, isImport)
}

func (e *Environment) define(name string, val Object, isMutable bool, isExported bool, isImport bool) (Object, error) {
	declaration := "val"
	if isMutable {
		declaration = "var"
	}
	binding, exists := e.Bindings[name]
	if exists && !binding.IsMutable {
		return nil, fmt.Errorf("%s `%s` is already defined as a 'val' and cannot be reassigned", declaration, name)
	} else if !exists {
		binding = &Binding{
			Value:     nil,
			IsMutable: isMutable,
		}
	}
	binding.Meta = Meta{
		IsImport: isImport,
		IsExport: isExported,
	}
	switch val := val.(type) {
	case *Function:
		fg, ok := binding.Value.(*FunctionGroup)
		if !ok {
			fg = &FunctionGroup{
				Functions: map[ast.FSig]Object{},
			}
		}
		fg.Functions[val.Signature] = val
		binding.Value = fg
	case *Foreign:
		fg, ok := binding.Value.(*FunctionGroup)
		if !ok {
			fg = &FunctionGroup{
				Functions: map[ast.FSig]Object{},
			}
		}
		fg.Functions[val.Signature] = val
		binding.Value = fg
	case *FunctionGroup:
		fg, ok := binding.Value.(*FunctionGroup)
		if !ok {
			fg = &FunctionGroup{
				Functions: map[ast.FSig]Object{},
			}
		}
		// Copy functions from val into fg
		for signature, fn := range val.Functions {
			fg.Functions[signature] = fn
		}
		binding.Value = fg
	default:
		binding.Value = val
	}
	e.Bindings[name] = binding
	log.Debug("binding: %v %v %v\n", binding.Value.Type(), name, binding.Meta)
	return val, nil
}

func (e *Environment) Assign(name string, val Object) (Object, error) {
	if binding, exists := e.Bindings[name]; exists {
		if !binding.IsMutable {
			return nil, fmt.Errorf("failed to assign to val '%s': value is immutible", name)
		}

		// since it's an assignment clear the import flag
		binding.Meta.IsImport = false

		switch val := val.(type) {
		case *Function:
			fg, ok := binding.Value.(*FunctionGroup)
			if !ok {
				fg = &FunctionGroup{
					Functions: map[ast.FSig]Object{},
				}
			}
			fg.Functions[val.Signature] = val
			binding.Value = fg
		case *Foreign:
			fg, ok := binding.Value.(*FunctionGroup)
			if !ok {
				fg = &FunctionGroup{
					Functions: map[ast.FSig]Object{},
				}
			}
			fg.Functions[val.Signature] = val
			binding.Value = fg
		case *FunctionGroup:
			binding.Value = val
		default:
			binding.Value = val
		}
		return val, nil
	}
	if e.outer != nil {
		return e.outer.Assign(name, val)
	}
	return nil, fmt.Errorf("failed to assign to '%s': not defined in any accessible scope", name)
}

func (e *Environment) GatherStackTrace(frame *StackFrame) []StackFrame {
	var trace []StackFrame
	line, col := parser.GetLineAndColumn(e.Src, frame.Position)
	frame.Line = line
	frame.Col = col
	trace = append([]StackFrame{*frame}, trace...)
	for e := e; e != nil; e = e.outer {
		if e.StackInfo != nil {
			line, col := parser.GetLineAndColumn(e.Src, e.StackInfo.Position)
			e.StackInfo.Line = line
			e.StackInfo.Col = col
			trace = append([]StackFrame{*e.StackInfo}, trace...) // Prepend to maintain call order
		}
	}
	return reverse(trace)
}

func reverse(slice []StackFrame) []StackFrame {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice
}

func (e *Environment) RegisterDefer(deferStmt ast.Statement) {
	log.Debug(">>> Stashing deferred block: %v", deferStmt)
	e.deferStack = append(e.deferStack, deferStmt)
}

func (e *Environment) ExecuteDeferred(evalFunc func(stmt ast.Statement)) {
	defer func() { e.deferStack = nil }() // Always clear defer stack
	for i := len(e.deferStack) - 1; i >= 0; i-- {
		evalFunc(e.deferStack[i]) // Execute each deferred statement
		log.Debug("<<< Executing deferred block: %s", e.deferStack[i])
	}
}
