package parser

import (
	"fmt"
	"reflect"
)

type node struct {
	name      string
	fieldName string
	typ       reflect.Type // non-pointer type
	value     string
	children  []*node
}

func newNode(name string) *node {
	return &node{name: name}
}

func newArrayNode(idx int) *node {
	name := fmt.Sprintf("[%d]", idx)
	return newNode(name)
}

func (nd *node) isArrayElem() bool {
	return len(nd.name) > 0 && nd.name[0] == '[' && nd.name[len(nd.name)-1] == ']'
}

func (nd *node) isArray() bool {
	return len(nd.children) > 0 && nd.children[0].isArrayElem()
}
