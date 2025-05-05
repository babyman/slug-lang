package object

import "slug/internal/parser"

// NewEnclosedEnvironment initializes an environment with a parent and optional stack frame.
func NewEnclosedEnvironment(outer *Environment, stackFrame *StackFrame) *Environment {
	env := NewEnvironment()
	env.outer = outer
	env.rootPath = outer.rootPath
	env.Path = outer.Path
	env.Src = outer.Src
	env.StackInfo = stackFrame
	return env
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{Store: s, outer: nil, rootPath: "."}
}

type Environment struct {
	Store     map[string]Object
	outer     *Environment
	Src       string
	Path      string
	rootPath  string      // Track the execution root path
	StackInfo *StackFrame // Optional stack frame information
}

// SetRootPath sets the root path for the environment
func (e *Environment) SetRootPath(path string) {
	e.rootPath = path
}

// GetRootPath retrieves the root path for the environment
func (e *Environment) GetRootPath() string {
	return e.rootPath
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.Store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Define(name string, val Object) (Object, bool) {
	if _, exists := e.Store[name]; exists {
		println("Failed to define variable: ", name, " already exists")
		return nil, false
	}
	e.Store[name] = val
	return val, true
}

func (e *Environment) Assign(name string, val Object) (Object, bool) {
	if _, exists := e.Store[name]; exists {
		e.Store[name] = val
		return val, true
	}
	if e.outer != nil {
		return e.outer.Assign(name, val)
	}
	return nil, false
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
