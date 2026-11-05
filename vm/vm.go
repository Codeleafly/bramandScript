package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"bramandscript/code"
	"bramandscript/compiler"
	"bramandscript/object"
)

const StackSize = 65536
const GlobalsSize = 65536
const MaxFrames = 1024

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}
var Null = &object.Null{}

type RuntimeError struct {
	Line    int
	Col     int
	Message string
}

func (re *RuntimeError) Error() string {
	return re.Message
}

func (vm *VM) wrapError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*RuntimeError); ok {
		return err
	}
	frame := vm.currentFrame()
	line := 1
	if frame != nil && frame.cl != nil && frame.cl.Fn != nil {
		if frame.ip >= 0 && frame.ip < len(frame.cl.Fn.SourceMap) {
			line = frame.cl.Fn.SourceMap[frame.ip]
		}
	}
	return &RuntimeError{Line: line, Col: 1, Message: err.Error()}
}

type GlobalsStore struct {
	mu   sync.RWMutex
	data []object.Object
}

type VM struct {
	constants []object.Object

	stack []object.Object
	sp    int // Always points to the next value. Top of stack is stack[sp-1]

	globalsStore *GlobalsStore

	frames      []*Frame
	framesIndex int

	profiler *Profiler
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{
		Instructions: bytecode.Instructions,
		SourceMap:    bytecode.SourceMap,
	}
	mainClosure := &object.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globalsStore: &GlobalsStore{data: make([]object.Object, GlobalsSize)},

		frames:      frames,
		framesIndex: 1,
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globalsStore = &GlobalsStore{data: s}
	return vm
}

func (vm *VM) getGlobal(index uint16) object.Object {
	vm.globalsStore.mu.RLock()
	defer vm.globalsStore.mu.RUnlock()
	return vm.globalsStore.data[index]
}

func (vm *VM) setGlobal(index uint16, val object.Object) {
	vm.globalsStore.mu.Lock()
	defer vm.globalsStore.mu.Unlock()
	vm.globalsStore.data[index] = val
}

func (vm *VM) newChildVM(cl *object.Closure, args []object.Object) *VM {
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

	childFrames := make([]*Frame, MaxFrames)
	frame := NewFrame(cl, 0)
	childFrames[0] = frame

	childStack := make([]object.Object, StackSize)
	for i, arg := range args {
		childStack[i] = arg
	}

	sp := cl.Fn.NumLocals
	if sp < len(args) {
		sp = len(args)
	}

	return &VM{
		constants:    vm.constants,
		stack:        childStack,
		sp:           sp,
		globalsStore: vm.globalsStore,
		frames:       childFrames,
		framesIndex:  1,
		profiler:     vm.profiler,
	}
}

var (
	activeVM   *VM
	activeVMMu sync.RWMutex
)

func SetActiveVM(vm *VM) {
	activeVMMu.Lock()
	defer activeVMMu.Unlock()
	activeVM = vm
}

func GetActiveVM() *VM {
	activeVMMu.RLock()
	defer activeVMMu.RUnlock()
	return activeVM
}

