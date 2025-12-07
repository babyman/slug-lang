package object

import (
	"fmt"
	"log/slog"
	"slug/internal/ast"
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
	s := make(map[string]*Binding)
	return &Environment{Bindings: s, Outer: nil}
}

type Environment struct {
	Bindings   map[string]*Binding
	Outer      *Environment
	Src        string
	Path       string
	ModuleFqn  string
	StackInfo  *StackFrame           // Optional stack frame information
	deferStack []*ast.DeferStatement // Stack for deferred statements
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

func (e *Environment) GatherStackTrace(frame *StackFrame) []StackFrame {
	var trace []StackFrame
	trace = append([]StackFrame{*frame}, trace...)
	for env := e; env != nil; env = env.Outer {
		if env.StackInfo != nil {
			sf := StackFrame{
				Src:      env.Src,
				File:     env.Path,
				Position: env.StackInfo.Position,
				Function: env.StackInfo.Function,
			}
			trace = append([]StackFrame{sf}, trace...) // Prepend to maintain call order
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

func (e *Environment) RegisterDefer(deferStmt *ast.DeferStatement) {
	slog.Debug("Stashing deferred block",
		slog.Any("deferred-statement", deferStmt))
	e.deferStack = append(e.deferStack, deferStmt)
}

// ExecuteDeferred runs deferred statements.
// It takes the current result of the block/function and returns the final result.
// If a deferred statement recovers or throws, the returned object will reflect that.
func (e *Environment) ExecuteDeferred(result Object, evalFunc func(stmt ast.Statement) Object, eqFunc func(Object, Object) bool) Object {
	defer func() { e.deferStack = nil }() // Always clear defer stack

	if e.deferStack == nil || len(e.deferStack) == 0 {
		return result
	}
	slog.Debug("Deferred execution starting",
		slog.Any("pre-result", result))
	currentResult := result

	for i := len(e.deferStack) - 1; i >= 0; i-- {
		ds := e.deferStack[i]

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
				// Scenario 3: Throwing new value (Chain)
				// If the user re-threw the EXACT SAME object, we keep it (Rethrow identity)
				if activeRuntimeErr != nil && newRtErr == activeRuntimeErr {
					currentResult = activeRuntimeErr
				} else {
					// New error, chain it if we had a previous error
					if activeRuntimeErr != nil {
						newRtErr.Cause = activeRuntimeErr
					}
					currentResult = newRtErr
				}
				continue
			}

			// Check for ReturnValue wrapper (Explicit Return)
			// This distinguishes `return x` (explicit) from `x` (implicit block result)
			if isError && ds.Mode == ast.DeferOnError {
				if retVal, ok := deferResult.(*ReturnValue); ok {
					val := retVal.Value
					// We are in error state.
					// Scenario 2: Rethrowing (returning the same payload)
					// If value == errorPayload, treat as rethrow -> keep current error.
					if eqFunc(val, errorPayload) {
						// Rethrow: keep propagating the original RuntimeError
						currentResult = activeRuntimeErr
					} else {
						// Scenario 1: Recovering
						// Explicit return of a different value -> Recover
						currentResult = val
					}
				} else if eqFunc(deferResult, errorPayload) {
					// Rethrow: keep propagating the original RuntimeError
					currentResult = activeRuntimeErr
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
