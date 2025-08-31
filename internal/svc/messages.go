package svc

import (
	"slug/internal/kernel"
	"slug/internal/logger"
)

// Log service messages
// ====================

type LogConfigure struct {
	Level logger.Level
}

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
