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
	"slug/internal/svc/eval"
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
	db *sql.DB
	tx *sql.Tx
}

var (
	msgTypeKey          = (&object.String{Value: "type"}).MapKey()
	connectionStringKey = (&object.String{Value: "connectionString"}).MapKey()
	sqlKey              = (&object.String{Value: "sql"}).MapKey()
	paramsKey           = (&object.String{Value: "params"}).MapKey()
)

func (fs *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if ok {
		to := replyTarget(msg)
		m, ok := p.Msg.(*object.Map)
		if !ok {
			ctx.SendAsync(to, errorStrResult("invalid message payload, map expected"))
			return kernel.Continue{}
		}

		msgType, ok := m.Pairs[msgTypeKey]
		if !ok {
			ctx.SendAsync(to, errorStrResult("invalid message payload"))
			return kernel.Continue{}
		}

		switch msgType.Value.Inspect() {
		case "connect":
			conn := &Connection{}
			connId, err := ctx.SpawnChild("mysql-conn", Operations, conn.Handler)
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
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
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
	} else {
		to := replyTarget(msg)
		m, ok := slugMsg.Msg.(*object.Map)
		if !ok {
			return kernel.Continue{}
		}

		msgType, ok := m.Pairs[msgTypeKey]
		if !ok {
			return kernel.Continue{}
		}

		switch msgType.Value.Inspect() {
		case "connect":
			if sc.db != nil {
				slog.Warn("mysql connection already established")
				return kernel.Continue{}
			}

			connStr := m.Pairs[connectionStringKey]

			var err error
			sc.db, err = sql.Open("mysql", connStr.Value.Inspect())
			if err != nil {
				slog.Error("failed to open mysql connection", slog.Any("error", err.Error()))
				ctx.SendAsync(to, errorResult(err))
				return kernel.Terminate{
					Reason: "Failed to open connection: " + err.Error(),
				}
			}
			// Test the connection
			if err := sc.db.Ping(); err != nil {
				ctx.SendAsync(to, errorResult(err))
				return kernel.Terminate{Reason: "Ping failed"}
			}

			ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
				Value: dec64.FromInt64(int64(ctx.Self)),
			}})

		case "query":
			var rows *sql.Rows
			var err error
			sqlStr := m.Pairs[sqlKey].Value.Inspect()
			params, ok := extractParameters(m)
			if !ok {
				ctx.SendAsync(to, errorStrResult("params must be a list"))
				return kernel.Continue{}
			}

			if sc.tx != nil {
				rows, err = sc.tx.Query(sqlStr, params...)
			} else {
				rows, err = sc.db.Query(sqlStr, params...)
			}
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
				return kernel.Continue{}
			}
			defer rows.Close()
			ctx.SendAsync(to, execSuccessRows(rows))

		case "exec":
			var result sql.Result
			var err error
			sqlStr := m.Pairs[sqlKey].Value.Inspect()
			params, ok := extractParameters(m)
			if !ok {
				ctx.SendAsync(to, errorStrResult("params must be a list"))
				return kernel.Continue{}
			}

			if sc.tx != nil {
				result, err = sc.tx.Exec(sqlStr, params...)
			} else {
				result, err = sc.db.Exec(sqlStr, params...)
			}
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
			} else {
				ctx.SendAsync(to, execSuccessResult(result))
			}

		case "begin":
			if sc.tx != nil {
				ctx.SendAsync(to, errorStrResult("transaction already in progress"))
				return kernel.Continue{}
			}
			tx, err := sc.db.Begin()
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
			} else {
				sc.tx = tx
				ctx.SendAsync(to, success())
			}

		case "commit":
			if sc.tx == nil {
				ctx.SendAsync(to, errorStrResult("no transaction in progress"))
				return kernel.Continue{}
			}
			err := sc.tx.Commit()
			sc.tx = nil
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
			} else {
				ctx.SendAsync(to, success())
			}

		case "rollback":
			if sc.tx == nil {
				ctx.SendAsync(to, errorStrResult("no transaction in progress"))
				return kernel.Continue{}
			}
			err := sc.tx.Rollback()
			sc.tx = nil
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
			} else {
				ctx.SendAsync(to, success())
			}

		case "close":
			if sc.tx != nil {
				sc.tx.Rollback()
				sc.tx = nil
			}
			if sc.db != nil {
				sc.db.Close()
			}
			ctx.SendAsync(to, success())
			return kernel.Terminate{
				Reason: "Connection closed",
			}
		}
	}
	return kernel.Continue{}
}

func success() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.TRUE,
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func execSuccessResult(result sql.Result) svc.SlugActorMessage {
	affected, _ := result.RowsAffected()
	lastInsertId, _ := result.LastInsertId()
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.TRUE,
	}
	resultMap.Pairs[(&object.String{Value: "rowsAffected"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "rowsAffected"},
		Value: &object.Number{Value: dec64.FromInt64(affected)},
	}
	if lastInsertId > 0 {
		resultMap.Pairs[(&object.String{Value: "lastInsertId"}).MapKey()] = object.MapPair{
			Key:   &object.String{Value: "lastInsertId"},
			Value: &object.Number{Value: dec64.FromInt64(lastInsertId)},
		}
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func execSuccessRows(rows *sql.Rows) svc.SlugActorMessage {
	columns, err := rows.Columns()
	if err != nil {
		return errorResult(err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return errorResult(err)
	}

	var resultRows []*object.Map
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		err := rows.Scan(pointers...)
		if err != nil {
			return errorResult(err)
		}
		rowMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
		for i, col := range columns {
			key := (&object.String{Value: col}).MapKey()

			var ct *sql.ColumnType
			if i < len(colTypes) {
				ct = colTypes[i]
			}

			rowMap.Pairs[key] = object.MapPair{
				Key:   &object.String{Value: col},
				Value: sqlValueToSlugObject(values[i], ct),
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	listElements := make([]object.Object, len(resultRows))
	for i, row := range resultRows {
		listElements[i] = row
	}
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.TRUE,
	}
	resultMap.Pairs[(&object.String{Value: "rows"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "rows"},
		Value: &object.List{Elements: listElements},
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func sqlValueToSlugObject(v any, ct *sql.ColumnType) object.Object {
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

func errorResult(err error) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.FALSE,
	}
	resultMap.Pairs[(&object.String{Value: "error"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "error"},
		Value: &object.Error{Message: err.Error()},
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func errorStrResult(err string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.FALSE,
	}
	resultMap.Pairs[(&object.String{Value: "error"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "error"},
		Value: &object.Error{Message: err},
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func replyTarget(msg kernel.Message) kernel.ActorID {
	if msg.ReplyTo > 0 {
		return msg.ReplyTo
	}
	return msg.From
}

func extractParameters(m *object.Map) ([]any, bool) {
	paramsObj, ok := m.Pairs[paramsKey]
	if !ok {
		return []any{}, true
	}

	paramsList, ok := paramsObj.Value.(*object.List)
	if !ok {
		return nil, false
	}

	params := make([]any, len(paramsList.Elements))
	for i, elem := range paramsList.Elements {
		params[i] = elem.Inspect()
	}
	return params, true
}