func ExecuteCallable(callable object.Object, args ...object.Object) (object.Object, error) {
	switch fn := callable.(type) {
	case *object.Builtin:
		return fn.Fn(args...), nil
	case *object.Closure:
		vm := GetActiveVM()
		var constants []object.Object
		var globalsStore *GlobalsStore
		var profiler *Profiler
		if vm != nil {
			constants = vm.constants
			globalsStore = vm.globalsStore
			profiler = vm.profiler
		}

		childFrames := make([]*Frame, MaxFrames)
		frame := NewFrame(fn, 0)
		childFrames[0] = frame

		childStack := make([]object.Object, StackSize)
		for i, arg := range args {
			childStack[i] = arg
		}

		sp := fn.Fn.NumLocals
		if sp < len(args) {
			sp = len(args)
		}

		childVM := &VM{
			constants:    constants,
			stack:        childStack,
			sp:           sp,
			globalsStore: globalsStore,
			frames:       childFrames,
			framesIndex:  1,
			profiler:     profiler,
		}
		oldVM := vm
		SetActiveVM(childVM)
		defer SetActiveVM(oldVM)

		err := childVM.Run()
		if err != nil {
			return nil, err
		}
		return childVM.LastPoppedStackElem(), nil
	default:
		return nil, fmt.Errorf("object is not callable: %s", callable.Type())
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) error {
	if vm.framesIndex >= MaxFrames {
		return fmt.Errorf("stack overflow: maximum call stack size of %d frames exceeded", MaxFrames)
	}
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
	return nil
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	if vm.sp <= 0 {
		return Null
	}
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) Run() error {
	SetActiveVM(vm)
	err := vm.runLoop()
	if err != nil {
		return vm.wrapError(err)
	}
	return nil
}

func (vm *VM) RunProfiled() (*ProfileReport, error) {
	vm.profiler = NewProfiler()
	vm.profiler.StartTime = time.Now()
	err := vm.Run()
	vm.profiler.EndTime = time.Now()
	report := vm.profiler.GenerateReport()
	return report, err
}

func (vm *VM) runLoop() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.framesIndex > 0 && vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		var t0 time.Time
		if vm.profiler != nil {
			t0 = time.Now()
		}

		switch op {
		case code.OpConstant:
			constIndex := binary.BigEndian.Uint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.pop()

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow,
			code.OpBitAnd, code.OpBitOr, code.OpBitXor, code.OpShiftLeft, code.OpShiftRight:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterThanEqual:
			err := vm.executeComparison(op)
			if err != nil {
				return err
			}

		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				return err
			}

		case code.OpBitNot:
			err := vm.executeBitNotOperator()
			if err != nil {
				return err
			}

		case code.OpAnd:
			right := vm.pop()
			left := vm.pop()
			vm.push(nativeBoolToBooleanObject(isTruthy(left) && isTruthy(right)))

		case code.OpOr:
			right := vm.pop()
			left := vm.pop()
			vm.push(nativeBoolToBooleanObject(isTruthy(left) || isTruthy(right)))

		case code.OpJump:
			pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1

		case code.OpJumpNotTruthy:
			pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

		case code.OpJumpTruthy:
			pos := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

		case code.OpDup:
			if vm.sp == 0 {
				return fmt.Errorf("stack underflow on dup")
			}
			err := vm.push(vm.stack[vm.sp-1])
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := binary.BigEndian.Uint16(ins[ip+1:])
			vm.currentFrame().ip += 2
			vm.setGlobal(globalIndex, vm.pop())

		case code.OpGetGlobal:
			globalIndex := binary.BigEndian.Uint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			val := vm.getGlobal(globalIndex)
			if val == nil {
				val = Null
			}
			err := vm.push(val)
			if err != nil {
				return err
			}

		case code.OpSetLocal:
			localIndex := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()
			vm.stack[frame.basePointer+localIndex] = vm.pop()

		case code.OpGetLocal:
			localIndex := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()
			err := vm.push(vm.stack[frame.basePointer+localIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			numElements := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			array := vm.buildArray(vm.sp-numElements, vm.sp)
			vm.sp = vm.sp - numElements

			err := vm.push(array)
			if err != nil {
				return err
			}

		case code.OpHash:
			numElements := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-numElements, vm.sp)
			if err != nil {
				return err
			}
			vm.sp = vm.sp - numElements

			err = vm.push(hash)
			if err != nil {
				return err
			}

		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()

			err := vm.executeIndexExpression(left, index)
			if err != nil {
				return err
			}

		case code.OpSetIndex:
			value := vm.pop()
			index := vm.pop()
			left := vm.pop()

			err := vm.executeSetIndex(left, index, value)
			if err != nil {
				return err
			}

		case code.OpCall:
			numArgs := int(ins[ip+1])
			vm.currentFrame().ip += 1

			err := vm.executeCall(numArgs)
			if err != nil {
				return err
			}

		case code.OpReturnValue:
			returnValue := vm.pop()

			frame := vm.popFrame()
			if frame.cl != nil && frame.cl.Instance != nil && len(frame.cl.Fn.FreeNames) > 0 {
				frame.cl.Instance.Mu.Lock()
				for i, name := range frame.cl.Fn.FreeNames {
					if i < len(frame.cl.Free) {
						strObj := &object.String{Value: name}
						frame.cl.Instance.Pairs[strObj.HashKey()] = object.HashPair{Key: strObj, Value: frame.cl.Free[i]}
					}
				}
				frame.cl.Instance.Mu.Unlock()
			}
			if vm.framesIndex == 0 {
				return nil
			}
			vm.sp = frame.basePointer - 1

			err := vm.push(returnValue)
			if err != nil {
				return err
			}

		case code.OpReturn:
			frame := vm.popFrame()
			if frame.cl != nil && frame.cl.Instance != nil && len(frame.cl.Fn.FreeNames) > 0 {
				frame.cl.Instance.Mu.Lock()
				for i, name := range frame.cl.Fn.FreeNames {
					if i < len(frame.cl.Free) {
						strObj := &object.String{Value: name}
						frame.cl.Instance.Pairs[strObj.HashKey()] = object.HashPair{Key: strObj, Value: frame.cl.Free[i]}
					}
				}
				frame.cl.Instance.Mu.Unlock()
			}
			if vm.framesIndex == 0 {
				return nil
			}
			vm.sp = frame.basePointer - 1

			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpSpawn:
			numArgs := int(ins[ip+1])
			vm.currentFrame().ip += 1

			callee := vm.stack[vm.sp-1-numArgs]
			args := make([]object.Object, numArgs)
			copy(args, vm.stack[vm.sp-numArgs:vm.sp])

			vm.sp = vm.sp - 1 - numArgs

			err := vm.push(Null)
			if err != nil {
				return err
			}

			switch fn := callee.(type) {
			case *object.Closure:
				if numArgs != fn.Fn.NumParameters {
					return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
						fn.Fn.NumParameters, numArgs)
				}
				go func(cl *object.Closure, taskArgs []object.Object) {
					childVM := vm.newChildVM(cl, taskArgs)
					_ = childVM.Run()
				}(fn, args)

			case *object.Builtin:
				go func(b *object.Builtin, taskArgs []object.Object) {
					b.Fn(taskArgs...)
				}(fn, args)

			default:
				return fmt.Errorf("calling non-function and non-builtin: %s", callee.Type())
			}

		case code.OpSend:
			val := vm.pop()
			chObj := vm.pop()

			channel, ok := chObj.(*object.Channel)
			if !ok {
				return fmt.Errorf("send operator <- requires CHANNEL, got %s", chObj.Type())
			}
			err := channel.Send(val)
			if err != nil {
				return err
			}
			err = vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpReceive:
			chObj := vm.pop()

			channel, ok := chObj.(*object.Channel)
			if !ok {
				return fmt.Errorf("receive operator <- requires CHANNEL, got %s", chObj.Type())
			}
			val, _ := channel.Recv()
			err := vm.push(val)
			if err != nil {
				return err
			}

		case code.OpGetBuiltin:
			builtinIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			definition := compiler.Builtins[builtinIndex]
			builtin := getBuiltinByName(definition.Name)
			err := vm.push(builtin)
			if err != nil {
				return err
			}

		case code.OpClosure:
			constIndex := binary.BigEndian.Uint16(ins[ip+1:])
			numFree := int(ins[ip+3])
			vm.currentFrame().ip += 3

			err := vm.pushClosure(int(constIndex), numFree)
			if err != nil {
				return err
			}

		case code.OpGetFree:
			freeIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			currentClosure := vm.currentFrame().cl
			err := vm.push(currentClosure.Free[freeIndex])
			if err != nil {
				return err
			}

		case code.OpSetFree:
			freeIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			currentClosure := vm.currentFrame().cl
			currentClosure.Free[freeIndex] = vm.pop()
		}

		if vm.profiler != nil {
			vm.profiler.RecordOpcode(op, time.Since(t0))
		}
	}

	return nil
}

