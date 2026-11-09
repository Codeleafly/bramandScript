package object

import "fmt"

type Option struct {
	HasValue bool
	Value    Object
}

func (o *Option) Type() ObjectType { return OPTION_OBJ }

func (o *Option) Inspect() string {
	if o.HasValue {
		if o.Value == nil {
			return "Some(null)"
		}
		return fmt.Sprintf("Some(%s)", o.Value.Inspect())
	}
	return "None"
}

func NewSome(v Object) *Option {
	if v == nil {
		v = &Null{}
	}
	return &Option{
		HasValue: true,
		Value:    v,
	}
}

func NewNone() *Option {
	return &Option{
		HasValue: false,
		Value:    &Null{},
	}
}
