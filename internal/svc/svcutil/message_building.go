package svcutil

import (
	"slug/internal/dec64"
	"slug/internal/object"
	"slug/internal/svc/eval"
)

var (
	MsgTypeKey          = (&object.String{Value: "type"}).MapKey()
	SqlKey              = (&object.String{Value: "sql"}).MapKey()
	ParamsKey           = (&object.String{Value: "params"}).MapKey()
	ConnectionStringKey = (&object.String{Value: "connectionString"}).MapKey()
)

func PutObj(resultMap *object.Map, key string, val object.Object) {
	keyStr := &object.String{Value: key}
	resultMap.Pairs[(keyStr).MapKey()] = object.MapPair{
		Key:   keyStr,
		Value: val,
	}
}

func PutString(resultMap *object.Map, key string, val string) {
	PutObj(resultMap, key, &object.String{Value: val})
}

func PutInt(resultMap *object.Map, key string, val int) {
	PutObj(resultMap, key, &object.Number{Value: dec64.FromInt(val)})
}

func PutBool(resultMap *object.Map, key string, val bool) {
	var value = eval.FALSE
	if val {
		value = eval.TRUE
	}
	PutObj(resultMap, key, value)
}

func PutError(resultMap *object.Map, key string, err error) {
	PutObj(resultMap, key, &object.Error{Message: err.Error()})
}
