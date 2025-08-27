package service

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/logger"
)

type LogConfigure struct {
	level logger.Level
}

type LogfMessage struct {
	source  kernel.ActorID
	level   logger.Level
	Message string
	Args    []any
}

type LogMessage struct {
	source  kernel.ActorID
	level   logger.Level
	Message string
}

var LogOperations = kernel.OpRights{
	reflect.TypeOf(LogConfigure{}): kernel.RightExec,
	reflect.TypeOf(LogfMessage{}):  kernel.RightWrite,
	reflect.TypeOf(LogMessage{}):   kernel.RightWrite,
}

var logSvc = logger.NewLogger("service", logger.INFO)

type Log struct {
}

func (l *Log) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case LogConfigure:
		logSvc.SetLevel(payload.level)
	case LogfMessage:
		switch payload.level {
		case logger.DEBUG:
			logSvc.Debugf("%d:"+payload.Message, append([]any{payload.source}, payload.Args...)...)
		case logger.INFO:
			logSvc.Infof("%d:"+payload.Message, append([]any{payload.source}, payload.Args...)...)
		case logger.WARN:
			logSvc.Warnf("%d:"+payload.Message, append([]any{payload.source}, payload.Args...)...)
		case logger.ERROR:
			logSvc.Errorf("%d:"+payload.Message, append([]any{payload.source}, payload.Args...)...)
		case logger.FATAL:
			logSvc.Fatalf("%d:"+payload.Message, append([]any{payload.source}, payload.Args...)...)
		}
	case LogMessage:
		switch payload.level {
		case logger.DEBUG:
			logSvc.Debug(fmt.Sprintf("%d:%s", payload.source, payload.Message))
		case logger.INFO:
			logSvc.Info(fmt.Sprintf("%d:%s", payload.source, payload.Message))
		case logger.WARN:
			logSvc.Warn(fmt.Sprintf("%d:%s", payload.source, payload.Message))
		case logger.ERROR:
			logSvc.Error(fmt.Sprintf("%d:%s", payload.source, payload.Message))
		case logger.FATAL:
			logSvc.Fatal(fmt.Sprintf("%d:%s", payload.source, payload.Message))
		}
	}
}
