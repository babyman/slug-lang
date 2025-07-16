package evaluator

import (
	"fmt"
	"slug/internal/ast"
	"slug/internal/dec64"
	"slug/internal/log"
	"slug/internal/object"
	"slug/internal/token"
	"strings"
)

var (
	NIL   = &object.Nil{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

type Evaluator struct {
	envStack []*object.Environment // Environment stack encapsulated in an evaluator struct
	Actor    *Actor                // can be null
}

func (e *Evaluator) PID() int64 {
	if e.Actor == nil {
		return 0
	}
	return e.Actor.MailboxPID
}

func (e *Evaluator) Receive(timeout int64) (object.Object, bool) {
	if e.Actor == nil {
		return nil, false
	}
	message, b := e.Actor.WaitForMessage(timeout)
	if !b {
		return nil, false
	}
	log.Debug("ACT: %d (%d) message received: %s\n", e.Actor.PID, e.Actor.MailboxPID, message.String())
	switch m := message.(type) {
	case UserMessage:
		return m.Payload.(object.Object), true
	case ActorExited:
		notification := &object.Map{}
		notification.Put(&object.String{Value: "tag"}, &object.String{Value: "__exit__"})
		notification.Put(&object.String{Value: "from"}, &object.Number{Value: dec64.FromInt64(m.MailboxPID)})
		notification.Put(&object.String{Value: "fn"}, m.Function)
		notification.Put(&object.String{Value: "args"}, &object.List{Elements: m.Args})
		notification.Put(&object.String{Value: "reason"}, &object.String{Value: m.Reason})
		notification.Put(&object.String{Value: "result"}, m.Result)

		if m.Result != nil {
			if lm, ok := (*m.LastMessage).(UserMessage); ok {
				notification.Put(&object.String{Value: "lastMessage"}, lm.Payload.(object.Object))
			}
		}

		messages := &object.List{}
		for _, msg := range m.QueuedMessages {
			messages.Elements = append(messages.Elements, msg.Payload.(object.Object))
		}
		notification.Put(&object.String{Value: "mailbox"}, messages)
		return notification, true
	default:
		log.Warn("ACT: %d (%d) Unknown message type: %T", e.Actor.PID, e.Actor.MailboxPID, m)
		notification := &object.Map{}
		notification.Put(&object.String{Value: "tag"}, &object.String{Value: "__unknown_message__"})
		notification.Put(&object.String{Value: "type"}, &object.String{Value: fmt.Sprintf("%T", m)})
		return notification, true
	}
}

func (e *Evaluator) Nil() *object.Nil {
	return NIL
}

func (e *Evaluator) PushEnv(env *object.Environment) {
	e.envStack = append(e.envStack, env)
	//log.Trace(">%s", strings.Repeat("-", len(e.envStack)))
}

func (e *Evaluator) CurrentEnv() *object.Environment {
	// Access the current environment from the top frame
	if len(e.envStack) == 0 {
		panic("Environment stack is empty in the current frame")
	}
	return e.envStack[len(e.envStack)-1]
}

func (e *Evaluator) PopEnv() {
	if len(e.envStack) == 0 {
		panic("Attempted to pop from an empty environment stack")
	}
	e.CurrentEnv().ExecuteDeferred(func(stmt ast.Statement) {
		e.Eval(stmt)
	})
	e.envStack = e.envStack[:len(e.envStack)-1]
	//log.Trace("<%s", strings.Repeat("-", len(e.envStack)))
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
		if _, err := e.patternMatches(node.Pattern, variable, false, isExported, false); err != nil {
			return newError(err.Error())
		}
		return e.applyTagsIfPresent(node.Tags, variable)

	case *ast.ValExpression:
		value := e.Eval(node.Value)
		if e.isError(value) {
			return value
		}
		isExported := hasExportTag(node.Tags)
		if _, err := e.patternMatches(node.Pattern, value, true, isExported, false); err != nil {
			return newError(err.Error())
		}
		e.applyTagsIfPresent(node.Tags, value)
		return value // Return the assigned value

	case *ast.ForeignFunctionDeclaration:
		return e.evalForeignFunctionDeclaration(node)

	// Expressions
	case *ast.NumberLiteral:
		return &object.Number{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

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
				return newError("left side of assignment must be an identifier")
			}

			// Evaluate right side
			right := e.Eval(node.Right)
			if e.isError(right) {
				return right
			}

			// Try to assign the value (variable is already defined)
			val, err := e.CurrentEnv().Assign(ident.Value, right)
			if err != nil {
				return newError(err.Error())
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
					return newError("spread operator can only be used on lists, got %s", spreadValue.Type())
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
			//log.Trace("function tail call")
			log.Info("Tail calling %s with %d arguments", node.Token.Literal, len(args))
			return &object.TailCall{
				FnName:    node.Token.Literal,
				Function:  function,
				Arguments: args,
			}
		}

		log.Info("Calling %s with %d arguments", node.Token.Literal, len(args))
		// For non-tail calls, invoke the function directly
		return e.ApplyFunction(node.Token.Literal, function, args)

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
		return e.evalIndexExpression(left, index)

	case *ast.SliceExpression:
		return e.evalSliceExpression(node)

	case *ast.MapLiteral:
		return e.evalMapLiteral(node)

	case *ast.ThrowStatement:
		return e.evalThrowStatement(node)

	case *ast.TryCatchStatement:
		return e.evalTryCatchStatement(node)

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
				result = e.ApplyFunction(returnVal.FnName, returnVal.Function, returnVal.Arguments)
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

func (e *Evaluator) LoadModule(pathParts []string) (*object.Module, error) {
	module, err := LoadModule(pathParts)
	if err != nil {
		return nil, err
	}
	if module.Env != nil {
		log.Debug("return loaded module: %v", module.Name)
		return module, nil
	}

	// if the module is not in the registry, create a new Module object and cache it
	moduleEnv := object.NewEnvironment()
	moduleEnv.Path = module.Path
	moduleEnv.ModuleFqn = module.Name
	moduleEnv.Src = module.Src

	// Evaluate the module
	module.Env = moduleEnv

	log.Debug("load module: %v\n", module.Name)
	e.PushEnv(moduleEnv)
	e.Eval(module.Program)
	e.PopEnv()
	log.Info("Module %s env len %d\n", module.Name, len(module.Env.Bindings))

	// Import the module into the current environment
	return module, nil
}

func (e *Evaluator) mapIdentifiersToStrings(identifiers []*ast.Identifier) []string {
	parts := []string{}
	for _, id := range identifiers {
		parts = append(parts, id.Value)
	}
	return parts
}

func (e *Evaluator) evalBlockStatement(block *ast.BlockStatement) object.Object {
	// Create a new environment with an associated stack frame
	//println("block")
	blockEnv := object.NewEnclosedEnvironment(e.CurrentEnv(), &object.StackFrame{
		Function: "block",
		File:     e.CurrentEnv().Path,
		Position: block.Token.Position,
	})
	e.PushEnv(blockEnv)
	defer e.PopEnv()

	// Variable to store the result of the evaluation
	var result object.Object

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

	return result
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
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func (e *Evaluator) evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.NUMBER_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return e.evalStringInfixExpression(operator, left, right)
	case operator == "*" && left.Type() == object.STRING_OBJ && right.Type() == object.NUMBER_OBJ:
		return e.evalStringMultiplication(left, right)
	case operator == "==":
		return e.NativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return e.NativeBoolToBooleanObject(left != right)
	case operator == ":+" && left.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)
	case operator == "+:" && right.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)
	case left.Type() == object.LIST_OBJ && right.Type() == object.LIST_OBJ:
		return e.evalListInfixExpression(operator, left, right)
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return e.evalBooleanInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ || right.Type() == object.STRING_OBJ:
		return e.evalStringPlusOtherInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
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
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Number).Value
	return &object.Number{Value: value.Neg()}
}

