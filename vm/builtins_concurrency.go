package vm

import (
	"fmt"
	"time"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("chan", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			capacity := 0
			if len(args) > 0 {
				if intObj, ok := args[0].(*object.Integer); ok {
					if intObj.Value < 0 {
						return &object.Error{Message: "channel capacity cannot be negative"}
					}
					capacity = int(intObj.Value)
				} else {
					return &object.Error{Message: fmt.Sprintf("channel capacity must be INTEGER, got %s", args[0].Type())}
				}
			}
			return &object.Channel{
				Ch:       make(chan object.Object, capacity),
				Capacity: capacity,
			}
		},
	})

	RegisterBuiltin("close", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			ch, ok := args[0].(*object.Channel)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `close` must be CHANNEL, got %s", args[0].Type())}
			}
			err := ch.Close()
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			return Null
		},
	})

	RegisterBuiltin("mutex", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Mutex{}
		},
	})

	RegisterBuiltin("wait_group", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.WaitGroup{}
		},
	})

	RegisterBuiltin("sleep", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			var ms int64
			switch arg := args[0].(type) {
			case *object.Integer:
				ms = arg.Value
			case *object.Float:
				ms = int64(arg.Value)
			default:
				return &object.Error{Message: fmt.Sprintf("argument to `sleep` must be number, got %s", args[0].Type())}
			}
			if ms > 0 {
				time.Sleep(time.Duration(ms) * time.Millisecond)
			}
			return Null
		},
	})
}
