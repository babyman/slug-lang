package runtime

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/foreign"
	"slug/internal/object"
	"slug/internal/token"
	"slug/internal/util"
	"strings"
	"sync"
	"time"
)

const (
	precision        = 14
	roundingStrategy = dec64.RoundHalfEven
)

type ByteOp func(a, b byte) byte

func AndBytes(a, b byte) byte { return a & b }
func OrBytes(a, b byte) byte  { return a | b }
func XorBytes(a, b byte) byte { return a ^ b }

type Task struct {
	ID           int64
	Runtime      *Runtime
	OwnerNursery *NurseryScope
	Result       object.Object
	Err          *object.RuntimeError
	Done         chan struct{} // Closed when the task is finished
	Observed     bool
	IsFinished   bool
	mu           sync.Mutex

	envStack     []*object.Environment // Environment stack encapsulated in an evaluator struct
	nurseryStack []*NurseryScope
	// callStack keeps track of the current function for things like `recur`
	callStack []struct {
		FnName string
		FnObj  object.Object
	}
}

func (e *Task) NextHandleID() int64 {
	return e.Runtime.NextHandleID()
}

func (e *Task) GetConfiguration() util.Configuration {
	return e.Runtime.Config
}

func (e *Task) Nil() *object.Nil {
	return object.NIL
}

func (e *Task) PushEnv(env *object.Environment) {
	if env.IsThreadNurseryScope {
		e.PushNurseryScope(&NurseryScope{
			Limit: make(chan struct{}, env.Limit),
		})
	}
	e.envStack = append(e.envStack, env)
	slog.Debug("push stack frame",
		slog.Int("stack-size", len(e.envStack)))
}

func (e *Task) CurrentEnv() *object.Environment {
	// Access the current environment from the top frame
	if len(e.envStack) == 0 {
		panic("Environment stack is empty in the current frame")
	}
	return e.envStack[len(e.envStack)-1]
}

func (e *Task) CurrentEnvStackSize() int {
	return len(e.envStack)
}

func (e *Task) PopEnv(result object.Object) object.Object {
	if len(e.envStack) == 0 {
		panic("Attempted to pop from an empty environment stack")
	}

	currentEnv := e.CurrentEnv()
	nurseryInjected := false

	if currentEnv.IsThreadNurseryScope {
		result, nurseryInjected = e.popNurseryScope(result)
	}

	// Execute deferred statements (cleanups)
	finalResult := currentEnv.ExecuteDeferred(result, func(stmt ast.Statement) object.Object {
		return e.Eval(stmt)
	})

	e.envStack = e.envStack[:len(e.envStack)-1]
	slog.Debug("pop stack frame",
		slog.Int("stack-size", len(e.envStack)),
		slog.Bool("nursery-injected", nurseryInjected),
	)

	return finalResult
}

func (e *Task) PushNurseryScope(scope *NurseryScope) {
	e.nurseryStack = append(e.nurseryStack, scope)
}

func (e *Task) currentNurseryScope() *NurseryScope {
	if len(e.nurseryStack) == 0 {
		panic("Nursery stack is empty in the current frame")
	}
	return e.nurseryStack[len(e.nurseryStack)-1]
}

func (e *Task) popNurseryScope(result object.Object) (object.Object, bool) {
	currentScope := e.currentNurseryScope()
	nurseryInjected := false

	// If we are exiting early (return or error), cancel children downward.
	switch result.(type) {
	case *object.ReturnValue:
		currentScope.CancelChildren(nil, nil, "parent scope exited early")
	case *object.RuntimeError:
		currentScope.CancelChildren(nil, result.(*object.RuntimeError), "parent scope failed")
	case *object.Error:
		currentScope.CancelChildren(nil, nil, "parent scope failed")
	}

	// Wait for children (join nursery)
	currentScope.WaitChildren()

	// If any child failed and the current result isn't already an error/return, propagate it upward.
	if currentScope.NurseryErr != nil {
		if result == nil || (result.Type() != object.ERROR_OBJ && result.Type() != object.RETURN_VALUE_OBJ) {
			result = currentScope.NurseryErr
			currentScope.NurseryErr = nil
			nurseryInjected = true
		}
	}

	e.nurseryStack = e.nurseryStack[:len(e.nurseryStack)-1]

	return result, nurseryInjected
}

// Helpers for tracking the current function (for `recur`)
func (e *Task) pushCallFrame(fnName string, fnObj object.Object) {
	e.callStack = append(e.callStack, struct {
		FnName string
		FnObj  object.Object
	}{FnName: fnName, FnObj: fnObj})
}

func (e *Task) currentCallFrame() (string, object.Object, bool) {
	if len(e.callStack) == 0 {
		return "", nil, false
	}
	top := e.callStack[len(e.callStack)-1]
	return top.FnName, top.FnObj, true
}

func (e *Task) popCallFrame() {
	if len(e.callStack) == 0 {
		return
	}
	e.callStack = e.callStack[:len(e.callStack)-1]
}

func (e *Task) Eval(node ast.Node) object.Object {
	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return e.evalProgram(node)

	case *ast.BlockStatement:
		return e.evalBlockStatement(node)

	case *ast.ExpressionStatement:
		return e.Eval(node.Expression)

	case *ast.ReturnStatement:
		val := e.Eval(node.ReturnValue)
		if e.isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.MatchExpression:
		return e.evalMatchExpression(node)

	case *ast.VarExpression:
		variable := e.Eval(node.Value)
		if e.isError(variable) {
			return variable
		}
		isExported := hasExportTag(node.Tags)
		if _, err := e.patternMatches(node.Pattern, variable, false, isExported, false, e.CurrentEnv()); err != nil {
			return e.newErrorWithPos(node.Token.Position, err.Error())
		}
		return e.applyTagsIfPresent(node.Tags, variable)

	case *ast.ValExpression:
		value := e.Eval(node.Value)
		if e.isError(value) {
			return value
		}
		isExported := hasExportTag(node.Tags)
		if _, err := e.patternMatches(node.Pattern, value, true, isExported, false, e.CurrentEnv()); err != nil {
			return e.newErrorWithPos(node.Token.Position, err.Error())
		}
		return e.applyTagsIfPresent(node.Tags, value)

	case *ast.ForeignFunctionDeclaration:
		return e.evalForeignFunctionDeclaration(node)

	// Expressions
	case *ast.NumberLiteral:
		return &object.Number{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.BytesLiteral:
		return &object.Bytes{Value: node.Value}

	case *ast.Boolean:
		return e.NativeBoolToBooleanObject(node.Value)

	case *ast.Nil:
		return object.NIL

	case *ast.PrefixExpression:
		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}
		return e.evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		// Special case for assignment
		if node.Operator == "=" {
			// Ensure left side is an identifier
			ident, ok := node.Left.(*ast.Identifier)
			if !ok {
				return e.newErrorWithPos(node.Token.Position, "left side of assignment must be an identifier")
			}

			// Evaluate right side
			right := e.Eval(node.Right)
			if e.isError(right) {
				return right
			}

			// Try to assign the value (variable is already defined)
			val, err := e.CurrentEnv().Assign(ident.Value, right)
			if err != nil {
				return e.newErrorWithPos(node.Token.Position, err.Error())
			}

			return val
		}

		// Regular infix expressions
		left := e.Eval(node.Left)
		if e.isError(left) {
			return left
		}

		// Short circuit for boolean operations
		if node.Operator == "&&" || node.Operator == "||" {
			return e.evalShortCircuitInfixExpression(left, node)
		}

		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}

		return e.evalInfixExpression(node.Operator, left, right)

	case *ast.IfExpression:
		return e.evalIfExpression(node)

	case *ast.Identifier:
		return e.evalIdentifier(node)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body

		return &object.Function{
			Parameters:  params,
			ParamIndex:  buildParamIndex(params),
			Env:         e.CurrentEnv(),
			Body:        body,
			Signature:   node.Signature,
			HasTailCall: node.HasTailCall,
		}

	case *ast.CallExpression:
		function := e.Eval(node.Function)
		if e.isError(function) {
			return function
		}

		positional, named, err := e.evalCallArguments(node.Token.Position, node.Arguments)
		if err != nil {
			return err
		}

		// If this is a tail call, wrap it in a TailCall object instead of evaluating
		if node.IsTailCall {
			slog.Debug("Tail call",
				slog.Any("function", node.Token.Literal),
				slog.Any("argument-count", len(positional)+len(named)))

			return &object.TailCall{
				FnName:         node.Token.Literal,
				Function:       function,
				Arguments:      positional,
				NamedArguments: named,
			}
		}

		slog.Debug("Function call",
			slog.Any("function", node.Token.Literal),
			slog.Any("argument-count", len(positional)+len(named)))
		// For non-tail calls, invoke the function directly
		return e.ApplyFunction(node.Token.Position, node.Token.Literal, function, positional, named)

	case *ast.RecurExpression:
		// Evaluate arguments (respecting spread and named args, same as call)
		positional, named, err := e.evalCallArguments(node.Token.Position, node.Arguments)
		if err != nil {
			return err
		}

		fnName, fnObj, ok := e.currentCallFrame()
		if !ok || fnObj == nil {
			// `recur` should only be valid inside a function body;
			// semantic checks should normally prevent this, but guard at runtime too.
			return e.newErrorWithPos(node.Token.Position, "recur used outside of a function")
		}

		slog.Debug("Tail recur",
			slog.Any("function", fnName),
			slog.Any("argument-count", len(positional)+len(named)))

		// Map directly to TailCall for the current function
		return &object.TailCall{
			FnName:         fnName,
			Function:       fnObj,
			Arguments:      positional,
			NamedArguments: named,
		}

	case *ast.ListLiteral:
		elements := e.evalExpressions(node.Elements)
		if len(elements) == 1 && e.isError(elements[0]) {
			return elements[0]
		}
		return &object.List{Elements: elements}

	case *ast.StructSchemaExpression:
		return e.evalStructSchemaExpression(node)

	case *ast.IndexExpression:
		left := e.Eval(node.Left)
		if e.isError(left) {
			return left
		}
		left = e.resolveValue(node.Token.Position, left)
		if e.isError(left) {
			return left
		}
		index := e.Eval(node.Index)
		if e.isError(index) {
			return index
		}
		return e.evalIndexExpression(node.Token.Position, left, index)

	case *ast.SliceExpression:
		return e.evalSliceExpression(node)

	case *ast.MapLiteral:
		return e.evalMapLiteral(node)

	case *ast.StructInitExpression:
		return e.evalStructInitExpression(node)

	case *ast.StructCopyExpression:
		return e.evalStructCopyExpression(node)

	case *ast.ThrowStatement:
		return e.evalThrowStatement(node)

	case *ast.DeferStatement:
		return e.evalDefer(node)

	case *ast.SpawnExpression:
		return e.evalSpawnExpression(node)

	case *ast.AwaitExpression:
		return e.evalAwaitExpression(node)
	}

	return nil
}

