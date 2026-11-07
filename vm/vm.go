package vm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"sync"
	"time"

	"bramandscript/code"
	"bramandscript/compiler"
	"bramandscript/object"
)

const StackSize = 65536
const GlobalsSize = 65536
const MaxFrames = 1024

const DefaultMaxFrames = 1024
const InitialFrames = 64
const InitialStackSize = 2048
const DefaultMaxStackSize = 1048576

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

	maxFrames    int
	maxStackSize int

	profiler *Profiler

	MaxSteps  uint64
	StepCount uint64
	Timeout   time.Duration
	StartTime time.Time
}

func (vm *VM) SetMaxSteps(steps uint64) {
	vm.MaxSteps = steps
}

func (vm *VM) SetTimeout(d time.Duration) {
	vm.Timeout = d
}

func (vm *VM) SetMaxFrames(max int) {
	vm.maxFrames = max
}

func (vm *VM) SetMaxStackSize(max int) {
	vm.maxStackSize = max
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{
		Instructions: bytecode.Instructions,
		SourceMap:    bytecode.SourceMap,
	}
	mainClosure := &object.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, InitialFrames)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, InitialStackSize),
		sp:    0,

		globalsStore: &GlobalsStore{data: make([]object.Object, 1024)},

		frames:       frames,
		framesIndex:  1,
		maxFrames:    DefaultMaxFrames,
		maxStackSize: DefaultMaxStackSize,
	}
}

func NewWithGlobalsStore(bytecode *compiler.Bytecode, s []object.Object) *VM {
	vm := New(bytecode)
	vm.globalsStore = &GlobalsStore{data: s}
	return vm
}

func (vm *VM) getGlobal(index int) object.Object {
	vm.globalsStore.mu.RLock()
	defer vm.globalsStore.mu.RUnlock()
	if index < 0 || index >= len(vm.globalsStore.data) {
		return Null
	}
	return vm.globalsStore.data[index]
}

func (vm *VM) setGlobal(index int, val object.Object) {
	vm.globalsStore.mu.Lock()
	defer vm.globalsStore.mu.Unlock()
	if index >= len(vm.globalsStore.data) {
		newCap := len(vm.globalsStore.data) * 2
		if newCap <= index {
			newCap = index + 1024
		}
		newData := make([]object.Object, newCap)
		copy(newData, vm.globalsStore.data)
		vm.globalsStore.data = newData
	}
	vm.globalsStore.data[index] = val
}

