package parser

import (
	"encoding/hex"
	"fmt"
	"reflect"
	"slug/internal/ast"
	"strings"
)

// RenderASTAsText produces a human-centric, indented, Slug-like representation of the AST.
// It is optimized for debugging precedence, binding, and pattern structure.
func RenderASTAsText(node ast.Node, indent int) string {
	if node == nil || (reflect.ValueOf(node).Kind() == reflect.Ptr && reflect.ValueOf(node).IsNil()) {
		return "nil"
	}

	sp := strings.Repeat("  ", indent)

	switch n := node.(type) {
	case *ast.Program:
		var sb strings.Builder
		for i, s := range n.Statements {
			if i > 0 {
				sb.WriteString("\n")
			}
			// Root level statements start at indent 0
			sb.WriteString(RenderASTAsText(s, 0))
		}
		return sb.String()

	case *ast.VarExpression:
		// VarExpression as a statement: we apply 'sp' here.
		// If it's used inside a block, 'sp' will be provided by the caller.
		return fmt.Sprintf("%s%svar %s = %s", sp, renderTags(n.Tags), RenderASTAsText(n.Pattern, 0), RenderASTAsText(n.Value, 0))

	case *ast.ValExpression:
		return fmt.Sprintf("%s%sval %s = %s", sp, renderTags(n.Tags), RenderASTAsText(n.Pattern, 0), RenderASTAsText(n.Value, 0))

	case *ast.ReturnStatement:
		return fmt.Sprintf("%sreturn %s", sp, RenderASTAsText(n.ReturnValue, 0))

	case *ast.ThrowStatement:
		return fmt.Sprintf("%sthrow %s", sp, RenderASTAsText(n.Value, 0))

	case *ast.ExpressionStatement:
		// The statement handles the line's starting indentation
		return sp + RenderASTAsText(n.Expression, 0)

	case *ast.BlockStatement:
		var sb strings.Builder
		sb.WriteString("{\n")
		for _, s := range n.Statements {
			// Statements inside the block are indented +1
			sb.WriteString(RenderASTAsText(s, indent+1))
			sb.WriteString("\n")
		}
		// The closing brace aligns with the parent's indent
		sb.WriteString(sp + "}")
		return sb.String()

	case *ast.FunctionLiteral:
		params := []string{}
		for _, p := range n.Parameters {
			params = append(params, RenderASTAsText(p, 0))
		}
		// Body block aligns its closing brace with 'indent'
		return fmt.Sprintf("fn(%s) %s", strings.Join(params, ", "), RenderASTAsText(n.Body, indent))

	case *ast.FunctionParameter:
		tags := renderTags(n.Tags)
		prefix := ""
		if n.IsVariadic {
			prefix = "..."
		}
		res := tags + prefix + RenderASTAsText(n.Name, 0)
		if n.Default != nil {
			res += " = " + RenderASTAsText(n.Default, 0)
		}
		return res

	case *ast.CallExpression:
		args := []string{}
		for _, a := range n.Arguments {
			args = append(args, RenderASTAsText(a, 0))
		}
		return fmt.Sprintf("%s(%s)", RenderASTAsText(n.Function, 0), strings.Join(args, ", "))

	case *ast.RecurExpression:
		args := []string{}
		for _, a := range n.Arguments {
			args = append(args, RenderASTAsText(a, 0))
		}
		return fmt.Sprintf("recur(%s)", strings.Join(args, ", "))

	case *ast.InfixExpression:
		return fmt.Sprintf("(%s %s %s)", RenderASTAsText(n.Left, 0), n.Operator, RenderASTAsText(n.Right, 0))

	case *ast.PrefixExpression:
		return fmt.Sprintf("(%s%s)", n.Operator, RenderASTAsText(n.Right, 0))

	case *ast.IfExpression:
		res := fmt.Sprintf("if %s %s", RenderASTAsText(n.Condition, 0), RenderASTAsText(n.ThenBranch, indent))
		if n.ElseBranch != nil {
			res += " else " + RenderASTAsText(n.ElseBranch, indent)
		}
		return res

	case *ast.MatchExpression:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("match %s {", RenderASTAsText(n.Value, 0)))
		for _, c := range n.Cases {
			sb.WriteString("\n")
			// Cases are structural lines, they handle their own 'sp'
			sb.WriteString(RenderASTAsText(c, indent+1))
		}
		sb.WriteString("\n" + sp + "}")
		return sb.String()

	case *ast.MatchCase:
		guard := ""
		if n.Guard != nil {
			guard = " if " + RenderASTAsText(n.Guard, 0)
		}
		// MatchCase is a structural line: it prepends its own 'sp'.
		// The body (Block or Expression) flows from the '=>'.
		return fmt.Sprintf("%s%s%s => %s", sp, RenderASTAsText(n.Pattern, 0), guard, RenderASTAsText(n.Body, indent))

	case *ast.IndexExpression:
		return fmt.Sprintf("%s[%s]", RenderASTAsText(n.Left, 0), RenderASTAsText(n.Index, 0))

	case *ast.SliceExpression:
		start := ""
		if n.Start != nil {
			start = RenderASTAsText(n.Start, 0)
		}
		end := ""
		if n.End != nil {
			end = RenderASTAsText(n.End, 0)
		}
		step := ""
		if n.Step != nil {
			step = ":" + RenderASTAsText(n.Step, 0)
		}
		return fmt.Sprintf("%s:%s%s", start, end, step)

	case *ast.Identifier:
		return n.Value
	case *ast.NumberLiteral:
		return n.Value.String()
	case *ast.StringLiteral:
		return fmt.Sprintf("%q", n.Value)
	case *ast.SymbolLiteral:
		return ":" + n.Value
	case *ast.Boolean:
		return fmt.Sprintf("%v", n.Value)
	case *ast.Nil:
		return "nil"
	case *ast.BytesLiteral:
		return fmt.Sprintf("0x%q", hex.EncodeToString(n.Value))

	case *ast.ListLiteral:
		elems := []string{}
		for _, e := range n.Elements {
			elems = append(elems, RenderASTAsText(e, 0))
		}
		return "[" + strings.Join(elems, ", ") + "]"

	case *ast.MapLiteral:
		pairs := []string{}
		for k, v := range n.Pairs {
			pairs = append(pairs, fmt.Sprintf("%s: %s", RenderASTAsText(k, 0), RenderASTAsText(v, 0)))
		}
		return "{" + strings.Join(pairs, ", ") + "}"

	case *ast.StructSchemaExpression:
		fields := []string{}
		for _, f := range n.Fields {
			field := ""
			if f.Hint != "" {
				field = f.Hint + " "
			}
			field += f.Name
			if f.Default != nil {
				field += " = " + RenderASTAsText(f.Default, 0)
			}
			fields = append(fields, field)
		}
		return "struct {" + strings.Join(fields, ", ") + "}"

	case *ast.StructInitExpression:
		fields := []string{}
		for _, f := range n.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", f.Name, RenderASTAsText(f.Value, 0)))
		}
		return fmt.Sprintf("%s {%s}", RenderASTAsText(n.Schema, 0), strings.Join(fields, ", "))

	case *ast.StructCopyExpression:
		fields := []string{}
		for _, f := range n.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", f.Name, RenderASTAsText(f.Value, 0)))
		}
		return fmt.Sprintf("%s copy {%s}", RenderASTAsText(n.Source, 0), strings.Join(fields, ", "))

	case *ast.WildcardPattern:
		return "_"
	case *ast.AllPattern:
		return "*"
	case *ast.BindingPattern:
		return RenderASTAsText(n.Name, 0) + " @ " + RenderASTAsText(n.Pattern, 0)
	case *ast.IdentifierPattern:
		return RenderASTAsText(n.Value, 0)
	case *ast.LiteralPattern:
		return RenderASTAsText(n.Value, 0)
	case *ast.PinnedIdentifierPattern:
		return "^" + RenderASTAsText(n.Value, 0)
	case *ast.SpreadPattern:
		res := "..."
		if n.Value != nil {
			res += n.Value.Value
		}
		return res

	case *ast.ListPattern:
		elems := []string{}
		for _, e := range n.Elements {
			elems = append(elems, RenderASTAsText(e, 0))
		}
		return "[" + strings.Join(elems, ", ") + "]"

	case *ast.MapPattern:
		pairs := []string{}
		for _, entry := range n.Pairs {
			pairs = append(pairs, fmt.Sprintf("%s: %s", RenderASTAsText(entry.Key, 0), RenderASTAsText(entry.Pattern, 0)))
		}
		if n.SelectAll {
			pairs = append(pairs, "*")
		}
		if n.Spread != nil {
			pairs = append(pairs, "..."+RenderASTAsText(n.Spread, 0))
		}
		delim := ""
		if n.Exact {
			delim = "|"
		}
		return "{" + delim + strings.Join(pairs, ", ") + delim + "}"

	case *ast.StructPattern:
		fields := []string{}
		for _, f := range n.Fields {
			fields = append(fields, fmt.Sprintf("%s: %s", f.Name, RenderASTAsText(f.Pattern, 0)))
		}
		return fmt.Sprintf("%s {%s}", RenderASTAsText(n.Schema, 0), strings.Join(fields, ", "))

	case *ast.DeferStatement:
		mode := ""
		switch n.Mode {
		case ast.DeferOnSuccess:
			mode = "onsuccess "
		case ast.DeferOnError:
			errName := ""
			if n.ErrorName != nil {
				errName = "(" + n.ErrorName.Value + ")"
			}
			mode = "onerror" + errName + " "
		}
		return fmt.Sprintf("%sdefer %s%s", sp, mode, RenderASTAsText(n.Call, 0))

	case *ast.ForeignFunctionDeclaration:
		params := []string{}
		for _, p := range n.Parameters {
			params = append(params, RenderASTAsText(p, 0))
		}
		return fmt.Sprintf("%s%sforeign %s = fn(%s)", sp, renderTags(n.Tags), RenderASTAsText(n.Name, 0), strings.Join(params, ", "))

	case *ast.SpreadExpression:
		return "..." + RenderASTAsText(n.Value, 0)

	case *ast.NotImplemented:
		return "???"

	default:
		return fmt.Sprintf("<unknown:%T>", n)
	}
}

func renderTags(tags []*ast.Tag) string {
	if len(tags) == 0 {
		return ""
	}
	var res []string
	for _, t := range tags {
		if len(t.Args) > 0 {
			args := []string{}
			for _, a := range t.Args {
				args = append(args, RenderASTAsText(a, 0))
			}
			res = append(res, fmt.Sprintf("@%s(%s)", t.Name, strings.Join(args, ", ")))
		} else {
			res = append(res, "@"+t.Name)
		}
	}
	return strings.Join(res, " ") + " "
}