func (e *Task) evalProgram(program *ast.Program) object.Object {
	//println("program")
	var result object.Object

	for _, statement := range program.Statements {
		result = e.Eval(statement)

		for {
			if returnVal, ok := result.(*object.TailCall); ok {
				result = e.ApplyFunction(0, returnVal.FnName, returnVal.Function, returnVal.Arguments, returnVal.NamedArguments)
			} else if returnVal, ok := result.(*object.ReturnValue); ok {
				rv, ok := returnVal.Value.(*object.TailCall)
				if ok {
					result = rv
				} else {
					break
				}
			} else {
				break
			}
		}

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.RuntimeError:
			return result
		case *object.Error:
			return result
		}
	}

	return result
}

func (e *Task) LoadModule(modName string) (*object.Module, error) {
	return e.Runtime.LoadModule(modName)
}

func (e *Task) mapIdentifiersToStrings(identifiers []*ast.Identifier) []string {
	parts := []string{}
	for _, id := range identifiers {
		parts = append(parts, id.Value)
	}
	return parts
}

func (e *Task) newBlockEnv(block *ast.BlockStatement) *object.Environment {
	// Create a new environment with an associated stack frame
	blockEnv := object.NewEnclosedEnvironment(e.CurrentEnv(), &object.StackFrame{
		Function: "block",
		File:     e.CurrentEnv().Path,
		Src:      e.CurrentEnv().Src,
		Position: block.Token.Position,
	})

	if block.IsNursery {
		blockEnv.IsThreadNurseryScope = true
		if block.Limit != nil {
			limitVal := e.Eval(block.Limit)
			if num, ok := limitVal.(*object.Number); ok {
				blockEnv.Limit = num.Value.ToInt()
			}
		} else {
			blockEnv.Limit = e.Runtime.Config.DefaultLimit
		}
	}

	return blockEnv
}

func (e *Task) evalBlockStatement(block *ast.BlockStatement) (result object.Object) {

	blockEnv := e.newBlockEnv(block)
	e.PushEnv(blockEnv)

	result = e.evalBlockStatementWithinEnv(block)
	return e.PopEnv(result)
}

func (e *Task) evalBlockStatementWithinEnv(block *ast.BlockStatement) (result object.Object) {

	result = object.NIL

	for _, statement := range block.Statements {

		result = e.Eval(statement)

		if result != nil {
			//println("stmt", result.Type())
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	if result != nil {
		return result
	}
	return object.NIL
}

func (e *Task) NativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return object.TRUE
	}
	return object.FALSE
}

func (e *Task) evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return e.evalBangOperatorExpression(right)
	case "-":
		return e.evalMinusPrefixOperatorExpression(right)
	case "~":
		return e.evalComplementPrefixOperatorExpression(right)
	default:
		// todo position
		return e.newErrorf("unknown operator: %s%s", operator, right.Type())
	}
}

func (e *Task) evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.NUMBER_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.evalNumberInfixExpression(operator, left, right)

	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return e.evalStringInfixExpression(operator, left, right)

	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return e.evalBooleanInfixExpression(operator, left, right)

	case operator == "+:" && right.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)
	case operator == ":+" && left.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)
	case left.Type() == object.LIST_OBJ && right.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)

	case operator == "+:" && right.Type() == object.BYTE_OBJ && left.Type() == object.NUMBER_OBJ:
		return e.evalBytesInfixExpression(operator, left, right)
	case operator == ":+" && left.Type() == object.BYTE_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.evalBytesInfixExpression(operator, left, right)
	case left.Type() == object.BYTE_OBJ && right.Type() == object.BYTE_OBJ:
		return e.evalBytesInfixExpression(operator, left, right)
	case operator == "&" && left.Type() == object.BYTE_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.doOp(right, left, AndBytes)
	case operator == "&" && right.Type() == object.BYTE_OBJ && left.Type() == object.NUMBER_OBJ:
		return e.doOp(left, right, AndBytes)
	case operator == "|" && left.Type() == object.BYTE_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.doOp(right, left, OrBytes)
	case operator == "|" && right.Type() == object.BYTE_OBJ && left.Type() == object.NUMBER_OBJ:
		return e.doOp(left, right, OrBytes)
	case operator == "^" && left.Type() == object.BYTE_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.doOp(right, left, XorBytes)
	case operator == "^" && right.Type() == object.BYTE_OBJ && left.Type() == object.NUMBER_OBJ:
		return e.doOp(left, right, XorBytes)

	case operator == "==":
		return e.NativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return e.NativeBoolToBooleanObject(left != right)
	case operator == "*" && left.Type() == object.STRING_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.evalStringMultiplication(left, right)
	case left.Type() == object.STRING_OBJ || right.Type() == object.STRING_OBJ:
		return e.evalStringPlusOtherInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return e.newErrorf("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case object.TRUE:
		return object.FALSE
	case object.FALSE:
		return object.TRUE
	case object.NIL:
		return object.TRUE
	default:
		return object.FALSE
	}
}

func (e *Task) evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.NUMBER_OBJ {
		return e.newErrorf("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Number).Value
	return &object.Number{Value: value.Neg()}
}

func (e *Task) evalComplementPrefixOperatorExpression(right object.Object) object.Object {
	switch v := right.(type) {
	case *object.Number:
		value := v.Value
		return &object.Number{Value: value.Not()}
	case *object.Bytes:
		value := v.Value
		complement := make([]byte, len(value))
		for i, b := range value {
			complement[i] = ^b
		}
		return &object.Bytes{Value: complement}
	default:
		return e.newErrorf("unknown operator: -%s", right.Type())
	}
}

func (e *Task) evalShortCircuitInfixExpression(left object.Object, node *ast.InfixExpression) object.Object {

	// Short circuit based on left value and operator
	switch node.Operator {
	case "&&":
		// If left is false, return false without evaluating right
		if !e.isTruthy(left) {
			return object.FALSE
		}
		// Otherwise, evaluate and return right
		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}
		if e.isTruthy(right) {
			return object.TRUE
		}
		return object.FALSE

	case "||":
		// If left is true, return true without evaluating right
		if e.isTruthy(left) {
			return object.TRUE
		}
		// Otherwise, evaluate and return right
		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}
		if e.isTruthy(right) {
			return object.TRUE
		}
		return object.FALSE

	default:
		return e.newErrorfWithPos(node.Token.Position, "unknown operator for short-circuit evaluation: %s", node.Operator)
	}
}

func (e *Task) evalBooleanInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value

	switch operator {
	case "==":
		return e.NativeBoolToBooleanObject(left == right)
	case "!=":
		return e.NativeBoolToBooleanObject(left != right)
	case "&&":
		return e.NativeBoolToBooleanObject(leftVal && rightVal)
	case "||":
		return e.NativeBoolToBooleanObject(leftVal || rightVal)
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) evalNumberInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Number).Value
	rightVal := right.(*object.Number).Value

	switch operator {
	case "+":
		return &object.Number{Value: leftVal.Add(rightVal)}
	case "-":
		return &object.Number{Value: leftVal.Sub(rightVal)}
	case "*":
		return &object.Number{Value: leftVal.Mul(rightVal)}
	case "/":
		return &object.Number{Value: leftVal.Div(rightVal, precision, roundingStrategy)}
	case "%":
		return &object.Number{Value: leftVal.Mod(rightVal)}
	case "&":
		return &object.Number{Value: leftVal.And(rightVal)}
	case "|":
		return &object.Number{Value: leftVal.Or(rightVal)}
	case "^":
		return &object.Number{Value: leftVal.Xor(rightVal)}
	case "<<":
		return &object.Number{Value: leftVal.ShiftLeft(rightVal)}
	case ">>":
		return &object.Number{Value: leftVal.ShiftRight(rightVal)}
	case "<":
		return e.NativeBoolToBooleanObject(leftVal.Lt(rightVal))
	case "<=":
		return e.NativeBoolToBooleanObject(leftVal.Lte(rightVal))
	case ">":
		return e.NativeBoolToBooleanObject(leftVal.Gt(rightVal))
	case ">=":
		return e.NativeBoolToBooleanObject(leftVal.Gte(rightVal))
	case "==":
		return e.NativeBoolToBooleanObject(leftVal.Eq(rightVal))
	case "!=":
		return e.NativeBoolToBooleanObject(!leftVal.Eq(rightVal))
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	case "==":
		return e.NativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return e.NativeBoolToBooleanObject(leftVal != rightVal)
	case "<":
		return e.NativeBoolToBooleanObject(leftVal < rightVal)
	case "<=":
		return e.NativeBoolToBooleanObject(leftVal <= rightVal)
	case ">":
		return e.NativeBoolToBooleanObject(leftVal > rightVal)
	case ">=":
		return e.NativeBoolToBooleanObject(leftVal >= rightVal)
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

}

