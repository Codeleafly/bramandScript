package vm

import (
	"fmt"

	"bramandscript/object"
)

func (vm *VM) executeCall(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]
	if callee == nil {
		return fmt.Errorf("calling null or undefined value")
	}
	switch callee := callee.(type) {
	case *object.Closure:
		return vm.callClosure(callee, numArgs)
	case *object.Builtin:
		return vm.callBuiltin(callee, numArgs)
	case *object.Hash:
		callee.Mu.RLock()
		callPair, hasCall := callee.Pairs[(&object.String{Value: "__call__"}).HashKey()]
		callee.Mu.RUnlock()
		if hasCall {
			if cl, isCl := callPair.Value.(*object.Closure); isCl {
				cl.Instance = callee
				vm.stack[vm.sp-1-numArgs] = cl
				return vm.callClosure(cl, numArgs)
			}
			if b, isB := callPair.Value.(*object.Builtin); isB {
				vm.stack[vm.sp-1-numArgs] = b
				return vm.callBuiltin(b, numArgs)
			}
		}
		return fmt.Errorf("calling non-function and non-builtin: %s", callee.Type())
	default:
		return fmt.Errorf("calling non-function and non-builtin: %s", callee.Type())
	}
}

func (vm *VM) callClosure(cl *object.Closure, numArgs int) error {
	if numArgs != cl.Fn.NumParameters {
		return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
			cl.Fn.NumParameters, numArgs)
	}

	if cl.Instance == nil && vm.framesIndex > 0 {
		parentFrame := vm.currentFrame()
		if parentFrame != nil && parentFrame.cl != nil && parentFrame.cl.Instance != nil {
			cl.Instance = parentFrame.cl.Instance
		}
	}

	if cl.Instance != nil && len(cl.Fn.FreeNames) > 0 {
		cl.Instance.Mu.RLock()
		for i, name := range cl.Fn.FreeNames {
			if i < len(cl.Free) {
				k := (&object.String{Value: name}).HashKey()
				if p, ok := cl.Instance.Pairs[k]; ok {
					cl.Free[i] = p.Value
				}
			}
		}
		cl.Instance.Mu.RUnlock()
	}

	frame := NewFrame(cl, vm.sp-numArgs)
	if err := vm.pushFrame(frame); err != nil {
		return err
	}

	vm.sp = frame.basePointer + cl.Fn.NumLocals
	return nil
}

func (vm *VM) pushClosure(constIndex, numFree int) error {
	constant := vm.constants[constIndex]
	function, ok := constant.(*object.CompiledFunction)
	if !ok {
		return fmt.Errorf("not a function: %+v", constant)
	}

	free := make([]object.Object, numFree)
	for i := 0; i < numFree; i++ {
		free[i] = vm.stack[vm.sp-numFree+i]
	}
	vm.sp = vm.sp - numFree

	closure := &object.Closure{Fn: function, Free: free}
	return vm.push(closure)
}

func (vm *VM) callBuiltin(builtin *object.Builtin, numArgs int) error {
	args := vm.stack[vm.sp-numArgs : vm.sp]
	result := builtin.Fn(args...)
	vm.sp = vm.sp - numArgs - 1

	if errObj, ok := result.(*object.Error); ok && errObj.Fatal {
		return fmt.Errorf("%s", errObj.Message)
	}

	if result != nil {
		vm.push(result)
	} else {
		vm.push(Null)
	}
	return nil
}
