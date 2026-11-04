package vm

import (
	"fmt"
	"math"
	"math/big"

	"bramandscript/code"
	"bramandscript/object"
)

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	if vm.sp < 2 {
		return fmt.Errorf("stack underflow during binary operation")
	}
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
	case leftType == object.ARRAY_OBJ && rightType == object.ARRAY_OBJ:
		if op == code.OpAdd {
			return vm.executeArrayConcatenation(left, right)
		}
		return fmt.Errorf("unsupported operator %d for arrays", op)
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
		if leftVal == math.MinInt64 && rightVal == -1 {
			return fmt.Errorf("integer overflow in division")
		}
		result = leftVal / rightVal
	case code.OpMod:
		if rightVal == 0 {
			return fmt.Errorf("modulo by zero")
		}
		if leftVal == math.MinInt64 && rightVal == -1 {
			result = 0
			break
		}
		result = leftVal % rightVal
	case code.OpPow:
		result = int64(math.Pow(float64(leftVal), float64(rightVal)))
	case code.OpBitAnd:
		result = leftVal & rightVal
	case code.OpBitOr:
		result = leftVal | rightVal
	case code.OpBitXor:
		result = leftVal ^ rightVal
	case code.OpShiftLeft:
		if rightVal < 0 || rightVal >= 64 {
			return fmt.Errorf("shift count out of range: %d", rightVal)
		}
		result = leftVal << uint(rightVal)
	case code.OpShiftRight:
		if rightVal < 0 || rightVal >= 64 {
			return fmt.Errorf("shift count out of range: %d", rightVal)
		}
		result = leftVal >> uint(rightVal)
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
	case code.OpMod:
		if rightVal == 0 {
			return fmt.Errorf("modulo by zero")
		}
		result = math.Mod(leftVal, rightVal)
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
		if rightVal.BitLen() > 24 || rightVal.Cmp(big.NewInt(1000000)) > 0 {
			return fmt.Errorf("exponent too large for BigInt pow (maximum exponent is 1000000)")
		}
		result.Exp(leftVal, rightVal, nil)
	case code.OpBitAnd:
		result.And(leftVal, rightVal)
	case code.OpBitOr:
		result.Or(leftVal, rightVal)
	case code.OpBitXor:
		result.Xor(leftVal, rightVal)
	case code.OpShiftLeft:
		if !rightVal.IsInt64() || rightVal.Int64() < 0 {
			return fmt.Errorf("invalid shift count: %s", rightVal.String())
		}
		result.Lsh(leftVal, uint(rightVal.Int64()))
	case code.OpShiftRight:
		if !rightVal.IsInt64() || rightVal.Int64() < 0 {
			return fmt.Errorf("invalid shift count: %s", rightVal.String())
		}
		result.Rsh(leftVal, uint(rightVal.Int64()))
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

func (vm *VM) executeArrayConcatenation(left, right object.Object) error {
	leftArr := left.(*object.Array)
	rightArr := right.(*object.Array)
	elements := make([]object.Object, len(leftArr.Elements)+len(rightArr.Elements))
	copy(elements, leftArr.Elements)
	copy(elements[len(leftArr.Elements):], rightArr.Elements)
	return vm.push(&object.Array{Elements: elements})
}

func (vm *VM) executeComparison(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	if left.Type() == object.NULL_OBJ || right.Type() == object.NULL_OBJ {
		switch op {
		case code.OpEqual:
			return vm.push(nativeBoolToBooleanObject(left.Type() == object.NULL_OBJ && right.Type() == object.NULL_OBJ))
		case code.OpNotEqual:
			return vm.push(nativeBoolToBooleanObject(!(left.Type() == object.NULL_OBJ && right.Type() == object.NULL_OBJ)))
		default:
			return vm.push(False)
		}
	}

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
	if left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ {
		return vm.executeStringComparison(op, left, right)
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBooleanObject(ObjectsEqual(left, right)))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBooleanObject(!ObjectsEqual(left, right)))
	default:
		return fmt.Errorf("unknown comparison operator: %d (%s %s)", op, left.Type(), right.Type())
	}
}

func (vm *VM) executeStringComparison(op code.Opcode, left, right object.Object) error {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

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
		return fmt.Errorf("unknown string comparison operator: %d", op)
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
	if isTruthy(operand) {
		return vm.push(False)
	}
	return vm.push(True)
}

func (vm *VM) executeBitNotOperator() error {
	operand := vm.pop()
	switch operand := operand.(type) {
	case *object.Integer:
		return vm.push(&object.Integer{Value: ^operand.Value})
	case *object.BigInt:
		res := new(big.Int).Not(operand.Value)
		return vm.push(&object.BigInt{Value: res})
	default:
		return fmt.Errorf("unsupported type for bitwise NOT: %s", operand.Type())
	}
}
