package parser

import (
	"reflect"
	"sync"
)

func typeFields(rt reflect.Type) []string {
	var current []reflect.Type
	next := []reflect.Type{rt}

	var count map[string]int
	names := map[string]bool{}

	visited := map[reflect.Type]bool{}

	var fields []string

	for len(next) > 0 {
		current, next = next, current[:0]
		count = map[string]int{}

		for _, t := range current {
			if visited[t] {
				continue
			}
			visited[t] = true

			for i := range t.NumField() {
				sf := t.Field(i)
				if !sf.IsExported() && !sf.Anonymous { // make sure it's exported
					continue
				}

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Pointer {
					ft = ft.Elem()
				}

				if !sf.Anonymous || ft.Kind() != reflect.Struct {
					count[sf.Name]++
					continue
				}

				next = append(next, ft)
				names[sf.Name] = true
			}
		}

		for field, v := range count {
			if !names[field] && v == 1 {
				fields = append(fields, field)
			}

			names[field] = true
		}
	}

	return fields
}

var fieldCache struct {
	sync.RWMutex
	m map[reflect.Type][]string
}

func cachedTypeFields(t reflect.Type) []string {
	fieldCache.RLock()
	f, ok := fieldCache.m[t]
	fieldCache.RUnlock()
	if ok {
		return f
	}

	f = typeFields(t)

	fieldCache.Lock()
	if fieldCache.m == nil {
		fieldCache.m = map[reflect.Type][]string{}
	}
	fieldCache.m[t] = f
	fieldCache.Unlock()

	return f
}
