package foreign

import (
	"database/sql"
	"fmt"
	"slug/internal/dec64"
	"slug/internal/object"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var (
	dbConnections  = map[int64]*sql.DB{}
	dbTransactions = map[int64]*sql.Tx{}
)

func fnIoDbConnect() *object.Foreign {
	return &object.Foreign{
		Name: "connect",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 2 {
				return ctx.NewError("connect expects 2 arguments: connectionString, driver")
			}
			connStr, _ := unpackString(args[0], "")
			driver, _ := unpackString(args[1], "")

			db, err := sql.Open(driver, connStr)
			if err != nil {
				return ctx.NewError("failed to open connection: %v", err)
			}

			if err := db.Ping(); err != nil {
				return ctx.NewError("failed to ping database: %v", err)
			}

			id := ctx.NextHandleID()
			dbConnections[id] = db
			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnIoDbQuery() *object.Foreign {
	return &object.Foreign{
		Name: "query",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) < 2 {
				return ctx.NewError("query expects at least 2 arguments: connection, sql")
			}
			id, _ := unpackNumber(args[0], "")
			query, _ := unpackString(args[1], "")

			db, ok := dbConnections[id]
			if !ok {
				return ctx.NewError("invalid connection handle")
			}

			params := make([]interface{}, len(args)-2)
			for i := 2; i < len(args); i++ {
				params[i-2] = args[i].Inspect() // Simple inspect for now
			}

			var rows *sql.Rows
			var err error

			tx, isTx := dbTransactions[id]
			if isTx {
				rows, err = tx.Query(query, params...)
			} else {
				rows, err = db.Query(query, params...)
			}

			if err != nil {
				return ctx.NewError("query failed: %v", err)
			}
			defer rows.Close()

			return renderRows(rows)
		},
	}
}

func fnIoDbExec() *object.Foreign {
	return &object.Foreign{
		Name: "exec",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			id, _ := unpackNumber(args[0], "")
			query, _ := unpackString(args[1], "")

			db, ok := dbConnections[id]
			if !ok {
				return ctx.NewError("invalid connection handle")
			}

			params := make([]interface{}, len(args)-2)
			for i := 2; i < len(args); i++ {
				params[i-2] = args[i].Inspect()
			}

			var result sql.Result
			var err error

			tx, isTx := dbTransactions[id]
			if isTx {
				result, err = tx.Exec(query, params...)
			} else {
				result, err = db.Exec(query, params...)
			}

			if err != nil {
				return ctx.NewError("exec failed: %v", err)
			}

			affected, _ := result.RowsAffected()
			lastID, _ := result.LastInsertId()

			resMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
			resMap.Put(&object.String{Value: "rowsAffected"}, &object.Number{Value: dec64.FromInt64(affected)})
			resMap.Put(&object.String{Value: "lastInsertId"}, &object.Number{Value: dec64.FromInt64(lastID)})
			return resMap
		},
	}
}

func fnIoDbClose() *object.Foreign {
	return &object.Foreign{
		Name: "close",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			id, _ := unpackNumber(args[0], "")
			if tx, ok := dbTransactions[id]; ok {
				tx.Rollback()
				delete(dbTransactions, id)
			}
			if db, ok := dbConnections[id]; ok {
				db.Close()
				delete(dbConnections, id)
			}
			return ctx.Nil()
		},
	}
}

// Helpers for transaction control
func fnIoDbBegin() *object.Foreign {
	return &object.Foreign{
		Name: "begin",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("begin expects 1 argument: connection")
			}
			id, _ := unpackNumber(args[0], "")

			db, ok := dbConnections[id]
			if !ok {
				return ctx.NewError("invalid connection handle")
			}

			tx, err := db.Begin()
			if err != nil {
				return ctx.NewError("failed to begin transaction: %v", err)
			}

			dbTransactions[id] = tx
			return args[0]
		},
	}
}

func fnIoDbCommit() *object.Foreign {
	return &object.Foreign{
		Name: "commit",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("commit expects 1 argument: connection")
			}
			id, _ := unpackNumber(args[0], "")

			tx, ok := dbTransactions[id]
			if !ok {
				return ctx.NewError("invalid transaction handle")
			}

			err := tx.Commit()
			if err != nil {
				return ctx.NewError("failed to commit transaction: %v", err)
			}

			delete(dbTransactions, id)
			return args[0]
		},
	}
}

func fnIoDbRollback() *object.Foreign {
	return &object.Foreign{
		Name: "rollback",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			if len(args) != 1 {
				return ctx.NewError("rollback expects 1 argument: connection")
			}
			id, _ := unpackNumber(args[0], "")

			tx, ok := dbTransactions[id]
			if !ok {
				return ctx.NewError("invalid transaction handle")
			}

			err := tx.Rollback()
			if err != nil {
				return ctx.NewError("failed to rollback transaction: %v", err)
			}

			delete(dbTransactions, id)
			return args[0]
		},
	}
}

func renderRows(rows *sql.Rows) object.Object {
	columns, _ := rows.Columns()
	types, _ := rows.ColumnTypes()
	var resultRows []object.Object

	for rows.Next() {
		values := make([]interface{}, len(columns))
		pointers := make([]interface{}, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		rows.Scan(pointers...)

		rowMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
		for i, col := range columns {
			// Pass column type info to help mapValue decide
			var typeName string
			if i < len(types) {
				typeName = types[i].DatabaseTypeName()
			}
			rowMap.Put(&object.String{Value: col}, mapValue(values[i], typeName))
		}
		resultRows = append(resultRows, rowMap)
	}
	return &object.List{Elements: resultRows}
}

func mapValue(v interface{}, dbType string) object.Object {
	if v == nil {
		return &object.Nil{}
	}
	switch x := v.(type) {
	case int64:
		return &object.Number{Value: dec64.FromInt64(x)}
	case float64:
		d, _ := dec64.FromString(strconv.FormatFloat(x, 'f', -1, 64))
		return &object.Number{Value: d}
	case []byte:
		// If the DB type hints at text, or if it's not a BLOB-like type, treat as string
		switch dbType {
		case "TEXT", "VARCHAR", "CHAR", "LONGTEXT", "MEDIUMTEXT", "TINYTEXT":
			return &object.String{Value: string(x)}
		case "BLOB", "LONGBLOB", "MEDIUMBLOB", "TINYBLOB", "BINARY", "VARBINARY":
			return &object.Bytes{Value: x}
		default:
			// Fallback: if it's valid UTF-8, it's probably a string the driver was being shy about
			return &object.String{Value: string(x)}
		}
	case string:
		return &object.String{Value: x}
	case bool:
		return &object.Boolean{Value: x}
	case time.Time:
		return &object.String{Value: x.Format(time.RFC3339)}
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}