func (e *Task) evalStringPlusOtherInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.Inspect()
	rightVal := right.Inspect()

	switch operator {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) evalStringMultiplication(
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.Number).Value.ToInt64()

	if rightVal < 0 {
		return e.newErrorf("repetition count must be a non-negative NUMBER, got %d", rightVal)
	}

	// Repeat the string
	repeated := strings.Repeat(leftVal, int(rightVal))
	return &object.String{Value: repeated}
}

func (e *Task) evalListInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {

	switch operator {
	case "==":
		return e.NativeBoolToBooleanObject(e.objectsEqual(left, right))
	case "!=":
		return e.NativeBoolToBooleanObject(!e.objectsEqual(left, right))
	case "+:":
		rightVal := right.(*object.List)
		length := len(rightVal.Elements) + 1
		newElements := make([]object.Object, length)
		copy(newElements, []object.Object{left})
		copy(newElements[1:], rightVal.Elements)
		return &object.List{Elements: newElements}
	case ":+":
		leftVal := left.(*object.List)
		length := len(leftVal.Elements) + 1
		newElements := make([]object.Object, length)
		copy(newElements, leftVal.Elements)
		copy(newElements[len(leftVal.Elements):], []object.Object{right})
		return &object.List{Elements: newElements}
	case "+":
		leftVal := left.(*object.List)
		rightVal := right.(*object.List)
		length := len(leftVal.Elements) + len(rightVal.Elements)
		if length > 0 {
			newElements := make([]object.Object, length)
			copy(newElements, leftVal.Elements)
			copy(newElements[len(leftVal.Elements):], rightVal.Elements)
			return &object.List{Elements: newElements}
		}
		return &object.List{Elements: []object.Object{}}
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) evalBytesInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {

	switch operator {
	case "==":
		return e.NativeBoolToBooleanObject(e.objectsEqual(left, right))
	case "!=":
		return e.NativeBoolToBooleanObject(!e.objectsEqual(left, right))
	case "&":
		return performBitwiseOperation(left, right, AndBytes)
	case "|":
		return performBitwiseOperation(left, right, OrBytes)
	case "^":
		return performBitwiseOperation(left, right, XorBytes)
	case "+:":
		byteValue, err := byteValue(left.(*object.Number))
		if err != nil {
			return e.newErrorf("cannot convert number to byte: %s", err.Error())
		}
		rightVal := right.(*object.Bytes)
		length := len(rightVal.Value) + 1
		newBytes := make([]byte, length)
		newBytes[0] = byteValue
		copy(newBytes[1:], rightVal.Value)
		return &object.Bytes{Value: newBytes}
	case ":+":
		byteValue, err := byteValue(right.(*object.Number))
		if err != nil {
			return e.newErrorf("cannot convert number to byte: %s", err.Error())
		}
		leftVal := left.(*object.Bytes)
		length := len(leftVal.Value) + 1
		newBytes := make([]byte, length)
		copy(newBytes, leftVal.Value)
		newBytes[length-1] = byteValue
		return &object.Bytes{Value: newBytes}
	case "+":
		leftVal := left.(*object.Bytes)
		rightVal := right.(*object.Bytes)
		length := len(leftVal.Value) + len(rightVal.Value)
		if length > 0 {
			newBytes := make([]byte, length)
			copy(newBytes, leftVal.Value)
			copy(newBytes[len(leftVal.Value):], rightVal.Value)
			return &object.Bytes{Value: newBytes}
		}
		return &object.Bytes{Value: []byte{}}
	default:
		return e.newErrorf("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Task) doOp(right object.Object, left object.Object, op ByteOp) object.Object {
	bv, err := byteValue(right.(*object.Number))
	if err != nil {
		return e.newErrorf("cannot convert number to byte: %s", err.Error())
	}
	short := []byte{bv}
	return performBitwiseByteOperation(left.(*object.Bytes).Value, short, op)
}

func performBitwiseOperation(left, right object.Object, op ByteOp) object.Object {
	leftVal := left.(*object.Bytes).Value
	rightVal := right.(*object.Bytes).Value
	return performBitwiseByteOperation(leftVal, rightVal, op)
}

func performBitwiseByteOperation(leftVal, rightVal []byte, op ByteOp) object.Object {

	// Repeat the shorter slice to match the longer length, then bitwise AND
	if len(leftVal) == 0 || len(rightVal) == 0 {
		return &object.Error{Message: "cannot perform bitwise operation on empty bytes"}
	}
	var long, short []byte
	if len(leftVal) >= len(rightVal) {
		long, short = leftVal, rightVal
	} else {
		long, short = rightVal, leftVal
	}
	out := make([]byte, len(long))
	for i := 0; i < len(long); i++ {
		out[i] = op(long[i], short[i%len(short)])
	}
	return &object.Bytes{Value: out}
}

func (e *Task) evalIfExpression(
	ie *ast.IfExpression,
) object.Object {
	condition := e.Eval(ie.Condition)
	if e.isError(condition) {
		return condition
	}

	if e.isTruthy(condition) {
		return e.Eval(ie.ThenBranch)
	} else if ie.ElseBranch != nil {
		return e.Eval(ie.ElseBranch)
	} else {
		return object.NIL
	}
}

func (e *Task) evalIdentifier(
	node *ast.Identifier,
) object.Object {

	if builtin, ok := e.Runtime.Builtins[node.Value]; ok {
		return builtin
	}

	if val, ok := e.CurrentEnv().Get(node.Value); ok {
		val = e.resolveValue(node.Token.Position, val)
		if e.isError(val) {
			return val
		}
		// If it is a Module, return the Module object itself
		if module, ok := val.(*object.Module); ok {
			return module
		}
		return val
	}

	return e.newErrorWithPos(node.Token.Position, "identifier not found: "+node.Value)
}

func (e *Task) resolveValue(pos int, obj object.Object) object.Object {
	for {
		ref, ok := obj.(*object.BindingRef)
		if !ok {
			return obj
		}
		if ref.Env == nil {
			return e.newErrorfWithPos(pos, "invalid binding reference: %s", ref.Inspect())
		}
		val, _, ok := ref.Env.GetLocalBindingValue(ref.Name)
		if !ok {
			return e.newErrorfWithPos(pos, "binding reference not found: %s", ref.Name)
		}
		if val == object.BINDING_UNINITIALIZED {
			return e.newErrorfWithPos(pos, "%s used before initialization (likely circular import)", ref.Name)
		}
		obj = val
	}
}

func (e *Task) isTruthy(obj object.Object) bool {
	switch obj {
	case object.NIL:
		return false
	case object.TRUE:
		return true
	case object.FALSE:
		return false
	default:
		return true
	}
}

func (e *Task) NewError(format string, a ...interface{}) *object.Error {
	return e.newErrorfWithPos(0, format, a...)
}

func (e *Task) newErrorfWithPos(pos int, format string, a ...interface{}) *object.Error {
	m := fmt.Sprintf(format, a...)
	return e.newErrorWithPos(pos, m)
}

func (e *Task) newErrorWithPos(pos int, m string) *object.Error {

	if pos == 0 {
		return &object.Error{Message: m}
	}

	env := e.CurrentEnv()

	line, col := util.GetLineAndColumn(env.Src, pos)

	var errorMsg bytes.Buffer
	errorMsg.WriteString(fmt.Sprintf("Error: %s\n", m))
	errorMsg.WriteString(fmt.Sprintf("    --> %s:%d:%d\n", env.Path, line, col))

	lines := util.GetContextLines(env.Src, line, col)
	errorMsg.WriteString(lines)

	return &object.Error{Message: errorMsg.String()}
}

func (e *Task) newErrorf(format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	return &object.Error{Message: msg}
}

func (e *Task) isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func (e *Task) evalExpressions(
	exps []ast.Expression,
) []object.Object {
	var result []object.Object

	for _, err := range exps {
		evaluated := e.Eval(err)
		if e.isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

type boundArguments struct {
	Values   []object.Object
	Provided []bool
}

func (e *Task) evalCallArguments(pos int, args []ast.Expression) ([]object.Object, map[string]object.Object, object.Object) {
	var positional []object.Object
	var named map[string]object.Object
	sawNamed := false

	for _, arg := range args {
		switch node := arg.(type) {
		case *ast.NamedArgument:
			sawNamed = true
			if named == nil {
				named = make(map[string]object.Object)
			}
			if _, exists := named[node.Name.Value]; exists {
				return nil, nil, e.newErrorfWithPos(node.Token.Position, "duplicate named argument: %s", node.Name.Value)
			}
			value := e.Eval(node.Value)
			if e.isError(value) {
				return nil, nil, value
			}
			named[node.Name.Value] = value
		case *ast.SpreadExpression:
			if sawNamed {
				return nil, nil, e.newErrorWithPos(node.Token.Position, "positional arguments must appear before named arguments")
			}
			spreadValue := e.Eval(node.Value)
			if e.isError(spreadValue) {
				return nil, nil, spreadValue
			}
			list, ok := spreadValue.(*object.List)
			if !ok {
				return nil, nil, e.newErrorfWithPos(node.Token.Position, "spread operator can only be used on lists, got %s", spreadValue.Type())
			}
			positional = append(positional, list.Elements...)
		default:
			if sawNamed {
				return nil, nil, e.newErrorWithPos(pos, "positional arguments must appear before named arguments")
			}
			evaluated := e.Eval(arg)
			if e.isError(evaluated) {
				return nil, nil, evaluated
			}
			positional = append(positional, evaluated)
		}
	}

	return positional, named, nil
}

func buildParamIndex(params []*ast.FunctionParameter) map[string]int {
	index := make(map[string]int, len(params))
	for i, param := range params {
		index[param.Name.Value] = i
	}
	return index
}

func (e *Task) bindArguments(
	pos int,
	fnObj object.Object,
	params []*ast.FunctionParameter,
	positional []object.Object,
	named map[string]object.Object,
) (*boundArguments, object.Object) {
	if params == nil {
		if len(named) > 0 {
			return nil, e.newErrorWithPos(pos, "named arguments are not supported for this function")
		}
		provided := make([]bool, len(positional))
		for i := range provided {
			provided[i] = true
		}
		return &boundArguments{Values: positional, Provided: provided}, nil
	}

	paramCount := len(params)
	values := make([]object.Object, paramCount)
	provided := make([]bool, paramCount)

	hasVariadic := paramCount > 0 && params[paramCount-1].IsVariadic
	variadicIndex := paramCount - 1

	if len(named) > 0 {
		var paramIndex map[string]int
		switch f := fnObj.(type) {
		case *object.Function:
			if f.ParamIndex == nil {
				f.ParamIndex = buildParamIndex(params)
			}
			paramIndex = f.ParamIndex
		case *object.Foreign:
			if f.ParamIndex == nil {
				f.ParamIndex = buildParamIndex(params)
			}
			paramIndex = f.ParamIndex
		default:
			paramIndex = buildParamIndex(params)
		}

		for name, val := range named {
			idx, ok := paramIndex[name]
			if !ok {
				return nil, e.newErrorfWithPos(pos, "unknown named parameter: %s", name)
			}
			if provided[idx] {
				return nil, e.newErrorfWithPos(pos, "duplicate assignment to parameter: %s", name)
			}
			if params[idx].IsVariadic {
				if _, ok := val.(*object.List); !ok {
					return nil, e.newErrorfWithPos(pos, "variadic parameter '%s' must be a list when passed by name", name)
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
				return nil, e.newErrorfWithPos(pos, "too many positional arguments")
			}
		} else {
			values[variadicIndex] = &object.List{Elements: remaining}
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
			return nil, e.newErrorfWithPos(pos, "too many positional arguments")
		}
	}

	for i, param := range params {
		if provided[i] {
			continue
		}
		if param.IsVariadic {
			values[i] = &object.List{Elements: []object.Object{}}
			continue
		}
		if param.Default != nil {
			defaultValue := e.evalDefaultParam(fnObj, param.Default)
			if e.isError(defaultValue) {
				return nil, defaultValue
			}
			values[i] = defaultValue
			continue
		}
		return nil, e.newErrorfWithPos(pos, "missing required parameter: %s", param.Name.Value)
	}

	return &boundArguments{Values: values, Provided: provided}, nil
}

func (e *Task) ApplyFunction(pos int, fnName string, fnObj object.Object, positional []object.Object, named map[string]object.Object) object.Object {
	fnObj = e.resolveValue(pos, fnObj)
	if e.isError(fnObj) {
		return fnObj
	}
	switch fn := fnObj.(type) {
	case *object.FunctionGroup:

		f, err := fn.DispatchToFunction(fnName, positional, named)
		if err != nil {
			return e.newErrorfWithPos(pos, "error calling function '%s': %s", fnName, err.Error())
		} else {
			return e.ApplyFunction(pos, fnName, f, positional, named)
		}

	case *object.Function:

		// Track current function for `recur`
		e.pushCallFrame(fnName, fn)
		defer e.popCallFrame()

		var result object.Object

		// Create the initial environment
		argsEnv, errObj := e.extendFunctionEnv(pos, fn, positional, named)
		if errObj != nil {
			return errObj
		}
		e.PushEnv(argsEnv)

		blockEnv := e.newBlockEnv(fn.Body)
		e.PushEnv(blockEnv)

		for {
			result = e.evalBlockStatementWithinEnv(fn.Body)

			_, ok := result.(*object.Error)
			if ok {
				break
			}

			nurseryScope := e.currentNurseryScope()
			if nurseryScope.NurseryErr != nil {
				_, ok := nurseryScope.NurseryErr.(*object.Error)
				if ok {
					// if the current nursery is erroring break out
					result = nurseryScope.NurseryErr
					//nurseryScope.NurseryErr = nil
					break
				}
			}

			// 1. Direct TailCall (e.g., from recur or tail-positioned call)
			if tc, ok := result.(*object.TailCall); ok {
				if tc.Function == fn {
					blockEnv.ResetForTCO()
					if errObj := e.rebindFunctionEnv(pos, argsEnv, fn, tc.Arguments, tc.NamedArguments); errObj != nil {
						result = errObj
						break
					}
					continue
				}
				// Call belongs to a different function. Resolve it now.
				result = e.ApplyFunction(pos, tc.FnName, tc.Function, tc.Arguments, tc.NamedArguments)
				break
			}

			// 2. ReturnValue (explicit return)
			if rv, ok := result.(*object.ReturnValue); ok {
				if tc, ok := rv.Value.(*object.TailCall); ok {
					if tc.Function == fn {
						blockEnv.ResetForTCO()
						if errObj := e.rebindFunctionEnv(pos, argsEnv, fn, tc.Arguments, tc.NamedArguments); errObj != nil {
							result = errObj
							break
						}
						continue
					}
					// Resolve TailCall for a different function
					result = e.ApplyFunction(pos, tc.FnName, tc.Function, tc.Arguments, tc.NamedArguments)
					break
				}
				// Unwrap the final value
				result = rv.Value
				break
			}

			// 3. Implicit return or Error
			break
		}

		// pop the block environment
		result = e.PopEnv(result)

		// PopEnv runs defers and joins children for the finalized call frame
		return e.PopEnv(result)

	case *object.Foreign:
		var result object.Object
		bound, errObj := e.bindArguments(pos, fn, fn.Parameters, positional, named)
		if errObj != nil {
			return errObj
		}
		callArgs := bound.Values
		if fn.Parameters != nil && len(fn.Parameters) > 0 && fn.Parameters[len(fn.Parameters)-1].IsVariadic {
			variadicIndex := len(fn.Parameters) - 1
			callArgs = append([]object.Object{}, bound.Values[:variadicIndex]...)
			if variadicVal, ok := bound.Values[variadicIndex].(*object.List); ok {
				callArgs = append(callArgs, variadicVal.Elements...)
			} else if bound.Values[variadicIndex] != nil {
				callArgs = append(callArgs, bound.Values[variadicIndex])
			}
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					println(r.(error).Error())
					result = e.newErrorfWithPos(pos, "error calling foreign function '%s'", fn.Name)
				}
			}()
			result = fn.Fn(e, callArgs...)
		}()

		// If a foreign function returns a plain Error, promote it to a RuntimeError payload so the language
		// can handle it uniformly (defer onerror, etc).
		if errObj, ok := result.(*object.Error); ok {
			payload := &object.Map{Pairs: map[object.MapKey]object.MapPair{}}
			foreign.PutString(payload, "type", "error")
			foreign.PutString(payload, "foreign", fn.Name)
			foreign.PutString(payload, "msg", errObj.Message)
			return e.runtimeError(pos, "error", payload)
		}

		return result

	default:
		if fn == nil {
			return e.newErrorWithPos(pos, "no function found!")
		}
		return e.newErrorfWithPos(pos, "not a function: %s", fn.Type())
	}
}

func (e *Task) extendFunctionEnv(
	pos int,
	fn *object.Function,
	positional []object.Object,
	named map[string]object.Object,
) (*object.Environment, object.Object) {
	env := object.NewEnclosedEnvironment(fn.Env, &object.StackFrame{
		Function: "call: " + e.callStack[len(e.callStack)-1].FnName,
		File:     fn.Env.Path,
		Position: fn.Body.Token.Position,
		Src:      fn.Env.Src,
	})

	bound, errObj := e.bindArguments(pos, fn, fn.Parameters, positional, named)
	if errObj != nil {
		return nil, errObj
	}

	for i, param := range fn.Parameters {
		env.Define(param.Name.Value, bound.Values[i], false, false)
	}

	return env, nil
}

func (e *Task) rebindFunctionEnv(
	pos int,
	env *object.Environment,
	fn *object.Function,
	positional []object.Object,
	named map[string]object.Object,
) object.Object {
	bound, errObj := e.bindArguments(pos, fn, fn.Parameters, positional, named)
	if errObj != nil {
		return errObj
	}

	for i, param := range fn.Parameters {
		env.Define(param.Name.Value, bound.Values[i], false, false)
	}

	return nil
}

func (e *Task) unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

// evalDefaultParam evaluates a default parameter expression at call time,
// but resolves names in the *defining module environment* of the function,
// per ADR-018.
//
// Conceptually: “defaults belong to the function, not the caller.”
func (e *Task) evalDefaultParam(fnObj object.Object, expr ast.Expression) object.Object {
	if expr == nil {
		return object.NIL
	}

	// Figure out which environment counts as the defining module env.
	// For Slug functions, we use fn.Env and walk outward to the module root.
	var defEnv *object.Environment
	switch f := fnObj.(type) {
	case *object.Function:
		defEnv = f.Env
	default:
		// Foreign functions currently don't carry a defining env; fall back to caller env.
		defEnv = e.CurrentEnv()
	}

	// Walk to module root (Outer == nil).
	if defEnv != nil {
		for defEnv.Outer != nil {
			defEnv = defEnv.Outer
		}
	}

	// Evaluate inside a fresh enclosed env so any transient bindings don't land in the module env.
	tmp := object.NewEnclosedEnvironment(defEnv, nil)

	e.PushEnv(tmp)
	val := e.Eval(expr)
	e.PopEnv(val)

	return val
}

func (e *Task) evalMapLiteral(
	node *ast.MapLiteral,
) object.Object {
	pairs := make(map[object.MapKey]object.MapPair)

	for keyNode, valueNode := range node.Pairs {
		key := e.Eval(keyNode)
		if e.isError(key) {
			return key
		}

		mapKey, ok := key.(object.Hashable)
		if !ok {
			return e.newErrorfWithPos(node.Token.Position, "unusable as map key: %s", key.Type())
		}

		value := e.Eval(valueNode)
		if e.isError(value) {
			return value
		}

		mapKeyHash := mapKey.MapKey()
		pairs[mapKeyHash] = object.MapPair{Key: key, Value: value}
	}

	return &object.Map{Pairs: pairs}
}

func (e *Task) evalStructSchemaExpression(node *ast.StructSchemaExpression) object.Object {
	schema := &object.StructSchema{
		Fields:     make([]object.StructSchemaField, 0, len(node.Fields)),
		FieldIndex: make(map[string]int, len(node.Fields)),
		Env:        e.CurrentEnv(),
	}

	for _, field := range node.Fields {
		if _, exists := schema.FieldIndex[field.Name]; exists {
			return e.newErrorfWithPos(field.Token.Position, "duplicate struct field: %s", field.Name)
		}
		if field.Hint != "" {
			if _, ok := object.TypeTags[field.Hint]; !ok {
				return e.newErrorfWithPos(field.Token.Position, "unknown struct field type hint: %s", field.Hint)
			}
		}

		schema.FieldIndex[field.Name] = len(schema.Fields)
		schema.Fields = append(schema.Fields, object.StructSchemaField{
			Name:    field.Name,
			Default: field.Default,
			Hint:    field.Hint,
		})
	}

	return schema
}

func (e *Task) evalStructInitExpression(node *ast.StructInitExpression) object.Object {
	schemaObj := e.Eval(node.Schema)
	if e.isError(schemaObj) {
		return schemaObj
	}
	schemaObj = e.resolveValue(node.Token.Position, schemaObj)
	if e.isError(schemaObj) {
		return schemaObj
	}

	schema, ok := schemaObj.(*object.StructSchema)
	if !ok {
		return e.newErrorfWithPos(node.Token.Position, "expected struct schema, got %s", schemaObj.Type())
	}

	values := make(map[string]object.Object, len(schema.Fields))
	for _, field := range node.Fields {
		if _, ok := schema.FieldIndex[field.Name]; !ok {
			return e.newErrorfWithPos(field.Token.Position, "unknown field '%s' for struct %s", field.Name, e.structSchemaName(schema))
		}
		if _, ok := values[field.Name]; ok {
			return e.newErrorfWithPos(field.Token.Position, "duplicate field '%s' in struct initializer", field.Name)
		}
		val := e.Eval(field.Value)
		if e.isError(val) {
			return val
		}
		values[field.Name] = val
	}

	for _, field := range schema.Fields {
		if _, ok := values[field.Name]; ok {
			continue
		}
		if field.Default != nil {
			val := e.evalStructDefault(schema, field.Default)
			if e.isError(val) {
				return val
			}
			values[field.Name] = val
		} else {
			values[field.Name] = object.NIL
		}
	}

	if err := e.validateStructHints(node.Token.Position, schema, values); err != nil {
		return err
	}

	return &object.StructValue{
		Schema: schema,
		Fields: values,
	}
}

func (e *Task) evalStructCopyExpression(node *ast.StructCopyExpression) object.Object {
	source := e.Eval(node.Source)
	if e.isError(source) {
		return source
	}
	source = e.resolveValue(node.Token.Position, source)
	if e.isError(source) {
		return source
	}

	structVal, ok := source.(*object.StructValue)
	if !ok {
		return e.newErrorfWithPos(node.Token.Position, "copy expects a struct value, got %s", source.Type())
	}

	values := make(map[string]object.Object, len(structVal.Fields))
	for name, value := range structVal.Fields {
		values[name] = value
	}

	seen := make(map[string]struct{}, len(node.Fields))
	for _, field := range node.Fields {
		if _, ok := seen[field.Name]; ok {
			return e.newErrorfWithPos(field.Token.Position, "duplicate field '%s' in struct copy", field.Name)
		}
		seen[field.Name] = struct{}{}

		if _, ok := structVal.Schema.FieldIndex[field.Name]; !ok {
			return e.newErrorfWithPos(field.Token.Position, "unknown field '%s' for struct %s", field.Name, e.structSchemaName(structVal.Schema))
		}
		val := e.Eval(field.Value)
		if e.isError(val) {
			return val
		}
		values[field.Name] = val
	}

	if err := e.validateStructHints(node.Token.Position, structVal.Schema, values); err != nil {
		return err
	}

	return &object.StructValue{
		Schema: structVal.Schema,
		Fields: values,
	}
}

func (e *Task) evalStructDefault(schema *object.StructSchema, expr ast.Expression) object.Object {
	if expr == nil {
		return object.NIL
	}

	defEnv := schema.Env
	if defEnv == nil {
		defEnv = e.CurrentEnv()
	}
	for defEnv != nil && defEnv.Outer != nil {
		defEnv = defEnv.Outer
	}

	tmp := object.NewEnclosedEnvironment(defEnv, nil)
	e.PushEnv(tmp)
	val := e.Eval(expr)
	e.PopEnv(val)
	return val
}

func (e *Task) validateStructHints(pos int, schema *object.StructSchema, values map[string]object.Object) object.Object {
	for _, field := range schema.Fields {
		if field.Hint == "" {
			continue
		}
		value := values[field.Name]
		if value == nil {
			value = object.NIL
		}
		if value.Type() == object.NIL_OBJ {
			continue
		}

		expected, ok := object.TypeTags[field.Hint]
		if !ok {
			return e.newErrorfWithPos(pos, "unknown struct field type hint: %s", field.Hint)
		}

		if field.Hint == object.FUNCTION_TAG {
			if value.Type() == object.FUNCTION_OBJ || value.Type() == object.FUNCTION_GROUP_OBJ {
				continue
			}
		}

		if string(value.Type()) != expected {
			return e.newErrorfWithPos(pos, "struct %s field %s expected %s, got %s", e.structSchemaName(schema), field.Name, field.Hint, value.Type())
		}
	}

	return nil
}

func (e *Task) structSchemaName(schema *object.StructSchema) string {
	if schema != nil && schema.Name != "" {
		return schema.Name
	}
	return "<anonymous>"
}

func (e *Task) evalMatchExpression(node *ast.MatchExpression) object.Object {
	// Evaluate the match value if provided
	var matchValue object.Object
	if node.Value != nil {
		matchValue = e.Eval(node.Value)
		if e.isError(matchValue) {
			return matchValue
		}
	}

	// Iterate through patterns
	for _, matchCase := range node.Cases {
		// Create a new scope for pattern variables
		result, matched := e.evalMatchCase(matchValue, matchCase)
		if matched {
			return result
		}
	}

	// No match found
	return object.NIL
}

func (e *Task) evalMatchCase(matchValue object.Object, matchCase *ast.MatchCase) (result object.Object, matched bool) {
	patternEnv := object.NewEnclosedEnvironment(e.CurrentEnv(), nil)
	e.PushEnv(patternEnv)
	result = nil
	defer func() { result = e.PopEnv(result) }()

	// Pinning always refers to the identifier in the enclosing lexical scope (outside the pattern env).
	pinEnv := patternEnv.Outer

	// Match against the provided value or evaluate the condition
	matched = false
	var err error
	if matchValue != nil {
		// Match the case's pattern against the matchValue
		matched, err = e.patternMatches(matchCase.Pattern, matchValue, false, false, false, pinEnv)
		if err != nil {
			result = e.newErrorfWithPos(matchCase.Token.Position, "pattern match error: %s", err.Error())
			return result, true
		}
	} else {
		// Valueless match condition
		matched = e.evaluatePatternAsCondition(matchCase.Pattern)
	}

	// Evaluate guard condition if pattern matches
	if matched && matchCase.Guard != nil {
		guardResult := e.Eval(matchCase.Guard)
		if e.isError(guardResult) {
			result = guardResult
			return result, true
		}
		matched = e.isTruthy(guardResult)
	}

	// If the pattern matched, evaluate the body
	if matched {
		result = e.Eval(matchCase.Body)
		return result, true
	}
	return nil, false
}

// e.patternMatches checks if a value matches a pattern and binds variables.
// pinEnv is the environment used for resolving pinned identifiers (^name); it must not include pattern bindings.
func (e *Task) patternMatches(
	pattern ast.MatchPattern,
	value object.Object,
	isConstant bool,
	isExport bool,
	isImport bool,
	pinEnv *object.Environment,
) (bool, error) {
	env := e.CurrentEnv()
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard matches anything
		return true, nil

	case *ast.PinnedIdentifierPattern:
		// Resolve before bindings; never bind; cannot be shadowed by pattern bindings.
		if pinEnv == nil {
			return false, fmt.Errorf("internal error: no enclosing environment for pinned identifier ^%s", p.Value.Value)
		}
		expected, ok := pinEnv.Get(p.Value.Value)
		if !ok {
			return false, fmt.Errorf("pinned identifier is undefined: %s", p.Value.Value)
		}
		expected = e.resolveValue(0, expected)
		if e.isError(expected) {
			return false, fmt.Errorf("pinned identifier could not be resolved: %s", p.Value.Value)
		}
		return e.objectsEqual(expected, value), nil

	case *ast.SpreadPattern:
		// SpreadPattern matches anything
		if p.Value != nil {
			if isConstant {
				_, err := env.DefineConstant(p.Value.Value, value, isExport, isImport)
				return err == nil, err
			} else {
				_, err := env.Define(p.Value.Value, value, isExport, isImport)
				return err == nil, err
			}
		}
		return true, nil

	case *ast.LiteralPattern:
		// Evaluate the literal and compare with the value
		literalValue := e.Eval(p.Value)
		if e.isError(literalValue) {
			return false, fmt.Errorf("error while evaluating literal pattern value: %s", literalValue)
		}
		return e.objectsEqual(literalValue, value), nil

	case *ast.IdentifierPattern:
		// Bind the value to the identifier
		if schema, ok := value.(*object.StructSchema); ok {
			if schema.Name == "" {
				schema.Name = p.Value.Value
			}
		}
		if isConstant {
			_, err := env.DefineConstant(p.Value.Value, value, isExport, isImport)
			return err == nil, err
		} else {
			_, err := env.Define(p.Value.Value, value, isExport, isImport)
			return err == nil, err
		}

	case *ast.MultiPattern:
		// Check if value matches any of the patterns
		for _, subPattern := range p.Patterns {
			encEnv := object.NewEnclosedEnvironment(env, nil)
			e.PushEnv(encEnv)
			matched, err := e.patternMatches(subPattern, value, isConstant, isExport, isImport, pinEnv)
			e.PopEnv(nil)
			if err != nil {
				return false, err
			}
			if matched {
				return true, nil
			}
		}
		return false, nil

	case *ast.ListPattern:
		// Check if the value is an list
		switch v := value.(type) {
		case *object.List:
			return e.patternMatchesList(env, p, v, isConstant, isExport, isImport, pinEnv)
		case *object.Bytes:
			return e.patternMatchesBytes(env, p, v, isConstant, isExport, isImport, pinEnv)
		default:
			return false, nil
		}

	case *ast.MapPattern:
		// Check if value is a map
		mapObj, ok := value.(*object.Map)
		if !ok {
			return false, nil
		}

		isImport := mapObj.HasTag(object.IMPORT_TAG)

		if p.SelectAll {
			// Copy all key-value pairs into current scope
			for _, pair := range mapObj.Pairs {
				if str, ok := pair.Key.(*object.String); ok {
					if isConstant {
						if _, err := env.DefineConstant(str.Value, pair.Value, isExport, isImport); err != nil {
							return false, err
						}
					} else {
						if _, err := env.Define(str.Value, pair.Value, isExport, isImport); err != nil {
							return false, err
						}
					}
				}
			}
			return true, nil
		}

		// Empty map pattern matches empty map
		if len(p.Pairs) == 0 {
			return len(mapObj.Pairs) == 0, nil
		}

		usedKeys := make([]object.MapKey, 0)

		// scoped environment to capture the match bindings, these will be copied to the parent env on success
		scoped := object.NewEnclosedEnvironment(env, nil)
		e.PushEnv(scoped)
		defer func() { e.PopEnv(nil) }() // Pattern matches shouldn't produce errors via defer usually

		// Check if all required fields are present
		for key, subPattern := range p.Pairs {
			if key == token.ELLIPSIS {
				// Skip wildcard placeholder for spread, we'll deal with that later
				continue
			}

			// Check if key exists in map
			keyObj := &object.String{Value: key}
			mapKey := keyObj.MapKey()
			usedKeys = append(usedKeys, mapKey)
			pair, ok := mapObj.Pairs[mapKey]
			if !ok {
				return false, nil
			}

			// Check if value matches subpattern
			matched, err := e.patternMatches(subPattern, pair.Value, isConstant, isExport, isImport, pinEnv)
			if !matched || err != nil {
				return false, err
			}
		}

		if p.Exact && len(usedKeys) != len(mapObj.Pairs) {
			return false, nil
		}

		// If a spread pattern is used, collect unused keys into a new map
		if p.Spread {
			pair, ok := p.Pairs[token.ELLIPSIS]
			if ok {
				if len(usedKeys) >= len(mapObj.Pairs) {
					// map is empty
					_, err := e.patternMatches(pair, &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}, isConstant, isExport, isImport, pinEnv)
					if err != nil {
						return false, err
					}
				} else {
					copiedPairs := make(map[object.MapKey]object.MapPair)
					for mapKey, pair := range mapObj.Pairs {
						isUsed := false
						for _, usedKey := range usedKeys {
							if mapKey == usedKey {
								isUsed = true
								break
							}
						}
						if !isUsed {
							copiedPairs[mapKey] = pair
						}
					}
					_, err := e.patternMatches(pair, &object.Map{Pairs: copiedPairs}, isConstant, isExport, isImport, pinEnv)
					if err != nil {
						return false, err
					}
				}
			}
		}

		// Copy bindings to parent env
		for name, binding := range scoped.Bindings {
			value, _ := scoped.Get(name)
			if binding.IsMutable {
				_, err := env.Define(name, value, isExport, isImport)
				if err != nil {
					return false, err
				}
			} else {
				_, err := env.DefineConstant(name, value, isExport, isImport)
				if err != nil {
					return false, err
				}
			}
		}

		return true, nil

	case *ast.StructPattern:
		structVal, ok := value.(*object.StructValue)
		if !ok {
			return false, nil
		}

		lookupEnv := pinEnv
		if lookupEnv == nil {
			lookupEnv = env
		}
		schemaObj, ok := lookupEnv.Get(p.Schema.Value)
		if !ok {
			return false, fmt.Errorf("struct schema not found: %s", p.Schema.Value)
		}
		schemaObj = e.resolveValue(0, schemaObj)
		if e.isError(schemaObj) {
			return false, fmt.Errorf("struct schema could not be resolved: %s", p.Schema.Value)
		}
		schema, ok := schemaObj.(*object.StructSchema)
		if !ok {
			return false, fmt.Errorf("pattern schema is not a struct: %s", p.Schema.Value)
		}
		if structVal.Schema != schema {
			return false, nil
		}

		scoped := object.NewEnclosedEnvironment(env, nil)
		e.PushEnv(scoped)
		defer func() { e.PopEnv(nil) }()

		for _, field := range p.Fields {
			if _, ok := schema.FieldIndex[field.Name]; !ok {
				return false, fmt.Errorf("unknown field in struct pattern: %s", field.Name)
			}
			val, ok := structVal.Fields[field.Name]
			if !ok {
				val = object.NIL
			}
			matched, err := e.patternMatches(field.Pattern, val, isConstant, isExport, isImport, pinEnv)
			if !matched || err != nil {
				return false, err
			}
		}

		for name, binding := range scoped.Bindings {
			value, _ := scoped.Get(name)
			if binding.IsMutable {
				_, err := env.Define(name, value, isExport, isImport)
				if err != nil {
					return false, err
				}
			} else {
				_, err := env.DefineConstant(name, value, isExport, isImport)
				if err != nil {
					return false, err
				}
			}
		}

		return true, nil
	}

	// Unhandled pattern type
	return false, nil
}

func (e *Task) patternMatchesList(
	env *object.Environment,
	listPattern *ast.ListPattern,
	list *object.List,
	isConstant bool,
	isExport bool,
	isImport bool,
	pinEnv *object.Environment,
) (bool, error) {

	// Empty list pattern matches empty list
	if len(listPattern.Elements) == 0 {
		return len(list.Elements) == 0, nil
	}

	_, isSpread := listPattern.Elements[len(listPattern.Elements)-1].(*ast.SpreadPattern)

	// Check if list length matches pattern length
	if (len(listPattern.Elements) != len(list.Elements) && !isSpread) || (len(listPattern.Elements) > len(list.Elements)+1 && isSpread) {
		return false, nil
	}

	// scoped environment to capture the match bindings, these will be copied to the parent env on success
	scoped := object.NewEnclosedEnvironment(env, nil)
	e.PushEnv(scoped)

	for i, elemPattern := range listPattern.Elements {
		if spread, isSpread := elemPattern.(*ast.SpreadPattern); isSpread {
			matched, err := e.patternMatches(spread, &object.List{Elements: list.Elements[i:]}, isConstant, isExport, isImport, pinEnv)
			if err != nil || !matched {
				e.PopEnv(nil)
				return false, err
			}
			break
		} else {
			matched, err := e.patternMatches(elemPattern, list.Elements[i], isConstant, isExport, isImport, pinEnv)
			if err != nil || !matched {
				e.PopEnv(nil)
				return false, err
			}
		}
	}

	// Copy bindings from scoped environment to parent environment
	for name, binding := range scoped.Bindings {
		value, _ := scoped.Get(name)
		if binding.IsMutable {
			if _, err := env.Define(name, value, isExport, isImport); err != nil {
				e.PopEnv(nil)
				return false, err
			}
		} else {
			if _, err := env.DefineConstant(name, value, isExport, isImport); err != nil {
				e.PopEnv(nil)
				return false, err
			}
		}
	}

	e.PopEnv(nil)

	return true, nil
}

func (e *Task) patternMatchesBytes(
	env *object.Environment,
	listPattern *ast.ListPattern,
	bytes *object.Bytes,
	isConstant bool,
	isExport bool,
	isImport bool,
	pinEnv *object.Environment,
) (bool, error) {

	// Empty bytes pattern matches empty bytes
	if len(listPattern.Elements) == 0 {
		return len(bytes.Value) == 0, nil
	}

	_, isSpread := listPattern.Elements[len(listPattern.Elements)-1].(*ast.SpreadPattern)

	// Check if bytes length matches pattern length
	if (len(listPattern.Elements) != len(bytes.Value) && !isSpread) || (len(listPattern.Elements) > len(bytes.Value)+1 && isSpread) {
		return false, nil
	}

	// scoped environment to capture the match bindings, these will be copied to the parent env on success
	scoped := object.NewEnclosedEnvironment(env, nil)
	e.PushEnv(scoped)

	for i, elemPattern := range listPattern.Elements {
		if spread, isSpread := elemPattern.(*ast.SpreadPattern); isSpread {
			matched, err := e.patternMatches(spread, &object.Bytes{Value: bytes.Value[i:]}, isConstant, isExport, isImport, pinEnv)
			if err != nil || !matched {
				e.PopEnv(nil)
				return false, err
			}
			break
		} else {
			matched, err := e.patternMatches(elemPattern, &object.Number{Value: dec64.FromInt(int(bytes.Value[i]))}, isConstant, isExport, isImport, pinEnv)
			if err != nil || !matched {
				e.PopEnv(nil)
				return false, err
			}
		}
	}

	// Copy bindings from scoped environment to parent environment
	for name, binding := range scoped.Bindings {
		value, _ := scoped.Get(name)
		if binding.IsMutable {
			if _, err := env.Define(name, value, isExport, isImport); err != nil {
				e.PopEnv(nil)
				return false, err
			}
		} else {
			if _, err := env.DefineConstant(name, value, isExport, isImport); err != nil {
				e.PopEnv(nil)
				return false, err
			}
		}
	}

	e.PopEnv(nil)

	return true, nil
}

// evaluatePatternAsCondition evaluates patterns as conditions for valueless match
func (e *Task) evaluatePatternAsCondition(pattern ast.MatchPattern) bool {
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard always matches
		return true

	case *ast.LiteralPattern:
		// Evaluate the literal and check if truthy
		result := e.Eval(p.Value)
		if e.isError(result) {
			return false
		}
		return e.isTruthy(result)

	case *ast.IdentifierPattern:
		// Look up identifier and check if truthy
		value, ok := e.CurrentEnv().Get(p.Value.Value)
		if !ok {
			return false
		}
		value = e.resolveValue(0, value)
		if e.isError(value) {
			return false
		}
		return e.isTruthy(value)

	case *ast.MultiPattern:
		// Check if any subpattern is truthy
		for _, subPattern := range p.Patterns {
			if e.evaluatePatternAsCondition(subPattern) {
				return true
			}
		}
		return false
	}

	return false
}

// objectsEqual compares two objects for equality
func (e *Task) objectsEqual(a, b object.Object) bool {
	if a.Type() != b.Type() {
		return false
	}

	switch aVal := a.(type) {
	case *object.Number:
		return aVal.Value.Eq(b.(*object.Number).Value)

	case *object.Boolean:
		return aVal.Value == b.(*object.Boolean).Value

	case *object.String:
		return aVal.Value == b.(*object.String).Value

	case *object.Nil:
		return true // nil equals nil

	case *object.List:
		bArr := b.(*object.List)
		if len(aVal.Elements) != len(bArr.Elements) {
			return false
		}

		for i, elem := range aVal.Elements {
			if !e.objectsEqual(elem, bArr.Elements[i]) {
				return false
			}
		}

		return true

	case *object.Bytes:
		bArr := b.(*object.Bytes)
		if len(aVal.Value) != len(bArr.Value) {
			return false
		}

		for i, elem := range aVal.Value {
			if elem != bArr.Value[i] {
				return false
			}
		}
		return true

	case *object.Map:
		mapObj := b.(*object.Map)
		if len(aVal.Pairs) != len(mapObj.Pairs) {
			return false
		}

		for k, v := range aVal.Pairs {
			bPair, ok := mapObj.Pairs[k]
			if !ok || !e.objectsEqual(v.Value, bPair.Value) {
				return false
			}
		}

		return true

	case *object.StructValue:
		other := b.(*object.StructValue)
		if aVal.Schema != other.Schema {
			return false
		}
		if aVal.Schema == nil {
			return false
		}
		for _, field := range aVal.Schema.Fields {
			leftVal, ok := aVal.Fields[field.Name]
			if !ok {
				leftVal = object.NIL
			}
			rightVal, ok := other.Fields[field.Name]
			if !ok {
				rightVal = object.NIL
			}
			if !e.objectsEqual(leftVal, rightVal) {
				return false
			}
		}
		return true
	}

	return false
}

func (e *Task) evalMapIndexExpression(pos int, obj, index object.Object) object.Object {
	mapObj := obj.(*object.Map)

	key, ok := index.(object.Hashable)
	if !ok {
		return e.newErrorfWithPos(pos, "unusable as map key: %s", index.Type())
	}

	pair, ok := mapObj.Pairs[key.MapKey()]
	if !ok {
		return object.NIL
	}

	return pair.Value
}

func (e *Task) evalStructIndexExpression(pos int, obj, index object.Object) object.Object {
	structObj := obj.(*object.StructValue)
	key, ok := index.(*object.String)
	if !ok {
		return e.newErrorfWithPos(pos, "struct field access expects string, got %s", index.Type())
	}
	if structObj.Schema == nil {
		return e.newErrorfWithPos(pos, "struct has no schema")
	}
	if _, ok := structObj.Schema.FieldIndex[key.Value]; !ok {
		return e.newErrorfWithPos(pos, "unknown field '%s' for struct %s", key.Value, e.structSchemaName(structObj.Schema))
	}
	val, ok := structObj.Fields[key.Value]
	if !ok {
		return object.NIL
	}
	return val
}

func (e *Task) runtimeErrorAt(pos int, typ string, fields map[string]object.Object) *object.RuntimeError {
	payload := &object.Map{}
	payload.Put(&object.String{Value: "type"}, &object.String{Value: typ})
	for k, v := range fields {
		payload.Put(&object.String{Value: k}, v)
	}

	return e.runtimeError(pos, typ, payload)
}

func (e *Task) evalThrowStatement(node *ast.ThrowStatement) object.Object {
	val := e.Eval(node.Value)
	if e.isError(val) {
		return val
	}
	return e.runtimeError(node.Token.Position, "throw", val)
}

func (e *Task) runtimeError(pos int, typ string, payload object.Object) *object.RuntimeError {
	env := e.CurrentEnv()
	return &object.RuntimeError{
		Payload: payload,
		StackTrace: e.GatherStackTrace(&object.StackFrame{
			Function: typ,
			File:     env.Path,
			Src:      env.Src,
			Position: pos,
		}),
	}
}

func (e *Task) GatherStackTrace(frame *object.StackFrame) []*object.StackFrame {
	var trace []*object.StackFrame
	if frame != nil {
		trace = append(trace, frame)
	}
	// Walk the envStack from top (current) to bottom
	for i := len(e.envStack) - 1; i >= 0; i-- {
		env := e.envStack[i]
		if env.StackInfo != nil {
			trace = append(trace, env.StackInfo)
		}
	}
	return trace // Already in correct order (most recent first)
}

func (e *Task) evalIndexExpression(pos int, left, index object.Object) object.Object {

	switch {
	case left.Type() == object.STRING_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if str, ok := left.(*object.String); ok {
				return e.evalStringSlice(str.Value, slice)
			}
		}
		return e.evalStringIndexExpression(pos, left, index)
	case left.Type() == object.LIST_OBJ && index.Type() == object.NUMBER_OBJ:
		return e.evalListIndexExpression(pos, left, index)
	case left.Type() == object.LIST_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if arr, ok := left.(*object.List); ok {
				return e.evalListSlice(arr.Elements, slice)
			}
		}
		return e.evalListIndexExpression(pos, left, index)
	case left.Type() == object.BYTE_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if arr, ok := left.(*object.Bytes); ok {
				return e.evalByteSlice(arr.Value, slice)
			}
		}
		return e.evalByteIndexExpression(left, index)
	case left.Type() == object.MAP_OBJ:
		return e.evalMapIndexExpression(pos, left, index)
	case left.Type() == object.STRUCT_OBJ:
		return e.evalStructIndexExpression(pos, left, index)
	default:
		return e.newErrorfWithPos(pos, "index operator not supported: %s", left.Type())
	}
}

