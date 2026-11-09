package vm

import (
	"fmt"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("Ok", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var val object.Object = Null
			if len(args) > 0 {
				val = args[0]
			}
			return object.NewOk(val)
		},
	})

	RegisterBuiltin("Err", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var errVal object.Object = &object.String{Value: "error"}
			if len(args) > 0 {
				errVal = args[0]
			}
			return object.NewErr(errVal)
		},
	})

	RegisterBuiltin("Some", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("Some expects 1 argument, got %d", len(args))}
			}
			return object.NewSome(args[0])
		},
	})

	RegisterBuiltin("None", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return object.NewNone()
		},
	})

	RegisterBuiltin("is_ok", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("is_ok expects 1 argument, got %d", len(args))}
			}
			if res, ok := args[0].(*object.Result); ok {
				return nativeBoolToBooleanObject(res.IsOk)
			}
			return False
		},
	})

	RegisterBuiltin("is_err", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("is_err expects 1 argument, got %d", len(args))}
			}
			if res, ok := args[0].(*object.Result); ok {
				return nativeBoolToBooleanObject(!res.IsOk)
			}
			return False
		},
	})

	RegisterBuiltin("is_some", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("is_some expects 1 argument, got %d", len(args))}
			}
			if opt, ok := args[0].(*object.Option); ok {
				return nativeBoolToBooleanObject(opt.HasValue)
			}
			return False
		},
	})

	RegisterBuiltin("is_none", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("is_none expects 1 argument, got %d", len(args))}
			}
			if opt, ok := args[0].(*object.Option); ok {
				return nativeBoolToBooleanObject(!opt.HasValue)
			}
			return False
		},
	})
}
