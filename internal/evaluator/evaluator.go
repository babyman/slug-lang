package evaluator

import (
	"fmt"
	"slug/internal/ast"
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
}

func (e *Evaluator) PushEnv(env *object.Environment) {
	e.envStack = append(e.envStack, env)
	//println(">", strings.Repeat("-", len(e.envStack)))
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
	//println("<", strings.Repeat("-", len(e.envStack)))
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

	case *ast.ImportStatement:
		return e.evalImportStatement(node)

	case *ast.MatchExpression:
		return e.evalMatchExpression(node)

	case *ast.VarStatement:
		val := e.Eval(node.Value)
		if e.isError(val) {
			return val
		}
		if _, err := e.patternMatches(node.Pattern, val, false); err != nil {
			return newError(err.Error())
		}

	case *ast.ValStatement:
		val := e.Eval(node.Value)
		if e.isError(val) {
			return val
		}
		if _, err := e.patternMatches(node.Pattern, val, true); err != nil {
			return newError(err.Error())
		}

	// Expressions
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

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
			//println("tail call")
			return &object.TailCall{
				Function:  function,
				Arguments: args,
			}
		}

		// For non-tail calls, invoke the function directly
		return e.applyFunction(function, args)

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

	case *ast.ForeignFunctionDeclaration:
		return e.evalForeignFunctionDeclaration(node)

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
				result = e.applyFunction(returnVal.Function, returnVal.Arguments)
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

func (e *Evaluator) evalImportStatement(importStatement *ast.ImportStatement) object.Object {
	module, err := LoadModule(e.mapIdentifiersToStrings(importStatement.PathParts))
	if err != nil {
		return newError(err.Error())
	}
	if module.Env != nil {
		return e.handleModuleImport(importStatement, module)
	}

	// if the module is not in the registry, create a new Module object and cache it
	moduleEnv := object.NewEnvironment()
	moduleEnv.Path = module.Path
	moduleEnv.ModuleFqn = module.Name
	moduleEnv.Src = module.Src

	// Evaluate the module
	module.Env = moduleEnv

	//println("import", module.Name)
	e.PushEnv(moduleEnv)
	e.Eval(module.Program)
	e.PopEnv()

	// Import the module into the current environment
	return e.handleModuleImport(importStatement, module)
}

