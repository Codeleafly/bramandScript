package object

import (
	"bytes"
	"strings"
)

type Tuple struct {
	Elements []Object
}

func (t *Tuple) Type() ObjectType { return TUPLE_OBJ }

func (t *Tuple) Inspect() string {
	var out bytes.Buffer
	elements := make([]string, len(t.Elements))
	for i, e := range t.Elements {
		if e == nil {
			elements[i] = "null"
		} else {
			elements[i] = e.Inspect()
		}
	}
	out.WriteString("(")
	out.WriteString(strings.Join(elements, ", "))
	if len(t.Elements) == 1 {
		out.WriteString(",")
	}
	out.WriteString(")")
	return out.String()
}

func NewTuple(elements ...Object) *Tuple {
	return &Tuple{Elements: elements}
}
