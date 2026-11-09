package vm

import (
	"fmt"

	"bramandscript/object"
)

func init() {
	spawnActorBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 3 {
				return &object.Error{Message: fmt.Sprintf("actor expects 1 to 3 arguments (handler, [initial_state, name]), got %d", len(args))}
			}

			handler := args[0]
			var initialState object.Object = &object.Null{}
			name := ""

			if len(args) >= 2 && args[1] != nil {
				initialState = args[1]
			}
			if len(args) >= 3 {
				if s, ok := args[2].(*object.String); ok {
					name = s.Value
				}
			}

			runner := func(h object.Object, callArgs []object.Object) (object.Object, error) {
				return ExecuteCallable(h, callArgs...)
			}

			act := object.NewActor(handler, initialState, runner)
			if name != "" {
				object.RegisterActor(name, act)
			}
			return act
		},
	}

	RegisterBuiltin("actor_spawn", spawnActorBuiltin)
	RegisterBuiltin("actor", spawnActorBuiltin)

	RegisterBuiltin("actor_register", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("actor_register expects 2 arguments (name, actor), got %d", len(args))}
			}
			nameObj, ok1 := args[0].(*object.String)
			actObj, ok2 := args[1].(*object.Actor)
			if !ok1 || !ok2 {
				return &object.Error{Message: "arguments to actor_register must be (STRING, ACTOR)"}
			}
			object.RegisterActor(nameObj.Value, actObj)
			return True
		},
	})

	RegisterBuiltin("actor_whereis", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("actor_whereis expects 1 argument (name), got %d", len(args))}
			}
			nameObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to actor_whereis must be STRING, got %s", args[0].Type())}
			}
			act := object.WhereisActor(nameObj.Value)
			if act == nil {
				return Null
			}
			return act
		},
	})

	RegisterBuiltin("actor_unregister", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("actor_unregister expects 1 argument (name), got %d", len(args))}
			}
			nameObj, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to actor_unregister must be STRING, got %s", args[0].Type())}
			}
			object.UnregisterActor(nameObj.Value)
			return True
		},
	})
}
