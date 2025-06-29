package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"math"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/log"
	"strings"
)

const (
	NIL_OBJ     = "NIL"
	BOOLEAN_OBJ = "BOOLEAN"
	NUMBER_OBJ  = "NUMBER"
	STRING_OBJ  = "STRING"

	LIST_OBJ = "LIST"
	MAP_OBJ  = "MAP"

	MODULE_OBJ         = "MODULE"
	FUNCTION_OBJ       = "FUNCTION"
	FUNCTION_GROUP_OBJ = "FUNCTION_GROUP"
	FOREIGN_OBJ        = "FOREIGN"
	ERROR_OBJ          = "ERROR"

	TAIL_CALL_OBJ    = "TAIL_CALL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
)

// EvaluatorContext provides the bridge between native Go code and the interpreter,
// allowing Foreign Function Interface (FFI) implementations to access the current
// execution context and helper methods.
type EvaluatorContext interface {
	CurrentEnv() *Environment
	PID() int64
	Receive(timeout int64) (Object, bool)
	NewError(message string, a ...interface{}) *Error
	Nil() *Nil
	NativeBoolToBooleanObject(input bool) *Boolean
	LoadModule(pathParts []string) (*Module, error)
}

type ForeignFunction func(ctx EvaluatorContext, args ...Object) Object

type ObjectType string

type MapKey struct {
	Type  ObjectType
	Value uint64
}

type Hashable interface {
	Object
	MapKey() MapKey
}

type Taggable interface {
	Object
	HasTag(tag string) bool
	GetTagParams(tag string) (List, bool)
}

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Number struct {
	Value dec64.Dec64
}

func (i *Number) Type() ObjectType { return NUMBER_OBJ }
func (i *Number) Inspect() string  { return i.Value.String() }
func (i *Number) MapKey() MapKey {
	return MapKey{Type: i.Type(), Value: uint64(i.Value.ToInt64())}
}

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }
func (b *Boolean) MapKey() MapKey {
	var value uint64

	if b.Value {
		value = 1
	} else {
		value = 0
	}

	return MapKey{Type: b.Type(), Value: value}
}

type Nil struct{}

func (n *Nil) Type() ObjectType { return NIL_OBJ }
func (n *Nil) Inspect() string  { return "nil" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Module struct {
	Name    string
	Path    string
	Src     string
	Env     *Environment
	Program *ast.Program
}

func (m *Module) Type() ObjectType { return MODULE_OBJ }

func (m *Module) Inspect() string {
	var out bytes.Buffer
	out.WriteString("module ")
	out.WriteString(m.Name)
	out.WriteString(" {")
	for key, val := range m.Env.Bindings {
		out.WriteString(fmt.Sprintf("\n  %s: %s,", key, val.Value.Inspect()))
	}
	out.WriteString("\n}")
	return out.String()
}

type Function struct {
	Signature   ast.FSig
	Tags        map[string]List
	Parameters  []*ast.FunctionParameter
	Body        *ast.BlockStatement
	Env         *Environment
	HasTailCall bool
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}

	out.WriteString("fn")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")

	return out.String()
}
func (f *Function) HasTag(tag string) bool {
	return hasTag(tag, f.Tags)
}
func (f *Function) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, f.Tags)
}

type FunctionGroup struct {
	Functions map[ast.FSig]Object
}

func (fg *FunctionGroup) Type() ObjectType { return FUNCTION_GROUP_OBJ }
func (fg *FunctionGroup) Inspect() string {
	var out bytes.Buffer
	out.WriteString("function group: [")
	entries := []string{}
	for signature, fn := range fg.Functions {
		entries = append(entries, fmt.Sprintf("%v => %s", signature, fn.Inspect()))
	}
	out.WriteString(strings.Join(entries, ", "))
	out.WriteString("]")
	return out.String()
}
func (fg *FunctionGroup) DispatchToFunction(args []Object) (Object, bool) {
	log.Info("dispatching to function group size: %d, param count %d", len(fg.Functions), len(args))

	n := len(args)
	var bestMatch Object
	var bestMax = math.MaxInt
	var foundNonVariadic bool

	for sig, fn := range fg.Functions {
		if n >= sig.Min && n <= sig.Max {
			isVariadic := sig.IsVariadic
			if sig.Max < bestMax || bestMatch == nil {
				bestMatch = fn
				bestMax = sig.Max
				foundNonVariadic = !isVariadic
			} else if sig.Max == bestMax {
				// Prefer non-variadic if Max values are equal
				if foundNonVariadic && isVariadic {
					continue
				}
				if !foundNonVariadic {
					bestMatch = fn
					foundNonVariadic = !isVariadic
				}
			}
		}
	}

	if bestMatch != nil {
		log.Debug("best match: %s", bestMatch.Inspect())
		return bestMatch, true
	} else {
		log.Debug("no match found for %v params", n)
	}

	return &Error{Message: "No suitable function found to dispatch"}, false
}
func (fg *FunctionGroup) HasTag(tag string) bool {
	for _, function := range fg.Functions {
		switch fn := function.(type) {
		case Taggable:
			if fn.HasTag(tag) {
				return true
			}
		}
	}
	return false
}
func (fg *FunctionGroup) GetTagParams(tag string) (List, bool) {
	for _, function := range fg.Functions {
		switch fn := function.(type) {
		case Taggable:
			v, ok := fn.GetTagParams(tag)
			if ok {
				return v, ok
			}
		}
	}
	return List{}, false
}

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) MapKey() MapKey {
	h := fnv.New64a()
	h.Write([]byte(s.Value))
	return MapKey{Type: s.Type(), Value: h.Sum64()}
}

