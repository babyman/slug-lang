package svc

import (
	"slug/internal/ast"
	"slug/internal/object"
)

const (
	SlugNamespace   = "slug:"
	CliService      = "cli"
	EvalService     = "evaluator"
	FsService       = "filesystem"
	LexerService    = "lexer"
	ModuleService   = "module-loader"
	ParserService   = "parser"
	ReplService     = "repl"
	ResolverService = "resolver"
	SOutService     = "system-out"
	SqliteService   = SlugNamespace + "sqlite"
	MysqlService    = SlugNamespace + "mysql"
	TcpService      = SlugNamespace + "tcp"
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
