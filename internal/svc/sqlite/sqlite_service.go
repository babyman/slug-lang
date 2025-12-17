package sqlite

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

	_ "github.com/mattn/go-sqlite3"
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
	if !ok {
		ctx.SendAsync(msg.From, errorStrResult("invalid message payload, SlugActorMessage expected"))
	} else {
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
			connId, err := ctx.SpawnChild("sqlite-conn", Operations, conn.Handler)
			if err != nil {
				ctx.SendAsync(to, errorResult(err))
				return kernel.Continue{}
			}
			ctx.GrantChildAccess(msg.From, connId, kernel.RightWrite, nil)
			ctx.ForwardAsync(connId, msg)
		}
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
			// log here
			return kernel.Continue{}
		}

		msgType, ok := m.Pairs[msgTypeKey]
		if !ok {
			// log here
			return kernel.Continue{}
		}

		switch msgType.Value.Inspect() {
		case "connect":

			if sc.db != nil {
				slog.Warn("sqlite connection already established")
				return kernel.Continue{}
			}

			connStr := m.Pairs[connectionStringKey]

			var err error
			sc.db, err = sql.Open("sqlite3", connStr.Value.Inspect())
			if err != nil {
				slog.Error("failed to open sqlite connection", slog.Any("error", err.Error()))
				ctx.SendAsync(to, errorResult(err))
				return kernel.Terminate{
					Reason: "Failed to open connection: " + err.Error(),
				}
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
			rowMap.Pairs[key] = object.MapPair{
				Key:   &object.String{Value: col},
				Value: &object.String{Value: fmt.Sprintf("%v", values[i])},
			}
		}
		resultRows = append(resultRows, rowMap)
	}
	// todo this can be more efficient?
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

	paramsObj := m.Pairs[paramsKey].Value

	paramsList, ok := paramsObj.(*object.List)
	if !ok {
		return nil, false
	}

	params := make([]any, len(paramsList.Elements))
	for i, elem := range paramsList.Elements {
		params[i] = elem.Inspect()
	}
	return params, true
}
