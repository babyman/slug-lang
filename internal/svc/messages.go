package svc

import (
	"slug/internal/ast"
	"slug/internal/object"
)

const (
	SlugNamespace   = "slug:"
	CliService      = "cli"
	EvalService     = "eval"
	FsService       = "fs"
	LexerService    = "lexer"
	ModuleService   = "mods"
	ParserService   = "parser"
	ReplService     = "repl"
	ResolverService = "resolver"
	SOutService     = "sout"
	SqliteService   = SlugNamespace + "sqlite"
)

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

type SlugActorMessage struct {
	Msg object.Object
}