func ObjectsEqual(left, right object.Object) bool {
	return objectsEqualDepth(left, right, 0)
}

func objectsEqualDepth(left, right object.Object, depth int) bool {
	if depth > 64 {
		return false
	}
	if left == right {
		return true
	}
	if left == nil || right == nil {
		return left == right
	}
	if left.Type() != right.Type() {
		if l, ok := left.(*object.Integer); ok {
			if r, ok := right.(*object.Float); ok {
				return float64(l.Value) == r.Value
			}
		}
		if l, ok := left.(*object.Float); ok {
			if r, ok := right.(*object.Integer); ok {
				return l.Value == float64(r.Value)
			}
		}
		return false
	}
	switch l := left.(type) {
	case *object.Integer:
		return l.Value == right.(*object.Integer).Value
	case *object.Float:
		return l.Value == right.(*object.Float).Value
	case *object.Boolean:
		return l.Value == right.(*object.Boolean).Value
	case *object.String:
		return l.Value == right.(*object.String).Value
	case *object.Null:
		return true
	case *object.BigInt:
		return l.Value.Cmp(right.(*object.BigInt).Value) == 0
	case *object.BigFloat:
		return l.Value.Cmp(right.(*object.BigFloat).Value) == 0
	case *object.Bytes:
		r := right.(*object.Bytes)
		if l == r {
			return true
		}
		l.Mu.RLock()
		r.Mu.RLock()
		defer l.Mu.RUnlock()
		defer r.Mu.RUnlock()
		return bytes.Equal(l.Value, r.Value)
	case *object.Array:
		r := right.(*object.Array)
		if l == r {
			return true
		}
		if len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if !objectsEqualDepth(l.Elements[i], r.Elements[i], depth+1) {
				return false
			}
		}
		return true
	case *object.Hash:
		r := right.(*object.Hash)
		if l == r {
			return true
		}
		l.Mu.RLock()
		r.Mu.RLock()
		defer l.Mu.RUnlock()
		defer r.Mu.RUnlock()
		if len(l.Pairs) != len(r.Pairs) {
			return false
		}
		for k, pair := range l.Pairs {
			rPair, ok := r.Pairs[k]
			if !ok {
				return false
			}
			if !objectsEqualDepth(pair.Value, rPair.Value, depth+1) {
				return false
			}
		}
		return true
	default:
		return left.Inspect() == right.Inspect()
	}
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	case *object.Integer:
		return obj.Value != 0
	case *object.Float:
		return obj.Value != 0.0
	default:
		return true
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return True
	}
	return False
}
