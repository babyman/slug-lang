package object

import (
	"fmt"
	"log/slog"
	"runtime"
	"slug/internal/ast"
	"sync"
	"sync/atomic"
)

var nextID atomic.Uint64

type Environment struct {
	ID        uint64
	Bindings  map[string]*Binding
	Outer     *Environment
	Src       string
	Path      string
	ModuleFqn string
	StackInfo *StackFrame           // Optional stack frame information
	Defers    []*ast.DeferStatement // Stack for deferred statements

	Limit                int
	IsThreadNurseryScope bool // marks a scope that can own spawned tasks

	mu sync.RWMutex
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

func nextEnvID() uint64 {
	return nextID.Add(1) // <<16 | int64(rand.Intn(0xFFFF))
}

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
		ID:       nextEnvID(),
		Bindings: make(map[string]*Binding),
		Defers:   make([]*ast.DeferStatement, 0),
	}
}

// NewRootEnvironment creates the base environment with a system-wide concurrency limit
func NewRootEnvironment(limit int) *Environment {
	slog.Debug("------ new root env ------\n",
		slog.Int("concurrency-limit", limit),
		slog.Int("gomaxprocs", runtime.GOMAXPROCS(0)),
	)
	env := NewEnvironment()
	env.IsThreadNurseryScope = true // root acts like a nursery scope for spawn ownership
	env.Limit = limit
	return env
}

func (e *Environment) ResetForTCO() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Bindings = make(map[string]*Binding)
	e.Defers = nil
}

func (e *Environment) ShallowCopy() *Environment {
	e.mu.RLock()
	defer e.mu.RUnlock()

	newEnv := &Environment{
		ID:        nextEnvID(),
		Bindings:  make(map[string]*Binding, len(e.Bindings)),
		Outer:     e.Outer,
		Src:       e.Src,
		Path:      e.Path,
		ModuleFqn: e.ModuleFqn,
	}

	for k, v := range e.Bindings {
		newEnv.Bindings[k] = v
	}

	return newEnv
}

func (e *Environment) GetBinding(name string) (*Binding, bool) {
	e.mu.RLock()
	binding, ok := e.Bindings[name]
	e.mu.RUnlock()

	if ok {
		return binding, true
	}
	if e.Outer != nil {
		return e.Outer.GetBinding(name)
	}
	return nil, false
}

// GetLocalBinding returns a binding from this environment only (it does not walk outers).
// This is useful for module-level binding references which should not be affected by shadowing.
func (e *Environment) GetLocalBinding(name string) (*Binding, bool) {
	e.mu.RLock()
	binding, ok := e.Bindings[name]
	e.mu.RUnlock()
	return binding, ok
}

// GetLocalBindingValue returns the current value of a local binding under a read lock.
// It does not walk outers. The returned *Binding is the same instance stored in the env.
func (e *Environment) GetLocalBindingValue(name string) (Object, *Binding, bool) {
	e.mu.RLock()
	binding, ok := e.Bindings[name]
	if !ok {
		e.mu.RUnlock()
		return nil, nil, false
	}
	val := binding.Value
	e.mu.RUnlock()
	return val, binding, true
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
	e.mu.Lock()
	defer e.mu.Unlock()

	declaration := "val"
	if isMutable {
		declaration = "var"
	}

	binding, exists := e.Bindings[name]
	if exists && !binding.IsMutable {
		// Allow the second phase of two-pass module loading to initialize a predeclared name.
		// Predeclare binds names to BINDING_UNINITIALIZED; the later `val/var` should set it once.
		if binding.Value == BINDING_UNINITIALIZED {
			// ok: initialization, not reassignment
		} else if binding.Meta.IsImport {
			// devx: allow locals to override imported bindings (warn instead of error).
			slog.Warn("imported name shadowed by local definition",
				slog.String("name", name),
				slog.String("module", e.ModuleFqn),
			)
		} else {
			return nil, fmt.Errorf("%s `%s` is already defined as a 'val' and cannot be reassigned", declaration, name)
		}
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

	var typ ObjectType = "<nil>"
	if binding.Value != nil {
		typ = binding.Value.Type()
	}

	slog.Debug("binding value",
		slog.Any("type", typ),
		slog.Any("name", name),
		slog.Any("meta", binding.Meta))
	return val, nil
}

func (e *Environment) Assign(name string, val Object) (Object, error) {
	e.mu.Lock()
	binding, exists := e.Bindings[name]
	if exists {
		defer e.mu.Unlock()
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
	e.mu.Unlock()

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
