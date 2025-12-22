package svcutil

import (
	"database/sql"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
)

type ConnectionState struct {
	DB *sql.DB
	Tx *sql.Tx
}

func HandleConnection(sc *ConnectionState, ctx *kernel.ActCtx, msg kernel.Message, driverMapping func(any, *sql.ColumnType) object.Object) kernel.HandlerSignal {
	slugMsg, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		svc.Reply(ctx, msg, kernel.UnknownOperation{})
		return kernel.Continue{}
	}

	to := ReplyTarget(msg)
	m, ok := slugMsg.Msg.(*object.Map)
	if !ok {
		return kernel.Continue{}
	}

	msgType, ok := m.Pairs[MsgTypeKey]
	if !ok {
		return kernel.Continue{}
	}

	switch msgType.Value.Inspect() {
	case "query":
		sqlStr := m.Pairs[SqlKey].Value.Inspect()
		params, _ := ExtractParameters(m)
		var rows *sql.Rows
		var err error
		if sc.Tx != nil {
			rows, err = sc.Tx.QueryContext(ctx.Context, sqlStr, params...)
		} else {
			rows, err = sc.DB.QueryContext(ctx.Context, sqlStr, params...)
		}
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		defer rows.Close()
		ctx.SendAsync(to, queryResult(rows, driverMapping))

	case "exec":
		sqlStr := m.Pairs[SqlKey].Value.Inspect()
		params, _ := ExtractParameters(m)
		var result sql.Result
		var err error
		if sc.Tx != nil {
			result, err = sc.Tx.ExecContext(ctx.Context, sqlStr, params...)
		} else {
			result, err = sc.DB.ExecContext(ctx.Context, sqlStr, params...)
		}
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err.Error()))
		} else {
			affected, _ := result.RowsAffected()
			lastInsertId, _ := result.LastInsertId()
			ctx.SendAsync(to, execResult(affected, lastInsertId))
		}

	case "begin":
		if sc.Tx != nil {
			ctx.SendAsync(to, ErrorResult("transaction already in progress"))
			return kernel.Continue{}
		}
		tx, err := sc.DB.BeginTx(ctx.Context, nil)
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err.Error()))
		} else {
			sc.Tx = tx
			ctx.SendAsync(to, beginResult())
		}

	case "commit":
		if sc.Tx == nil {
			ctx.SendAsync(to, ErrorResult("no transaction in progress"))
			return kernel.Continue{}
		}
		err := sc.Tx.Commit()
		sc.Tx = nil
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, commitResult())
		}

	case "rollback":
		if sc.Tx == nil {
			ctx.SendAsync(to, ErrorResult("no transaction in progress"))
			return kernel.Continue{}
		}
		err := sc.Tx.Rollback()
		sc.Tx = nil
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, rollbackResult())
		}

	case "close":
		if sc.Tx != nil {
			sc.Tx.Rollback()
		}
		if sc.DB != nil {
			sc.DB.Close()
		}
		ctx.SendAsync(to, CloseResult(ctx.Self))
		return kernel.Terminate{Reason: "Connection closed"}
	}
	return kernel.Continue{}
}

func beginResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "begin")
	return svc.SlugActorMessage{Msg: resultMap}
}

func commitResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "commit")
	return svc.SlugActorMessage{Msg: resultMap}
}

func rollbackResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "rollback")
	return svc.SlugActorMessage{Msg: resultMap}
}

func execResult(affected int64, lastInsertId int64) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "exec")
	PutInt(resultMap, "rowsAffected", int(affected))
	if lastInsertId > 0 {
		PutInt(resultMap, "lastInsertId", int(lastInsertId))
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func queryResult(rows *sql.Rows, driverMapping func(any, *sql.ColumnType) object.Object) svc.SlugActorMessage {
	columns, err := rows.Columns()
	if err != nil {
		return ErrorResult(err.Error())
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return ErrorResult(err.Error())
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
			return ErrorResult(err.Error())
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
				Value: driverMapping(values[i], ct),
			}
		}
		resultRows = append(resultRows, rowMap)
	}

	listElements := make([]object.Object, len(resultRows))
	for i, row := range resultRows {
		listElements[i] = row
	}

	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutString(resultMap, "type", "query")
	PutBool(resultMap, "ok", true)
	PutObj(resultMap, "rows", &object.List{Elements: listElements})
	return svc.SlugActorMessage{Msg: resultMap}
}
