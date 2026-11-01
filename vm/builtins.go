package vm

import (
	"bufio"
	"bramandscript/object"
	"fmt"
	"os"
	"strconv"
	"strings"
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
				case *object.Hash:
					return &object.Integer{Value: int64(len(arg.Pairs))}
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
				fmt.Println()
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
	case "print_raw":
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
	case "input":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) > 0 {
					fmt.Print(args[0].Inspect())
				}
				reader := bufio.NewReader(os.Stdin)
				text, _ := reader.ReadString('\n')
				return &object.String{Value: strings.TrimRight(text, "\r\n")}
			},
		}
	case "int":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					return &object.Integer{Value: int64(arg.Value)}
				case *object.String:
					v, err := strconv.ParseInt(arg.Value, 0, 64)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to integer", arg.Value)}
					}
					return &object.Integer{Value: v}
				case *object.Boolean:
					if arg.Value {
						return &object.Integer{Value: 1}
					}
					return &object.Integer{Value: 0}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to integer", arg.Type())}
				}
			},
		}
	case "float":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Float:
					return arg
				case *object.Integer:
					return &object.Float{Value: float64(arg.Value)}
				case *object.String:
					v, err := strconv.ParseFloat(arg.Value, 64)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to float", arg.Value)}
					}
					return &object.Float{Value: v}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to float", arg.Type())}
				}
			},
		}
	case "str":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: args[0].Inspect()}
			},
		}
	case "bool":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return nativeBoolToBooleanObject(isTruthy(args[0]))
			},
		}
	case "keys":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `keys` must be HASH, got %s", args[0].Type())}
				}
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Key)
				}
				return &object.Array{Elements: elements}
			},
		}
	case "values":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `values` must be HASH, got %s", args[0].Type())}
				}
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Value)
				}
				return &object.Array{Elements: elements}
			},
		}
	case "slice":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `slice` must be ARRAY, got %s", args[0].Type())}
				}
				start, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "second argument to `slice` must be INTEGER"}
				}

				length := int64(len(arr.Elements))
				s := start.Value
				if s < 0 {
					s = 0
				}
				if s > length {
					s = length
				}

				e := length
				if len(args) == 3 {
					end, ok := args[2].(*object.Integer)
					if !ok {
						return &object.Error{Message: "third argument to `slice` must be INTEGER"}
					}
					e = end.Value
					if e > length {
						e = length
					}
					if e < s {
						e = s
					}
				}

				newElements := make([]object.Object, e-s)
				copy(newElements, arr.Elements[s:e])
				return &object.Array{Elements: newElements}
			},
		}
	case "exit":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				code := 0
				if len(args) > 0 {
					if intObj, ok := args[0].(*object.Integer); ok {
						code = int(intObj.Value)
					}
				}
				os.Exit(code)
				return Null
			},
		}
	default:
		return nil
	}
}
