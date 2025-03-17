package parser

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type filler struct{}

func (f filler) fill(nd *node, rv reflect.Value) error {
	rv = indirect(rv)
	typ, val := nd.typ, nd.value

	switch typ.Kind() {
	case reflect.Interface:
		rv.Set(reflect.ValueOf(val))
	case reflect.String:
		rv.SetString(val)
	case reflect.Int, reflect.Int64:
		return f.setInt(rv, val, 64)
	case reflect.Int8:
		return f.setInt(rv, val, 8)
	case reflect.Int16:
		return f.setInt(rv, val, 16)
	case reflect.Int32:
		return f.setInt(rv, val, 32)
	case reflect.Uint, reflect.Uint64:
		return f.setUint(rv, val, 64)
	case reflect.Uint8:
		return f.setUint(rv, val, 8)
	case reflect.Uint16:
		return f.setUint(rv, val, 16)
	case reflect.Uint32:
		return f.setUint(rv, val, 32)
	case reflect.Float32:
		return f.setFloat(rv, val, 32)
	case reflect.Float64:
		return f.setFloat(rv, val, 64)
	case reflect.Bool:
		return f.setBool(rv, val)
	case reflect.Struct:
		return f.setStruct(nd, rv)
	case reflect.Map:
		return f.setMap(nd, rv)
	case reflect.Slice:
		return f.setSlice(nd, rv)
	case reflect.Array:
		return f.setSliceArray(nd, rv)
	default:
		panic("unreachable")
	}

	return nil
}

func (f filler) setInt(rv reflect.Value, strNum string, bitSize int) error {
	if _, ok := rv.Interface().(time.Duration); ok {
		dur, err := time.ParseDuration(strNum)
		if err != nil {
			return err
		}
		rv.SetInt(int64(dur))

		return nil
	}

	val, err := strconv.ParseInt(strNum, 10, bitSize)
	if err != nil {
		return err
	}
	rv.SetInt(val)

	return nil
}

func (f filler) setUint(rv reflect.Value, strNum string, bitSize int) error {
	val, err := strconv.ParseUint(strNum, 10, bitSize)
	if err != nil {
		return err
	}
	rv.SetUint(val)

	return nil
}

func (f filler) setFloat(rv reflect.Value, strNum string, bitSize int) error {
	val, err := strconv.ParseFloat(strNum, bitSize)
	if err != nil {
		return err
	}
	rv.SetFloat(val)

	return nil
}

func (f filler) setBool(rv reflect.Value, strBool string) error {
	val, err := strconv.ParseBool(strBool)
	if err != nil {
		return err
	}
	rv.SetBool(val)

	return nil
}

func (f filler) setStruct(nd *node, rv reflect.Value) error {
	for _, child := range nd.children {
		fd := rv.FieldByName(child.fieldName)
		if fd == (reflect.Value{}) { // field not found
			return fmt.Errorf("named field (%s in %q) not found", child.fieldName, rv.Type())
		}

		err := f.fill(child, fd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f filler) setMap(nd *node, rv reflect.Value) error {
	rt := nd.typ
	rm := reflect.MakeMap(rt)

	for _, child := range nd.children {
		elem := reflect.New(rt.Elem()).Elem()
		err := f.fill(child, elem)
		if err != nil {
			return err
		}

		rm.SetMapIndex(reflect.ValueOf(child.name), elem)
	}
	rv.Set(rm)

	return nil
}

func (f filler) setSlice(nd *node, rv reflect.Value) error {
	rs := reflect.MakeSlice(nd.typ, len(nd.children), len(nd.children))
	err := f.setSliceArray(nd, rs)
	if err != nil {
		return err
	}
	rv.Set(rs)

	return nil
}

func (f filler) setSliceArray(nd *node, rv reflect.Value) error {
	rt := nd.typ
	for i, child := range nd.children {
		elem := reflect.New(rt.Elem()).Elem()
		err := f.fill(child, elem)
		if err != nil {
			return err
		}

		rv.Index(i).Set(elem)
	}

	return nil
}
