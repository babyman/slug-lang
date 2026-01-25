package object

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/util"
	"strings"
	"unicode/utf8"
)

const (
	NIL_OBJ     = "NIL"
	BOOLEAN_OBJ = "BOOLEAN"
	NUMBER_OBJ  = "NUMBER"
	STRING_OBJ  = "STRING"
	BYTE_OBJ    = "BYTES"

	LIST_OBJ          = "LIST"
	MAP_OBJ           = "MAP"
	STRUCT_SCHEMA_OBJ = "STRUCT_SCHEMA"
	STRUCT_OBJ        = "STRUCT"

	MODULE_OBJ         = "MODULE"
	FUNCTION_OBJ       = "FUNCTION"
	FUNCTION_GROUP_OBJ = "FUNCTION_GROUP"
	FOREIGN_OBJ        = "FOREIGN"
	ERROR_OBJ          = "ERROR"

	TAIL_CALL_OBJ     = "TAIL_CALL"
	RETURN_VALUE_OBJ  = "RETURN_VALUE"
	TASK_HANDLE_OBJ   = "TASK"
	BINDING_REF_OBJ   = "BINDING_REF"
	UNINITIALIZED_OBJ = "UNINITIALIZED"
)

const (
	IMPORT_TAG   = "@import"
	EXPORT_TAG   = "@export"
	FUNCTION_TAG = "@fun"
	TASK_TAG     = "@task"
)

var TypeTags = map[string]string{
	"@num":       NUMBER_OBJ,
	"@str":       STRING_OBJ,
	"@map":       MAP_OBJ,
	"@list":      LIST_OBJ,
	"@bytes":     BYTE_OBJ,
	"@bool":      BOOLEAN_OBJ,
	FUNCTION_TAG: FUNCTION_OBJ,
	TASK_TAG:     TASK_HANDLE_OBJ,
}

var (
	NIL   = &Nil{}
	TRUE  = &Boolean{Value: true}
	FALSE = &Boolean{Value: false}
)

// EvaluatorContext provides the bridge between native Go code and the interpreter,
// allowing Foreign Function Interface (FFI) implementations to access the current
// execution context and helper methods.
type EvaluatorContext interface {
	CurrentEnv() *Environment
	ApplyFunction(pos int, fnName string, fnObj Object, positional []Object, named map[string]Object) Object
	NewError(message string, a ...interface{}) *Error
	Nil() *Nil
	NativeBoolToBooleanObject(input bool) *Boolean
	LoadModule(pathParts string) (*Module, error)
	GetConfiguration() util.Configuration
	NextHandleID() int64
}

type ForeignFunction func(ctx EvaluatorContext, args ...Object) Object

type ObjectType string

type Hashable interface {
	Object
	MapKey() MapKey
}

type Taggable interface {
	Object
	HasTag(tag string) bool
	GetTagParams(tag string) (List, bool)
	GetTags() map[string]List
	SetTag(tag string, params List)
}

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Number struct {
	Tags  map[string]List
	Value dec64.Dec64
}

func (n *Number) Type() ObjectType { return NUMBER_OBJ }
func (n *Number) Inspect() string  { return n.Value.String() }
func (n *Number) MapKey() MapKey {
	return MapKey{Type: n.Type(), Value: uint64(n.Value.ToInt64())}
}
func (n *Number) HasTag(tag string) bool {
	return hasTag(tag, n.Tags)
}
func (n *Number) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, n.Tags)
}
func (n *Number) GetTags() map[string]List {
	return getTags(&n.Tags)
}
func (n *Number) SetTag(tag string, params List) {
	setTag(&n.Tags, tag, params)
}

type Boolean struct {
	Tags  map[string]List
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
func (b *Boolean) HasTag(tag string) bool {
	return hasTag(tag, b.Tags)
}
func (b *Boolean) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, b.Tags)
}
func (b *Boolean) GetTags() map[string]List {
	return getTags(&b.Tags)
}
func (b *Boolean) SetTag(tag string, params List) {
	setTag(&b.Tags, tag, params)
}

