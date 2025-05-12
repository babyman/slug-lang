package evaluator

import (
	"fmt"
	"slug/internal/ast"
	"slug/internal/object"
	"slug/internal/token"
)

var (
	NIL   = &object.Nil{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {

	// Statements
	case *ast.Program:
		return evalProgram(node, env)

	case *ast.BlockStatement:
		return evalBlockStatement(node, env)

	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}

	case *ast.ImportStatement:
		return evalImportStatement(node, env)

	case *ast.MatchExpression:
		return evalMatchExpression(node, env)

	case *ast.VarStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		if _, err := patternMatches(node.Pattern, val, env, false); err != nil {
			return newError(err.Error())
		}

	case *ast.ValStatement:
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		if _, err := patternMatches(node.Pattern, val, env, true); err != nil {
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
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)

	case *ast.InfixExpression:
		// Special case for assignment
		if node.Operator == "=" {
			// Ensure left side is an identifier
			ident, ok := node.Left.(*ast.Identifier)
			if !ok {
				return newError("left side of assignment must be an identifier")
			}

			// Evaluate right side
			right := Eval(node.Right, env)
			if isError(right) {
				return right
			}

			// Try to assign the value (variable is already defined)
			val, err := env.Assign(ident.Value, right)
			if err != nil {
				return newError(err.Error())
			}

			return val
		}

		// Regular infix expressions
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)

	case *ast.IfExpression:
		return evalIfExpression(node, env)

	case *ast.Identifier:
		return evalIdentifier(node, env)

	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		return &object.Function{Parameters: params, Env: env, Body: body}

	case *ast.CallExpression:
		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)

	case *ast.ListLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.List{Elements: elements}

	case *ast.IndexExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}
		index := Eval(node.Index, env)
		if isError(index) {
			return index
		}
		return evalIndexExpression(left, index)

	case *ast.SliceExpression:
		return evalSliceExpression(node, env)

	case *ast.MapLiteral:
		return evalMapLiteral(node, env)

	case *ast.ThrowStatement:
		return evalThrowStatement(node, env)

	case *ast.TryCatchStatement:
		return evalTryCatchStatement(node, env)

	case *ast.ForeignFunctionDeclaration:
		return evalForeignFunctionDeclaration(node, env)

	}

	return nil
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement, env)

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

func evalImportStatement(importStatement *ast.ImportStatement, env *object.Environment) object.Object {

	module, err := LoadModule(mapIdentifiersToStrings(importStatement.PathParts))
	if err != nil {
		return newError(err.Error())
	}
	if module.Env != nil {
		return handleModuleImport(importStatement, env, module)
	}

	// if the module is not in the registry, create a new Module object and cache it
	moduleEnv := object.NewEnvironment()
	moduleEnv.Path = module.Path
	moduleEnv.ModuleFqn = module.Name
	moduleEnv.Src = module.Src

	// Evaluate the module
	module.Env = moduleEnv

	Eval(module.Program, moduleEnv)

	// Import the module into the current environment
	return handleModuleImport(importStatement, env, module)
}

