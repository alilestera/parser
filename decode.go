package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/BurntSushi/toml"
)

func Decode(data []byte, ext string, v any) error {
	return newDecoder(data, ext).decode(v)
}

func DecodeFile(path string, v any) error {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = Decode(data, filepath.Ext(path), v)
	if err != nil {
		return fmt.Errorf("decode file %q error: %w", path, err)
	}

	return nil
}

type decoder struct {
	data []byte
	ext  string

	res  map[string]any
	root *node
	meta metadata
	fi   filler
}

func newDecoder(data []byte, ext string) *decoder {
	return &decoder{
		data: data,
		ext:  ext,
	}
}

func (dec *decoder) decode(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer {
		s := "%q"
		if reflect.TypeOf(v) == nil {
			s = "%v"
		}

		return fmt.Errorf("cannot decode to non-pointer "+s, reflect.TypeOf(v))
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot decode to nil value of %q", reflect.TypeOf(v))
	}

	// Check if `v` is a supported type: struct, map or any
	rv = indirect(rv)
	if rv.Kind() != reflect.Struct && rv.Kind() != reflect.Map && !(rv.Kind() == reflect.Interface && rv.NumMethod() == 0) {
		return fmt.Errorf("cannot decode to %q", rv.Type())
	}

	var err error

	if err = dec.unmarshal(); err != nil {
		return fmt.Errorf("unmarshal data error: %w", err)
	}

	// build a tree of untyped nodes
	if err = dec.buildNodes(); err != nil {
		return err
	}

	if err = dec.meta.add(dec.root, rv.Type()); err != nil {
		return fmt.Errorf("add metadata error: %w", err)
	}

	if err = dec.fi.fill(dec.root, rv); err != nil {
		return err
	}

	return nil
}

func (dec *decoder) unmarshal() error {
	switch dec.ext {
	case ".toml", "toml":
		if err := toml.Unmarshal(dec.data, &dec.res); err != nil {
			return err
		}
	case ".yaml", ".yml", ".json", "yaml", "yml", "json":
		if err := yaml.Unmarshal(dec.data, &dec.res); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported file extension %q", dec.ext)
	}

	return nil
}

func (dec *decoder) buildNodes() error {
	dec.root = newNode("root")
	rvMap := reflect.ValueOf(dec.res)

	err := decodeRawMap(dec.root, rvMap)
	if err != nil {
		return err
	}

	return nil
}

func decodeRawMap(nd *node, rvMap reflect.Value) error {
	for _, key := range rvMap.MapKeys() {
		child := newNode(key.String())
		rv := reflect.ValueOf(rvMap.MapIndex(key).Interface())

		err := decodeRawValue(child, rv)
		if err != nil {
			return fmt.Errorf("decode map key %q error: %w", key.String(), err)
		}

		nd.children = append(nd.children, child)
	}

	return nil
}

func decodeRawSlice(nd *node, rvSlice reflect.Value) error {
	for i := range rvSlice.Len() {
		child := newArrayNode(i)
		rv := reflect.ValueOf(rvSlice.Index(i).Interface())

		err := decodeRawValue(child, rv)
		if err != nil {
			return fmt.Errorf("decode slice index [%d] error: %w", i, err)
		}

		nd.children = append(nd.children, child)
	}

	return nil
}

func decodeRawValue(nd *node, rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.String:
		nd.value = rv.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		nd.value = strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		nd.value = strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		nd.value = strings.TrimSuffix(strconv.FormatFloat(rv.Float(), 'f', 6, 64), ".000000")
	case reflect.Bool:
		nd.value = strconv.FormatBool(rv.Bool())
	case reflect.Slice:
		err := decodeRawSlice(nd, rv)
		if err != nil {
			return err
		}
	case reflect.Map:
		err := decodeRawMap(nd, rv)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode value from unsupported type %s", rv.Kind().String())
	}

	return nil
}
