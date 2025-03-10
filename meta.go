package parser

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	zeroSliceType = reflect.TypeOf([]any{})
	zeroMapType   = reflect.TypeOf(map[string]any{})
)

type metadata struct{}

func (md metadata) add(nd *node, rt reflect.Type) error {
	for rt.Kind() == reflect.Pointer {
		// Follow pointers
		rt = rt.Elem()
	}

	k := rt.Kind()
	nd.kind = k

	if !isSupportedKind(k) {
		return fmt.Errorf("unsupported type %q", rt)
	}

	switch k {
	case reflect.Struct:
		return md.addStruct(nd, rt)
	case reflect.Slice:
		return md.addSliceArray(nd, rt)
	case reflect.Array:
		return md.addArray(nd, rt)
	case reflect.Map:
		return md.addMap(nd, rt)
	case reflect.Interface:
		// Only empty interfaces are supported
		if rt.NumMethod() > 0 {
			return fmt.Errorf("unsupported type %q", rt)
		}

		return md.addAnything(nd)
	default:
		return nil
	}
}

func (md metadata) addStruct(nd *node, rt reflect.Type) error {
	fields := cachedTypeFields(rt)

	for _, child := range nd.children {
		for _, field := range fields {
			if strings.EqualFold(field, child.name) {
				child.fieldName = field
			}
		}

		if child.fieldName != "" {
			structField, _ := rt.FieldByName(child.fieldName)
			err := md.add(child, structField.Type)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (md metadata) addSliceArray(nd *node, rt reflect.Type) error {
	for _, child := range nd.children {
		err := md.add(child, rt.Elem())
		if err != nil {
			return err
		}
	}

	return nil
}

func (md metadata) addArray(nd *node, rt reflect.Type) error {
	if rt.Len() != len(nd.children) {
		return fmt.Errorf("expected array length %d, but got %d", rt.Len(), len(nd.children))
	}

	return md.addSliceArray(nd, rt)
}

func (md metadata) addMap(nd *node, rt reflect.Type) error {
	keyKind := rt.Key().Kind()
	if keyKind != reflect.String && !isEmptyIface(rt.Key()) {
		return fmt.Errorf("map with non-string key type (%s in %q) is not supported", keyKind, rt)
	}

	for _, child := range nd.children {
		err := md.add(child, rt.Elem())
		if err != nil {
			return err
		}
	}

	return nil
}

func (md metadata) addAnything(nd *node) error {
	if len(nd.children) == 0 {
		// case unique value, treats it as type `string`
		// but do not change the kind
		return nil
	}

	if nd.isArray() {
		// case array value, treats it as type `[]any`
		return md.addSliceArray(nd, zeroSliceType)
	}

	// case map value, treats it as type `map[string]any`
	return md.addMap(nd, zeroMapType)
}

func isSupportedKind(kind reflect.Kind) bool {
	switch kind {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String,
		reflect.Struct,
		reflect.Slice,
		reflect.Array,
		reflect.Map,
		reflect.Interface:
		return true
	default:
		return false
	}
}
