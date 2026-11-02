package vm

import (
	"bramandscript/code"
	"bramandscript/compiler"
	"bramandscript/object"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"strings"
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
		if vm != nil {
			constants = vm.constants
			globalsStore = vm.globalsStore
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
	SetActiveVM(vm)
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
	case leftType == object.BIG_INT_OBJ || rightType == object.BIG_INT_OBJ:
		return vm.executeBinaryBigIntOperation(op, left, right)
	case leftType == object.BIG_FLOAT_OBJ || rightType == object.BIG_FLOAT_OBJ:
		return vm.executeBinaryBigFloatOperation(op, left, right)
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

func toBigInt(obj object.Object) (*big.Int, bool) {
	switch o := obj.(type) {
	case *object.BigInt:
		return new(big.Int).Set(o.Value), true
	case *object.Integer:
		return big.NewInt(o.Value), true
	case *object.String:
		bi := new(big.Int)
		if _, ok := bi.SetString(o.Value, 0); ok {
			return bi, true
		}
		return nil, false
	default:
		return nil, false
	}
}

func toBigFloat(obj object.Object) (*big.Float, bool) {
	switch o := obj.(type) {
	case *object.BigFloat:
		return new(big.Float).Set(o.Value), true
	case *object.BigInt:
		return new(big.Float).SetInt(o.Value), true
	case *object.Integer:
		return new(big.Float).SetInt64(o.Value), true
	case *object.Float:
		return new(big.Float).SetFloat64(o.Value), true
	case *object.String:
		bf := new(big.Float)
		if _, ok := bf.SetString(o.Value); ok {
			return bf, true
		}
		return nil, false
	default:
		return nil, false
	}
}

func (vm *VM) executeBinaryBigIntOperation(op code.Opcode, left, right object.Object) error {
	leftVal, ok1 := toBigInt(left)
	rightVal, ok2 := toBigInt(right)
	if !ok1 || !ok2 {
		return fmt.Errorf("type mismatch: %s and %s for BigInt operation", left.Type(), right.Type())
	}

	result := new(big.Int)
	switch op {
	case code.OpAdd:
		result.Add(leftVal, rightVal)
	case code.OpSub:
		result.Sub(leftVal, rightVal)
	case code.OpMul:
		result.Mul(leftVal, rightVal)
	case code.OpDiv:
		if rightVal.Sign() == 0 {
			return fmt.Errorf("division by zero")
		}
		result.Quo(leftVal, rightVal)
	case code.OpMod:
		if rightVal.Sign() == 0 {
			return fmt.Errorf("division by zero")
		}
		result.Rem(leftVal, rightVal)
	case code.OpPow:
		if rightVal.Sign() < 0 {
			return fmt.Errorf("negative exponent not supported for BigInt pow")
		}
		result.Exp(leftVal, rightVal, nil)
	default:
		return fmt.Errorf("unknown big int operator: %d", op)
	}

	return vm.push(&object.BigInt{Value: result})
}

func (vm *VM) executeBinaryBigFloatOperation(op code.Opcode, left, right object.Object) error {
	leftVal, ok1 := toBigFloat(left)
	rightVal, ok2 := toBigFloat(right)
	if !ok1 || !ok2 {
		return fmt.Errorf("type mismatch: %s and %s for BigFloat operation", left.Type(), right.Type())
	}

	result := new(big.Float)
	switch op {
	case code.OpAdd:
		result.Add(leftVal, rightVal)
	case code.OpSub:
		result.Sub(leftVal, rightVal)
	case code.OpMul:
		result.Mul(leftVal, rightVal)
	case code.OpDiv:
		if rightVal.Sign() == 0 {
			return fmt.Errorf("division by zero")
		}
		result.Quo(leftVal, rightVal)
	default:
		return fmt.Errorf("unknown big float operator: %d", op)
	}

	return vm.push(&object.BigFloat{Value: result})
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
	if left.Type() == object.BIG_INT_OBJ || right.Type() == object.BIG_INT_OBJ {
		return vm.executeBigIntComparison(op, left, right)
	}
	if left.Type() == object.BIG_FLOAT_OBJ || right.Type() == object.BIG_FLOAT_OBJ {
		return vm.executeBigFloatComparison(op, left, right)
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

func (vm *VM) executeBigIntComparison(op code.Opcode, left, right object.Object) error {
	leftVal, ok1 := toBigInt(left)
	rightVal, ok2 := toBigInt(right)
	if !ok1 || !ok2 {
		return fmt.Errorf("type mismatch: cannot compare %s with %s", left.Type(), right.Type())
	}

	cmp := leftVal.Cmp(rightVal)
	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(cmp == 0))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(cmp != 0))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(cmp > 0))
	case code.OpGreaterThanEqual:
		return vm.push(nativeBoolToBooleanObject(cmp >= 0))
	default:
		return fmt.Errorf("unknown comparison operator: %d", op)
	}
}