func (e *Task) evalSliceExpression(node *ast.SliceExpression) object.Object {
	start := e.Eval(node.Start)
	if e.isError(start) {
		return start
	}
	end := e.Eval(node.End)
	if e.isError(end) {
		return end
	}
	step := e.Eval(node.Step)
	if e.isError(step) {
		return step
	}
	return &object.Slice{
		Start: start,
		End:   end,
		Step:  step,
	}
}

func (e *Task) evalListIndexExpression(pos int, list, index object.Object) object.Object {
	listObject := list.(*object.List)
	num, ok := index.(*object.Number)
	if !ok {
		return e.newErrorfWithPos(pos, "index operator not supported: %s", index.Type())
	}
	idx := num.Value.ToInt64()
	max := int64(len(listObject.Elements) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return object.NIL
	}

	return listObject.Elements[idx]
}

func (e *Task) evalByteIndexExpression(list, index object.Object) object.Object {
	bytesObject := list.(*object.Bytes)
	idx := index.(*object.Number).Value.ToInt64()
	max := int64(len(bytesObject.Value) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return object.NIL
	}

	return &object.Number{Value: dec64.FromInt(int(bytesObject.Value[idx]))}
}

func (e *Task) evalStringIndexExpression(pos int, str, index object.Object) object.Object {
	stringObject := str.(*object.String)
	num, ok := index.(*object.Number)

	if !ok {
		return e.newErrorfWithPos(pos, "index operator not supported: %s", index.Type())
	}
	idx := num.Value.ToInt64()

	runes := []rune(stringObject.Value)
	max := int64(len(runes) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return object.NIL
	}

	return &object.String{Value: string(runes[idx])}
}

