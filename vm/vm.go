package vm

import (
	"bramandscript/code"
	"bramandscript/compiler"
	"bramandscript/object"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
)

const StackSize = 2048
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
	}
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
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
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) Run() error {
	err := vm.runLoop()
	if err != nil {
		return vm.wrapError(err)
	}
	return nil
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

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv, code.OpMod, code.OpPow:
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
			localIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

			frame := vm.currentFrame()
			vm.stack[frame.basePointer+localIndex] = vm.pop()

		case code.OpGetLocal:
			localIndex := int(ins[ip+1])
			vm.currentFrame().ip += 1

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
	}

	return nil
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	switch {
	case leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ:
		return vm.executeBinaryIntegerOperation(op, left, right)
	case (leftType == object.INTEGER_OBJ || leftType == object.FLOAT_OBJ) &&
		(rightType == object.INTEGER_OBJ || rightType == object.FLOAT_OBJ):
		return vm.executeBinaryFloatOperation(op, left, right)
	case leftType == object.STRING_OBJ || rightType == object.STRING_OBJ:
		if op == code.OpAdd {
			return vm.executeStringConcatenation(left, right)
		}
		return fmt.Errorf("unsupported operator %d for strings", op)
	default:
		return fmt.Errorf("unsupported types for binary operation: %s %d %s",
			leftType, op, rightType)
	}
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	var result int64

	switch op {
	case code.OpAdd:
		result = leftVal + rightVal
	case code.OpSub:
		result = leftVal - rightVal
	case code.OpMul:
		result = leftVal * rightVal
	case code.OpDiv:
		if rightVal == 0 {
			return fmt.Errorf("division by zero")
		}
		result = leftVal / rightVal
	case code.OpMod:
		if rightVal == 0 {
			return fmt.Errorf("modulo by zero")
		}
		result = leftVal % rightVal
	case code.OpPow:
		result = int64(math.Pow(float64(leftVal), float64(rightVal)))
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeBinaryFloatOperation(op code.Opcode, left, right object.Object) error {
	var leftVal, rightVal float64

	if l, ok := left.(*object.Integer); ok {
		leftVal = float64(l.Value)
	} else {
		leftVal = left.(*object.Float).Value
	}

	if r, ok := right.(*object.Integer); ok {
		rightVal = float64(r.Value)
	} else {
		rightVal = right.(*object.Float).Value
	}

	var result float64

	switch op {
	case code.OpAdd:
		result = leftVal + rightVal
	case code.OpSub:
		result = leftVal - rightVal
	case code.OpMul:
		result = leftVal * rightVal
	case code.OpDiv:
		if rightVal == 0 {
			return fmt.Errorf("division by zero")
		}
		result = leftVal / rightVal
	case code.OpPow:
		result = math.Pow(leftVal, rightVal)
	default:
		return fmt.Errorf("unknown float operator: %d", op)
	}

	return vm.push(&object.Float{Value: result})
}

func (vm *VM) executeStringConcatenation(left, right object.Object) error {
	return vm.push(&object.String{Value: left.Inspect() + right.Inspect()})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
		return vm.executeIntegerComparison(op, left, right)
	}
	if (left.Type() == object.INTEGER_OBJ || left.Type() == object.FLOAT_OBJ) &&
		(right.Type() == object.INTEGER_OBJ || right.Type() == object.FLOAT_OBJ) {
		return vm.executeFloatComparison(op, left, right)
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(left.Inspect() == right.Inspect()))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(left.Inspect() != right.Inspect()))
	default:
		return fmt.Errorf("unknown comparison operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeIntegerComparison(op code.Opcode, left, right object.Object) error {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal == rightVal))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal != rightVal))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftVal > rightVal))
	case code.OpGreaterThanEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal >= rightVal))
	default:
		return fmt.Errorf("unknown integer comparison operator: %d", op)
	}
}

func (vm *VM) executeFloatComparison(op code.Opcode, left, right object.Object) error {
	var leftVal, rightVal float64
	if l, ok := left.(*object.Integer); ok {
		leftVal = float64(l.Value)
	} else {
		leftVal = left.(*object.Float).Value
	}
	if r, ok := right.(*object.Integer); ok {
		rightVal = float64(r.Value)
	} else {
		rightVal = right.(*object.Float).Value
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal == rightVal))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal != rightVal))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(leftVal > rightVal))
	case code.OpGreaterThanEqual:
		return vm.push(nativeBoolToBooleanObject(leftVal >= rightVal))
	default:
		return fmt.Errorf("unknown float comparison operator: %d", op)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	switch operand := operand.(type) {
	case *object.Integer:
		return vm.push(&object.Integer{Value: -operand.Value})
	case *object.Float:
		return vm.push(&object.Float{Value: -operand.Value})
	default:
		return fmt.Errorf("unsupported type for negation: %s", operand.Type())
	}
}

func (vm *VM) executeBangOperator() error {
	operand := vm.pop()
	switch operand {
	case True:
		return vm.push(False)
	case False:
		return vm.push(True)
	case Null:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)
	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}
	return &object.Array{Elements: elements}
}

