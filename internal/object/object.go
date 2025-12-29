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
	"sync"
	"unicode/utf8"
)

const (
	NIL_OBJ     = "NIL"
	BOOLEAN_OBJ = "BOOLEAN"
	NUMBER_OBJ  = "NUMBER"
	STRING_OBJ  = "STRING"
	BYTE_OBJ    = "BYTES"

	LIST_OBJ = "LIST"
	MAP_OBJ  = "MAP"

	MODULE_OBJ         = "MODULE"
	FUNCTION_OBJ       = "FUNCTION"
	FUNCTION_GROUP_OBJ = "FUNCTION_GROUP"
	FOREIGN_OBJ        = "FOREIGN"
	ERROR_OBJ          = "ERROR"

	TAIL_CALL_OBJ    = "TAIL_CALL"
	RETURN_VALUE_OBJ = "RETURN_VALUE"
	TASK_HANDLE_OBJ  = "TASK_HANDLE"
)

const (
	IMPORT_TAG   = "@import"
	EXPORT_TAG   = "@export"
	FUNCTION_TAG = "@fun"
)

var TypeTags = map[string]string{
	"@num":       NUMBER_OBJ,
	"@str":       STRING_OBJ,
	"@map":       MAP_OBJ,
	"@list":      LIST_OBJ,
	"@bytes":     BYTE_OBJ,
	"@bool":      BOOLEAN_OBJ,
	FUNCTION_TAG: FUNCTION_OBJ,
}

// EvaluatorContext provides the bridge between native Go code and the interpreter,
// allowing Foreign Function Interface (FFI) implementations to access the current
// execution context and helper methods.
type EvaluatorContext interface {
	CurrentEnv() *Environment
	ApplyFunction(pos int, fnName string, fnObj Object, args []Object) Object
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

// TaskHandle represents a concurrent task spawned via 'spawn'
type TaskHandle struct {
	ID         int64
	Result     Object
	Err        *RuntimeError
	Done       chan struct{} // Closed when the task is finished
	IsFinished bool
	mu         sync.Mutex
}

func (th *TaskHandle) Type() ObjectType { return TASK_HANDLE_OBJ }
func (th *TaskHandle) Inspect() string {
	return fmt.Sprintf("<task %d>", th.ID)
}

// Complete sets the result and signals any waiters
func (th *TaskHandle) Complete(res Object) {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.IsFinished {
		return
	}

	if rtErr, ok := res.(*RuntimeError); ok {
		th.Err = rtErr
	} else {
		th.Result = res
	}

	th.IsFinished = true
	close(th.Done)
}

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
	Limit       ast.Expression
	IsAsync     bool
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
func (fg *FunctionGroup) GetTags() map[string]List {
	result := make(map[string]List)
	for _, function := range fg.Functions {
		if taggableFn, ok := function.(Taggable); ok {
			// Retrieve tags from each taggable function
			for tag, params := range taggableFn.GetTags() {
				// Merge/accumulate tags
				result[tag] = params
			}
		}
	}
	return result

}
func (fg *FunctionGroup) SetTag(tag string, params List) {
	for _, function := range fg.Functions {
		if taggableFn, ok := function.(Taggable); ok {
			taggableFn.SetTag(tag, params)
		}
	}
}
func (fg *FunctionGroup) DispatchToFunction(fnName string, args []Object) (Object, error) {
	slog.Debug("dispatching to function group",
		slog.Any("group-size", len(fg.Functions)),
		slog.Any("parameter-count", len(args)))

	n := len(args)
	var bestMatch Object
	var bestMax = math.MaxInt
	var bestScore = -1
	var foundNonVariadic bool

	for sig, fn := range fg.Functions {
		if n >= sig.Min && n <= sig.Max {
			isVariadic := sig.IsVariadic

			var score = 0
			switch f := fn.(type) {
			case *Function:
				score = evaluateFunctionMatch(f.Parameters, args)
			case *Foreign:
				score = evaluateFunctionMatch(f.Parameters, args)
			}

			//fmt.Printf("----- sig max %d, score: %d\n", sig.Max, score)

			if score >= 0 && sig.Max < bestMax || (sig.Max == bestMax && score > bestScore) ||
				(sig.Max == bestMax && score == bestScore && (!foundNonVariadic || !isVariadic)) {
				bestMatch = fn
				bestMax = sig.Max
				bestScore = score
				foundNonVariadic = !isVariadic
			}
		}
	}

	if bestMatch != nil {
		slog.Debug("best match: %s",
			slog.Any("match", bestMatch.Inspect()))
		//fmt.Printf("-----\nbest match: %s, best max %d, score: %d\n\n", bestMatch.Inspect(), bestMax, bestScore)
		return bestMatch, nil
	} else {
		slog.Debug("no match found",
			slog.Any("parameter-count", n))
	}

	var a strings.Builder
	for i, arg := range args {
		a.WriteString(string(arg.Type()))
		if i < len(args)-1 {
			a.WriteString(", ")
		}
	}
	if fnName == "" {
		fnName = "<anonymous>"
	}
	err := fmt.Sprintf("No suitable function (%s) found to dispatch", a.String())
	return &Error{Message: err}, errors.New(err)
}

func evaluateFunctionMatch(params []*ast.FunctionParameter, args []Object) int {
	score := 0 // Start with zero matches
	for i, param := range params {
		if i >= len(args) {
			break
		}
		arg := args[i]
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

type RuntimeError struct {
	Payload    Object        // The error payload (must be a Map object)
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
	FnName    string
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
