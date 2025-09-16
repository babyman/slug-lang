package svc

import (
	"slug/internal/ast"
	"slug/internal/kernel"
	"slug/internal/logger"
)

const SOutService = "sout"
const LogService = "log"
const ModuleService = "mods"
const CliService = "cli"
const FsService = "fs"
const LexerService = "lexer"
const ParserService = "parser"
const EvalService = "eval"
const ReplService = "repl"

// Log service messages
// ====================

type LogfMessage struct {
	Source  kernel.ActorID
	Level   logger.Level
	Message string
	Args    []any
}

type LogMessage struct {
	Source  kernel.ActorID
	Level   logger.Level
	Message string
}

// SOut service messages
// ====================

type SOutPrintln struct {
	Str  string
	Args []any
}

// Evaluator service messages
// ==========================

type EvaluateProgram struct {
	Source  string
	Args    []string
	Program *ast.Program
}
