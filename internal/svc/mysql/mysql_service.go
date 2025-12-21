package mysql

import (
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"slug/internal/dec64"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/svcutil"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SlugActorMessage{}): kernel.RightWrite,
}

type Service struct {
}

type Connection struct {
	state svcutil.ConnectionState
}

func (fs *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if ok {
		to := svcutil.ReplyTarget(msg)
		m, ok := p.Msg.(*object.Map)
		if !ok {
			ctx.SendAsync(to, svcutil.ErrorResult("invalid message payload, map expected"))
			return kernel.Continue{}
		}

		msgType, ok := m.Pairs[svcutil.MsgTypeKey]
		if !ok {
			ctx.SendAsync(to, svcutil.ErrorResult("invalid message payload"))
			return kernel.Continue{}
		}

		switch msgType.Value.Inspect() {
		case "connect":
			conn := &Connection{}
			connId, err := ctx.SpawnChild("mysql-conn", Operations, conn.Handler)
			if err != nil {
				ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
				return kernel.Continue{}
			}
			ctx.GrantChildAccess(msg.From, connId, kernel.RightWrite, nil)
			ctx.ForwardAsync(connId, msg)
		}
	} else {
		slog.Debug("invalid message payload", slog.Any("payload", msg.Payload))
	}
	return kernel.Continue{}
}

func (sc *Connection) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	slugMsg, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, _ := slugMsg.Msg.(*object.Map)
	msgType, _ := m.Pairs[svcutil.MsgTypeKey]

	if msgType.Value.Inspect() == "connect" {
		if sc.state.DB != nil {
			return kernel.Continue{}
		}
		connStr := m.Pairs[svcutil.ConnectionStringKey].Value.Inspect()
		db, err := sql.Open("mysql", connStr)
		if err != nil {
			slog.Error("failed to open mysql connection", slog.Any("error", err.Error()))
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Terminate{
				Reason: "Failed to open connection: " + err.Error(),
			}
		}
		sc.state.DB = db

		// Test the connection
		if err := sc.state.DB.Ping(); err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Terminate{Reason: "Ping failed"}
		}

		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})
		return kernel.Continue{}
	}

	return svcutil.HandleConnection(&sc.state, ctx, msg, TypeMapper)
}

func TypeMapper(v any, ct *sql.ColumnType) object.Object {
	if v == nil {
		return &object.Nil{}
	}

	switch x := v.(type) {
	case int:
		return &object.Number{Value: dec64.FromInt64(int64(x))}
	case int64:
		return &object.Number{Value: dec64.FromInt64(x)}
	case float64:
		s := strconv.FormatFloat(x, 'g', -1, 64)
		if d, err := dec64.FromString(s); err == nil {
			return &object.Number{Value: d}
		}
		return &object.String{Value: s}
	case bool:
		return &object.Boolean{Value: x}
	case time.Time:
		return &object.String{Value: x.Format(time.RFC3339Nano)}
	case []byte:
		if ct != nil {
			decl := strings.ToUpper(ct.DatabaseTypeName())
			// MySQL drivers often return DECIMAL and BIT as []byte
			if decl == "DECIMAL" || decl == "NEWDECIMAL" {
				if d, err := dec64.FromString(string(x)); err == nil {
					return &object.Number{Value: d}
				}
				return &object.String{Value: string(x)}
			}
			if strings.Contains(decl, "CHAR") || strings.Contains(decl, "TEXT") || strings.Contains(decl, "VARCHAR") {
				return &object.String{Value: string(x)}
			}
		}
		return &object.Bytes{Value: x}
	case string:
		return &object.String{Value: x}
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}
