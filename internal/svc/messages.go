package svc

import (
	"slug/internal/ast"
)

const SOutService = "sout"
const ModuleService = "mods"
const ResolverService = "resolver"
const CliService = "cli"
const FsService = "fs"
const LexerService = "lexer"
const ParserService = "parser"
const EvalService = "eval"
const ReplService = "repl"

// SOut service messages
// ====================

type SOutPrintf struct {
	Str  string
	Args []any
}

// Evaluator service messages
// ==========================

type EvaluateProgram struct {
	Name    string
	Path    string
	Source  string
	Program *ast.Program
	Args    []string
}

type EvaluateResult struct {
	Result string
	Error  error
}