type Foreign struct {
	Signature ast.FSig
	Tags      map[string]List
	Fn        ForeignFunction
	Name      string
	Arity     int
}

func (f *Foreign) Type() ObjectType { return FOREIGN_OBJ }
func (f *Foreign) Inspect() string {
	return "foreign " + f.Name + "(" + fmt.Sprintf("%d", f.Arity) + ") { <native fn> }"
}
func (f *Foreign) HasTag(tag string) bool {
	return hasTag(tag, f.Tags)
}
func (f *Foreign) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, f.Tags)
}

type List struct {
	Elements []Object
}

func (ao *List) Type() ObjectType { return LIST_OBJ }
func (ao *List) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range ao.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

type MapPair struct {
	Key   Object
	Value Object
}

type Map struct {
	Tags  map[string]List
	Pairs map[MapKey]MapPair
}

func (m *Map) Type() ObjectType { return MAP_OBJ }
func (m *Map) Inspect() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, pair := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			pair.Key.Inspect(), pair.Value.Inspect()))
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// Put simplify adding objects to a map
func (m *Map) Put(k Hashable, v Object) *Map {
	if m.Pairs == nil {
		m.Pairs = map[MapKey]MapPair{}
	}
	m.Pairs[k.MapKey()] = MapPair{
		Key:   k,
		Value: v,
	}
	return m
}
func (m *Map) HasTag(tag string) bool {
	return hasTag(tag, m.Tags)
}
func (m *Map) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, m.Tags)
}

type RuntimeError struct {
	Payload    Object       // The error payload (must be a Map object)
	StackTrace []StackFrame // Stack frames for error propagation
}

func (re *RuntimeError) Type() ObjectType { return ERROR_OBJ }
func (re *RuntimeError) Inspect() string {
	var out bytes.Buffer
	out.WriteString("RuntimeError: ")
	out.WriteString(re.Payload.Inspect())
	out.WriteString("\nStack trace:")
	for _, frame := range re.StackTrace {
		out.WriteString(fmt.Sprintf("\n at [%3d:%2d] %-8s - %s", frame.Line, frame.Col, frame.Function, frame.File))
	}
	return out.String()
}

type StackFrame struct {
	Function string
	File     string
	Position int
	Line     int // lazy populated line number
	Col      int // lazy populated column number
}

type Slice struct {
	Start Object
	End   Object
	Step  Object
}

func (s *Slice) Type() ObjectType { return "SLICE" }
func (s *Slice) Inspect() string {
	var out bytes.Buffer
	if s.Start != nil {
		out.WriteString(s.Start.Inspect())
	}
	out.WriteString(":")
	if s.End != nil {
		out.WriteString(s.End.Inspect())
	}
	if s.Step != nil {
		out.WriteString(":")
		out.WriteString(s.Step.Inspect())
	}
	return out.String()
}

// TailCall is a special object used for tail call optimization
type TailCall struct {
	Function  Object
	Arguments []Object
}

func (tc *TailCall) Type() ObjectType { return TAIL_CALL_OBJ }
func (tc *TailCall) Inspect() string {
	var out bytes.Buffer
	out.WriteString("tailcall(")
	out.WriteString(tc.Function.Inspect())
	out.WriteString(", [")

	args := []string{}
	for _, arg := range tc.Arguments {
		args = append(args, arg.Inspect())
	}
	out.WriteString(strings.Join(args, ", "))

	out.WriteString("])")
	return out.String()
}

func hasTag(tag string, tags map[string]List) bool {
	if tags == nil {
		return false
	}
	_, exists := tags[tag]
	return exists
}

func getTagParams(tag string, tags map[string]List) (List, bool) {
	if tags == nil {
		return List{}, false
	}
	list, ok := tags[tag]
	return list, ok
}
