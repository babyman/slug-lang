package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"slug/internal/ast"
)

// WalkAST recursively traverses an AST and serializes it into a map structure for JSON output.
func WalkAST(node ast.Node) interface{} {
	switch n := node.(type) {
	case *ast.Program:
		statements := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			statements[i] = WalkAST(s)
		}
		return map[string]interface{}{
			"0.type":       "Program",
			"1.statements": statements,
		}

	case *ast.VarStatement:
		return map[string]interface{}{
			"0.type":     "VarStatement",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.pattern":  WalkAST(n.Pattern),
			"4.value":    WalkAST(n.Value),
		}

	case *ast.ReturnStatement:
		return map[string]interface{}{
			"0.type":        "ReturnStatement",
			"1.position":    n.Token.Position,
			"2.token":       n.TokenLiteral(),
			"3.returnValue": WalkAST(n.ReturnValue),
		}

	case *ast.ImportStatement:
		symbols := []interface{}{}
		for _, sym := range n.Symbols {
			symbols = append(symbols, sym.String())
		}
		return map[string]interface{}{
			"0.type":      "ImportStatement",
			"1.position":  n.Token.Position,
			"2.token":     n.TokenLiteral(),
			"3.pathParts": n.PathAsString(),
			"4.cases":     symbols,
			"5.wildcard":  n.Wildcard,
		}

	case *ast.ExpressionStatement:
		return map[string]interface{}{
			"0.type":       "ExpressionStatement",
			"1.position":   n.Token.Position,
			"2.token":      n.TokenLiteral(),
			"3.expression": WalkAST(n.Expression),
		}

	case *ast.BlockStatement:
		statements := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			statements[i] = WalkAST(s)
		}
		return map[string]interface{}{
			"0.type":       "BlockStatement",
			"1.position":   n.Token.Position,
			"2.token":      n.TokenLiteral(),
			"3.statements": statements,
		}

	case *ast.Identifier:
		return map[string]interface{}{
			"0.type":     "Identifier",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.value":    n.Value,
		}

	case *ast.Boolean:
		return map[string]interface{}{
			"0.type":     "Boolean",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.value":    n.Value,
		}

	case *ast.Nil:
		return map[string]interface{}{
			"0.type":     "Nil",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
		}

	case *ast.IntegerLiteral:
		return map[string]interface{}{
			"0.type":     "IntegerLiteral",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.value":    n.Value,
		}

	case *ast.InfixExpression:
		return map[string]interface{}{
			"0.type":     "InfixExpression",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.left":     WalkAST(n.Left),
			"4.operator": n.Operator,
			"5.right":    WalkAST(n.Right),
		}

	case *ast.PrefixExpression:
		return map[string]interface{}{
			"0.type":     "PrefixExpression",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.operator": n.Operator,
			"4.right":    WalkAST(n.Right),
		}

	case *ast.StringLiteral:
		return map[string]interface{}{
			"0.type":     "StringLiteral",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.value":    n.Value,
		}

	case *ast.ArrayLiteral:
		elements := make([]interface{}, len(n.Elements))
		for i, el := range n.Elements {
			elements[i] = WalkAST(el)
		}
		return map[string]interface{}{
			"0.type":     "ArrayLiteral",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.elements": elements,
		}

	case *ast.IfExpression:
		return map[string]interface{}{
			"0.type":      "IfExpression",
			"1.position":  n.Token.Position,
			"2.token":     n.TokenLiteral(),
			"3.condition": WalkAST(n.Condition),
			"4.then":      WalkAST(n.ThenBranch),
			"5.else":      WalkAST(n.ElseBranch),
		}

	case *ast.FunctionLiteral:
		parameters := make([]interface{}, len(n.Parameters))
		for i, param := range n.Parameters {
			parameters[i] = WalkAST(param)
		}
		return map[string]interface{}{
			"0.type":       "FunctionLiteral",
			"1.position":   n.Token.Position,
			"2.token":      n.TokenLiteral(),
			"3.parameters": parameters,
			"4.body":       WalkAST(n.Body),
		}

	case *ast.CallExpression:
		args := make([]interface{}, len(n.Arguments))
		for i, arg := range n.Arguments {
			args[i] = WalkAST(arg)
		}
		return map[string]interface{}{
			"0.type":      "CallExpression",
			"1.position":  n.Token.Position,
			"2.token":     n.TokenLiteral(),
			"3.function":  WalkAST(n.Function),
			"4.arguments": args,
		}

	case *ast.MapLiteral:
		pairs := map[string]interface{}{}
		for key, value := range n.Pairs {
			pairs[fmt.Sprintf("%v", WalkAST(key))] = WalkAST(value)
		}
		return map[string]interface{}{
			"0.type":     "MapLiteral",
			"1.position": n.Token.Position,
			"2.token":    n.TokenLiteral(),
			"3.pairs":    pairs,
		}

	case *ast.FunctionParameter:
		return map[string]interface{}{
			"0.type":       "FunctionParameter",
			"2.token":      n.TokenLiteral(),
			"3.name":       WalkAST(n.Name),
			"4.isVariadic": n.IsVariadic,
			"5.default":    WalkAST(n.Default),
		}

	case *ast.MatchExpression:
		var cases []interface{}
		for _, c := range n.Cases {
			cases = append(cases, WalkAST(c))
		}
		return map[string]interface{}{
			"0.type":       "MatchExpression",
			"1.position":   n.Token.Position,
			"2.expression": WalkAST(n.Value),
			"3.cases":      cases,
		}

	case *ast.MatchCase:
		return map[string]interface{}{
			"0.type":     "MatchCase",
			"1.position": n.Token.Position,
			"2.pattern":  WalkAST(n.Pattern),
			"3.guard":    WalkAST(n.Guard),
			"4.body":     WalkAST(n.Body),
		}

	case *ast.WildcardPattern:
		return map[string]interface{}{
			"0.type":     "WildcardPattern",
			"1.position": n.Token.Position,
			"2.token":    n.Token.Literal,
		}

	case *ast.SpreadPattern:
		var id interface{}
		if n.Value != nil {
			id = WalkAST(n.Value)
		}
		return map[string]interface{}{
			"0.type":     "SpreadPattern",
			"1.position": n.Token.Position,
			"2.token":    n.Token.Literal,
			"3.value":    id,
		}

	case *ast.LiteralPattern:
		return map[string]interface{}{
			"0.type":     "LiteralPattern",
			"1.position": n.Token.Position,
			"2.value":    WalkAST(n.Value),
		}

	case *ast.IdentifierPattern:
		return map[string]interface{}{
			"0.type":       "IdentifierPattern",
			"1.position":   n.Token.Position,
			"2.identifier": WalkAST(n.Value),
		}

	case *ast.MultiPattern:
		patterns := make([]interface{}, len(n.Patterns))
		for i, p := range n.Patterns {
			patterns[i] = WalkAST(p)
		}
		return map[string]interface{}{
			"0.type":     "MultiPattern",
			"1.position": n.Token.Position,
			"2.patterns": patterns,
		}

	case *ast.ArrayPattern:
		elements := make([]interface{}, len(n.Elements))
		for i, el := range n.Elements {
			elements[i] = WalkAST(el)
		}
		return map[string]interface{}{
			"0.type":     "ArrayPattern",
			"1.position": n.Token.Position,
			"2.elements": elements,
		}

	case *ast.MapPattern:
		pairs := map[string]interface{}{}
		for key, value := range n.Pairs {
			pairs[fmt.Sprintf("%v", key)] = WalkAST(value)
		}
		return map[string]interface{}{
			"0.type":     "MapPattern",
			"1.position": n.Token.Position,
			"2.pairs":    pairs,
		}

	case *ast.DestructureBinding:
		return map[string]interface{}{
			"0.type":     "DestructureBinding",
			"1.position": n.Token.Position,
			"2.head":     WalkAST(n.Head),
			"3.tail":     WalkAST(n.Tail),
		}

	case *ast.ThrowStatement:
		return map[string]interface{}{
			"0.type":  "ThrowStatement",
			"1.token": n.TokenLiteral(),
			"2.value": WalkAST(n.Value),
		}

	case *ast.TryCatchStatement:
		return map[string]interface{}{
			"0.type":       "TryCatchStatement",
			"1.token":      n.TokenLiteral(),
			"2.tryBlock":   WalkAST(n.TryBlock),
			"3.catchBlock": WalkAST(n.CatchBlock),
		}

	default:
		return map[string]interface{}{
			"0.type": "Unknown",
			"1.node": n,
		}
	}
}

// WriteASTToJSON takes a root AST node and writes it to a JSON file.
func WriteASTToJSON(node ast.Node, filename string) error {
	astMap := WalkAST(node)

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")  // Pretty-print the JSON
	encoder.SetEscapeHTML(false) // Disable escaping of characters like <, >, &

	if err := encoder.Encode(astMap); err != nil {
		return fmt.Errorf("failed to write JSON: %v", err)
	}
	return nil
}
