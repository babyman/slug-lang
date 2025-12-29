package object

import (
	"fmt"
	"log/slog"
	"slug/internal/ast"
	"sync"
)

// NewEnclosedEnvironment initializes an environment with a parent and optional stack frame.
func NewEnclosedEnvironment(outer *Environment, stackFrame *StackFrame) *Environment {
	slog.Debug("------ new env ------\n")
	env := NewEnvironment()
	env.Outer = outer
	env.Path = outer.Path
	env.ModuleFqn = outer.ModuleFqn
	env.Src = outer.Src
	env.StackInfo = stackFrame
	return env
}

func NewEnvironment() *Environment {
	return &Environment{
		Bindings: make(map[string]*Binding),
		Defers:   make([]*ast.DeferStatement, 0),
		Children: make([]*TaskHandle, 0),
	}
}

// NewRootEnvironment creates the base environment with a system-wide concurrency limit
func NewRootEnvironment(limit int) *Environment {
	env := NewEnvironment()
	if limit > 0 {
		env.Limit = make(chan struct{}, limit)
	}
	return env
}

type Environment struct {
	Bindings  map[string]*Binding
	Outer     *Environment
	Src       string
	Path      string
	ModuleFqn string
	StackInfo *StackFrame           // Optional stack frame information
	Defers    []*ast.DeferStatement // Stack for deferred statements

	// Concurrency tracking
	Children []*TaskHandle // Tasks owned by this scope
	Limit    chan struct{} // Semaphore for 'async limit N'
	mu       sync.Mutex
}

type Binding struct {
	Value Object // can be a FunctionGroup
	Err   *RuntimeError
	Meta  Meta
	//MetaIndex map[string]Meta // todo add metadata for function group functions
	IsMutable bool
}

type Meta struct {
	IsImport bool
	IsExport bool
}

// AddChild registers a task handle with this environment
func (e *Environment) AddChild(th *TaskHandle) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Children = append(e.Children, th)
}

// WaitChildren blocks until all direct children of this scope have settled
func (e *Environment) WaitChildren() {
	// We copy the slice to avoid holding the lock during blocking waits
	e.mu.Lock()
	children := make([]*TaskHandle, len(e.Children))
	copy(children, e.Children)
	e.mu.Unlock()

	for _, child := range children {
		<-child.Done
	}
}

func (e *Environment) GetBinding(name string) (*Binding, bool) {
	if binding, ok := e.Bindings[name]; ok {
		return binding, true
	}
	if e.Outer != nil {
		return e.Outer.GetBinding(name)
	}
	return nil, false
}

func (e *Environment) Get(name string) (Object, bool) {
	binding, ok := e.GetBinding(name)
	if !ok {
		return nil, false
	}
	slog.Debug("Found binding",
		slog.Any("name", name),
		slog.Any("binding", binding))
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
	//fmt.Printf("binding: %v %v %v %v\n", binding.Value.Type(), name, val.Inspect(), binding.Meta)
	slog.Debug("binding value",
		slog.Any("type", binding.Value.Type()),
		slog.Any("name", name),
		slog.Any("meta", binding.Meta))
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
		//fmt.Printf("assigning: %v %v %v %v\n", binding.Value.Type(), name, binding.Value, binding.Meta)
		slog.Debug("assigning bound value",
			slog.Any("type", binding.Value.Type()),
			slog.Any("name", name),
			slog.Any("meta", binding.Meta))
		return val, nil
	}
	if e.Outer != nil {
		return e.Outer.Assign(name, val)
	}
	return nil, fmt.Errorf("failed to assign to '%s': not defined in any accessible scope", name)
}

func (e *Environment) RegisterDefer(deferStmt *ast.DeferStatement) {
	slog.Debug("Stashing deferred block",
		slog.Any("deferred-statement", deferStmt))
	e.Defers = append(e.Defers, deferStmt)
}

// ExecuteDeferred runs deferred statements.
// It takes the current result of the block/function and returns the final result.
// If a deferred statement recovers or throws, the returned object will reflect that.
func (e *Environment) ExecuteDeferred(result Object, evalFunc func(stmt ast.Statement) Object) Object {
	defer func() { e.Defers = nil }() // Always clear defer stack

	if e.Defers == nil || len(e.Defers) == 0 {
		return result
	}
	slog.Debug("Deferred execution starting",
		slog.Any("pre-result", result))
	currentResult := result

	for i := len(e.Defers) - 1; i >= 0; i-- {
		ds := e.Defers[i]

		// 1. Analyze current state
		isError := false
		var errorPayload Object
		var activeRuntimeErr *RuntimeError

		if currentResult != nil {
			if rtErr, ok := currentResult.(*RuntimeError); ok {
				isError = true
				activeRuntimeErr = rtErr
				errorPayload = rtErr.Payload
			} else if _, ok := currentResult.(*Error); ok {
				// Internal/Native errors are also errors
				isError = true
				errorPayload = currentResult
			}
		}

		// 2. Determine if handler should run
		shouldRun := false
		switch ds.Mode {
		case ast.DeferAlways:
			shouldRun = true
		case ast.DeferOnSuccess:
			shouldRun = !isError
		case ast.DeferOnError:
			shouldRun = isError
		}

		if shouldRun {
			if isError && ds.Mode == ast.DeferOnError && ds.ErrorName != nil {
				// Force bind the error variable in the current environment
				e.Bindings[ds.ErrorName.Value] = &Binding{
					Value:     errorPayload,
					Err:       activeRuntimeErr,
					IsMutable: false,
					Meta:      Meta{},
				}
			}

			// 3. Execute the deferred block
			deferResult := evalFunc(ds.Call)

			slog.Debug("Executed deferred block",
				slog.Any("is-error", isError),
				slog.Any("block", ds.Call.String()),
				slog.Any("defer-result", deferResult.Inspect()),
			)

			// 4. Handle the result of the deferred block
			// If the block returned a RuntimeError (threw), we chain or replace.
			if newRtErr, ok := deferResult.(*RuntimeError); ok {
				if activeRuntimeErr != nil {
					newRtErr.Cause = activeRuntimeErr
				}
				currentResult = newRtErr
				continue
			}

			// Check for ReturnValue wrapper (Explicit Return)
			// This distinguishes `return x` (explicit) from `x` (implicit block result)
			if isError && ds.Mode == ast.DeferOnError {
				if retVal, ok := deferResult.(*ReturnValue); ok {
					val := retVal.Value
					currentResult = val
				} else {
					currentResult = deferResult
				}
			}
		}
	}

	slog.Debug("Deferred execution complete",
		slog.Any("result", currentResult))

	return currentResult
}
