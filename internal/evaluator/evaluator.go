package evaluator

import (
	"fmt"
	"io/ioutil"
	"os"
	"slug/internal/ast"
	"slug/internal/lexer"
	"slug/internal/object"
	"slug/internal/parser"
	"slug/internal/token"
	"strings"
)

var (
	NIL   = &object.Nil{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

var ModuleRegistry = make(map[string]*object.Module)

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
		patternMatches(node.Pattern, val, env)

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
			val, ok := env.Assign(ident.Value, right)
			if !ok {
				return newError("cannot assign to '%s'", ident.Value)
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

	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}

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

	case *ast.HashLiteral:
		return evalHashLiteral(node, env)

	case *ast.ThrowStatement:
		return evalThrowStatement(node, env)

	case *ast.TryCatchStatement:
		return evalTryCatchStatement(node, env)

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
	// Generate the cache key from PathParts
	key := strings.Join(mapIdentifiersToStrings(importStatement.PathParts), ".")

	// Check if the module is already in the registry
	if module, exists := ModuleRegistry[key]; exists {
		// Module is already loaded, return it
		return handleModuleImport(importStatement, env, module)
	}

	// Step 1: Resolve the root path
	rootPath := env.GetRootPath() // Retrieve the root path set by --root

	// if the module is not in the registry, create a new Module object and cache it
	moduleName := importStatement.PathParts[len(importStatement.PathParts)-1].Value
	moduleEnv := object.NewEnvironment()
	moduleEnv.SetRootPath(rootPath)
	module := &object.Module{Name: moduleName, Env: moduleEnv}
	ModuleRegistry[key] = module

	// Step 2: Resolve module path to file path
	moduleRelativePath := strings.Join(mapIdentifiersToStrings(importStatement.PathParts), "/")

	modulePath := fmt.Sprintf("%s/%s.slug", rootPath, moduleRelativePath)

	// Try to read the module from the resolved path
	moduleSrc, err := ioutil.ReadFile(modulePath)
	fallbackPath := ""
	if err != nil {
		// If not found, attempt fallback to ${SLUG_HOME}/lib
		slugHome := os.Getenv("SLUG_HOME")
		if slugHome == "" {
			return newError("environment variable SLUG_HOME is not set")
		}

		fallbackPath = fmt.Sprintf("%s/lib/%s.slug", slugHome, moduleRelativePath)
		moduleSrc, err = ioutil.ReadFile(fallbackPath)
		if err != nil {
			return newError("error reading module '%s': %s", fallbackPath, err)
		}
	}

	// Step 3: Parse and evaluate the module
	src := string(moduleSrc)
	moduleEnv.Src = src
	moduleEnv.Path = moduleRelativePath
	l := lexer.New(src)
	p := parser.New(l, src)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return newError("parsing errors in module '%s': %s", key, strings.Join(p.Errors(), ", "))
	}

	// Evaluate the program within the module's environment
	Eval(program, moduleEnv)

	// Import the module into the current environment
	return handleModuleImport(importStatement, env, module)
}

