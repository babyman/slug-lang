package foreign

import (
	"slug/internal/dec64"
	"slug/internal/object"
	"sort"
)

func fnSortWithComparator() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {
			// Check if there are exactly two arguments: the list and the comparator.
			if len(args) != 2 {
				return ctx.NewError("wrong number of arguments. got=%d, want=2", len(args))
			}

			// Ensure the first argument is a LIST_OBJ (the list to sort).
			listObj, ok := args[0].(*object.List)
			if !ok {
				return ctx.NewError("first argument to `sortWithComparator` must be a LIST, got=%s", args[0].Type())
			}

			// Ensure the second argument is a FUNCTION_GROUP_OBJ (the comparator function group).
			compFnGroup, okFg := args[1].(*object.FunctionGroup)
			compFn, okFn := args[1].(*object.Function)
			if !okFg && !okFn {
				return ctx.NewError("second argument to `sortWithComparator` must be a FUNCTION_GROUP, got=%s", args[1].Type())
			}

			var call func(args []object.Object) object.Object
			if okFg {
				call = func(args []object.Object) object.Object {
					return ctx.ApplyFunction("", compFnGroup, args)
				}
			} else {
				call = func(args []object.Object) object.Object {
					return ctx.ApplyFunction("", compFn, args)
				}
			}

			// Sorting logic using the custom comparator.
			elements := listObj.Elements
			sortedElements := make([]object.Object, len(elements))
			copy(sortedElements, elements)

			// Use Go's sort.Slice with a custom comparison using the provided comparator.
			sort.Slice(sortedElements, func(i, j int) bool {
				// Apply the comparator function to the pair of elements.
				args := []object.Object{sortedElements[i], sortedElements[j]}
				callResult := call(args)

				// Ensure the comparator result is a NUMBER_OBJ.
				resultObj, ok := callResult.(*object.Number)
				if !ok {
					return false
				}

				// A negative number indicates a < b, zero indicates a == b, and positive indicates a > b.
				return resultObj.Value.Lt(dec64.ZERO)
			})

			// Return a new sorted List object.
			return &object.List{Elements: sortedElements}
		},
	}
}
