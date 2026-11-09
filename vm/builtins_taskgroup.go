package vm

import (
	"fmt"
	"sync"

	"bramandscript/object"
)

type taskGroupInstance struct {
	mu        sync.Mutex
	tasks     []*object.Future
	cancelled bool
	reason    string
}

func newTaskGroup() object.Object {
	tg := &taskGroupInstance{
		tasks: make([]*object.Future, 0),
	}

	spawnFn := func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "tg.spawn expects a callable argument"}
		}
		callable := args[0]
		fut := object.NewFuture()

		tg.mu.Lock()
		if tg.cancelled {
			fut.Reject(fmt.Errorf("%s", tg.reason))
			tg.mu.Unlock()
			return fut
		}
		tg.tasks = append(tg.tasks, fut)
		tg.mu.Unlock()

		go func() {
			tg.mu.Lock()
			if tg.cancelled {
				fut.Reject(fmt.Errorf("%s", tg.reason))
				tg.mu.Unlock()
				return
			}
			tg.mu.Unlock()

			var callArgs []object.Object
			if len(args) > 1 {
				callArgs = args[1:]
			}
			res, err := ExecuteCallable(callable, callArgs...)
			if err != nil {
				tg.mu.Lock()
				if !tg.cancelled {
					tg.cancelled = true
					tg.reason = fmt.Sprintf("cascade cancellation: %v", err)
					for _, other := range tg.tasks {
						if other != fut {
							other.Cancel(tg.reason)
						}
					}
				}
				tg.mu.Unlock()
				fut.Reject(err)
			} else {
				fut.Resolve(res)
			}
		}()

		return fut
	}

	waitAllFn := func(args ...object.Object) object.Object {
		tg.mu.Lock()
		taskList := make([]*object.Future, len(tg.tasks))
		copy(taskList, tg.tasks)
		tg.mu.Unlock()

		results := make([]object.Object, 0, len(taskList))
		for _, fut := range taskList {
			val, err := fut.Await()
			if err != nil {
				return object.NewErr(&object.String{Value: err.Error()})
			}
			if val == nil {
				val = Null
			}
			results = append(results, val)
		}
		return object.NewOk(&object.Array{Elements: results})
	}

	type waitAnyResult struct {
		val object.Object
		err error
	}

	waitAnyFn := func(args ...object.Object) object.Object {
		tg.mu.Lock()
		taskList := make([]*object.Future, len(tg.tasks))
		copy(taskList, tg.tasks)
		tg.mu.Unlock()

		if len(taskList) == 0 {
			return object.NewOk(Null)
		}

		resCh := make(chan waitAnyResult, len(taskList))
		for _, f := range taskList {
			go func(fut *object.Future) {
				v, e := fut.Await()
				resCh <- waitAnyResult{val: v, err: e}
			}(f)
		}

		first := <-resCh
		if first.err != nil {
			return object.NewErr(&object.String{Value: first.err.Error()})
		}
		if first.val == nil {
			first.val = Null
		}
		return object.NewOk(first.val)
	}

	cancelFn := func(args ...object.Object) object.Object {
		reason := "task group cancelled"
		if len(args) > 0 {
			reason = args[0].Inspect()
		}
		tg.mu.Lock()
		tg.cancelled = true
		tg.reason = reason
		for _, fut := range tg.tasks {
			fut.Cancel(reason)
		}
		tg.mu.Unlock()
		return True
	}

	isCancelledFn := func(args ...object.Object) object.Object {
		tg.mu.Lock()
		defer tg.mu.Unlock()
		if tg.cancelled {
			return True
		}
		return False
	}

	countFn := func(args ...object.Object) object.Object {
		tg.mu.Lock()
		defer tg.mu.Unlock()
		return &object.Integer{Value: int64(len(tg.tasks))}
	}

	pairs := make(map[object.HashKey]object.HashPair)
	insertMethod := func(name string, fn func(args ...object.Object) object.Object) {
		key := &object.String{Value: name}
		val := &object.Builtin{Fn: fn}
		pairs[key.HashKey()] = object.HashPair{Key: key, Value: val}
	}

	insertMethod("spawn", spawnFn)
	insertMethod("wait_all", waitAllFn)
	insertMethod("wait_any", waitAnyFn)
	insertMethod("cancel", cancelFn)
	insertMethod("is_cancelled", isCancelledFn)
	insertMethod("count", countFn)

	return &object.Hash{Pairs: pairs}
}

func init() {
	RegisterBuiltin("task_group", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return newTaskGroup()
		},
	})
}