func (e *Evaluator) handleModuleImport(importStatement *ast.ImportStatement, module *object.Module) object.Object {
	// Handle named symbols import, wildcard import, or namespace import
	env := e.CurrentEnv()

	if importStatement.Wildcard {
		// Import all symbols into the current environment
		for name, val := range module.Env.Store {
			if val.IsConstant {
				if _, err := env.DefineConstant(name, val.Value); err != nil {
					return newError(err.Error())
				}
			} else {
				if _, err := env.Define(name, val.Value); err != nil {
					return newError(err.Error())
				}
			}
		}
	} else if len(importStatement.Symbols) > 0 {
		// Import specific symbols
		for _, sym := range importStatement.Symbols {
			if binding, ok := module.Env.GetBinding(sym.Name.Value); ok {
				alias := sym.Name.Value
				if sym.Alias != nil {
					alias = sym.Alias.Value
				}
				if binding.IsConstant {
					if _, err := env.DefineConstant(alias, binding.Value); err != nil {
						return newError(err.Error())
					}
				} else {
					if _, err := env.Define(alias, binding.Value); err != nil {
						return newError(err.Error())
					}
				}
			} else {
				return newError("symbol '%s' not found in module '%s'", sym.Name.Value, module.Name)
			}
		}
	}

	return NIL
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

func nativeBoolToBooleanObject(input bool) *object.Boolean {
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
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return e.evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return e.evalStringInfixExpression(operator, left, right)
	case operator == "*" && left.Type() == object.STRING_OBJ && right.Type() == object.INTEGER_OBJ:
		return e.evalStringMultiplication(left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
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
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func (e *Evaluator) evalComplementPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: ^value}
}

func (e *Evaluator) evalBooleanInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value

	switch operator {
	case "&&":
		return nativeBoolToBooleanObject(leftVal && rightVal)
	case "||":
		return nativeBoolToBooleanObject(leftVal || rightVal)
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func (e *Evaluator) evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "%":
		return &object.Integer{Value: leftVal % rightVal}
	case "&":
		return &object.Integer{Value: leftVal & rightVal}
	case "|":
		return &object.Integer{Value: leftVal | rightVal}
	case "^":
		return &object.Integer{Value: leftVal ^ rightVal}
	case "<<":
		return &object.Integer{Value: leftVal << rightVal}
	case ">>":
		return &object.Integer{Value: leftVal >> rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
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
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
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
	rightVal := right.(*object.Integer).Value

	if rightVal < 0 {
		return newError("repetition count must be a non-negative INTEGER, got %d", rightVal)
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

func (e *Evaluator) applyFunction(fnObj object.Object, args []object.Object) object.Object {
	switch fn := fnObj.(type) {
	case *object.Function:

		// Create a new call frame and push it
		//println("call", fn.HasTailCall)
		e.PushEnv(e.extendFunctionEnv(fn, args))

		// Evaluate function body
		result := e.Eval(fn.Body)

		e.PopEnv()

		for {
			if returnVal, ok := result.(*object.TailCall); ok {
				result = e.applyFunction(returnVal.Function, returnVal.Arguments)
			} else if returnVal, ok := result.(*object.ReturnValue); ok {
				result = returnVal.Value
			} else {
				break
			}
		}

		return result

	case *object.Foreign:
		//println(fn.Inspect())
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func (e *Evaluator) extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env, &object.StackFrame{
		Function: "function", // String representation of the function
		File:     fn.Env.Path,
		Position: fn.Body.Token.Position,
	})
	numArgs := len(args)

	for i, param := range fn.Parameters {

		// Handle variadic arguments
		if param.IsVariadic {
			env.Define(param.Name.Value, &object.List{
				Elements: args[i:], // Remaining args as a list
			})
			break
		}

		// Handle default values
		if i >= numArgs {
			if param.Default != nil {
				defaultValue := e.Eval(param.Default)
				env.Define(param.Name.Value, defaultValue)
			} else {
				env.Define(param.Name.Value, NIL)
			}
		} else {
			env.Define(param.Name.Value, args[i])
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
		matched, err = e.patternMatches(matchCase.Pattern, matchValue, false)
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
func (e *Evaluator) patternMatches(pattern ast.MatchPattern, value object.Object, isConstant bool) (bool, error) {
	env := e.CurrentEnv()
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard matches anything
		return true, nil

	case *ast.SpreadPattern:
		// SpreadPattern matches anything
		if p.Value != nil {
			if isConstant {
				_, err := env.DefineConstant(p.Value.Value, value)
				return err == nil, err
			} else {
				_, err := env.Define(p.Value.Value, value)
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
			_, err := env.DefineConstant(p.Value.Value, value)
			return err == nil, err
		} else {
			_, err := env.Define(p.Value.Value, value)
			return err == nil, err
		}

	case *ast.MultiPattern:
		// Check if value matches any of the patterns
		for _, subPattern := range p.Patterns {
			encEnv := object.NewEnclosedEnvironment(env, nil)
			e.PushEnv(encEnv)
			matched, err := e.patternMatches(subPattern, value, isConstant)
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
				matched, err := e.patternMatches(spread, &object.List{Elements: arr.Elements[i:]}, isConstant)
				if err != nil || !matched {
					return false, err
				}
				break
			} else {
				matched, err := e.patternMatches(elemPattern, arr.Elements[i], isConstant)
				if err != nil || !matched {
					return false, err
				}
			}
		}

		// Copy bindings from scoped environment to parent environment
		for name, val := range scoped.Store {
			if val.IsConstant {
				if _, err := env.DefineConstant(name, val.Value); err != nil {
					return false, err
				}
			} else {
				if _, err := env.Define(name, val.Value); err != nil {
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

		// Empty map pattern matches empty map
		if len(p.Pairs) == 0 {
			return len(mapObj.Pairs) == 0, nil
		}

		usedKeys := make([]object.MapKey, 0)

		// scoped environment to capture the match bindings, these will be copied to the parent env on success
		scoped := object.NewEnclosedEnvironment(env, nil)
		e.PushEnv(scoped)

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
			matched, err := e.patternMatches(subPattern, pair.Value, isConstant)
			if !matched || err != nil {
				return false, err
			}
		}

		// If a spread pattern is used, collect unused keys into a new map
		if p.Spread {
			pair, ok := p.Pairs[token.ELLIPSIS]
			if ok {
				if len(usedKeys) >= len(mapObj.Pairs) {
					// map is empty
					_, err := e.patternMatches(pair, &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}, isConstant)
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
					_, err := e.patternMatches(pair, &object.Map{Pairs: copiedPairs}, isConstant)
					if err != nil {
						return false, err
					}

				}
			}
		}

		// Copy bindings to parent env
		for name, val := range scoped.Store {
			if val.IsConstant {
				_, err := env.DefineConstant(name, val.Value)
				if err != nil {
					return false, err
				}
			} else {
				_, err := env.Define(name, val.Value)
				if err != nil {
					return false, err
				}
			}
		}

		e.PopEnv()

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
	case *object.Integer:
		return aVal.Value == b.(*object.Integer).Value

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
	_, bindError := catchEnv.Define(catchMatch.Value.String(), err.Payload)
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
	case left.Type() == object.LIST_OBJ && index.Type() == object.INTEGER_OBJ:
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
	idx := index.(*object.Integer).Value
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
	idx := index.(*object.Integer).Value
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
		start = int(slice.Start.(*object.Integer).Value)
	}
	if slice.End != nil {
		end = int(slice.End.(*object.Integer).Value)
	}
	if slice.Step != nil {
		step = int(slice.Step.(*object.Integer).Value)
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

func (e *Evaluator) evalForeignFunctionDeclaration(stmt *ast.ForeignFunctionDeclaration) object.Object {
	env := e.CurrentEnv()
	modulePath := env.ModuleFqn
	functionName := stmt.Name.Value

	fqn := modulePath + "." + functionName

	if foreignFn, exists := foreignFunctions[fqn]; exists {
		foreignFn.Name = functionName
		foreignFn.Arity = len(stmt.Parameters)
		_, err := env.Define(functionName, foreignFn)
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
