package evaluator

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/token"
	"slug/internal/util"
	"strings"
	"sync/atomic"
)

var (
	NIL   = &object.Nil{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

const (
	precision        = 14
	roundingStrategy = dec64.RoundHalfUp
)

type ByteOp func(a, b byte) byte

func AndBytes(a, b byte) byte { return a & b }
func OrBytes(a, b byte) byte  { return a | b }
func XorBytes(a, b byte) byte { return a ^ b }

type Evaluator struct {
	Config         util.Configuration
	Sandbox        bool
	AllowedImports []string
	Modules        map[string]*object.Module

	envStack []*object.Environment // Environment stack encapsulated in an evaluator struct
	// callStack keeps track of the current function for things like `recur`
	// todo: fixme: this is a stack that will grow and eventually fail in a tailcall scenario.
	callStack []struct {
		FnName string
		FnObj  object.Object
	}
	nextID atomic.Int64
}

func (e *Evaluator) NextHandleID() int64 {
	return e.nextID.Add(1)<<16 | int64(rand.Intn(0xFFFF))
}

func (e *Evaluator) GetConfiguration() util.Configuration {
	return e.Config
}

func (e *Evaluator) Nil() *object.Nil {
	return NIL
}

func (e *Evaluator) PushEnv(env *object.Environment) {
	e.envStack = append(e.envStack, env)
	slog.Debug("push stack frame",
		slog.Int("stack-size", len(e.envStack)))
}

func (e *Evaluator) CurrentEnv() *object.Environment {
	// Access the current environment from the top frame
	if len(e.envStack) == 0 {
		panic("Environment stack is empty in the current frame")
	}
	return e.envStack[len(e.envStack)-1]
}

func (e *Evaluator) PopEnv(result object.Object) object.Object {
	if len(e.envStack) == 0 {
		panic("Attempted to pop from an empty environment stack")
	}

	finalResult := e.CurrentEnv().ExecuteDeferred(result, func(stmt ast.Statement) object.Object {
		return e.Eval(stmt)
	})

	e.envStack = e.envStack[:len(e.envStack)-1]
	slog.Debug("pop stack frame",
		slog.Int("stack-size", len(e.envStack)))

	return finalResult
}

// Helpers for tracking the current function (for `recur`)
func (e *Evaluator) pushCallFrame(fnName string, fnObj object.Object) {
	e.callStack = append(e.callStack, struct {
		FnName string
		FnObj  object.Object
	}{FnName: fnName, FnObj: fnObj})
}

func (e *Evaluator) popCallFrame() {
	if len(e.callStack) == 0 {
		return
	}
	e.callStack = e.callStack[:len(e.callStack)-1]
}

func (e *Evaluator) currentCallFrame() (string, object.Object, bool) {
	if len(e.callStack) == 0 {
		return "", nil, false
	}
	top := e.callStack[len(e.callStack)-1]
	return top.FnName, top.FnObj, true
}

func (e *Evaluator) Eval(node ast.Node) object.Object {
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
		return NIL

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

		var args []object.Object
		for _, arg := range node.Arguments {
			if spreadExpr, ok := arg.(*ast.SpreadExpression); ok {
				spreadValue := e.Eval(spreadExpr.Value)
				if e.isError(spreadValue) {
					return spreadValue
				}

				// Ensure the spread value is a list
				list, ok := spreadValue.(*object.List)
				if !ok {
					return e.newErrorfWithPos(spreadExpr.Token.Position, "spread operator can only be used on lists, got %s", spreadValue.Type())
				}

				// Append all elements of the list to args
				args = append(args, list.Elements...)
			} else {
				evaluatedArg := e.Eval(arg)
				if e.isError(evaluatedArg) {
					return evaluatedArg
				}
				args = append(args, evaluatedArg)
			}
		}

		// If this is a tail call, wrap it in a TailCall object instead of evaluating
		if node.IsTailCall {
			slog.Debug("Tail call",
				slog.Any("function", node.Token.Literal),
				slog.Any("argument-count", len(args)))

			return &object.TailCall{
				FnName:    node.Token.Literal,
				Function:  function,
				Arguments: args,
			}
		}

		slog.Debug("Function call",
			slog.Any("function", node.Token.Literal),
			slog.Any("argument-count", len(args)))
		// For non-tail calls, invoke the function directly
		return e.ApplyFunction(node.Token.Position, node.Token.Literal, function, args)

	case *ast.RecurExpression:
		// Evaluate arguments (respecting spread, same as call)
		var args []object.Object
		for _, arg := range node.Arguments {
			if spreadExpr, ok := arg.(*ast.SpreadExpression); ok {
				spreadValue := e.Eval(spreadExpr.Value)
				if e.isError(spreadValue) {
					return spreadValue
				}

				list, ok := spreadValue.(*object.List)
				if !ok {
					return e.newErrorfWithPos(spreadExpr.Token.Position, "spread operator can only be used on lists, got %s", spreadValue.Type())
				}
				args = append(args, list.Elements...)
			} else {
				evaluatedArg := e.Eval(arg)
				if e.isError(evaluatedArg) {
					return evaluatedArg
				}
				args = append(args, evaluatedArg)
			}
		}

		fnName, fnObj, ok := e.currentCallFrame()
		if !ok || fnObj == nil {
			// `recur` should only be valid inside a function body;
			// semantic checks should normally prevent this, but guard at runtime too.
			return e.newErrorWithPos(node.Token.Position, "recur used outside of a function")
		}

		slog.Debug("Tail recur",
			slog.Any("function", fnName),
			slog.Any("argument-count", len(args)))

		// Map directly to TailCall for the current function
		return &object.TailCall{
			FnName:    fnName,
			Function:  fnObj,
			Arguments: args,
		}

	case *ast.ListLiteral:
		elements := e.evalExpressions(node.Elements)
		if len(elements) == 1 && e.isError(elements[0]) {
			return elements[0]
		}
		return &object.List{Elements: elements}

	case *ast.IndexExpression:
		left := e.Eval(node.Left)
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

	case *ast.ThrowStatement:
		return e.evalThrowStatement(node)

	case *ast.DeferStatement:
		return e.evalDefer(node)

	}

	return nil
}

func (e *Evaluator) evalProgram(program *ast.Program) object.Object {
	//println("program")
	var result object.Object

	for _, statement := range program.Statements {
		result = e.Eval(statement)

		for {
			if returnVal, ok := result.(*object.TailCall); ok {
				result = e.ApplyFunction(0, returnVal.FnName, returnVal.Function, returnVal.Arguments)
				//println("tail call", result.Type())
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

func (e *Evaluator) LoadModule(modName string) (*object.Module, error) {

	if e.Sandbox {
		allowed := false
		for _, m := range e.AllowedImports {
			if m == modName {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, errors.New("module not allowed")
		}
	}

	if e.Modules == nil {
		e.Modules = make(map[string]*object.Module)
	}

	if mod, ok := e.Modules[modName]; ok {
		return mod, nil
	}

	// 1. Resolve module name to relative file path (e.g., "slug.std" -> "slug/std.slug")
	pathParts := strings.Split(modName, ".")
	relPath := filepath.Join(pathParts...) + ".slug"

	// 2. Search Paths: Check local RootPath, then SLUG_HOME/lib
	var fullPath string
	var source []byte
	var err error

	// Try local RootPath (directory of the entry script)
	fullPath = filepath.Join(e.Config.RootPath, relPath)
	source, errFirst := os.ReadFile(fullPath)

	if errFirst != nil {
		//// Fallback to $SLUG_HOME/lib
		if e.Config.SlugHome != "" {
			fullPath = filepath.Join(e.Config.SlugHome, "lib", relPath)
			source, err = os.ReadFile(fullPath)
			if err != nil {
				return nil, fmt.Errorf("could not load module %s: local error: %v, lib error: %v", modName, errFirst, err)
			}
		} else {
			return nil, fmt.Errorf("could not load module %s: %v (SLUG_HOME not set)", modName, errFirst)
		}
	}

	// 3. Tokenize and Parse
	l := lexer.New(string(source))
	p := parser.New(l, fullPath, string(source))
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors in module %s:\n%s", modName, strings.Join(p.Errors(), "\n"))
	}

	if e.Config.DebugJsonAST {
		json, err := parser.RenderASTAsJSON(program)
		if err != nil {
			slog.Error("Failed to render AST as JSON",
				slog.Any("error", err))
		} else {
			jsonPath := fullPath + ".ast.json"
			err = os.WriteFile(jsonPath, []byte(json), 0644)
			if err != nil {
				slog.Error("Failed to write AST as JSON")
			}
		}
	}
	if e.Config.DebugTxtAST {
		txtPath := fullPath + ".ast.txt"
		text := parser.RenderASTAsText(program, 0)
		err = os.WriteFile(txtPath, []byte(text), 0644)
		if err != nil {
			slog.Error("Failed to write AST as JSON")
		}
	}

	// 4. Setup Module Object and Environment
	moduleEnv := object.NewEnvironment()
	moduleEnv.Path = fullPath
	moduleEnv.ModuleFqn = modName
	moduleEnv.Src = string(source)

	module := &object.Module{
		Name:    modName,
		Path:    fullPath,
		Src:     string(source),
		Program: program,
		Env:     moduleEnv,
	}

	e.Modules[modName] = module

	// 5. Evaluate the module in its own environment
	slog.Debug("loading module", slog.String("name", modName), slog.String("path", fullPath))

	e.PushEnv(moduleEnv)
	out := e.Eval(program)
	// We pop the env, but the moduleEnv now contains all the defined bindings
	e.PopEnv(out)

	if e.isError(out) {
		return nil, fmt.Errorf("runtime error while loading module %s: %s", modName, out.Inspect())
	}

	return module, nil
}

func (e *Evaluator) mapIdentifiersToStrings(identifiers []*ast.Identifier) []string {
	parts := []string{}
	for _, id := range identifiers {
		parts = append(parts, id.Value)
	}
	return parts
}

func (e *Evaluator) evalBlockStatement(block *ast.BlockStatement) (result object.Object) {
	// Create a new environment with an associated stack frame
	//println("block")
	blockEnv := object.NewEnclosedEnvironment(e.CurrentEnv(), &object.StackFrame{
		Function: "block",
		File:     e.CurrentEnv().Path,
		Src:      e.CurrentEnv().Src,
		Position: block.Token.Position,
	})
	e.PushEnv(blockEnv)
	result = NIL
	defer func() { result = e.PopEnv(result) }()

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
	return NIL
}

func (e *Evaluator) NativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func (e *Evaluator) evalPrefixExpression(operator string, right object.Object) object.Object {
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

func (e *Evaluator) evalInfixExpression(
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

func (e *Evaluator) evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NIL:
		return TRUE
	default:
		return FALSE
	}
}

func (e *Evaluator) evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.NUMBER_OBJ {
		return e.newErrorf("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Number).Value
	return &object.Number{Value: value.Neg()}
}

func (e *Evaluator) evalComplementPrefixOperatorExpression(right object.Object) object.Object {
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

func (e *Evaluator) evalShortCircuitInfixExpression(left object.Object, node *ast.InfixExpression) object.Object {

	// Short circuit based on left value and operator
	switch node.Operator {
	case "&&":
		// If left is false, return false without evaluating right
		if !e.isTruthy(left) {
			return FALSE
		}
		// Otherwise, evaluate and return right
		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}
		if e.isTruthy(right) {
			return TRUE
		}
		return FALSE

	case "||":
		// If left is true, return true without evaluating right
		if e.isTruthy(left) {
			return TRUE
		}
		// Otherwise, evaluate and return right
		right := e.Eval(node.Right)
		if e.isError(right) {
			return right
		}
		if e.isTruthy(right) {
			return TRUE
		}
		return FALSE

	default:
		return e.newErrorfWithPos(node.Token.Position, "unknown operator for short-circuit evaluation: %s", node.Operator)
	}
}

func (e *Evaluator) evalBooleanInfixExpression(
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

func (e *Evaluator) evalNumberInfixExpression(
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

func (e *Evaluator) evalStringInfixExpression(
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

func (e *Evaluator) evalStringPlusOtherInfixExpression(
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

func (e *Evaluator) evalStringMultiplication(
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

func (e *Evaluator) evalListInfixExpression(
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

func (e *Evaluator) evalBytesInfixExpression(
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

func (e *Evaluator) doOp(right object.Object, left object.Object, op ByteOp) object.Object {
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

func (e *Evaluator) evalIfExpression(
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
		return NIL
	}
}

func (e *Evaluator) evalIdentifier(
	node *ast.Identifier,
) object.Object {

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	if val, ok := e.CurrentEnv().Get(node.Value); ok {
		// If it is a Module, return the Module object itself
		if module, ok := val.(*object.Module); ok {
			return module
		}
		return val
	}

	return e.newErrorWithPos(node.Token.Position, "identifier not found: "+node.Value)
}

func (e *Evaluator) isTruthy(obj object.Object) bool {
	switch obj {
	case NIL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func (e *Evaluator) NewError(format string, a ...interface{}) *object.Error {
	return e.newErrorfWithPos(0, format, a...)
}

func (e *Evaluator) newErrorfWithPos(pos int, format string, a ...interface{}) *object.Error {
	m := fmt.Sprintf(format, a...)
	return e.newErrorWithPos(pos, m)
}

func (e *Evaluator) newErrorWithPos(pos int, m string) *object.Error {

	if pos == 0 {
		return &object.Error{Message: m}
	}

	env := e.CurrentEnv()

	line, col := util.GetLineAndColumn(env.Src, pos)

	var errorMsg bytes.Buffer
	errorMsg.WriteString(fmt.Sprintf("\nError: %s\n", m))
	errorMsg.WriteString(fmt.Sprintf("    --> %s:%d:%d\n", env.Path, line, col))

	lines := util.GetContextLines(env.Src, line, col)
	errorMsg.WriteString(lines)

	return &object.Error{Message: errorMsg.String()}
}

func (e *Evaluator) newErrorf(format string, a ...interface{}) *object.Error {
	msg := fmt.Sprintf(format, a...)
	return &object.Error{Message: msg}
}

func (e *Evaluator) isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func (e *Evaluator) evalExpressions(
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

func (e *Evaluator) ApplyFunction(pos int, fnName string, fnObj object.Object, args []object.Object) object.Object {
	switch fn := fnObj.(type) {
	case *object.FunctionGroup:

		f, err := fn.DispatchToFunction(fnName, args)
		if err != nil {
			return e.newErrorfWithPos(pos, "error calling function '%s': %s", fnName, err.Error())
		} else {
			return e.ApplyFunction(pos, fnName, f, args)
		}

	case *object.Function:

		// Track current function for `recur`
		e.pushCallFrame(fnName, fn)
		defer e.popCallFrame()

		// Create a new call frame and push it
		e.PushEnv(e.extendFunctionEnv(fn, args))

		// Evaluate function body
		result := e.Eval(fn.Body)

		result = e.PopEnv(result)

		for {
			if returnVal, ok := result.(*object.TailCall); ok {
				result = e.ApplyFunction(pos, fnName, returnVal.Function, returnVal.Arguments)
			} else if returnVal, ok := result.(*object.ReturnValue); ok {
				result = returnVal.Value
			} else {
				break
			}
		}

		return result

	case *object.Foreign:
		var result object.Object
		func() {
			defer func() {
				if r := recover(); r != nil {
					println(r.(error).Error())
					result = e.newErrorfWithPos(pos, "error calling foreign function '%s'", fn.Name)
				}
			}()
			result = fn.Fn(e, args...)
		}()
		return result

	default:
		if fn == nil {
			return e.newErrorWithPos(pos, "no function found!")
		}
		return e.newErrorfWithPos(pos, "not a function: %s", fn.Type())
	}
}

func (e *Evaluator) extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env, &object.StackFrame{
		Function: "call: " + e.callStack[len(e.callStack)-1].FnName,
		File:     fn.Env.Path,
		Position: fn.Body.Token.Position,
		Src:      fn.Env.Src,
	})
	numArgs := len(args)

	for i, param := range fn.Parameters {

		// Handle variadic arguments
		if param.IsVariadic && len(args) >= i {
			env.Define(param.Name.Value, &object.List{
				Elements: args[i:], // Remaining args as a list
			}, false, false)
			break
		}

		// Handle default values
		if i >= numArgs {
			if param.Default != nil {
				defaultValue := e.Eval(param.Default)
				env.Define(param.Name.Value, defaultValue, false, false)
			} else {
				env.Define(param.Name.Value, NIL, false, false)
			}
		} else {
			env.Define(param.Name.Value, args[i], false, false)
		}
	}

	return env
}

func (e *Evaluator) unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func (e *Evaluator) evalMapLiteral(
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

func (e *Evaluator) evalMatchExpression(node *ast.MatchExpression) object.Object {
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
	return NIL
}

func (e *Evaluator) evalMatchCase(matchValue object.Object, matchCase *ast.MatchCase) (result object.Object, matched bool) {
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
func (e *Evaluator) patternMatches(
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
	}

	// Unhandled pattern type
	return false, nil
}

func (e *Evaluator) patternMatchesList(
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

func (e *Evaluator) patternMatchesBytes(
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
func (e *Evaluator) evaluatePatternAsCondition(pattern ast.MatchPattern) bool {
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
func (e *Evaluator) objectsEqual(a, b object.Object) bool {
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
	}

	return false
}

func (e *Evaluator) evalMapIndexExpression(pos int, obj, index object.Object) object.Object {
	mapObj := obj.(*object.Map)

	key, ok := index.(object.Hashable)
	if !ok {
		return e.newErrorfWithPos(pos, "unusable as map key: %s", index.Type())
	}

	pair, ok := mapObj.Pairs[key.MapKey()]
	if !ok {
		return NIL
	}

	return pair.Value
}

func (e *Evaluator) evalThrowStatement(node *ast.ThrowStatement) object.Object {
	val := e.Eval(node.Value)
	if e.isError(val) {
		return val
	}
	env := e.CurrentEnv()
	return &object.RuntimeError{
		Payload: val,
		StackTrace: e.GatherStackTrace(&object.StackFrame{
			Function: "throw",
			File:     env.Path,
			Src:      env.Src,
			Position: node.Token.Position,
		}),
	}
}

func (e *Evaluator) GatherStackTrace(frame *object.StackFrame) []*object.StackFrame {
	var trace []*object.StackFrame
	if frame != nil {
		trace = append(trace, frame)
	}
	// Walk the envStack from top (current) to bottom
	for i := len(e.envStack) - 1; i >= 0; i-- {
		env := e.envStack[i]
		if env.StackInfo != nil {
			//sf := object.StackFrame{
			//	Src:      env.Src,
			//	File:     env.Path,
			//	Position: env.StackInfo.Position,
			//	Function: env.StackInfo.Function,
			//}
			trace = append(trace, env.StackInfo)
		}
	}
	return trace // Already in correct order (most recent first)
}

func (e *Evaluator) evalIndexExpression(pos int, left, index object.Object) object.Object {

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
	default:
		return e.newErrorfWithPos(pos, "index operator not supported: %s", left.Type())
	}
}

func (e *Evaluator) evalSliceExpression(node *ast.SliceExpression) object.Object {
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

func (e *Evaluator) evalListIndexExpression(pos int, list, index object.Object) object.Object {
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
		return NIL
	}

	return listObject.Elements[idx]
}

func (e *Evaluator) evalByteIndexExpression(list, index object.Object) object.Object {
	bytesObject := list.(*object.Bytes)
	idx := index.(*object.Number).Value.ToInt64()
	max := int64(len(bytesObject.Value) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return NIL
	}

	return &object.Number{Value: dec64.FromInt(int(bytesObject.Value[idx]))}
}

func (e *Evaluator) evalStringIndexExpression(pos int, str, index object.Object) object.Object {
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
		return NIL
	}

	return &object.String{Value: string(runes[idx])}
}

func (e *Evaluator) evalListSlice(elements []object.Object, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(elements), slice)
	var result []object.Object
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.List{Elements: result}
}

func (e *Evaluator) evalByteSlice(elements []byte, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(elements), slice)
	var result []byte
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.Bytes{Value: result}
}

func (e *Evaluator) evalStringSlice(value string, slice *object.Slice) object.Object {
	runes := []rune(value)
	start, end, step := e.computeSliceIndices(len(runes), slice)
	var b strings.Builder
	for i := start; i < end; i += step {
		b.WriteRune(runes[i])
	}
	return &object.String{Value: b.String()}
}

func (e *Evaluator) computeSliceIndices(length int, slice *object.Slice) (int, int, int) {
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

func (e *Evaluator) evalForeignFunctionDeclaration(ff *ast.ForeignFunctionDeclaration) object.Object {
	env := e.CurrentEnv()
	modulePath := env.ModuleFqn
	functionName := ff.Name.Value

	fqn := modulePath + "." + functionName

	if foreignFn, exists := lookupForeign(fqn); exists {
		foreignFn.Tags = e.evalTags(ff.Tags)
		foreignFn.Parameters = ff.Parameters
		foreignFn.Name = functionName
		foreignFn.Signature = ff.Signature
		isExported := hasExportTag(ff.Tags)
		_, err := env.Define(functionName, foreignFn, isExported, false)
		if err != nil {
			return e.newErrorWithPos(ff.Token.Position, err.Error())
		}
		return NIL
	}
	return e.newErrorfWithPos(ff.Token.Position, "unknown foreign function %s", fqn)
}

func (e *Evaluator) evalDefer(deferStmt *ast.DeferStatement) object.Object {
	// Register the defer statement into the environment's defer stack
	e.CurrentEnv().RegisterDefer(deferStmt)
	return nil // Defer statements do not produce a direct result
}

func (e *Evaluator) applyTagsIfPresent(tags []*ast.Tag, val object.Object) object.Object {
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

func (e *Evaluator) evalTags(tags []*ast.Tag) map[string]object.List {
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

func hasExportTag(tags []*ast.Tag) bool {
	for _, tag := range tags {
		if tag.Name == object.EXPORT_TAG {
			return true
		}
	}
	return false
}

func byteValue(n *object.Number) (byte, error) {

	value := n.Value.ToInt()
	if value < 0 || value > 255 {
		return 0, errors.New("byte must be between 0 and 255, got " + n.Inspect())
	}
	return byte(value), nil
}
