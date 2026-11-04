package vm

import (
	"fmt"

	"bramandscript/object"
)

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

	hashObject.Mu.RLock()
	pair, ok := hashObject.Pairs[key.HashKey()]
	hashObject.Mu.RUnlock()
	if ok {
		val := pair.Value
		if cl, isCl := val.(*object.Closure); isCl {
			hashObject.Mu.RLock()
			_, isStruct := hashObject.Pairs[(&object.String{Value: "__struct__"}).HashKey()]
			hashObject.Mu.RUnlock()
			if isStruct {
				cl.Instance = hashObject
			}
		}
		return vm.push(val)
	}

	if strIdx, ok := index.(*object.String); ok {
		if strIdx.Value == "length" || strIdx.Value == "len" {
			hashObject.Mu.RLock()
			l := len(hashObject.Pairs)
			hashObject.Mu.RUnlock()
			return vm.push(&object.Integer{Value: int64(l)})
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
		l.Mu.Lock()
		defer l.Mu.Unlock()
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
		l.Mu.Lock()
		l.Pairs[key.HashKey()] = object.HashPair{Key: index, Value: value}
		l.Mu.Unlock()
		return nil

	default:
		return fmt.Errorf("index assignment not supported for type: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	arrayObject.Mu.RLock()
	defer arrayObject.Mu.RUnlock()
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(arrayObject.Elements[i])
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeStringIndex(left, index)
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
	default:
		// Check math and concurrency methods
		if handled, err := vm.executeMathAndConcurrencyMethod(left, index); handled {
			return err
		}
		// Check networking methods
		if handled, err := vm.executeNetMethod(left, index); handled {
			return err
		}
		return fmt.Errorf("index operator not supported: %s[%s]", left.Type(), index.Type())
	}
}
