package object

import "fmt"

type Result struct {
	IsOk  bool
	Value Object
	Err   Object
}

func (r *Result) Type() ObjectType { return RESULT_OBJ }

func (r *Result) Inspect() string {
	if r.IsOk {
		if r.Value == nil {
			return "Ok()"
		}
		return fmt.Sprintf("Ok(%s)", r.Value.Inspect())
	}
	if r.Err == nil {
		return "Err()"
	}
	return fmt.Sprintf("Err(%s)", r.Err.Inspect())
}

func NewOk(v Object) *Result {
	if v == nil {
		v = &Null{}
	}
	return &Result{
		IsOk:  true,
		Value: v,
		Err:   nil,
	}
}

func NewErr(e Object) *Result {
	if e == nil {
		e = &String{Value: "error"}
	}
	return &Result{
		IsOk:  false,
		Value: nil,
		Err:   e,
	}
}