func (e *Evaluator) evalComplementPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.NUMBER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Number).Value
	return &object.Number{Value: ^value}
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
		return newError("unknown operator for short-circuit evaluation: %s", node.Operator)
	}
}

func (e *Evaluator) evalBooleanInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value

	switch operator {
	case "&&":
		return e.NativeBoolToBooleanObject(leftVal && rightVal)
	case "||":
		return e.NativeBoolToBooleanObject(leftVal || rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Evaluator) evalIntegerInfixExpression(
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
		return &object.Number{Value: leftVal.Div(rightVal, 14, dec64.RoundHalfUp)} // todo make this a constant at least, should really be config
	case "%":
		return &object.Number{Value: leftVal.Mod(rightVal)}
	case "&":
		return &object.Number{Value: leftVal & rightVal}
	case "|":
		return &object.Number{Value: leftVal | rightVal}
	case "^":
		return &object.Number{Value: leftVal ^ rightVal}
	case "<<":
		return &object.Number{Value: leftVal << rightVal}
	case ">>":
		return &object.Number{Value: leftVal >> rightVal}
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
		return newError("unknown operator: %s %s %s",
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
	default:
		return newError("unknown operator: %s %s %s",
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
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Evaluator) evalStringMultiplication(
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.Number).Value.ToInt64()

	if rightVal < 0 {
		return newError("repetition count must be a non-negative NUMBER, got %d", rightVal)
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
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
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

	return newError("identifier not found: " + node.Value)
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
	return newError(format, a...)
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
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

func (e *Evaluator) ApplyFunction(fnName string, fnObj object.Object, args []object.Object) object.Object {
	switch fn := fnObj.(type) {
	case *object.FunctionGroup:

		f, ok := fn.DispatchToFunction(fnName, args)
		if ok {
			return e.ApplyFunction(fnName, f, args)
		} else {
			return f
		}

	case *object.Function:

		// Create a new call frame and push it
		//println("call", fn.HasTailCall)
		e.PushEnv(e.extendFunctionEnv(fn, args))

		// Evaluate function body
		result := e.Eval(fn.Body)

		e.PopEnv()

		for {
			if returnVal, ok := result.(*object.TailCall); ok {
				result = e.ApplyFunction(fnName, returnVal.Function, returnVal.Arguments)
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
					result = newError("error calling foreign function %s", fn.Name)
				}
			}()
			result = fn.Fn(e, args...)
		}()
		return result

	default:
		if fn == nil {
			return newError("no function found!")
		}
		return newError("not a function: %s", fn.Type())
	}
}

func (e *Evaluator) extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env, &object.StackFrame{
		Function: "call", // String representation of the function
		File:     fn.Env.Path,
		Position: fn.Body.Token.Position,
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
			return newError("unusable as map key: %s", key.Type())
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

func (e *Evaluator) evalMatchCase(matchValue object.Object, matchCase *ast.MatchCase) (object.Object, bool) {
	patternEnv := object.NewEnclosedEnvironment(e.CurrentEnv(), nil)
	e.PushEnv(patternEnv)
	defer e.PopEnv()

	// Match against the provided value or evaluate the condition
	matched := false
	var err error
	if matchValue != nil {
		// Match the case's pattern against the matchValue
		matched, err = e.patternMatches(matchCase.Pattern, matchValue, false, false, false)
		if err != nil {
			return newError("pattern match error: %s", err.Error()), true
		}
	} else {
		// Valueless match condition
		matched = e.evaluatePatternAsCondition(matchCase.Pattern)
	}

	// Evaluate guard condition if pattern matches
	if matched && matchCase.Guard != nil {
		guardResult := e.Eval(matchCase.Guard)
		if e.isError(guardResult) {
			return guardResult, true
		}
		matched = e.isTruthy(guardResult)
	}

	// If the pattern matched, evaluate the body
	if matched {
		return e.Eval(matchCase.Body), true
	}
	return nil, false
}

// e.patternMatches checks if a value matches a pattern and binds variables
func (e *Evaluator) patternMatches(pattern ast.MatchPattern, value object.Object, isConstant bool, isExport bool, isImport bool) (bool, error) {
	env := e.CurrentEnv()
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard matches anything
		return true, nil

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
			matched, err := e.patternMatches(subPattern, value, isConstant, isExport, isImport)
			e.PopEnv()
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
		arr, ok := value.(*object.List)
		if !ok {
			return false, nil
		}

		// Empty list pattern matches empty list
		if len(p.Elements) == 0 {
			return len(arr.Elements) == 0, nil
		}

		_, isSpread := p.Elements[len(p.Elements)-1].(*ast.SpreadPattern)

		// Check if list length matches pattern length
		if (len(p.Elements) != len(arr.Elements) && !isSpread) || (len(p.Elements) > len(arr.Elements)+1 && isSpread) {
			return false, nil
		}

		// scoped environment to capture the match bindings, these will be copied to the parent env on success
		scoped := object.NewEnclosedEnvironment(env, nil)
		e.PushEnv(scoped)

		for i, elemPattern := range p.Elements {
			if spread, isSpread := elemPattern.(*ast.SpreadPattern); isSpread {
				matched, err := e.patternMatches(spread, &object.List{Elements: arr.Elements[i:]}, isConstant, isExport, isImport)
				if err != nil || !matched {
					return false, err
				}
				break
			} else {
				matched, err := e.patternMatches(elemPattern, arr.Elements[i], isConstant, isExport, isImport)
				if err != nil || !matched {
					return false, err
				}
			}
		}

		// Copy bindings from scoped environment to parent environment
		for name, binding := range scoped.Bindings {
			value, _ := scoped.Get(name)
			if binding.IsMutable {
				if _, err := env.Define(name, value, isExport, isImport); err != nil {
					return false, err
				}
			} else {
				if _, err := env.DefineConstant(name, value, isExport, isImport); err != nil {
					return false, err
				}
			}
		}

		e.PopEnv()

		return true, nil

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
		defer e.PopEnv()

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
			matched, err := e.patternMatches(subPattern, pair.Value, isConstant, isExport, isImport)
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
					_, err := e.patternMatches(pair, &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}, isConstant, isExport, isImport)
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
					_, err := e.patternMatches(pair, &object.Map{Pairs: copiedPairs}, isConstant, isExport, isImport)
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

func (e *Evaluator) evalMapIndexExpression(obj, index object.Object) object.Object {
	mapObj := obj.(*object.Map)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as map key: %s", index.Type())
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
	if val.Type() != object.MAP_OBJ {
		return newError("throw argument must be a Map, got %s", val.Type())
	}
	env := e.CurrentEnv()
	return &object.RuntimeError{
		Payload: val,
		StackTrace: env.GatherStackTrace(&object.StackFrame{
			Function: "throw",
			File:     env.Path,
			Position: node.Token.Position,
		}),
	}
}

func (e *Evaluator) evalTryCatchStatement(node *ast.TryCatchStatement) object.Object {
	// Execute the try block
	tryResult := e.Eval(node.TryBlock)

	err, isRuntimeError := tryResult.(*object.RuntimeError)

	// If there's no error, return the result of the try block
	if !isRuntimeError {
		return tryResult
	}

	// Match the error Payload against catch block patterns
	catchMatch := node.CatchBlock

	// Otherwise, handle the error (it must be a RuntimeError)
	env := e.CurrentEnv()
	catchEnv := object.NewEnclosedEnvironment(env, nil)
	e.PushEnv(catchEnv)
	defer e.PopEnv()

	// bind the error payload to the catch pattern
	_, bindError := catchEnv.Define(catchMatch.Value.String(), err.Payload, false, false)
	if bindError != nil {
		return newError(bindError.Error())
	}

	return e.evalMatchExpression(catchMatch)
}

func (e *Evaluator) evalIndexExpression(left, index object.Object) object.Object {

	switch {
	case left.Type() == object.STRING_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if str, ok := left.(*object.String); ok {
				return e.evalStringSlice(str.Value, slice)
			}
		}
		return e.evalStringIndexExpression(left, index)
	case left.Type() == object.LIST_OBJ && index.Type() == object.NUMBER_OBJ:
		return e.evalListIndexExpression(left, index)
	case left.Type() == object.LIST_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if arr, ok := left.(*object.List); ok {
				return e.evalListSlice(arr.Elements, slice)
			}
		}
		return e.evalListIndexExpression(left, index)
	case left.Type() == object.MAP_OBJ:
		return e.evalMapIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
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

func (e *Evaluator) evalListIndexExpression(list, index object.Object) object.Object {
	listObject := list.(*object.List)
	idx := index.(*object.Number).Value.ToInt64()
	max := int64(len(listObject.Elements) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return NIL
	}

	return listObject.Elements[idx]
}

func (e *Evaluator) evalStringIndexExpression(str, index object.Object) object.Object {
	stringObject := str.(*object.String)
	idx := index.(*object.Number).Value.ToInt64()
	max := int64(len(stringObject.Value) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return NIL
	}

	return &object.String{Value: string(stringObject.Value[idx])}
}

func (e *Evaluator) evalListSlice(elements []object.Object, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(elements), slice)
	var result []object.Object
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.List{Elements: result}
}

func (e *Evaluator) evalStringSlice(value string, slice *object.Slice) object.Object {
	start, end, step := e.computeSliceIndices(len(value), slice)
	var result string
	for i := start; i < end; i += step {
		result += string(value[i])
	}
	return &object.String{Value: result}
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
			return newError(err.Error())
		}
		return NIL
	}
	return newError("unknown foreign function %s", fqn)
}

func (e *Evaluator) evalDefer(deferStmt *ast.DeferStatement) object.Object {
	// Register the defer statement into the environment's defer stack
	e.CurrentEnv().RegisterDefer(deferStmt.Call)
	return nil // Defer statements do not produce a direct result
}

func (e *Evaluator) applyTagsIfPresent(tags []*ast.Tag, val object.Object) object.Object {
	if tags != nil {
		switch t := val.(type) {
		case *object.Function:
			t.Tags = e.evalTags(tags)
		case *object.Foreign:
			t.Tags = e.evalTags(tags)
		case *object.Map:
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
