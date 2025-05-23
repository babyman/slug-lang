package object

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"slug/internal/ast"
	"strings"
)

type ForeignFunction func(args ...Object) Object

type ObjectType string

const (
	NIL_OBJ   = "NIL"
	ERROR_OBJ = "ERROR"

	INTEGER_OBJ = "INTEGER"
	BOOLEAN_OBJ = "BOOLEAN"
	STRING_OBJ  = "STRING"

	RETURN_VALUE_OBJ = "RETURN_VALUE"
	TAIL_CALL_OBJ    = "TAIL_CALL"

	MODULE_OBJ   = "MODULE"
	FUNCTION_OBJ = "FUNCTION"
	FOREIGN_OBJ  = "BUILTIN"

	LIST_OBJ = "LIST"
	MAP_OBJ  = "MAP"
)

type MapKey struct {
	Type  ObjectType
	Value uint64
}

type Hashable interface {
	Object
	MapKey() MapKey
}

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }
func (i *Integer) MapKey() MapKey {
	return MapKey{Type: i.Type(), Value: uint64(i.Value)}
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
	Name    string // Module name/namespace (e.g., `Arithmetic` or an alias)
	Path    string
	Src     string
	Env     *Environment // Module-scoped environment with variables and functions
	Program *ast.Program
}

func (m *Module) Type() ObjectType { return MODULE_OBJ }

func (m *Module) Inspect() string {
	var out bytes.Buffer
	out.WriteString("module ")
	out.WriteString(m.Name)
	out.WriteString(" {")
	for key, val := range m.Env.Store {
		out.WriteString(fmt.Sprintf("\n  %s: %s,", key, val.Value.Inspect()))
	}
	out.WriteString("\n}")
	return out.String()
}

type Function struct {
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
	Fn    ForeignFunction
	Name  string
	Arity int
}

func (b *Foreign) Type() ObjectType { return FOREIGN_OBJ }
func (b *Foreign) Inspect() string {
	return "foreign " + b.Name + "(" + fmt.Sprintf("%d", b.Arity) + ") { <native fn> }"
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
	Pairs map[MapKey]MapPair
}

func (h *Map) Type() ObjectType { return MAP_OBJ }
func (h *Map) Inspect() string {
	var out bytes.Buffer

	pairs := []string{}
	for _, pair := range h.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			pair.Key.Inspect(), pair.Value.Inspect()))
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

// Put simplify adding objects to a map
func (h *Map) Put(k Hashable, v Object) *Map {
	if h.Pairs == nil {
		h.Pairs = map[MapKey]MapPair{}
	}
	h.Pairs[k.MapKey()] = MapPair{
		Key:   k,
		Value: v,
	}
	return h
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
