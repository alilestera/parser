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

// Decode decodes the contents of data into a pointer v.
// ext is the file extension for the format of data,
// which must be one of the following: toml, yaml or json.
// The last target of v pointed to must be a struct, map or empty interface.
// If no error occurs, v will be filled with the decoded data.
//
// The process can be roughly divided into four steps:
// 1. Call function from different dependency packages to decode the data
// into a map[string]any value.
// 2. Build a tree of nodes from the top down, these nodes contain only value.
// 3. Assign metadata such as actual type to these nodes depending on
// the type of target.
// 4. Fill value to v depending on the metadata and value from nodes.
//
// During the process, if occurs an empty interface that actual type
// cannot be determined, then treated it as a string.
func Decode(data []byte, ext string, v any) error {
	return newDecoder(data, ext).decode(v)
}

// DecodeFile decodes the contents of given file into a pointer v.
//
// See [Decode] function for a description of decoding process.
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
	ext  string // file extension

	mapping map[string]any
	root    *node
	metadata
	filler
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

		return fmt.Errorf("expected a pointer type but got "+s, reflect.TypeOf(v))
	}
	if rv.IsNil() {
		return fmt.Errorf("cannot decode to nil value of %q", reflect.TypeOf(v))
	}

	// Check if `v` is a supported type: struct, map or empty interface.
	rv = indirect(rv)
	if rv.Kind() != reflect.Struct && rv.Kind() != reflect.Map && !(rv.Kind() == reflect.Interface && rv.NumMethod() == 0) {
		return fmt.Errorf("cannot decode to unsupported type %q. Supported types are struct, map, or empty interface", rv.Type())
	}

	var err error

	if err = dec.unmarshal(); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}
	// Build a tree of nodes from the top down, these nodes contain only value.
	if err = dec.buildNodes(); err != nil {
		return fmt.Errorf("failed to build nodes: %w", err)
	}

	// Start at the root, assign metadata to nodes.
	if err = dec.metadata.add(dec.root, rv.Type()); err != nil {
		return fmt.Errorf("failed to add metadata: %w", err)
	}
	if err = dec.filler.fill(dec.root, rv); err != nil {
		return fmt.Errorf("failed to fill value: %w", err)
	}

	return nil
}

func (dec *decoder) unmarshal() error {
	switch dec.ext {
	case ".toml", "toml":
		if err := toml.Unmarshal(dec.data, &dec.mapping); err != nil {
			return err
		}
	case ".yaml", "yaml", ".yml", "yml", ".json", "json":
		if err := yaml.Unmarshal(dec.data, &dec.mapping); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported extension %q", dec.ext)
	}

	return nil
}

func (dec *decoder) buildNodes() error {
	dec.root = newNode("root")
	rvMap := reflect.ValueOf(dec.mapping)

	err := decodeRawMap(dec.root, rvMap)
	if err != nil {
		return err
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
		if err := decodeRawSlice(nd, rv); err != nil {
			return err
		}
	case reflect.Map:
		if err := decodeRawMap(nd, rv); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot decode value from unsupported type %s", rv.Kind().String())
	}

	return nil
}

func decodeRawMap(nd *node, rvMap reflect.Value) error {
	for _, key := range rvMap.MapKeys() {
		child := newNode(key.String())
		rv := reflect.ValueOf(rvMap.MapIndex(key).Interface())

		err := decodeRawValue(child, rv)
		if err != nil {
			return fmt.Errorf("decode map key %q: %w", key.String(), err)
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
			return fmt.Errorf("decode slice index [%d]: %w", i, err)
		}

		nd.children = append(nd.children, child)
	}

	return nil
}
