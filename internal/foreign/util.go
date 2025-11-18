package foreign

import (
	"slug/internal/object"
)

// ToFunctionArgument determines if `obj` is a valid `Function` or within a `FunctionGroup`, matching the provided arguments.
// Returns the found `Function` and true if successful; otherwise returns nil and false.
func ToFunctionArgument(obj object.Object, args []object.Object) (*object.Function, bool) {

	functionGroup, ok := obj.(*object.FunctionGroup)
	if ok {
		toFunction, ok := functionGroup.DispatchToFunction("", args)
		if !ok {
			return nil, false
		}
		foundFunction, ok := toFunction.(*object.Function)
		return foundFunction, ok
	} else {
		foundFunction, ok := obj.(*object.Function)
		return foundFunction, ok
	}
}
