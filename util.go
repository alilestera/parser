package parser

import "reflect"

// indirect returns the value pointed to by a pointer.
//
// Pointers are followed until the value is not a pointer.
// New values are allocated for each nil pointer.
func indirect(rv reflect.Value) reflect.Value {
	if rv.Kind() != reflect.Pointer {
		return rv
	}

	if rv.IsNil() {
		rv.Set(reflect.New(rv.Type().Elem()))
	}

	return indirect(reflect.Indirect(rv))
}

// isEmptyIface returns true if the given type is an empty interface.
func isEmptyIface(rt reflect.Type) bool {
	return rt.Kind() == reflect.Interface && rt.NumMethod() == 0
}