func handleModuleImport(importStatement *ast.ImportStatement, env *object.Environment, module *object.Module) object.Object {
	// Handle named symbols import, wildcard import, or namespace import
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

func mapIdentifiersToStrings(identifiers []*ast.Identifier) []string {
	parts := []string{}
	for _, id := range identifiers {
		parts = append(parts, id.Value)
	}
	return parts
}

func evalBlockStatement(block *ast.BlockStatement, env *object.Environment) object.Object {
	// Create a new environment with an associated stack frame
	blockEnv := object.NewEnclosedEnvironment(env, &object.StackFrame{
		Function: "block",
		File:     env.Path,
		Position: block.Token.Position,
	})

	var result object.Object
	for _, statement := range block.Statements {
		result = Eval(statement, blockEnv)
		if result != nil {
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

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	case "~":
		return evalComplementPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case operator == ":+" && left.Type() == object.LIST_OBJ:
		return evalListInfixExpression(operator, left, right)
	case operator == "+:" && right.Type() == object.LIST_OBJ:
		return evalListInfixExpression(operator, left, right)
	case left.Type() == object.LIST_OBJ && right.Type() == object.LIST_OBJ:
		return evalListInfixExpression(operator, left, right)
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ || right.Type() == object.STRING_OBJ:
		return evalStringPlusOtherInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s",
			left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
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

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalComplementPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: ^value}
}

func evalBooleanInfixExpression(
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

func evalIntegerInfixExpression(
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

func evalStringInfixExpression(
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

func evalListInfixExpression(
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

func evalStringPlusOtherInfixExpression(
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

func evalIfExpression(
	ie *ast.IfExpression,
	env *object.Environment,
) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.ThenBranch, env)
	} else if ie.ElseBranch != nil {
		return Eval(ie.ElseBranch, env)
	} else {
		return NIL
	}
}

func evalIdentifier(
	node *ast.Identifier,
	env *object.Environment,
) object.Object {
	if val, ok := env.Get(node.Value); ok {
		// If it is a Module, return the Module object itself
		if module, ok := val.(*object.Module); ok {
			return module
		}
		return val
	}

	return newError("identifier not found: " + node.Value)
}

func isTruthy(obj object.Object) bool {
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

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		return unwrapReturnValue(evaluated)

	case *object.Foreign:
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(
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

		// Handle destructuring (h:t)
		if param.Destructure != nil {
			if i >= numArgs {
				env.Define(param.Destructure.Head.Value, NIL)
				env.Define(param.Destructure.Tail.Value, NIL)
			} else {
				arg := args[i]

				list := arg.(*object.List)
				if len(list.Elements) > 0 {
					env.Define(param.Destructure.Head.Value, list.Elements[0])
					env.Define(param.Destructure.Tail.Value, &object.List{
						Elements: list.Elements[1:],
					})
				} else {
					env.Define(param.Destructure.Head.Value, NIL)
					env.Define(param.Destructure.Tail.Value, &object.List{})
				}
			}
			continue
		}

		// Handle default values
		if i >= numArgs {
			if param.Default != nil {
				defaultValue := Eval(param.Default, env)
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

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func evalMapLiteral(
	node *ast.MapLiteral,
	env *object.Environment,
) object.Object {
	pairs := make(map[object.MapKey]object.MapPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		mapKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as map key: %s", key.Type())
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		mapKeyHash := mapKey.MapKey()
		pairs[mapKeyHash] = object.MapPair{Key: key, Value: value}
	}

	return &object.Map{Pairs: pairs}
}

func evalMatchExpression(node *ast.MatchExpression, env *object.Environment) object.Object {
	// Evaluate the match value if provided
	var matchValue object.Object
	if node.Value != nil {
		matchValue = Eval(node.Value, env)
		if isError(matchValue) {
			return matchValue
		}
	}

	// Iterate through patterns
	for _, matchCase := range node.Cases {
		// Create a new scope for pattern variables
		patternEnv := object.NewEnclosedEnvironment(env, nil)

		// Match against the provided value or evaluate the condition
		matched := false
		var err error
		if matchValue != nil {
			// Match the case's pattern against the matchValue
			matched, err = patternMatches(matchCase.Pattern, matchValue, patternEnv, false)
			if err != nil {
				return newError("pattern match error: %s", err.Error())
			}
		} else {
			// Valueless match condition
			matched = evaluatePatternAsCondition(matchCase.Pattern, patternEnv)
		}

		// Evaluate guard condition if pattern matches
		if matched && matchCase.Guard != nil {
			guardResult := Eval(matchCase.Guard, patternEnv)
			if isError(guardResult) {
				return guardResult
			}
			matched = isTruthy(guardResult)
		}

		// If the pattern matched, evaluate the body
		if matched {
			return Eval(matchCase.Body, patternEnv)
		}
	}

	// No match found
	return NIL
}

// patternMatches checks if a value matches a pattern and binds variables
func patternMatches(pattern ast.MatchPattern, value object.Object, env *object.Environment, isConstant bool) (bool, error) {
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
		literalValue := Eval(p.Value, env)
		if isError(literalValue) {
			return false, fmt.Errorf("error while evaluating literal pattern value: %s", literalValue)
		}
		return objectsEqual(literalValue, value), nil

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
			matched, err := patternMatches(subPattern, value, object.NewEnclosedEnvironment(env, nil), isConstant)
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

		for i, elemPattern := range p.Elements {
			if spread, isSpread := elemPattern.(*ast.SpreadPattern); isSpread {
				matched, err := patternMatches(spread, &object.List{Elements: arr.Elements[i:]}, scoped, isConstant)
				if err != nil || !matched {
					return false, err
				}
				break
			} else {
				matched, err := patternMatches(elemPattern, arr.Elements[i], scoped, isConstant)
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
			matched, err := patternMatches(subPattern, pair.Value, scoped, isConstant)
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
					_, err := patternMatches(pair, &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}, scoped, isConstant)
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
					_, err := patternMatches(pair, &object.Map{Pairs: copiedPairs}, scoped, isConstant)
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

		return true, nil
	}

	// Unhandled pattern type
	return false, nil
}

// evaluatePatternAsCondition evaluates patterns as conditions for valueless match
func evaluatePatternAsCondition(pattern ast.MatchPattern, env *object.Environment) bool {
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard always matches
		return true

	case *ast.LiteralPattern:
		// Evaluate the literal and check if truthy
		result := Eval(p.Value, env)
		if isError(result) {
			return false
		}
		return isTruthy(result)

	case *ast.IdentifierPattern:
		// Look up identifier and check if truthy
		value, ok := env.Get(p.Value.Value)
		if !ok {
			return false
		}
		return isTruthy(value)

	case *ast.MultiPattern:
		// Check if any subpattern is truthy
		for _, subPattern := range p.Patterns {
			if evaluatePatternAsCondition(subPattern, env) {
				return true
			}
		}
		return false
	}

	return false
}

// objectsEqual compares two objects for equality
func objectsEqual(a, b object.Object) bool {
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
			if !objectsEqual(elem, bArr.Elements[i]) {
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
			if !ok || !objectsEqual(v.Value, bPair.Value) {
				return false
			}
		}

		return true
	}

	return false
}

func evalMapIndexExpression(obj, index object.Object) object.Object {
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

func evalThrowStatement(node *ast.ThrowStatement, env *object.Environment) object.Object {
	val := Eval(node.Value, env)
	if isError(val) {
		return val
	}
	if val.Type() != object.MAP_OBJ {
		return newError("throw argument must be a Map, got %s", val.Type())
	}
	return &object.RuntimeError{
		Payload: val,
		StackTrace: env.GatherStackTrace(&object.StackFrame{
			Function: "throw",
			File:     env.Path,
			Position: node.Token.Position,
		}),
	}
}

func evalTryCatchStatement(node *ast.TryCatchStatement, env *object.Environment) object.Object {
	// Execute the try block
	tryResult := Eval(node.TryBlock, env)

	err, isRuntimeError := tryResult.(*object.RuntimeError)

	// If there's no error, return the result of the try block
	if !isRuntimeError {
		return tryResult
	}

	// Match the error Payload against catch block patterns
	catchMatch := node.CatchBlock

	// Otherwise, handle the error (it must be a RuntimeError)
	catchEnv := object.NewEnclosedEnvironment(env, nil)

	// bind the error payload to the catch pattern
	_, bindError := catchEnv.Define(catchMatch.Value.String(), err.Payload)
	if bindError != nil {
		return newError(bindError.Error())
	}

	return evalMatchExpression(catchMatch, catchEnv)
}

func evalIndexExpression(left, index object.Object) object.Object {

	switch {
	case left.Type() == object.STRING_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if str, ok := left.(*object.String); ok {
				return evalStringSlice(str.Value, slice)
			}
		}
		return evalStringIndexExpression(left, index)
	case left.Type() == object.LIST_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalListIndexExpression(left, index)
	case left.Type() == object.LIST_OBJ:
		if slice, ok := index.(*object.Slice); ok {
			if arr, ok := left.(*object.List); ok {
				return evalListSlice(arr.Elements, slice)
			}
		}
		return evalListIndexExpression(left, index)
	case left.Type() == object.MAP_OBJ:
		return evalMapIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalSliceExpression(node *ast.SliceExpression, env *object.Environment) object.Object {
	start := Eval(node.Start, env)
	if isError(start) {
		return start
	}
	end := Eval(node.End, env)
	if isError(end) {
		return end
	}
	step := Eval(node.Step, env)
	if isError(step) {
		return step
	}
	return &object.Slice{
		Start: start,
		End:   end,
		Step:  step,
	}
}

func evalListIndexExpression(list, index object.Object) object.Object {
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

func evalStringIndexExpression(str, index object.Object) object.Object {
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

func evalListSlice(elements []object.Object, slice *object.Slice) object.Object {
	start, end, step := computeSliceIndices(len(elements), slice)
	var result []object.Object
	for i := start; i < end; i += step {
		result = append(result, elements[i])
	}
	return &object.List{Elements: result}
}

func evalStringSlice(value string, slice *object.Slice) object.Object {
	start, end, step := computeSliceIndices(len(value), slice)
	var result string
	for i := start; i < end; i += step {
		result += string(value[i])
	}
	return &object.String{Value: result}
}

func computeSliceIndices(length int, slice *object.Slice) (int, int, int) {
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

func evalForeignFunctionDeclaration(stmt *ast.ForeignFunctionDeclaration, env *object.Environment) object.Object {
	modulePath := env.ModuleFqn
	functionName := stmt.Name.Value

	fqn := modulePath + "." + functionName

	if foreignFn, exists := foreignFunctions[fqn]; exists {
		_, err := env.Define(functionName, foreignFn)
		if err != nil {
			return newError(err.Error())
		}
		return NIL
	}
	return newError("unknown foreign function %s", fqn)
}
