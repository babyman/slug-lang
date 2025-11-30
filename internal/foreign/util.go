package foreign

import (
	"errors"
	"slug/internal/object"
)

// ToFunctionArgument determines if `obj` is a valid `Function` or within a `FunctionGroup`, matching the provided arguments.
// Returns the found `Function` and true if successful; otherwise returns nil and false.
func ToFunctionArgument(obj object.Object, args []object.Object) (*object.Function, error) {

	functionGroup, ok := obj.(*object.FunctionGroup)
	if ok {
		toFunction, err := functionGroup.DispatchToFunction("", args)
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
