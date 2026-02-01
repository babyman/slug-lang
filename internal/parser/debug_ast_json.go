package parser

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"slug/internal/ast"
)

// WalkAST recursively traverses an AST and serializes it into a machine-centric map structure.
// This output is designed for stability, canonical representation, and tool-chain consumption.
func WalkAST(node ast.Node) interface{} {
	if node == nil || (reflect.ValueOf(node).Kind() == reflect.Ptr && reflect.ValueOf(node).IsNil()) {
		return nil
	}

	switch n := node.(type) {
	case *ast.Program:
		statements := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			statements[i] = WalkAST(s)
		}
		return map[string]interface{}{
			"type":       "Program",
			"statements": statements,
		}

	case *ast.VarExpression:
		return map[string]interface{}{
			"type":    "VarExpression",
			"tags":    walkTags(n.Tags),
			"token":   n.TokenLiteral(),
			"pattern": WalkAST(n.Pattern),
			"value":   WalkAST(n.Value),
		}

	case *ast.ValExpression:
		return map[string]interface{}{
			"type":    "ValExpression",
			"tags":    walkTags(n.Tags),
			"token":   n.TokenLiteral(),
			"pattern": WalkAST(n.Pattern),
			"value":   WalkAST(n.Value),
		}

	case *ast.ReturnStatement:
		return map[string]interface{}{
			"type":        "ReturnStatement",
			"token":       n.TokenLiteral(),
			"returnValue": WalkAST(n.ReturnValue),
		}

	case *ast.ExpressionStatement:
		return map[string]interface{}{
			"type":       "ExpressionStatement",
			"token":      n.TokenLiteral(),
			"expression": WalkAST(n.Expression),
		}

	case *ast.BlockStatement:
		statements := make([]interface{}, len(n.Statements))
		for i, s := range n.Statements {
			statements[i] = WalkAST(s)
		}
		return map[string]interface{}{
			"type":       "BlockStatement",
			"token":      n.TokenLiteral(),
			"statements": statements,
		}

	case *ast.Identifier:
		return map[string]interface{}{
			"type":  "Identifier",
			"token": safeTokenLiteral(n),
			"value": n.Value,
		}

	case *ast.Boolean:
		return map[string]interface{}{
			"type":  "Boolean",
			"token": n.TokenLiteral(),
			"value": n.Value,
		}

	case *ast.Nil:
		return map[string]interface{}{
			"type":  "Nil",
			"token": n.TokenLiteral(),
		}

	case *ast.NumberLiteral:
		return map[string]interface{}{
			"type":  "NumberLiteral",
			"token": safeTokenLiteral(n),
			"value": n.Value.String(), // Dec64 as string for precision
		}

	case *ast.StringLiteral:
		return map[string]interface{}{
			"type":  "StringLiteral",
			"token": n.TokenLiteral(),
			"value": n.Value,
		}

	case *ast.SymbolLiteral:
		return map[string]interface{}{
			"type":  "SymbolLiteral",
			"token": n.TokenLiteral(),
			"value": n.Value,
		}

	case *ast.BytesLiteral:
		return map[string]interface{}{
			"type":  "BytesLiteral",
			"token": safeTokenLiteral(n),
			"value": hex.EncodeToString(n.Value),
		}

	case *ast.InfixExpression:
		return map[string]interface{}{
			"type":     "InfixExpression",
			"token":    n.TokenLiteral(),
			"left":     WalkAST(n.Left),
			"operator": n.Operator,
			"right":    WalkAST(n.Right),
		}

	case *ast.PrefixExpression:
		return map[string]interface{}{
			"type":     "PrefixExpression",
			"token":    n.TokenLiteral(),
			"operator": n.Operator,
			"right":    WalkAST(n.Right),
		}

	case *ast.IfExpression:
		return map[string]interface{}{
			"type":       "IfExpression",
			"token":      safeTokenLiteral(n),
			"condition":  WalkAST(n.Condition),
			"thenBranch": WalkAST(n.ThenBranch),
			"elseBranch": WalkAST(n.ElseBranch),
		}

	case *ast.FunctionLiteral:
		params := make([]interface{}, len(n.Parameters))
		for i, p := range n.Parameters {
			params[i] = WalkAST(p)
		}
		return map[string]interface{}{
			"type":        "FunctionLiteral",
			"token":       n.TokenLiteral(),
			"parameters":  params,
			"body":        WalkAST(n.Body),
			"hasTailCall": n.HasTailCall,
		}

	case *ast.CallExpression:
		args := make([]interface{}, len(n.Arguments))
		for i, arg := range n.Arguments {
			args[i] = WalkAST(arg)
		}
		return map[string]interface{}{
			"type":       "CallExpression",
			"token":      safeTokenLiteral(n),
			"function":   WalkAST(n.Function),
			"arguments":  args,
			"isTailCall": n.IsTailCall,
		}

	case *ast.RecurExpression:
		args := make([]interface{}, len(n.Arguments))
		for i, arg := range n.Arguments {
			args[i] = WalkAST(arg)
		}
		return map[string]interface{}{
			"type":      "RecurExpression",
			"token":     n.TokenLiteral(),
			"arguments": args,
		}

	case *ast.ListLiteral:
		elements := make([]interface{}, len(n.Elements))
		for i, el := range n.Elements {
			elements[i] = WalkAST(el)
		}
		return map[string]interface{}{
			"type":     "ListLiteral",
			"token":    n.TokenLiteral(),
			"elements": elements,
		}

	case *ast.MapLiteral:
		type pair struct {
			Key   interface{} `json:"key"`
			Value interface{} `json:"value"`
		}
		pairs := make([]pair, 0, len(n.Pairs))
		for k, v := range n.Pairs {
			pairs = append(pairs, pair{Key: WalkAST(k), Value: WalkAST(v)})
		}
		return map[string]interface{}{
			"type":  "MapLiteral",
			"token": n.TokenLiteral(),
			"pairs": pairs,
		}

	case *ast.StructSchemaExpression:
		fields := make([]interface{}, len(n.Fields))
		for i, f := range n.Fields {
			fields[i] = map[string]interface{}{
				"name":    f.Name,
				"hint":    f.Hint,
				"default": WalkAST(f.Default),
			}
		}
		return map[string]interface{}{
			"type":   "StructSchemaExpression",
			"token":  n.TokenLiteral(),
			"fields": fields,
		}

	case *ast.StructInitExpression:
		fields := make([]interface{}, len(n.Fields))
		for i, f := range n.Fields {
			fields[i] = map[string]interface{}{
				"name":  f.Name,
				"value": WalkAST(f.Value),
			}
		}
		return map[string]interface{}{
			"type":   "StructInitExpression",
			"token":  n.TokenLiteral(),
			"schema": WalkAST(n.Schema),
			"fields": fields,
		}

	case *ast.StructCopyExpression:
		fields := make([]interface{}, len(n.Fields))
		for i, f := range n.Fields {
			fields[i] = map[string]interface{}{
				"name":  f.Name,
				"value": WalkAST(f.Value),
			}
		}
		return map[string]interface{}{
			"type":   "StructCopyExpression",
			"token":  n.TokenLiteral(),
			"source": WalkAST(n.Source),
			"fields": fields,
		}

	case *ast.IndexExpression:
		return map[string]interface{}{
			"type":  "IndexExpression",
			"token": safeTokenLiteral(n),
			"left":  WalkAST(n.Left),
			"index": WalkAST(n.Index),
		}

	case *ast.SliceExpression:
		return map[string]interface{}{
			"type":  "SliceExpression",
			"token": safeTokenLiteral(n),
			"start": WalkAST(n.Start),
			"end":   WalkAST(n.End),
			"step":  WalkAST(n.Step),
		}

	case *ast.MatchExpression:
		cases := make([]interface{}, len(n.Cases))
		for i, c := range n.Cases {
			cases[i] = WalkAST(c)
		}
		return map[string]interface{}{
			"type":  "MatchExpression",
			"token": n.TokenLiteral(),
			"value": WalkAST(n.Value),
			"cases": cases,
		}

	case *ast.MatchCase:
		return map[string]interface{}{
			"type":    "MatchCase",
			"token":   n.TokenLiteral(),
			"pattern": WalkAST(n.Pattern),
			"guard":   WalkAST(n.Guard),
			"body":    WalkAST(n.Body),
		}

	// Pattern Nodes
	case *ast.WildcardPattern:
		return map[string]interface{}{"type": "WildcardPattern", "token": n.TokenLiteral()}
	case *ast.AllPattern:
		return map[string]interface{}{"type": "AllPattern", "token": n.TokenLiteral()}
	case *ast.BindingPattern:
		return map[string]interface{}{"type": "BindingPattern", "identifier": WalkAST(n.Name), "pattern": WalkAST(n.Pattern)}
	case *ast.LiteralPattern:
		return map[string]interface{}{"type": "LiteralPattern", "value": WalkAST(n.Value)}
	case *ast.IdentifierPattern:
		return map[string]interface{}{"type": "IdentifierPattern", "identifier": WalkAST(n.Value)}
	case *ast.PinnedIdentifierPattern:
		return map[string]interface{}{"type": "PinnedIdentifierPattern", "identifier": WalkAST(n.Value)}
	case *ast.SpreadPattern:
		return map[string]interface{}{"type": "SpreadPattern", "token": safeTokenLiteral(n), "identifier": WalkAST(n.Value)}
	case *ast.ListPattern:
		elements := make([]interface{}, len(n.Elements))
		for i, el := range n.Elements {
			elements[i] = WalkAST(el)
		}
		return map[string]interface{}{"type": "ListPattern", "elements": elements}
	case *ast.MapPattern:
		type pair struct {
			Key     interface{} `json:"key"`
			Pattern interface{} `json:"pattern"`
		}
		pairs := make([]pair, 0, len(n.Pairs))
		for _, entry := range n.Pairs {
			pairs = append(pairs, pair{Key: WalkAST(entry.Key), Pattern: WalkAST(entry.Pattern)})
		}
		return map[string]interface{}{
			"type":      "MapPattern",
			"pairs":     pairs,
			"exact":     n.Exact,
			"selectAll": n.SelectAll,
			"spread":    WalkAST(n.Spread),
		}

	case *ast.StructPattern:
		fields := make([]interface{}, len(n.Fields))
		for i, f := range n.Fields {
			fields[i] = map[string]interface{}{
				"name":    f.Name,
				"pattern": WalkAST(f.Pattern),
			}
		}
		return map[string]interface{}{
			"type":   "StructPattern",
			"token":  n.TokenLiteral(),
			"schema": WalkAST(n.Schema),
			"fields": fields,
		}

	case *ast.FunctionParameter:
		return map[string]interface{}{
			"type":         "FunctionParameter",
			"tags":         walkTags(n.Tags),
			"name":         WalkAST(n.Name),
			"defaultValue": WalkAST(n.Default),
			"isVariadic":   n.IsVariadic,
		}

	case *ast.ThrowStatement:
		return map[string]interface{}{
			"type":  "ThrowStatement",
			"token": safeTokenLiteral(n),
			"value": WalkAST(n.Value),
		}

	case *ast.DeferStatement:
		return map[string]interface{}{
			"type":      "DeferStatement",
			"token":     safeTokenLiteral(n),
			"call":      WalkAST(n.Call),
			"mode":      int(n.Mode),
			"errorName": WalkAST(n.ErrorName),
		}

	case *ast.ForeignFunctionDeclaration:
		params := make([]interface{}, len(n.Parameters))
		for i, p := range n.Parameters {
			params[i] = WalkAST(p)
		}
		return map[string]interface{}{
			"type":       "ForeignFunctionDeclaration",
			"tags":       walkTags(n.Tags),
			"token":      safeTokenLiteral(n),
			"name":       WalkAST(n.Name),
			"parameters": params,
		}

	case *ast.SpreadExpression:
		return map[string]interface{}{
			"type":  "SpreadExpression",
			"token": safeTokenLiteral(n),
			"value": WalkAST(n.Value),
		}

	case *ast.NotImplemented:
		return map[string]interface{}{"type": "NotImplemented", "token": safeTokenLiteral(n)}

	default:
		return map[string]interface{}{
			"type": "Unknown",
			"node": fmt.Sprintf("%T", n),
		}
	}
}

func safeTokenLiteral(node ast.Node) string {
	if node == nil || (reflect.ValueOf(node).Kind() == reflect.Ptr && reflect.ValueOf(node).IsNil()) {
		return ""
	}
	return node.TokenLiteral()
}

func walkTags(tags []*ast.Tag) []interface{} {
	if tags == nil {
		return []interface{}{}
	}
	result := make([]interface{}, len(tags))
	for i, t := range tags {
		args := make([]interface{}, len(t.Args))
		for j, arg := range t.Args {
			args[j] = WalkAST(arg)
		}
		result[i] = map[string]interface{}{
			"type":      "Tag",
			"name":      t.Name,
			"arguments": args,
		}
	}
	return result
}

func RenderASTAsJSON(node ast.Node) (string, error) {
	astMap := WalkAST(node)
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(astMap); err != nil {
		return "", fmt.Errorf("failed to encode JSON: %v", err)
	}
	return buf.String(), nil
}