func (vm *VM) buildHash(startIndex, endIndex int) (object.Object, error) {
	hashedPairs := make(map[object.HashKey]object.HashPair)

	for i := startIndex; i < endIndex; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]

		pair := object.HashPair{Key: key, Value: value}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable as hash key: %s", key.Type())
		}

		hashedPairs[hashKey.HashKey()] = pair
	}

	return &object.Hash{Pairs: hashedPairs}, nil
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeStringIndex(left, index)
	case left.Type() == object.HASH_OBJ && index.Type() == object.STRING_OBJ:
		prop := index.(*object.String).Value
		if prop == "length" || prop == "len" {
			return vm.push(&object.Integer{Value: int64(len(left.(*object.Hash).Pairs))})
		}
		return vm.executeHashIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)
	case (left.Type() == object.ARRAY_OBJ || left.Type() == object.STRING_OBJ) && index.Type() == object.STRING_OBJ:
		prop := index.(*object.String).Value
		if prop == "length" || prop == "len" {
			switch l := left.(type) {
			case *object.Array:
				return vm.push(&object.Integer{Value: int64(len(l.Elements))})
			case *object.String:
				return vm.push(&object.Integer{Value: int64(len([]rune(l.Value)))})
			}
		}
		return vm.push(Null)
	case left.Type() == object.MUTEX_OBJ && index.Type() == object.STRING_OBJ:
		mtx := left.(*object.Mutex)
		prop := index.(*object.String).Value
		switch prop {
		case "lock":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					mtx.Lock()
					return Null
				},
			})
		case "unlock":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					mtx.Unlock()
					return Null
				},
			})
		case "try_lock":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return nativeBoolToBooleanObject(mtx.TryLock())
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on mutex", prop)
		}
	case left.Type() == object.WAIT_GROUP_OBJ && index.Type() == object.STRING_OBJ:
		wg := left.(*object.WaitGroup)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					delta := 1
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok {
							delta = int(intObj.Value)
						}
					}
					wg.Add(delta)
					return Null
				},
			})
		case "done":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					wg.Done()
					return Null
				},
			})
		case "wait":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					wg.Wait()
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on wait_group", prop)
		}
	case left.Type() == object.CHANNEL_OBJ && index.Type() == object.STRING_OBJ:
		ch := left.(*object.Channel)
		prop := index.(*object.String).Value
		switch prop {
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					err := ch.Close()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "capacity", "cap":
			return vm.push(&object.Integer{Value: int64(ch.Capacity)})
		default:
			return fmt.Errorf("unknown method '%s' on channel", prop)
		}
	default:
		return fmt.Errorf("index operator not supported: %s[%s]", left.Type(), index.Type())
	}
}

func (vm *VM) executeStringIndex(str, index object.Object) error {
	strObject := str.(*object.String)
	runes := []rune(strObject.Value)
	i := index.(*object.Integer).Value
	max := int64(len(runes) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(&object.String{Value: string(runes[i])})
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if ok {
		return vm.push(pair.Value)
	}

	if intIdx, ok := index.(*object.Integer); ok {
		i := intIdx.Value
		if i >= 0 && i < int64(len(hashObject.Pairs)) {
			curr := int64(0)
			for _, p := range hashObject.Pairs {
				if curr == i {
					return vm.push(p.Key)
				}
				curr++
			}
		}
	}

	return vm.push(Null)
}

func (vm *VM) executeSetIndex(left, index, value object.Object) error {
	switch l := left.(type) {
	case *object.Array:
		idx, ok := index.(*object.Integer)
		if !ok {
			return fmt.Errorf("array index must be an INTEGER, got %s", index.Type())
		}
		i := idx.Value
		if i < 0 || i >= int64(len(l.Elements)) {
			return fmt.Errorf("index out of bounds: %d (array length is %d)", i, len(l.Elements))
		}
		l.Elements[i] = value
		return nil

	case *object.Hash:
		key, ok := index.(object.Hashable)
		if !ok {
			return fmt.Errorf("unusable as hash key: %s", index.Type())
		}
		l.Pairs[key.HashKey()] = object.HashPair{Key: index, Value: value}
		return nil

	default:
		return fmt.Errorf("index assignment not supported for type: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(arrayObject.Elements[i])
}


func (vm *VM) executeCall(numArgs int) error {
	callee := vm.stack[vm.sp-1-numArgs]
	switch callee := callee.(type) {
	case *object.Closure:
		return vm.callClosure(callee, numArgs)
	case *object.Builtin:
		return vm.callBuiltin(callee, numArgs)
	default:
		return fmt.Errorf("calling non-function and non-builtin: %s", callee.Type())
	}
}

func (vm *VM) callClosure(cl *object.Closure, numArgs int) error {
	if numArgs != cl.Fn.NumParameters {
		return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
			cl.Fn.NumParameters, numArgs)
	}

	frame := NewFrame(cl, vm.sp-numArgs)
	vm.pushFrame(frame)

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

	if result != nil {
		vm.push(result)
	} else {
		vm.push(Null)
	}
	return nil
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	case *object.Integer:
		return obj.Value != 0
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