func (vm *VM) handleException(errObj object.Object) bool {
	for vm.framesIndex > 0 {
		frame := vm.currentFrame()
		if len(frame.handlers) > 0 {
			handler, _ := frame.PopHandler()
			vm.sp = handler.SP
			_ = vm.push(errObj)
			frame.ip = handler.CatchIP - 1
			return true
		}
		if vm.framesIndex > 1 {
			vm.popFrame()
		} else {
			break
		}
	}
	return false
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

	childFrames := make([]*Frame, InitialFrames)
	frame := NewFrame(cl, 0)
	childFrames[0] = frame

	childStack := make([]object.Object, InitialStackSize)
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
		maxFrames:    vm.maxFrames,
		maxStackSize: vm.maxStackSize,
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
		maxFrames := DefaultMaxFrames
		maxStackSize := DefaultMaxStackSize
		if vm != nil {
			constants = vm.constants
			globalsStore = vm.globalsStore
			profiler = vm.profiler
			maxFrames = vm.maxFrames
			maxStackSize = vm.maxStackSize
		}

		childFrames := make([]*Frame, InitialFrames)
		frame := NewFrame(fn, 0)
		childFrames[0] = frame

		childStack := make([]object.Object, InitialStackSize)
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
			maxFrames:    maxFrames,
			maxStackSize: maxStackSize,
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
	limit := vm.maxFrames
	if limit <= 0 {
		limit = DefaultMaxFrames
	}
	if vm.framesIndex >= limit {
		return fmt.Errorf("stack overflow: maximum call stack size of %d frames exceeded", limit)
	}
	if vm.framesIndex >= len(vm.frames) {
		newCap := len(vm.frames) * 2
		if newCap > limit {
			newCap = limit
		}
		newFrames := make([]*Frame, newCap)
		copy(newFrames, vm.frames)
		vm.frames = newFrames
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
	limit := vm.maxStackSize
	if limit <= 0 {
		limit = DefaultMaxStackSize
	}
	if vm.sp >= limit {
		return fmt.Errorf("stack overflow: maximum stack size of %d exceeded", limit)
	}
	if vm.sp >= len(vm.stack) {
		newCap := len(vm.stack) * 2
		if newCap > limit {
			newCap = limit
		}
		newStack := make([]object.Object, newCap)
		copy(newStack, vm.stack)
		vm.stack = newStack
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
	if vm.Timeout > 0 && vm.StartTime.IsZero() {
		vm.StartTime = time.Now()
	}
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
		if vm.MaxSteps > 0 {
			vm.StepCount++
			if vm.StepCount >= vm.MaxSteps {
				return fmt.Errorf("execution step limit exceeded (%d steps)", vm.MaxSteps)
			}
		}
		if vm.Timeout > 0 && (vm.StepCount&1023 == 0) {
			if time.Since(vm.StartTime) > vm.Timeout {
				return fmt.Errorf("execution timeout exceeded (%v)", vm.Timeout)
			}
		}

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
			constIndex := binary.BigEndian.Uint64(ins[ip+1:])
			vm.currentFrame().ip += 8

			err := vm.push(vm.constants[constIndex])
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpPop:
			vm.pop()

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow,
			code.OpBitAnd, code.OpBitOr, code.OpBitXor, code.OpShiftLeft, code.OpShiftRight:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterThanEqual:
			err := vm.executeComparison(op)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpMinus:
			err := vm.executeMinusOperator()
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperator()
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpBitNot:
			err := vm.executeBitNotOperator()
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
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
			pos := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1

		case code.OpJumpNotTruthy:
			pos := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip += 8

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

		case code.OpJumpTruthy:
			pos := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip += 8

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
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpSetGlobal:
			globalIndex := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip += 8
			vm.setGlobal(globalIndex, vm.pop())

		case code.OpGetGlobal:
			globalIndex := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip += 8

			val := vm.getGlobal(globalIndex)
			if val == nil {
				val = Null
			}
			err := vm.push(val)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpPushHandler:
			catchIP := int(binary.BigEndian.Uint64(ins[ip+1:]))
			vm.currentFrame().ip += 8
			vm.currentFrame().PushHandler(catchIP, vm.sp)

		case code.OpPopHandler:
			vm.currentFrame().PopHandler()

		case code.OpThrow:
			val := vm.pop()
			if !vm.handleException(val) {
				return fmt.Errorf("%s", val.Inspect())
			}

		case code.OpSetLocal:
			localIndex := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()
			slot := frame.basePointer + localIndex
			val := vm.pop()
			if cell, ok := vm.stack[slot].(*object.Cell); ok {
				cell.Mu.Lock()
				cell.Value = val
				cell.Mu.Unlock()
			} else {
				vm.stack[slot] = val
			}

		case code.OpGetLocal:
			localIndex := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()
			val := vm.stack[frame.basePointer+localIndex]
			if cell, ok := val.(*object.Cell); ok {
				cell.Mu.RLock()
				val = cell.Value
				cell.Mu.RUnlock()
			}
			err := vm.push(val)
			if err != nil {
				return err
			}

		case code.OpGetLocalCell:
			localIndex := int(binary.BigEndian.Uint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()
			slot := frame.basePointer + localIndex
			val := vm.stack[slot]
			cell, isCell := val.(*object.Cell)
			if !isCell {
				if val == nil {
					val = Null
				}
				cell = &object.Cell{Value: val}
				vm.stack[slot] = cell
			}
			err := vm.push(cell)
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
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpSetIndex:
			value := vm.pop()
			index := vm.pop()
			left := vm.pop()

			err := vm.executeSetIndex(left, index, value)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpCall:
			numArgs := int(ins[ip+1])
			vm.currentFrame().ip += 1

			err := vm.executeCall(numArgs)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpReturnValue:
			returnValue := vm.pop()
			if cell, ok := returnValue.(*object.Cell); ok {
				cell.Mu.RLock()
				returnValue = cell.Value
				cell.Mu.RUnlock()
			}

			frame := vm.popFrame()
			if frame.cl != nil && frame.cl.Instance != nil && len(frame.cl.Fn.FreeNames) > 0 {
				frame.cl.Instance.Mu.Lock()
				for i, name := range frame.cl.Fn.FreeNames {
					if i < len(frame.cl.Free) {
						strObj := &object.String{Value: name}
						hk := strObj.HashKey()
						if _, exists := frame.cl.Instance.Pairs[hk]; exists {
							freeVal := frame.cl.Free[i]
							if cell, ok := freeVal.(*object.Cell); ok {
								cell.Mu.RLock()
								freeVal = cell.Value
								cell.Mu.RUnlock()
							}
							frame.cl.Instance.Pairs[hk] = object.HashPair{Key: strObj, Value: freeVal}
						}
					}
				}
				frame.cl.Instance.Mu.Unlock()
			}
			if vm.framesIndex == 0 {
				return nil
			}
			for i := frame.basePointer; i < vm.sp; i++ {
				vm.stack[i] = nil
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
						hk := strObj.HashKey()
						if _, exists := frame.cl.Instance.Pairs[hk]; exists {
							freeVal := frame.cl.Free[i]
							if cell, ok := freeVal.(*object.Cell); ok {
								cell.Mu.RLock()
								freeVal = cell.Value
								cell.Mu.RUnlock()
							}
							frame.cl.Instance.Pairs[hk] = object.HashPair{Key: strObj, Value: freeVal}
						}
					}
				}
				frame.cl.Instance.Mu.Unlock()
			}
			if vm.framesIndex == 0 {
				return nil
			}
			for i := frame.basePointer; i < vm.sp; i++ {
				vm.stack[i] = nil
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
			constIndex := binary.BigEndian.Uint64(ins[ip+1:])
			numFree := int(ins[ip+9])
			vm.currentFrame().ip += 9

			err := vm.pushClosure(int(constIndex), numFree)
			if err != nil {
				if vm.handleException(&object.Error{Message: err.Error()}) {
					continue
				}
				return err
			}

		case code.OpGetFree:
			freeIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			currentClosure := vm.currentFrame().cl
			val := currentClosure.Free[freeIndex]
			if cell, ok := val.(*object.Cell); ok {
				cell.Mu.RLock()
				val = cell.Value
				cell.Mu.RUnlock()
			}
			err := vm.push(val)
			if err != nil {
				return err
			}

		case code.OpSetFree:
			freeIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			currentClosure := vm.currentFrame().cl
			val := vm.pop()
			if cell, ok := currentClosure.Free[freeIndex].(*object.Cell); ok {
				cell.Mu.Lock()
				cell.Value = val
				cell.Mu.Unlock()
			} else {
				currentClosure.Free[freeIndex] = val
			}

		case code.OpGetFreeCell:
			freeIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			currentClosure := vm.currentFrame().cl
			err := vm.push(currentClosure.Free[freeIndex])
			if err != nil {
				return err
			}

		case code.OpCurrentClosure:
			currentClosure := vm.currentFrame().cl
			err := vm.push(currentClosure)
			if err != nil {
				return err
			}
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
			if r, ok := right.(*object.BigInt); ok {
				return r.Value.Cmp(big.NewInt(l.Value)) == 0
			}
		}
		if l, ok := left.(*object.BigInt); ok {
			if r, ok := right.(*object.Integer); ok {
				return l.Value.Cmp(big.NewInt(r.Value)) == 0
			}
		}
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
		if l, ok := left.(*object.Vector); ok {
			if r, ok := right.(*object.Array); ok {
				if len(l.Elements) != len(r.Elements) {
					return false
				}
				for i := range l.Elements {
					if !objectsEqualDepth(&object.Float{Value: l.Elements[i]}, r.Elements[i], depth+1) {
						return false
					}
				}
				return true
			}
		}
		if l, ok := left.(*object.Array); ok {
			if r, ok := right.(*object.Vector); ok {
				if len(l.Elements) != len(r.Elements) {
					return false
				}
				for i := range l.Elements {
					if !objectsEqualDepth(l.Elements[i], &object.Float{Value: r.Elements[i]}, depth+1) {
						return false
					}
				}
				return true
			}
		}
		return false
	}
	switch l := left.(type) {
	case *object.Vector:
		r := right.(*object.Vector)
		if len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if l.Elements[i] != r.Elements[i] {
				return false
			}
		}
		return true
	case *object.Matrix:
		r := right.(*object.Matrix)
		if l.Rows != r.Rows || l.Cols != r.Cols {
			return false
		}
		for i := 0; i < l.Rows; i++ {
			for j := 0; j < l.Cols; j++ {
				if l.Data[i][j] != r.Data[i][j] {
					return false
				}
			}
		}
		return true
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