func (e *Task) evalListSlice(elements []object.Object, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(elements), slice)
	var result []object.Object
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.List{Elements: result}
}

func (e *Task) evalByteSlice(elements []byte, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(elements), slice)
	var result []byte
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.Bytes{Value: result}
}

func (e *Task) evalStringSlice(value string, slice *object.Slice) object.Object {
	runes := []rune(value)
	start, end, step := e.computeSliceIndices(len(runes), slice)
	var b strings.Builder
	for i := start; i < end; i += step {
		b.WriteRune(runes[i])
	}
	return &object.String{Value: b.String()}
}

func (e *Task) computeSliceIndices(length int, slice *object.Slice) (int, int, int) {
	start := 0
	end := length
	step := 1

	if slice.Start != nil {
		start = int(slice.Start.(*object.Number).Value.ToInt64())
	}
	if slice.End != nil {
		end = int(slice.End.(*object.Number).Value.ToInt64())
	}
	if slice.Step != nil {
		step = int(slice.Step.(*object.Number).Value.ToInt64())
	}

	if start < 0 {
		start += length
	}
	if end < 0 {
		end += length
	}
	if start < 0 {
		start = 0
	}
	if end > length {
		end = length
	}
	if step <= 0 {
		// todo error on 0, consider negative step
		step = 1
	}

	return start, end, step
}

