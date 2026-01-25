package foreign

import (
	"errors"
	"fmt"
	"slug/internal/dec64"
	"slug/internal/object"
)

// ToFunctionArgument determines if `obj` is a valid `Function` or within a `FunctionGroup`, matching the provided arguments.
// Returns the found `Function` and true if successful; otherwise returns nil and false.
func ToFunctionArgument(obj object.Object, args []object.Object) (*object.Function, error) {

	functionGroup, ok := obj.(*object.FunctionGroup)
	if ok {
		toFunction, err := functionGroup.DispatchToFunction("", args, nil)
		if err != nil {
			return nil, err
		}
		foundFunction, ok := toFunction.(*object.Function)
		if !ok {
			return nil, errors.New("found function is not a Function")
		}
		return foundFunction, nil
	} else {
		foundFunction, ok := obj.(*object.Function)
		if !ok {
			return nil, errors.New("found function is not a Function")
		}
		return foundFunction, nil
	}
}

func unpackString(arg object.Object, argName string) (string, error) {

	if arg.Type() != object.STRING_OBJ {
		return "", fmt.Errorf("argument to `%s` must be a STRING, got=%s", argName, arg.Type())
	}
	value := arg.(*object.String)
	return value.Value, nil
}

func unpackNumber(arg object.Object, argName string) (int64, error) {

	if arg.Type() != object.NUMBER_OBJ {
		return -1, fmt.Errorf("argument to `%s` must be a NUMBER, got=%s", argName, arg.Type())
	}
	value := arg.(*object.Number)
	return value.Value.ToInt64(), nil
}

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
		putObj(resultMap, key, object.TRUE)
	} else {
		putObj(resultMap, key, object.FALSE)
	}
}

// ToNative converts a Slug object to its closest matching Go value.
func ToNative(obj object.Object) interface{} {
	switch o := obj.(type) {
	case *object.Number:
		// dec64 can be converted to float64 or int64 depending on needs
		if o.Value.IsFloat() {
			return o.Value.ToFloat64()
		} else {
			return o.Value.ToInt64()
		}
	case *object.Boolean:
		return o.Value
	case *object.String:
		return o.Value
	case *object.Bytes:
		return o.Value
	case *object.List:
		res := make([]interface{}, len(o.Elements))
		for i, el := range o.Elements {
			res[i] = ToNative(el)
		}
		return res
	case *object.Map:
		res := make(map[interface{}]interface{})
		for _, pair := range o.Pairs {
			res[ToNative(pair.Key)] = ToNative(pair.Value)
		}
		return res
	case *object.Nil:
		return nil
	default:
		// For functions, modules, or errors, we return the object itself
		// or its inspection string as a fallback.
		return o
	}
}

func NativeToSlugObject(val interface{}) object.Object {
	switch v := val.(type) {
	case string:
		return &object.String{Value: v}
	case int64:
		return &object.Number{Value: dec64.FromInt64(v)}
	case float64:
		return &object.Number{Value: dec64.FromFloat64(v)}
	case bool:
		if v {
			return object.TRUE
		}
		return object.FALSE
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, item := range v {
			elements[i] = NativeToSlugObject(item)
		}
		return &object.List{Elements: elements}
	default:
		return object.NIL
	}
}