func (vm *VM) executeBigFloatComparison(op code.Opcode, left, right object.Object) error {
	leftVal, ok1 := toBigFloat(left)
	rightVal, ok2 := toBigFloat(right)
	if !ok1 || !ok2 {
		return fmt.Errorf("type mismatch: cannot compare %s with %s", left.Type(), right.Type())
	}

	cmp := leftVal.Cmp(rightVal)
	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(cmp == 0))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(cmp != 0))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBooleanObject(cmp > 0))
	case code.OpGreaterThanEqual:
		return vm.push(nativeBoolToBooleanObject(cmp >= 0))
	default:
		return fmt.Errorf("unknown comparison operator: %d", op)
	}
}

func (vm *VM) executeMinusOperator() error {
	operand := vm.pop()

	switch operand := operand.(type) {
	case *object.Integer:
		return vm.push(&object.Integer{Value: -operand.Value})
	case *object.Float:
		return vm.push(&object.Float{Value: -operand.Value})
	case *object.BigInt:
		return vm.push(operand.Neg())
	case *object.BigFloat:
		return vm.push(operand.Neg())
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
		case "send":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: fmt.Sprintf("send requires 1 argument, got %d", len(args))}
					}
					err := ch.Send(args[0])
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "recv", "receive":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					val, ok := ch.Recv()
					if !ok || val == nil {
						return Null
					}
					return val
				},
			})
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

	case left.Type() == object.VECTOR_OBJ && index.Type() == object.INTEGER_OBJ:
		v := left.(*object.Vector)
		i := index.(*object.Integer).Value
		if i < 0 || i >= int64(len(v.Elements)) {
			return vm.push(Null)
		}
		return vm.push(&object.Float{Value: v.Elements[i]})

	case left.Type() == object.VECTOR_OBJ && index.Type() == object.STRING_OBJ:
		v := left.(*object.Vector)
		prop := index.(*object.String).Value
		switch prop {
		case "dim", "len", "length":
			return vm.push(&object.Integer{Value: int64(v.Dim())})
		case "elements":
			return vm.push(v.ToArray())
		case "norm", "magnitude":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Float{Value: v.Norm()}
				},
			})
		case "normalize":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					res, err := v.Normalize()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "dot":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "dot requires 1 argument"}
					}
					other, ok := args[0].(*object.Vector)
					if !ok {
						return &object.Error{Message: "argument to dot must be Vector"}
					}
					d, err := v.Dot(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Float{Value: d}
				},
			})
		case "cross":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "cross requires 1 argument"}
					}
					other, ok := args[0].(*object.Vector)
					if !ok {
						return &object.Error{Message: "argument to cross must be Vector"}
					}
					res, err := v.Cross(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "add":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "add requires 1 argument"}
					}
					other, ok := args[0].(*object.Vector)
					if !ok {
						return &object.Error{Message: "argument to add must be Vector"}
					}
					res, err := v.Add(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "sub":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "sub requires 1 argument"}
					}
					other, ok := args[0].(*object.Vector)
					if !ok {
						return &object.Error{Message: "argument to sub must be Vector"}
					}
					res, err := v.Sub(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "scale":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "scale requires 1 argument"}
					}
					s, ok := toFloat(args[0])
					if !ok {
						return &object.Error{Message: "scalar must be number"}
					}
					return v.Scale(s)
				},
			})
		case "to_array":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return v.ToArray()
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on Vector", prop)
		}

	case left.Type() == object.MATRIX_OBJ && index.Type() == object.INTEGER_OBJ:
		m := left.(*object.Matrix)
		i := index.(*object.Integer).Value
		if i < 0 || i >= int64(m.Rows) {
			return vm.push(Null)
		}
		cols := make([]object.Object, m.Cols)
		for j := 0; j < m.Cols; j++ {
			cols[j] = &object.Float{Value: m.Data[i][j]}
		}
		return vm.push(&object.Array{Elements: cols})

	case left.Type() == object.MATRIX_OBJ && index.Type() == object.STRING_OBJ:
		m := left.(*object.Matrix)
		prop := index.(*object.String).Value
		switch prop {
		case "rows":
			return vm.push(&object.Integer{Value: int64(m.Rows)})
		case "cols":
			return vm.push(&object.Integer{Value: int64(m.Cols)})
		case "data", "elements":
			return vm.push(m.ToArray())
		case "get":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "get requires 2 arguments (row, col)"}
					}
					r, ok1 := args[0].(*object.Integer)
					c, ok2 := args[1].(*object.Integer)
					if !ok1 || !ok2 {
						return &object.Error{Message: "matrix coordinates must be integers"}
					}
					val, err := m.Get(int(r.Value), int(c.Value))
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Float{Value: val}
				},
			})
		case "set":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 3 {
						return &object.Error{Message: "set requires 3 arguments (row, col, value)"}
					}
					r, ok1 := args[0].(*object.Integer)
					c, ok2 := args[1].(*object.Integer)
					v, ok3 := toFloat(args[2])
					if !ok1 || !ok2 || !ok3 {
						return &object.Error{Message: "invalid arguments to matrix.set"}
					}
					err := m.Set(int(r.Value), int(c.Value), v)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "mul", "mul_vec":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "mul requires 1 argument"}
					}
					switch other := args[0].(type) {
					case *object.Matrix:
						res, err := m.Mul(other)
						if err != nil {
							return &object.Error{Message: err.Error()}
						}
						return res
					case *object.Vector:
						res, err := m.MulVec(other)
						if err != nil {
							return &object.Error{Message: err.Error()}
						}
						return res
					default:
						return &object.Error{Message: "argument to matrix.mul must be Matrix or Vector"}
					}
				},
			})
		case "add":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "add requires 1 argument"}
					}
					other, ok := args[0].(*object.Matrix)
					if !ok {
						return &object.Error{Message: "argument to matrix.add must be Matrix"}
					}
					res, err := m.Add(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "sub":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "sub requires 1 argument"}
					}
					other, ok := args[0].(*object.Matrix)
					if !ok {
						return &object.Error{Message: "argument to matrix.sub must be Matrix"}
					}
					res, err := m.Sub(other)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "scale":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "scale requires 1 argument"}
					}
					s, ok := toFloat(args[0])
					if !ok {
						return &object.Error{Message: "scalar must be number"}
					}
					return m.Scale(s)
				},
			})
		case "transpose":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return m.Transpose()
				},
			})
		case "det":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					d, err := m.Det()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Float{Value: d}
				},
			})
		case "inv":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					inv, err := m.Inv()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return inv
				},
			})
		case "to_array":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return m.ToArray()
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on Matrix", prop)
		}

	case left.Type() == object.BIG_INT_OBJ && index.Type() == object.STRING_OBJ:
		bi := left.(*object.BigInt)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "add requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.add must be integer or BigInt"}
					}
					return bi.Add(&object.BigInt{Value: o})
				},
			})
		case "sub":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "sub requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.sub must be integer or BigInt"}
					}
					return bi.Sub(&object.BigInt{Value: o})
				},
			})
		case "mul":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "mul requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.mul must be integer or BigInt"}
					}
					return bi.Mul(&object.BigInt{Value: o})
				},
			})
		case "div":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "div requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.div must be integer or BigInt"}
					}
					res, err := bi.Div(&object.BigInt{Value: o})
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "mod":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "mod requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.mod must be integer or BigInt"}
					}
					res, err := bi.Mod(&object.BigInt{Value: o})
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "pow":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "pow requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.pow must be integer or BigInt"}
					}
					return bi.Pow(&object.BigInt{Value: o})
				},
			})
		case "cmp":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "cmp requires 1 argument"}
					}
					o, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigInt.cmp must be integer or BigInt"}
					}
					return &object.Integer{Value: int64(bi.Cmp(&object.BigInt{Value: o}))}
				},
			})
		case "abs":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bi.Abs()
				},
			})
		case "neg":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bi.Neg()
				},
			})
		case "str", "to_str", "to_string":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: bi.Value.String()}
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on BigInt", prop)
		}

	case left.Type() == object.BIG_FLOAT_OBJ && index.Type() == object.STRING_OBJ:
		bf := left.(*object.BigFloat)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "add requires 1 argument"}
					}
					o, ok := toBigFloat(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigFloat.add must be number or BigFloat"}
					}
					return bf.Add(&object.BigFloat{Value: o})
				},
			})
		case "sub":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "sub requires 1 argument"}
					}
					o, ok := toBigFloat(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigFloat.sub must be number or BigFloat"}
					}
					return bf.Sub(&object.BigFloat{Value: o})
				},
			})
		case "mul":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "mul requires 1 argument"}
					}
					o, ok := toBigFloat(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigFloat.mul must be number or BigFloat"}
					}
					return bf.Mul(&object.BigFloat{Value: o})
				},
			})
		case "div":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "div requires 1 argument"}
					}
					o, ok := toBigFloat(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigFloat.div must be number or BigFloat"}
					}
					res, err := bf.Div(&object.BigFloat{Value: o})
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "sqrt":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					res, err := bf.Sqrt()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "cmp":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "cmp requires 1 argument"}
					}
					o, ok := toBigFloat(args[0])
					if !ok {
						return &object.Error{Message: "argument to BigFloat.cmp must be number or BigFloat"}
					}
					return &object.Integer{Value: int64(bf.Cmp(&object.BigFloat{Value: o}))}
				},
			})
		case "abs":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bf.Abs()
				},
			})
		case "neg":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bf.Neg()
				},
			})
		case "str", "to_str", "to_string":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					prec := -1
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok {
							prec = int(intObj.Value)
						}
					}
					return &object.String{Value: bf.Value.Text('g', prec)}
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on BigFloat", prop)
		}

	case left.Type() == object.TCP_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.TcpServer)
		prop := index.(*object.String).Value
		switch prop {
		case "addr":
			return vm.push(&object.String{Value: s.Addr()})
		case "accept":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					conn, err := s.Accept()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return conn
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on TcpServer", prop)
		}

	case left.Type() == object.TCP_CONN_OBJ && index.Type() == object.STRING_OBJ:
		c := left.(*object.TcpConn)
		prop := index.(*object.String).Value
		switch prop {
		case "local_addr":
			return vm.push(&object.String{Value: c.LocalAddr()})
		case "remote_addr":
			return vm.push(&object.String{Value: c.RemoteAddr()})
		case "write", "send":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "write requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := c.Write(data)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "read", "recv":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
						}
					}
					data, err := c.Read(maxBytes)
					if err != nil {
						if err == io.EOF {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: string(data)}
				},
			})
		case "read_line":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					line, err := c.ReadLine()
					if err != nil {
						if err == io.EOF && len(line) == 0 {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: line}
				},
			})
		case "set_timeout":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "set_timeout requires 1 argument (ms)"}
					}
					if intObj, ok := args[0].(*object.Integer); ok {
						c.SetTimeout(intObj.Value)
					}
					return Null
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := c.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on TcpConn", prop)
		}

	case left.Type() == object.UDP_SOCKET_OBJ && index.Type() == object.STRING_OBJ:
		u := left.(*object.UdpSocket)
		prop := index.(*object.String).Value
		switch prop {
		case "local_addr":
			return vm.push(&object.String{Value: u.LocalAddr()})
		case "send_to":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "send_to requires 2 arguments (data, addr)"}
					}
					addrObj, ok := args[1].(*object.String)
					if !ok {
						return &object.Error{Message: "address must be string"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := u.SendTo(data, addrObj.Value)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "recv_from":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
						}
					}
					data, addr, err := u.RecvFrom(maxBytes)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					pairs := make(map[object.HashKey]object.HashPair)
					dataKey := &object.String{Value: "data"}
					addrKey := &object.String{Value: "addr"}
					pairs[dataKey.HashKey()] = object.HashPair{Key: dataKey, Value: &object.String{Value: string(data)}}
					pairs[addrKey.HashKey()] = object.HashPair{Key: addrKey, Value: &object.String{Value: addr}}
					return &object.Hash{Pairs: pairs}
				},
			})
		case "write", "send":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "write requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					n, err := u.Write(data)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Integer{Value: int64(n)}
				},
			})
		case "read", "recv":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					maxBytes := 4096
					if len(args) > 0 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							maxBytes = int(intObj.Value)
						}
					}
					data, err := u.Read(maxBytes)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: string(data)}
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := u.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on UdpSocket", prop)
		}

	case left.Type() == object.WS_CONN_OBJ && index.Type() == object.STRING_OBJ:
		w := left.(*object.WsConn)
		prop := index.(*object.String).Value
		switch prop {
		case "is_open":
			return vm.push(nativeBoolToBooleanObject(w.IsOpen()))
		case "send":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "send requires 1 argument"}
					}
					var msg string
					if strObj, ok := args[0].(*object.String); ok {
						msg = strObj.Value
					} else {
						msg = args[0].Inspect()
					}
					if err := w.Send(msg); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "send_binary":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "send_binary requires 1 argument"}
					}
					var data []byte
					if strObj, ok := args[0].(*object.String); ok {
						data = []byte(strObj.Value)
					} else {
						data = []byte(args[0].Inspect())
					}
					if err := w.SendBinary(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "recv":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					msg, err := w.Recv()
					if err != nil {
						if err == io.EOF {
							return Null
						}
						return &object.Error{Message: err.Error()}
					}
					return &object.String{Value: msg}
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := w.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on WsConn", prop)
		}

	case left.Type() == object.WS_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.WsServer)
		prop := index.(*object.String).Value
		switch prop {
		case "addr":
			return vm.push(&object.String{Value: s.Addr()})
		case "path":
			return vm.push(&object.String{Value: s.Path})
		case "accept":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					conn, err := s.Accept()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return conn
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on WsServer", prop)
		}

	case left.Type() == object.HTTP_SERVER_OBJ && index.Type() == object.STRING_OBJ:
		s := left.(*object.HttpServer)
		prop := index.(*object.String).Value
		switch prop {
		case "get", "post", "put", "delete":
			method := strings.ToUpper(prop)
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: fmt.Sprintf("%s route requires 2 arguments (path, handler)", prop)}
					}
					pathObj, ok := args[0].(*object.String)
					if !ok {
						return &object.Error{Message: "route path must be string"}
					}
					s.AddRoute(method, pathObj.Value, args[1])
					return left
				},
			})
		case "use":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "use requires 1 argument (middleware)"}
					}
					s.Use(args[0])
					return left
				},
			})
		case "static":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "static requires 2 arguments (prefix, dir)"}
					}
					prefixObj, ok1 := args[0].(*object.String)
					dirObj, ok2 := args[1].(*object.String)
					if !ok1 || !ok2 {
						return &object.Error{Message: "static prefix and dir must be strings"}
					}
					s.Static(prefixObj.Value, dirObj.Value)
					return left
				},
			})
		case "listen":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) < 1 {
						return &object.Error{Message: "listen requires at least 1 argument (port or addr)"}
					}
					addr := "127.0.0.1:8080"
					if strObj, ok := args[0].(*object.String); ok {
						addr = strObj.Value
					} else if intObj, ok := args[0].(*object.Integer); ok {
						addr = fmt.Sprintf("127.0.0.1:%d", intObj.Value)
					}
					l, err := net.Listen("tcp", addr)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					s.Listener = l

					handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						s.Mu.RLock()
						for prefix, dir := range s.StaticDirs {
							if strings.HasPrefix(r.URL.Path, prefix) {
								s.Mu.RUnlock()
								http.StripPrefix(prefix, http.FileServer(http.Dir(dir))).ServeHTTP(w, r)
								return
							}
						}
						s.Mu.RUnlock()

						bodyBytes, _ := io.ReadAll(r.Body)
						bodyStr := string(bodyBytes)

						reqPairs := make(map[object.HashKey]object.HashPair)
						setStr := func(k, v string) {
							kObj := &object.String{Value: k}
							reqPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
						}
						setStr("method", r.Method)
						setStr("path", r.URL.Path)
						setStr("body", bodyStr)

						qPairs := make(map[object.HashKey]object.HashPair)
						for qk, qvs := range r.URL.Query() {
							if len(qvs) > 0 {
								kObj := &object.String{Value: qk}
								qPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: qvs[0]}}
							}
						}
						qKey := &object.String{Value: "query"}
						reqPairs[qKey.HashKey()] = object.HashPair{Key: qKey, Value: &object.Hash{Pairs: qPairs}}

						hdrPairs := make(map[object.HashKey]object.HashPair)
						for hk, hvs := range r.Header {
							if len(hvs) > 0 {
								kObj := &object.String{Value: hk}
								hdrPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: hvs[0]}}
							}
						}
						hKey := &object.String{Value: "headers"}
						reqPairs[hKey.HashKey()] = object.HashPair{Key: hKey, Value: &object.Hash{Pairs: hdrPairs}}

						jsonKey := &object.String{Value: "json"}
						reqPairs[jsonKey.HashKey()] = object.HashPair{
							Key: jsonKey,
							Value: &object.Builtin{
								Fn: func(a ...object.Object) object.Object {
									var v interface{}
									if err := json.Unmarshal([]byte(bodyStr), &v); err != nil {
										return &object.Error{Message: fmt.Sprintf("invalid JSON: %s", err)}
									}
									return nativeToBramandObject(v)
								},
							},
						}

						reqObj := &object.Hash{Pairs: reqPairs}
						rwObj := &object.HttpResponseWriter{W: w}

						// Middlewares
						s.Mu.RLock()
						mws := make([]object.Object, len(s.Middlewares))
						copy(mws, s.Middlewares)
						s.Mu.RUnlock()

						for _, mw := range mws {
							_, err := ExecuteCallable(mw, reqObj, rwObj)
							if err != nil {
								http.Error(w, err.Error(), 500)
								return
							}
							if rwObj.Written {
								return
							}
						}

						s.Mu.RLock()
						mRoutes := s.Routes[r.Method]
						var routeHandler object.Object
						if mRoutes != nil {
							routeHandler = mRoutes[r.URL.Path]
						}
						s.Mu.RUnlock()

						if routeHandler != nil {
							_, err := ExecuteCallable(routeHandler, reqObj, rwObj)
							if err != nil {
								http.Error(w, err.Error(), 500)
								return
							}
							if !rwObj.Written {
								_ = rwObj.Send(nil)
							}
							return
						}

						http.NotFound(w, r)
					})

					s.Server = &http.Server{Handler: handler}
					go func() {
						_ = s.Server.Serve(l)
					}()

					if len(args) >= 2 {
						_, _ = ExecuteCallable(args[1])
					}
					return left
				},
			})
		case "close":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if err := s.Close(); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on HttpServer", prop)
		}

	case left.Type() == object.HTTP_RESPONSE_WRITER_OBJ && index.Type() == object.STRING_OBJ:
		rw := left.(*object.HttpResponseWriter)
		prop := index.(*object.String).Value
		switch prop {
		case "send":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					var data []byte
					if len(args) > 0 {
						if strObj, ok := args[0].(*object.String); ok {
							data = []byte(strObj.Value)
						} else {
							data = []byte(args[0].Inspect())
						}
					}
					if err := rw.Send(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "json":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "json requires 1 argument"}
					}
					rw.SetHeader("Content-Type", "application/json")
					native := bramandObjectToNative(args[0])
					data, err := json.Marshal(native)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					if err := rw.Send(data); err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "status":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "status requires 1 argument"}
					}
					if intObj, ok := args[0].(*object.Integer); ok {
						rw.SendStatus(int(intObj.Value))
					}
					return left
				},
			})
		case "set_header", "setHeader":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 2 {
						return &object.Error{Message: "set_header requires 2 arguments (key, value)"}
					}
					kObj, ok1 := args[0].(*object.String)
					vObj, ok2 := args[1].(*object.String)
					if !ok1 || !ok2 {
						return &object.Error{Message: "header key and value must be strings"}
					}
					rw.SetHeader(kObj.Value, vObj.Value)
					return left
				},
			})
		case "redirect":
			return vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) < 1 {
						return &object.Error{Message: "redirect requires at least 1 argument (url)"}
					}
					urlStr := args[0].Inspect()
					if sObj, ok := args[0].(*object.String); ok {
						urlStr = sObj.Value
					}
					code := 302
					if len(args) >= 2 {
						if intObj, ok := args[1].(*object.Integer); ok {
							code = int(intObj.Value)
						}
					}
					rw.SendStatus(code)
					rw.SetHeader("Location", urlStr)
					_ = rw.Send(nil)
					return Null
				},
			})
		default:
			return fmt.Errorf("unknown method '%s' on HttpResponseWriter", prop)
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
	if callee == nil {
		return fmt.Errorf("calling null or undefined value")
	}
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

func ObjectsEqual(left, right object.Object) bool {
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
	case *object.Array:
		r := right.(*object.Array)
		if len(l.Elements) != len(r.Elements) {
			return false
		}
		for i := range l.Elements {
			if !ObjectsEqual(l.Elements[i], r.Elements[i]) {
				return false
			}
		}
		return true
	case *object.Hash:
		r := right.(*object.Hash)
		if len(l.Pairs) != len(r.Pairs) {
			return false
		}
		for k, pair := range l.Pairs {
			rPair, ok := r.Pairs[k]
			if !ok {
				return false
			}
			if !ObjectsEqual(pair.Value, rPair.Value) {
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
