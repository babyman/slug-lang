package log

import (
	"fmt"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.LogConfigure{}): kernel.RightExec,
	reflect.TypeOf(svc.LogfMessage{}):  kernel.RightWrite,
	reflect.TypeOf(svc.LogMessage{}):   kernel.RightWrite,
}

var logSvc = logger.NewLogger("service", logger.INFO)

type LogService struct {
}

func (l *LogService) Handler(ctx *kernel.ActCtx, msg kernel.Message) {
	switch payload := msg.Payload.(type) {
	case svc.LogConfigure:
		logSvc.SetLevel(payload.Level)
		svc.Reply(ctx, msg, nil)
	case svc.LogfMessage:
		switch payload.Level {
		case logger.DEBUG:
			logSvc.Debugf("%d:"+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.INFO:
			logSvc.Infof("%d:"+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.WARN:
			logSvc.Warnf("%d:"+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.ERROR:
			logSvc.Errorf("%d:"+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		case logger.FATAL:
			logSvc.Fatalf("%d:"+payload.Message, append([]any{payload.Source}, payload.Args...)...)
		}
		svc.Reply(ctx, msg, nil)
	case svc.LogMessage:
		switch payload.Level {
		case logger.DEBUG:
			logSvc.Debug(fmt.Sprintf("%d:%s", payload.Source, payload.Message))
		case logger.INFO:
			logSvc.Info(fmt.Sprintf("%d:%s", payload.Source, payload.Message))
		case logger.WARN:
			logSvc.Warn(fmt.Sprintf("%d:%s", payload.Source, payload.Message))
		case logger.ERROR:
			logSvc.Error(fmt.Sprintf("%d:%s", payload.Source, payload.Message))
		case logger.FATAL:
			logSvc.Fatal(fmt.Sprintf("%d:%s", payload.Source, payload.Message))
		}
		svc.Reply(ctx, msg, nil)
	default:
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	}
}