func (e *Task) evalForeignFunctionDeclaration(ff *ast.ForeignFunctionDeclaration) object.Object {
	env := e.CurrentEnv()
	modulePath := env.ModuleFqn
	functionName := ff.Name.Value

	fqn := modulePath + "." + functionName

	if foreignFn, exists := e.Runtime.LookupForeign(fqn); exists {
		foreignFn.Tags = e.evalTags(ff.Tags)
		foreignFn.Parameters = ff.Parameters
		foreignFn.ParamIndex = buildParamIndex(ff.Parameters)
		foreignFn.Name = functionName
		foreignFn.Signature = ff.Signature
		isExported := hasExportTag(ff.Tags)
		_, err := env.Define(functionName, foreignFn, isExported, false)
		if err != nil {
			return e.newErrorWithPos(ff.Token.Position, err.Error())
		}
		return object.NIL
	}
	return e.newErrorfWithPos(ff.Token.Position, "unknown foreign function %s", fqn)
}

func (e *Task) evalDefer(deferStmt *ast.DeferStatement) object.Object {
	// Register the defer statement into the environment's defer stack
	e.CurrentEnv().RegisterDefer(deferStmt)
	return nil // Defer statements do not produce a direct result
}

func (e *Task) evalSpawnExpression(node *ast.SpawnExpression) object.Object {
	currentEnv := e.CurrentEnv()
	nurseryScope := e.currentNurseryScope()

	taskEval := &Task{
		Runtime: e.Runtime,
		ID:      e.NextHandleID(),
		Done:    make(chan struct{}),
	}
	// IMPORTANT: register child on the owner scope, not necessarily currentEnv
	nurseryScope.AddChild(taskEval)
	taskEval.PushNurseryScope(nurseryScope)

	// Use ShallowCopy to capture current local variables.
	// This prevents ResetForTCO from wiping variables that the spawned task needs.
	taskEnv := currentEnv.ShallowCopy()

	go func() {
		// lexical limit lookup (currentEnv chain)
		limitChan := nurseryScope.Limit
		if limitChan != nil {
			limitChan <- struct{}{}
			defer func() { <-limitChan }()
		}

		taskEval.PushEnv(taskEnv)
		result := taskEval.Eval(node.Body)

		if fn, ok := result.(*object.Function); ok && len(fn.Parameters) == 0 {
			result = taskEval.ApplyFunction(node.Token.Position, "spawned_task", fn, []object.Object{}, nil)
		}
		result = taskEval.PopEnv(result)
		taskEval.Complete(result)

		if taskEval.CurrentEnvStackSize() != 0 {
			panic("task environment stack not empty after evaluation")
		}

		// Check for ANY error type to trigger fail-fast
		if taskEval.Err != nil && !taskEval.Observed {
			nurseryScope.NoteChildFailure(taskEval, taskEval.Err)
		} else if e.isError(result) {
			nurseryScope.NoteChildFailure(taskEval, result)
		}
	}()

	return taskEval
}

