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
	MaxKey              = (&object.String{Value: "max"}).MapKey()
	DataKey             = (&object.String{Value: "data"}).MapKey()
)

func putObj(resultMap *object.Map, key string, val object.Object) {
	keyStr := &object.String{Value: key}
	resultMap.Pairs[(keyStr).MapKey()] = object.MapPair{
		Key:   keyStr,
		Value: val,
	}
}

func GetObj(m *object.Map, key object.MapKey) (object.Object, bool) {
	pair, ok := m.Pairs[key]
	if !ok {
		return nil, false
	}
	return pair.Value, true
}

func PutString(resultMap *object.Map, key string, val string) {
	putObj(resultMap, key, &object.String{Value: val})
}

func GetString(m *object.Map, key object.MapKey) (string, bool) {
	pair, ok := GetObj(m, key)
	if !ok {
		return "", false
	}
	str, ok := pair.(*object.String)
	if !ok {
		return "", false
	}
	return str.Value, true
}

func GetStringWithDefault(m *object.Map, key object.MapKey, def string) string {
	str, ok := GetString(m, key)
	if !ok {
		return def
	}
	return str
}

func PutBytes(resultMap *object.Map, key string, val []byte) {
	putObj(resultMap, key, &object.Bytes{Value: val})
}

func PutInt(resultMap *object.Map, key string, val int) {
	putObj(resultMap, key, &object.Number{Value: dec64.FromInt(val)})
}

func PutInt64(resultMap *object.Map, key string, val int64) {
	putObj(resultMap, key, &object.Number{Value: dec64.FromInt64(val)})
}

func GetInt(m *object.Map, key object.MapKey) (int, bool) {
	pair, ok := GetObj(m, key)
	if !ok {
		return 0, false
	}
	str, ok := pair.(*object.Number)
	if !ok {
		return 0, false
	}
	return str.Value.ToInt(), true
}

func GetIntWithDefault(m *object.Map, key object.MapKey, def int) int {
	val, ok := GetInt(m, key)
	if !ok {
		return def
	}
	return val
}

func PutList(resultMap *object.Map, key string, val []object.Object) {
	putObj(resultMap, key, &object.List{Elements: val})
}

func PutBool(resultMap *object.Map, key string, val bool) {
	if val {
		putObj(resultMap, key, eval.TRUE)
	} else {
		putObj(resultMap, key, eval.FALSE)
	}
}
