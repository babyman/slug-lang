package object

import (
	"fmt"
	"slug/internal/ast"
	"slug/internal/parser"
)

// NewEnclosedEnvironment initializes an environment with a parent and optional stack frame.
func NewEnclosedEnvironment(outer *Environment, stackFrame *StackFrame) *Environment {
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
	return &Environment{Store: s, outer: nil}
}

type Binding struct {
	Value      Object
	IsConstant bool
}

type Environment struct {
	Store      map[string]*Binding
	outer      *Environment
	Src        string
	Path       string
	ModuleFqn  string
	StackInfo  *StackFrame     // Optional stack frame information
	deferStack []ast.Statement // Stack for deferred statements
}

func (e *Environment) GetBinding(name string) (*Binding, bool) {
	if binding, ok := e.Store[name]; ok {
		return binding, true
	}
	if e.outer != nil {
		return e.outer.GetBinding(name)
	}
	return nil, false
}

func (e *Environment) Get(name string) (Object, bool) {
	b, ok := e.GetBinding(name)
	return b.Value, ok
}

func (e *Environment) DefineConstant(name string, val Object) (Object, error) {
	if v, exists := e.Store[name]; exists && v.IsConstant {
		return nil, fmt.Errorf("val `%s` is already defined and cannot be reassigned", name)
	}
	e.Store[name] = &Binding{Value: val, IsConstant: true}
	return val, nil
}

// Define adds a new variable with the given name and value to the environment and returns the value
func (e *Environment) Define(name string, val Object) (Object, error) {
	if v, exists := e.Store[name]; exists && v.IsConstant {
		return nil, fmt.Errorf("var `%s` is already defined as a 'val' and cannot be reassigned", name)
	}
	e.Store[name] = &Binding{Value: val, IsConstant: false}
	return val, nil
}

func (e *Environment) Assign(name string, val Object) (Object, error) {
	if v, exists := e.Store[name]; exists {
		if v.IsConstant {
			return nil, fmt.Errorf("failed to assign to val '%s': value is immutible", name)
		}
		e.Store[name] = &Binding{Value: val, IsConstant: false}
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
	//println(">>> Stashing deferred block: ", deferStmt.String())
	e.deferStack = append(e.deferStack, deferStmt)
}

func (e *Environment) ExecuteDeferred(evalFunc func(stmt ast.Statement)) {
	defer func() { e.deferStack = nil }() // Always clear defer stack
	for i := len(e.deferStack) - 1; i >= 0; i-- {
		evalFunc(e.deferStack[i]) // Execute each deferred statement
		//println("<<< Executing deferred block: ", e.deferStack[i].String())
	}
}
