package vm

import (
	"bramandscript/object"
	"fmt"
	"time"
)

func getBuiltinByName(name string) *object.Builtin {
	switch name {
	case "len":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Array:
					return &object.Integer{Value: int64(len(arg.Elements))}
				case *object.String:
					return &object.Integer{Value: int64(len(arg.Value))}
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Type())}
				}
			},
		}
	case "print":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(arg.Inspect())
				}
				return Null
			},
		}
	case "println":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(arg.Inspect())
				}
				fmt.Println()
				return Null
			},
		}
	case "type":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: string(args[0].Type())}
			},
		}
	case "push":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				if args[0].Type() != object.ARRAY_OBJ {
					return &object.Error{Message: fmt.Sprintf("first argument to `push` must be ARRAY, got %s", args[0].Type())}
				}
				arr := args[0].(*object.Array)
				length := len(arr.Elements)
				newElements := make([]object.Object, length+1)
				copy(newElements, arr.Elements)
				newElements[length] = args[1]
				return &object.Array{Elements: newElements}
			},
		}
	case "now":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Integer{Value: time.Now().Unix()}
			},
		}
	default:
		return nil
	}
}
