package log

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(kernel.ConfigureSystem{}): kernel.RightExec,
	reflect.TypeOf(svc.LogfMessage{}):        kernel.RightWrite,
	reflect.TypeOf(svc.LogMessage{}):         kernel.RightWrite,
}

var logSvc = logger.NewLogger("service", logger.FATAL)

type LogService struct {
}

func (l *LogService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case kernel.ConfigureSystem:
		logSvc.SetLevel(payload.LogLevel)
		if payload.LogPath != "" {
			logSvc.SetLogFile(payload.LogPath)
		}
		svc.Reply(ctx, msg, nil)
	case svc.LogfMessage:
		switch payload.Level {
		case logger.TRACE:
			logSvc.Tracef("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.DEBUG:
			logSvc.Debugf("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.INFO:
			logSvc.Infof("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.WARN:
			logSvc.Warnf("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.ERROR:
			logSvc.Errorf("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.FATAL:
			logSvc.Fatalf("ActorID %d: "+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		}
		svc.Reply(ctx, msg, nil)
	case svc.LogMessage:
		switch payload.Level {
		case logger.TRACE:
			logSvc.Trace(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		case logger.DEBUG:
			logSvc.Debug(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		case logger.INFO:
			logSvc.Info(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		case logger.WARN:
			logSvc.Warn(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		case logger.ERROR:
			logSvc.Error(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		case logger.FATAL:
			logSvc.Fatal(fmt.Sprintf("ActorID %d: %s", payload.Source, payload.Message))
		}
		svc.Reply(ctx, msg, nil)
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
	return kernel.Continue{}
}
