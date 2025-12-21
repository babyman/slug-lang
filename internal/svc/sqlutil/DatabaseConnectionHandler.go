package sqlutil

import (
	"database/sql"
	"slug/internal/dec64"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/eval"
)

type ConnectionState struct {
	DB *sql.DB
	Tx *sql.Tx
}

var (
	MsgTypeKey          = (&object.String{Value: "type"}).MapKey()
	SqlKey              = (&object.String{Value: "sql"}).MapKey()
	ParamsKey           = (&object.String{Value: "params"}).MapKey()
	ConnectionStringKey = (&object.String{Value: "connectionString"}).MapKey()
)

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
			rows, err = sc.Tx.Query(sqlStr, params...)
		} else {
			rows, err = sc.DB.Query(sqlStr, params...)
		}
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err))
			return kernel.Continue{}
		}
		defer rows.Close()
		ctx.SendAsync(to, ExecSuccessRows(rows, driverMapping))

	case "exec":
		sqlStr := m.Pairs[SqlKey].Value.Inspect()
		params, _ := ExtractParameters(m)
		var result sql.Result
		var err error
		if sc.Tx != nil {
			result, err = sc.Tx.Exec(sqlStr, params...)
		} else {
			result, err = sc.DB.Exec(sqlStr, params...)
		}
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err))
		} else {
			ctx.SendAsync(to, ExecSuccessResult(result))
		}

	case "begin":
		if sc.Tx != nil {
			ctx.SendAsync(to, ErrorStrResult("transaction already in progress"))
			return kernel.Continue{}
		}
		tx, err := sc.DB.Begin()
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err))
		} else {
			sc.Tx = tx
			ctx.SendAsync(to, Success())
		}

	case "commit":
		if sc.Tx == nil {
			ctx.SendAsync(to, ErrorStrResult("no transaction in progress"))
			return kernel.Continue{}
		}
		err := sc.Tx.Commit()
		sc.Tx = nil
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err))
		} else {
			ctx.SendAsync(to, Success())
		}

	case "rollback":
		if sc.Tx == nil {
			ctx.SendAsync(to, ErrorStrResult("no transaction in progress"))
			return kernel.Continue{}
		}
		err := sc.Tx.Rollback()
		sc.Tx = nil
		if err != nil {
			ctx.SendAsync(to, ErrorResult(err))
		} else {
			ctx.SendAsync(to, Success())
		}

	case "close":
		if sc.Tx != nil {
			sc.Tx.Rollback()
		}
		if sc.DB != nil {
			sc.DB.Close()
		}
		ctx.SendAsync(to, Success())
		return kernel.Terminate{Reason: "Connection closed"}
	}
	return kernel.Continue{}
}

func Success() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	resultMap.Pairs[(&object.String{Value: "ok"}).MapKey()] = object.MapPair{
		Key:   &object.String{Value: "ok"},
		Value: eval.TRUE,
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func ExecSuccessResult(result sql.Result) svc.SlugActorMessage {
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

func ExecSuccessRows(rows *sql.Rows, driverMapping func(any, *sql.ColumnType) object.Object) svc.SlugActorMessage {
	columns, err := rows.Columns()
	if err != nil {
		return ErrorResult(err)
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return ErrorResult(err)
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
			return ErrorResult(err)
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

func ErrorResult(err error) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutBool(resultMap, "ok", false)
	PutError(resultMap, "error", err)
	return svc.SlugActorMessage{Msg: resultMap}
}

func ErrorStrResult(err string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	PutBool(resultMap, "ok", false)
	PutString(resultMap, "error", err)
	return svc.SlugActorMessage{Msg: resultMap}
}

func ReplyTarget(msg kernel.Message) kernel.ActorID {

	if msg.ReplyTo > 0 {
		return msg.ReplyTo
	}

	return msg.From
}

func ExtractParameters(m *object.Map) ([]any, bool) {

	paramsObj := m.Pairs[ParamsKey].Value

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
