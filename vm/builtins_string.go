package vm

import (
	"fmt"
	"strings"

	"bramandscript/object"
)

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"split": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2", len(args))}
				}
				strObj, ok1 := args[0].(*object.String)
				sepObj, ok2 := args[1].(*object.String)
				if !ok1 || !ok2 {
					return &object.Error{Message: "arguments to `split` must be (STRING, STRING)"}
				}
				parts := strings.Split(strObj.Value, sepObj.Value)
				elems := make([]object.Object, len(parts))
				for i, p := range parts {
					elems[i] = &object.String{Value: p}
				}
				return &object.Array{Elements: elems}
			},
		},
		"join": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2", len(args))}
				}
				arr, ok1 := args[0].(*object.Array)
				sepObj, ok2 := args[1].(*object.String)
				if !ok1 || !ok2 {
					return &object.Error{Message: "arguments to `join` must be (ARRAY, STRING)"}
				}
				arr.Mu.RLock()
				defer arr.Mu.RUnlock()
				parts := make([]string, len(arr.Elements))
				for i, el := range arr.Elements {
					if s, ok := el.(*object.String); ok {
						parts[i] = s.Value
					} else {
						parts[i] = el.Inspect()
					}
				}
				return &object.String{Value: strings.Join(parts, sepObj.Value)}
			},
		},
		"trim": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: "first argument to `trim` must be STRING"}
				}
				if len(args) == 2 {
					cutset, ok := args[1].(*object.String)
					if !ok {
						return &object.Error{Message: "second argument to `trim` must be STRING"}
					}
					return &object.String{Value: strings.Trim(strObj.Value, cutset.Value)}
				}
				return &object.String{Value: strings.TrimSpace(strObj.Value)}
			},
		},
		"to_lower": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: "to_lower requires 1 argument"}
				}
				s := args[0].Inspect()
				if strObj, ok := args[0].(*object.String); ok {
					s = strObj.Value
				}
				return &object.String{Value: strings.ToLower(s)}
			},
		},
		"to_upper": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: "to_upper requires 1 argument"}
				}
				s := args[0].Inspect()
				if strObj, ok := args[0].(*object.String); ok {
					s = strObj.Value
				}
				return &object.String{Value: strings.ToUpper(s)}
			},
		},
		"replace": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 3 {
					return &object.Error{Message: "replace requires 3 arguments: (str, old, new)"}
				}
				strObj, ok1 := args[0].(*object.String)
				oldObj, ok2 := args[1].(*object.String)
				newObj, ok3 := args[2].(*object.String)
				if !ok1 || !ok2 || !ok3 {
					return &object.Error{Message: "all arguments to `replace` must be strings"}
				}
				return &object.String{Value: strings.ReplaceAll(strObj.Value, oldObj.Value, newObj.Value)}
			},
		},
	})
}
