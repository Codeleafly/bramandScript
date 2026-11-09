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

func (t *Tuple) HashKey() HashKey {
	var h uint64 = 14695981039346656037
	for _, el := range t.Elements {
		if hashable, ok := el.(Hashable); ok {
			hk := hashable.HashKey()
			h ^= hk.Value
			h *= 1099511628211
		} else {
			str := ""
			if el != nil {
				str = el.Inspect()
			}
			for i := 0; i < len(str); i++ {
				h ^= uint64(str[i])
				h *= 1099511628211
			}
		}
	}
	return HashKey{Type: t.Type(), Value: h}
}