type String struct {
	Tags  map[string]List
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }
func (s *String) MapKey() MapKey {
	h := fnv.New64a()
	for _, r := range s.Value {
		var buf [4]byte
		n := utf8.EncodeRune(buf[:], r)
		h.Write(buf[:n])
	}
	return MapKey{Type: s.Type(), Value: h.Sum64()}
}
func (s *String) HasTag(tag string) bool {
	return hasTag(tag, s.Tags)
}
func (s *String) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, s.Tags)
}
func (s *String) GetTags() map[string]List {
	return getTags(&s.Tags)
}
func (s *String) SetTag(tag string, params List) {
	setTag(&s.Tags, tag, params)
}

type Bytes struct {
	Tags  map[string]List
	Value []byte
}

func (b *Bytes) Type() ObjectType { return BYTE_OBJ }
func (b *Bytes) Inspect() string {
	return `0x"` + hex.EncodeToString(b.Value) + `"`
}
func (b *Bytes) MapKey() MapKey {
	h := fnv.New64a()
	h.Write(b.Value)
	return MapKey{Type: b.Type(), Value: h.Sum64()}
}
func (b *Bytes) HasTag(tag string) bool {
	return hasTag(tag, b.Tags)
}
func (b *Bytes) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, b.Tags)
}
func (b *Bytes) GetTags() map[string]List {
	return getTags(&b.Tags)
}
func (b *Bytes) SetTag(tag string, params List) {
	setTag(&b.Tags, tag, params)
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
func (e *Error) Inspect() string  { return e.Message }

// Uninitialized is a sentinel used during two-phase module loading.
// It indicates that a top-level binding name exists (declared) but its value
// has not yet been initialized (executed).
type Uninitialized struct{}

func (u *Uninitialized) Type() ObjectType { return UNINITIALIZED_OBJ }
func (u *Uninitialized) Inspect() string  { return "<uninitialized>" }

// BINDING_UNINITIALIZED is the singleton sentinel instance used by the runtime.
var BINDING_UNINITIALIZED = &Uninitialized{}

// BindingRef is an internal indirection used to model live bindings across module imports.
// It is intentionally invisible at the language level; the evaluator should transparently
// dereference it to the current value of the referenced binding.
type BindingRef struct {
	Env  *Environment
	Name string
}

func (br *BindingRef) Type() ObjectType { return BINDING_REF_OBJ }
func (br *BindingRef) Inspect() string  { return fmt.Sprintf("<bindingref %s>", br.Name) }

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
	ParamIndex  map[string]int
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
func (f *Function) GetTags() map[string]List {
	return getTags(&f.Tags)
}
func (f *Function) SetTag(tag string, params List) {
	setTag(&f.Tags, tag, params)
}

type FunctionGroup struct {
	// Functions holds implementations owned by this group.
	Functions map[ast.FSig]Object
	// Delegates optionally points at other groups whose implementations should be considered
	// during dispatch. This enables composite groups (e.g. merged imports) while keeping
	// live updates when delegate groups are extended.
	Delegates []*FunctionGroup
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

// allGroups returns fg plus any delegated function groups (if present).
func (fg *FunctionGroup) allGroups() []*FunctionGroup {
	if fg.Delegates == nil || len(fg.Delegates) == 0 {
		return []*FunctionGroup{fg}
	}
	groups := make([]*FunctionGroup, 0, 1+len(fg.Delegates))
	groups = append(groups, fg)
	groups = append(groups, fg.Delegates...)
	return groups
}

func (fg *FunctionGroup) HasTag(tag string) bool {
	for _, g := range fg.allGroups() {
		for _, function := range g.Functions {
			if taggableFn, ok := function.(Taggable); ok {
				if taggableFn.HasTag(tag) {
					return true
				}
			}
		}
	}
	return false
}

func (fg *FunctionGroup) GetTagParams(tag string) (List, bool) {
	for _, g := range fg.allGroups() {
		for _, function := range g.Functions {
			if taggableFn, ok := function.(Taggable); ok {
				if v, ok := taggableFn.GetTagParams(tag); ok {
					return v, true
				}
			}
		}
	}
	return List{}, false
}

func (fg *FunctionGroup) GetTags() map[string]List {
	result := make(map[string]List)
	for _, g := range fg.allGroups() {
		for _, function := range g.Functions {
			if taggableFn, ok := function.(Taggable); ok {
				for t, params := range taggableFn.GetTags() {
					result[t] = params
				}
			}
		}
	}
	return result
}

func (fg *FunctionGroup) SetTag(tag string, params List) {
	for _, g := range fg.allGroups() {
		for _, function := range g.Functions {
			if taggableFn, ok := function.(Taggable); ok {
				taggableFn.SetTag(tag, params)
			}
		}
	}
}

type BoundArguments struct {
	Values   []Object
	Provided []bool
}

func bindArgumentsForDispatch(params []*ast.FunctionParameter, positional []Object, named map[string]Object) (*BoundArguments, error) {
	if params == nil {
		if len(named) > 0 {
			return nil, fmt.Errorf("named arguments are not supported for this function")
		}
		provided := make([]bool, len(positional))
		for i := range provided {
			provided[i] = true
		}
		return &BoundArguments{Values: positional, Provided: provided}, nil
	}

	paramCount := len(params)
	values := make([]Object, paramCount)
	provided := make([]bool, paramCount)

	hasVariadic := paramCount > 0 && params[paramCount-1].IsVariadic
	variadicIndex := paramCount - 1

	if len(named) > 0 {
		paramIndex := make(map[string]int, paramCount)
		for i, param := range params {
			paramIndex[param.Name.Value] = i
		}
		for name, val := range named {
			idx, ok := paramIndex[name]
			if !ok {
				return nil, fmt.Errorf("unknown named parameter: %s", name)
			}
			if provided[idx] {
				return nil, fmt.Errorf("duplicate assignment to parameter: %s", name)
			}
			if params[idx].IsVariadic {
				if _, ok := val.(*List); !ok {
					return nil, fmt.Errorf("variadic parameter '%s' must be a list when passed by name", name)
				}
			}
			values[idx] = val
			provided[idx] = true
		}
	}

	posIndex := 0
	if hasVariadic {
		for i := 0; i < variadicIndex; i++ {
			if posIndex >= len(positional) {
				break
			}
			if provided[i] {
				continue
			}
			values[i] = positional[posIndex]
			provided[i] = true
			posIndex++
		}

		remaining := positional[posIndex:]
		if provided[variadicIndex] {
			if len(remaining) > 0 {
				return nil, fmt.Errorf("too many positional arguments")
			}
		} else {
			values[variadicIndex] = &List{Elements: remaining}
			provided[variadicIndex] = true
		}
	} else {
		for i := 0; i < paramCount; i++ {
			if posIndex >= len(positional) {
				break
			}
			if provided[i] {
				continue
			}
			values[i] = positional[posIndex]
			provided[i] = true
			posIndex++
		}
		if posIndex < len(positional) {
			return nil, fmt.Errorf("too many positional arguments")
		}
	}

	for i, param := range params {
		if provided[i] {
			continue
		}
		if param.IsVariadic {
			continue
		}
		if param.Default != nil {
			continue
		}
		return nil, fmt.Errorf("missing required parameter: %s", param.Name.Value)
	}

	return &BoundArguments{Values: values, Provided: provided}, nil
}

func (fg *FunctionGroup) DispatchToFunction(fnName string, positional []Object, named map[string]Object) (Object, error) {
	slog.Debug("dispatching to function group",
		slog.Any("group-size", len(fg.Functions)),
		slog.Any("parameter-count", len(positional)+len(named)))

	n := len(positional) + len(named)
	var bestMatch Object
	var bestMax = math.MaxInt
	var bestScore = -1
	var foundNonVariadic bool
	var firstBindErr error

	for _, g := range fg.allGroups() {
		for sig, fn := range g.Functions {
			if n >= sig.Min && n <= sig.Max {
				isVariadic := sig.IsVariadic

				score := 0
				switch f := fn.(type) {
				case *Function:
					bound, err := bindArgumentsForDispatch(f.Parameters, positional, named)
					if err != nil {
						if firstBindErr == nil {
							firstBindErr = err
						}
						continue
					}
					score = evaluateFunctionMatch(f.Parameters, bound)
				case *Foreign:
					bound, err := bindArgumentsForDispatch(f.Parameters, positional, named)
					if err != nil {
						if firstBindErr == nil {
							firstBindErr = err
						}
						continue
					}
					score = evaluateFunctionMatch(f.Parameters, bound)
				}

				if (score >= 0 && sig.Max < bestMax) ||
					(sig.Max == bestMax && score > bestScore) ||
					(sig.Max == bestMax && score == bestScore && (!foundNonVariadic || !isVariadic)) {
					bestMatch = fn
					bestMax = sig.Max
					bestScore = score
					foundNonVariadic = !isVariadic
				}
			}
		}
	}

	if bestMatch != nil {
		slog.Debug("best match",
			slog.Any("match", bestMatch.Inspect()))
		return bestMatch, nil
	}

	slog.Debug("no match found", slog.Any("parameter-count", n))

	if firstBindErr != nil {
		return &Error{Message: firstBindErr.Error()}, firstBindErr
	}

	var a strings.Builder
	for i, arg := range positional {
		a.WriteString(string(arg.Type()))
		if i < len(positional)-1 {
			a.WriteString(", ")
		}
	}
	if fnName == "" {
		fnName = "<anonymous>"
	}
	err := fmt.Sprintf("No suitable function (%s) found to dispatch", a.String())
	return &Error{Message: err}, errors.New(err)
}

func evaluateFunctionMatch(params []*ast.FunctionParameter, bound *BoundArguments) int {
	score := 0 // Start with zero matches
	for i, param := range params {
		if i >= len(bound.Values) {
			break
		}
		if !bound.Provided[i] {
			continue
		}
		arg := bound.Values[i]
		// Check for matching tags
		for _, tag := range param.Tags {
			if tagType, exists := TypeTags[tag.Name]; exists {
				// special case the function tag since it can match FUNCTION_OBJ and FUNCTION_GROUP_OBJ
				if string(arg.Type()) == tagType || (tag.Name == FUNCTION_TAG && arg.Type() == FUNCTION_GROUP_OBJ) {
					score++
					break
				} else if arg.Type() == NIL_OBJ {
					score++
					break
				} else {
					// we have a type tag and it's not a match
					return -1
				}
			} else if arg.Type() != NIL_OBJ {
				slog.Warn("no match for argument type",
					slog.Any("type", arg.Type()))
			}
		}
	}
	return score
}

type Foreign struct {
	Signature  ast.FSig
	Tags       map[string]List
	Parameters []*ast.FunctionParameter
	ParamIndex map[string]int
	Fn         ForeignFunction
	Name       string
}

func (f *Foreign) Type() ObjectType { return FOREIGN_OBJ }
func (f *Foreign) Inspect() string {
	return "foreign " + f.Name + "(" + fmt.Sprintf("%v", f.Signature) + ") { <native fn> }"
}
func (f *Foreign) HasTag(tag string) bool {
	return hasTag(tag, f.Tags)
}
func (f *Foreign) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, f.Tags)
}
func (f *Foreign) GetTags() map[string]List {
	return getTags(&f.Tags)
}
func (f *Foreign) SetTag(tag string, params List) {
	setTag(&f.Tags, tag, params)
}