func (e *Task) evalAwaitExpression(node *ast.AwaitExpression) object.Object {
	obj := e.Eval(node.Value)
	if e.isError(obj) {
		return obj
	}

	handle, ok := obj.(*Task)
	if !ok {
		return e.newErrorfWithPos(node.Token.Position, "await expects a task handle, got %s", obj.Type())
	}

	if node.Timeout != nil {
		timeoutVal := e.Eval(node.Timeout)
		if e.isError(timeoutVal) {
			return timeoutVal
		}

		var duration int64
		if num, ok := timeoutVal.(*object.Number); ok {
			duration = num.Value.ToInt64()
		} else {
			return e.newErrorfWithPos(node.Token.Position, "timeout must be a number (ms)")
		}

		timer := time.NewTimer(time.Duration(duration) * time.Millisecond)
		defer timer.Stop()

		handle.Observed = true
		owner := handle.OwnerNursery
		owner.RemoveChild(handle)

		select {
		case <-handle.Done:
			// Task finished in time
		case <-timer.C:
			// Timeout: cancel task + raise timeout error
			handle.Cancel(nil, fmt.Sprintf("handle %d cancelled due to await timeout", handle.ID))
			return e.runtimeErrorAt(node.Token.Position, "timeout", map[string]object.Object{
				"ms":     &object.Number{Value: dec64.FromInt(int(duration))},
				"handle": &object.Number{Value: dec64.FromInt64(handle.ID)},
			})
		}
	} else {
		<-handle.Done
	}

	if handle.Err != nil {
		return handle.Err
	}
	return handle.Result
}