func handleModuleImport(importStatement *ast.ImportStatement, env *object.Environment, module *object.Module) object.Object {
	// Handle named symbols import, wildcard import, or namespace import
	if importStatement.Wildcard {
		// Import all symbols into the current environment
		for name, val := range module.Env.Store {
			_, ok := env.Define(name, val)
			if !ok {
				return newError("cannot assign to '%s'", name)
			}
		}
	} else if len(importStatement.Symbols) > 0 {
		// Import specific symbols
		for _, sym := range importStatement.Symbols {
			if val, ok := module.Env.Get(sym.Name.Value); ok {
				alias := sym.Name.Value
				if sym.Alias != nil {
					alias = sym.Alias.Value
				}
				_, ok := env.Define(alias, val)
				if !ok {
					return newError("cannot assign to '%s'", alias)
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

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
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

	case *object.Builtin:
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
			env.Define(param.Name.Value, &object.Array{
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

				list := arg.(*object.Array)
				if len(list.Elements) > 0 {
					env.Define(param.Destructure.Head.Value, list.Elements[0])
					env.Define(param.Destructure.Tail.Value, &object.Array{
						Elements: list.Elements[1:],
					})
				} else {
					env.Define(param.Destructure.Head.Value, NIL)
					env.Define(param.Destructure.Tail.Value, &object.Array{})
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

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.HASH_OBJ:
		return evalHashIndexExpression(left, index)
	default:
		return newError("index operator not supported: %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if idx < 0 {
		idx = max + idx + 1
	}

	if idx < 0 || idx > max {
		return NIL
	}

	return arrayObject.Elements[idx]
}

func evalHashLiteral(
	node *ast.HashLiteral,
	env *object.Environment,
) object.Object {
	pairs := make(map[object.HashKey]object.HashPair)

	for keyNode, valueNode := range node.Pairs {
		key := Eval(keyNode, env)
		if isError(key) {
			return key
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return newError("unusable as hash key: %s", key.Type())
		}

		value := Eval(valueNode, env)
		if isError(value) {
			return value
		}

		hashed := hashKey.HashKey()
		pairs[hashed] = object.HashPair{Key: key, Value: value}
	}

	return &object.Hash{Pairs: pairs}
}

func evalMatchExpression(node *ast.MatchExpression, env *object.Environment) object.Object {
	// If there's a match value, evaluate it
	var matchValue object.Object
	if node.Value != nil {
		matchValue = Eval(node.Value, env)
		if isError(matchValue) {
			return matchValue
		}
	}

	// Try each pattern in sequence
	for _, matchCase := range node.Cases {
		// Create a new scope for pattern variables
		patternEnv := object.NewEnclosedEnvironment(env, nil)

		// Check if the pattern matches
		matched := false
		if matchValue != nil {
			// Match against the provided value
			matched = patternMatches(matchCase.Pattern, matchValue, patternEnv)
		} else {
			// Valueless match - evaluate pattern as a condition
			matched = evaluatePatternAsCondition(matchCase.Pattern, patternEnv)
		}

		// Check guard condition if pattern matched
		if matched && matchCase.Guard != nil {
			guardResult := Eval(matchCase.Guard, patternEnv)
			if isError(guardResult) {
				return guardResult
			}
			matched = isTruthy(guardResult)
		}

		// If matched, evaluate the body with bindings from the pattern
		if matched {
			return Eval(matchCase.Body, patternEnv)
		}
	}

	// If no patterns matched, return nil
	return NIL
}

// patternMatches checks if a value matches a pattern and binds variables
func patternMatches(pattern ast.MatchPattern, value object.Object, env *object.Environment) bool {
	switch p := pattern.(type) {
	case *ast.WildcardPattern:
		// Wildcard matches anything
		return true

	case *ast.SpreadPattern:
		// SpreadPattern matches anything
		if p.Value != nil {
			env.Define(p.Value.Value, value)
		}
		return true

	case *ast.LiteralPattern:
		// Evaluate the literal and compare with the value
		literalValue := Eval(p.Value, env)
		if isError(literalValue) {
			return false
		}
		return objectsEqual(literalValue, value)

	case *ast.IdentifierPattern:
		// Bind the value to the identifier
		env.Define(p.Value.Value, value)
		return true

	case *ast.MultiPattern:
		// Check if value matches any of the patterns
		for _, subPattern := range p.Patterns {
			if patternMatches(subPattern, value, object.NewEnclosedEnvironment(env, nil)) {
				return true
			}
		}
		return false

	case *ast.ArrayPattern:
		// Check if the value is an array
		arr, ok := value.(*object.Array)
		if !ok {
			return false
		}

		// Empty array pattern matches empty array
		if len(p.Elements) == 0 {
			return len(arr.Elements) == 0
		}

		_, isSpread := p.Elements[len(p.Elements)-1].(*ast.SpreadPattern)

		// Check if array length matches pattern length
		if (len(p.Elements) != len(arr.Elements) && !isSpread) || (len(p.Elements) > len(arr.Elements)+1 && isSpread) {
			return false
		}

		// scoped environment to capture the match bindings, these will be copied to the parent env on success
		scoped := object.NewEnclosedEnvironment(env, nil)

		// Check each element against its pattern
		for i, elemPattern := range p.Elements {
			_, ok := elemPattern.(*ast.SpreadPattern)
			if ok {
				matches := patternMatches(elemPattern, &object.Array{Elements: arr.Elements[i:]}, scoped)
				if matches {
					// Copy bindings from scoped environment to parent environment
					for name, val := range scoped.Store {
						env.Define(name, val)
					}
				}
				return matches
			} else if !patternMatches(elemPattern, arr.Elements[i], scoped) {
				return false
			}
		}

		// Copy bindings from scoped environment to parent environment
		for name, val := range scoped.Store {
			env.Define(name, val)
		}

		return true

	case *ast.HashPattern:
		// Check if value is a hash
		hash, ok := value.(*object.Hash)
		if !ok {
			return false
		}

		// Empty hash pattern matches empty hash
		if len(p.Pairs) == 0 {
			return len(hash.Pairs) == 0
		}

		usedKeys := make([]object.HashKey, 0)

		// scoped environment to capture the match bindings, these will be copied to the parent env on success
		scoped := object.NewEnclosedEnvironment(env, nil)

		// Check if all required fields are present
		for key, subPattern := range p.Pairs {
			if key == token.ELLIPSIS {
				// Skip wildcard placeholder for spread, we'll deal with that later
				continue
			}

			// Check if key exists in hash
			keyObj := &object.String{Value: key}
			hashKey := keyObj.HashKey()
			usedKeys = append(usedKeys, hashKey)
			pair, ok := hash.Pairs[hashKey]
			if !ok {
				return false
			}

			// Check if value matches subpattern
			if !patternMatches(subPattern, pair.Value, scoped) {
				return false
			}
		}

		// If a spread pattern is used, collect unused keys into a new hash
		if p.Spread {
			pair, ok := p.Pairs[token.ELLIPSIS]
			if ok {
				if len(usedKeys) >= len(hash.Pairs) {
					// map is empty
					patternMatches(pair, &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}, scoped)
				} else {
					copiedPairs := make(map[object.HashKey]object.HashPair)
					for hashKey, pair := range hash.Pairs {
						isUsed := false
						for _, usedKey := range usedKeys {
							if hashKey == usedKey {
								isUsed = true
								break
							}
						}
						if !isUsed {
							copiedPairs[hashKey] = pair
						}
					}
					patternMatches(pair, &object.Hash{Pairs: copiedPairs}, scoped)
				}
			}
		}

		if !p.Spread && len(usedKeys) != len(hash.Pairs) {
			return false
		}

		// Copy bindings from scoped environment to parent environment
		for name, val := range scoped.Store {
			env.Define(name, val)
		}

		return true
	}

	return false
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

	case *object.Array:
		bArr := b.(*object.Array)
		if len(aVal.Elements) != len(bArr.Elements) {
			return false
		}

		for i, elem := range aVal.Elements {
			if !objectsEqual(elem, bArr.Elements[i]) {
				return false
			}
		}

		return true

	case *object.Hash:
		bHash := b.(*object.Hash)
		if len(aVal.Pairs) != len(bHash.Pairs) {
			return false
		}

		for k, v := range aVal.Pairs {
			bPair, ok := bHash.Pairs[k]
			if !ok || !objectsEqual(v.Value, bPair.Value) {
				return false
			}
		}

		return true
	}

	return false
}

func evalHashIndexExpression(hash, index object.Object) object.Object {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return newError("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
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
	if val.Type() != object.HASH_OBJ {
		return newError("throw argument must be a Hash, got %s", val.Type())
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
	catchEnv.Define(catchMatch.Value.String(), err.Payload)

	return evalMatchExpression(catchMatch, catchEnv)
}