type List struct {
	Tags     map[string]List
	Elements []Object
}

func (l *List) Type() ObjectType { return LIST_OBJ }
func (l *List) Inspect() string {
	var out bytes.Buffer

	elements := []string{}
	for _, e := range l.Elements {
		elements = append(elements, e.Inspect())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}
func (l *List) HasTag(tag string) bool {
	return hasTag(tag, l.Tags)
}
func (l *List) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, l.Tags)
}
func (l *List) GetTags() map[string]List {
	return getTags(&l.Tags)
}
func (l *List) SetTag(tag string, params List) {
	setTag(&l.Tags, tag, params)
}

type MapKey struct {
	Type  ObjectType
	Value uint64
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

	tags := []string{}
	for k, tag := range m.Tags {
		tags = append(tags, fmt.Sprintf("%v: %v",
			k, tag.Inspect()))
	}

	pairs := []string{}
	for _, pair := range m.Pairs {
		pairs = append(pairs, fmt.Sprintf("%s: %s",
			pair.Key.Inspect(), pair.Value.Inspect()))
	}

	out.WriteString(strings.Join(tags, " "))
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
func (m *Map) PutPair(k MapKey, v MapPair) *Map {
	if m.Pairs == nil {
		m.Pairs = map[MapKey]MapPair{}
	}
	m.Pairs[k] = v
	return m
}
func (m *Map) Get(k Hashable) (Object, bool) {
	pair, ok := m.Pairs[k.MapKey()]
	return pair.Value, ok
}
func (m *Map) HasTag(tag string) bool {
	return hasTag(tag, m.Tags)
}
func (m *Map) GetTagParams(tag string) (List, bool) {
	return getTagParams(tag, m.Tags)
}
func (m *Map) GetTags() map[string]List {
	return getTags(&m.Tags)
}
func (m *Map) SetTag(tag string, params List) {
	setTag(&m.Tags, tag, params)
}

type StructSchemaField struct {
	Name    string
	Default ast.Expression
	Hint    string
}

type StructSchema struct {
	Name       string
	Fields     []StructSchemaField
	FieldIndex map[string]int
	Env        *Environment
}

func (s *StructSchema) Type() ObjectType { return STRUCT_SCHEMA_OBJ }
func (s *StructSchema) Inspect() string {
	var out bytes.Buffer
	if s.Name != "" {
		out.WriteString(s.Name)
		out.WriteString(" ")
	}
	out.WriteString("struct {")
	parts := []string{}
	for _, field := range s.Fields {
		var b strings.Builder
		if field.Hint != "" {
			b.WriteString(field.Hint)
			b.WriteString(" ")
		}
		b.WriteString(field.Name)
		if field.Default != nil {
			b.WriteString(" = ")
			b.WriteString(field.Default.String())
		}
		parts = append(parts, b.String())
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("}")
	return out.String()
}

type StructValue struct {
	Schema *StructSchema
	Fields map[string]Object
}

func (s *StructValue) Type() ObjectType { return STRUCT_OBJ }
func (s *StructValue) Inspect() string {
	var out bytes.Buffer
	if s.Schema != nil && s.Schema.Name != "" {
		out.WriteString(s.Schema.Name)
	} else {
		out.WriteString("struct")
	}
	out.WriteString(" {")
	parts := []string{}
	if s.Schema != nil {
		for _, field := range s.Schema.Fields {
			val, ok := s.Fields[field.Name]
			if !ok {
				continue
			}
			parts = append(parts, field.Name+": "+val.Inspect())
		}
	}
	out.WriteString(strings.Join(parts, ", "))
	out.WriteString("}")
	return out.String()
}

type RuntimeError struct {
	Payload    Object
	StackTrace []*StackFrame // Stack frames for error propagation
	Cause      *RuntimeError // The error that caused this error (for chaining)
}

func (re *RuntimeError) Type() ObjectType { return ERROR_OBJ }
func (re *RuntimeError) Inspect() string {
	return RenderStacktrace(re)
}

type StackFrame struct {
	Function string
	File     string
	Src      string
	Position int
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
	FnName         string
	Function       Object
	Arguments      []Object
	NamedArguments map[string]Object
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

	if len(tc.NamedArguments) > 0 {
		if len(args) > 0 {
			out.WriteString(", ")
		}
		named := []string{}
		for name, arg := range tc.NamedArguments {
			named = append(named, name+" = "+arg.Inspect())
		}
		out.WriteString(strings.Join(named, ", "))
	}

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

// Helper to retrieve all tags of a Taggable object
func getTags(tags *map[string]List) map[string]List {
	if *tags == nil {
		*tags = make(map[string]List)
	}
	return *tags
}

// Helper to set (add/update) a tag for a Taggable object
func setTag(tags *map[string]List, tag string, params List) {
	if *tags == nil {
		*tags = make(map[string]List)
	}
	(*tags)[tag] = params
}