func (e *Task) applyTagsIfPresent(tags []*ast.Tag, val object.Object) object.Object {
	if tags != nil {
		switch t := val.(type) {
		case *object.Boolean:
			t.Tags = e.evalTags(tags)
		case *object.Number:
			t.Tags = e.evalTags(tags)
		case *object.String:
			t.Tags = e.evalTags(tags)
		case *object.Function:
			t.Tags = e.evalTags(tags)
		case *object.Foreign:
			t.Tags = e.evalTags(tags)
		case *object.Map:
			t.Tags = e.evalTags(tags)
		case *object.List:
			t.Tags = e.evalTags(tags)
		}
	}
	return val
}

func (e *Task) evalTags(tags []*ast.Tag) map[string]object.List {
	result := make(map[string]object.List)
	for _, tag := range tags {
		var argList []object.Object
		for _, arg := range tag.Args {
			val := e.Eval(arg)
			argList = append(argList, val)
		}
		result[tag.Name] = object.List{Elements: argList}
	}
	return result
}

func byteValue(n *object.Number) (byte, error) {

	value := n.Value.ToInt()
	if value < 0 || value > 255 {
		return 0, errors.New("byte must be between 0 and 255, got " + n.Inspect())
	}
	return byte(value), nil
}

func (th *Task) Type() object.ObjectType { return object.TASK_HANDLE_OBJ }
func (th *Task) Inspect() string {
	return fmt.Sprintf("<task %d>", th.ID)
}

// Complete sets the result and signals any waiters
func (th *Task) Complete(res object.Object) {
	th.mu.Lock()
	defer th.mu.Unlock()

	if th.IsFinished {
		return
	}

	if rtErr, ok := res.(*object.RuntimeError); ok {
		th.Err = rtErr
	} else {
		th.Result = res
	}

	th.IsFinished = true
	close(th.Done)
}

// Cancel force-settles the task as cancelled (idempotent).
// The underlying goroutine may continue running, but its result is ignored.
func (th *Task) Cancel(cause *object.RuntimeError, reason string) {
	payload := &object.Map{}
	payload.Put(&object.String{Value: "type"}, &object.String{Value: "cancelled"})
	payload.Put(&object.String{Value: "reason"}, &object.String{Value: reason})

	rt := &object.RuntimeError{
		Payload: payload,
		Cause:   cause,
	}

	th.Complete(rt)
}
