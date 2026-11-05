package vm

import (
	"encoding/binary"
	"fmt"
	"strings"

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

	hashObject.Mu.RLock()
	embedPair, hasEmbeds := hashObject.Pairs[(&object.String{Value: "__embeds__"}).HashKey()]
	hashObject.Mu.RUnlock()
	if hasEmbeds {
		if embedArr, isArr := embedPair.Value.(*object.Array); isArr {
			embedArr.Mu.RLock()
			for _, embObj := range embedArr.Elements {
				if embHash, isH := embObj.(*object.Hash); isH {
					embHash.Mu.RLock()
					p, ok := embHash.Pairs[key.HashKey()]
					embHash.Mu.RUnlock()
					if ok {
						embedArr.Mu.RUnlock()
						val := p.Value
						if cl, isCl := val.(*object.Closure); isCl {
							cl.Instance = embHash
						}
						return vm.push(val)
					}
				}
			}
			embedArr.Mu.RUnlock()
		}
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

	case *object.Bytes:
		idx, ok := index.(*object.Integer)
		if !ok {
			return fmt.Errorf("bytes index must be an INTEGER, got %s", index.Type())
		}
		val, ok := value.(*object.Integer)
		if !ok {
			return fmt.Errorf("bytes element must be an INTEGER, got %s", value.Type())
		}
		l.Mu.Lock()
		defer l.Mu.Unlock()
		i := idx.Value
		if i < 0 || i >= int64(len(l.Value)) {
			return fmt.Errorf("index out of bounds: %d (bytes length is %d)", i, len(l.Value))
		}
		l.Value[i] = byte(val.Value)
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

func (vm *VM) executeBytesIndex(bytesObj, index object.Object) error {
	b := bytesObj.(*object.Bytes)
	b.Mu.RLock()
	defer b.Mu.RUnlock()
	i := index.(*object.Integer).Value
	max := int64(len(b.Value) - 1)

	if i < 0 || i > max {
		return vm.push(Null)
	}

	return vm.push(&object.Integer{Value: int64(b.Value[i])})
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeStringIndex(left, index)
	case left.Type() == object.BYTES_OBJ && index.Type() == object.INTEGER_OBJ:
		return vm.executeBytesIndex(left, index)
	case left.Type() == object.HASH_OBJ:
		return vm.executeHashIndex(left, index)
	case left.Type() == object.ERROR_OBJ && index.Type() == object.STRING_OBJ:
		errObj := left.(*object.Error)
		prop := index.(*object.String).Value
		if prop == "message" || prop == "msg" {
			return vm.push(&object.String{Value: errObj.Message})
		}
		return vm.push(Null)
	case (left.Type() == object.ARRAY_OBJ || left.Type() == object.STRING_OBJ || left.Type() == object.BYTES_OBJ) && index.Type() == object.STRING_OBJ:
		prop := index.(*object.String).Value
		if prop == "length" || prop == "len" {
			switch l := left.(type) {
			case *object.Array:
				return vm.push(&object.Integer{Value: int64(len(l.Elements))})
			case *object.String:
				return vm.push(&object.Integer{Value: int64(len([]rune(l.Value)))})
			case *object.Bytes:
				l.Mu.RLock()
				defer l.Mu.RUnlock()
				return vm.push(&object.Integer{Value: int64(len(l.Value))})
			}
		}
		if left.Type() == object.BYTES_OBJ {
			if handled, err := vm.executeBytesMethod(left, index); handled {
				return err
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
		// Check bytes methods
		if handled, err := vm.executeBytesMethod(left, index); handled {
			return err
		}
		return fmt.Errorf("index operator not supported: %s[%s]", left.Type(), index.Type())
	}
}

func (vm *VM) executeBytesMethod(left, index object.Object) (bool, error) {
	b, ok := left.(*object.Bytes)
	if !ok {
		return false, nil
	}
	prop, ok := index.(*object.String)
	if !ok {
		return false, nil
	}

	switch prop.Value {
	case "read_u8":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("read_u8 expected 1 argument (offset), got %d", len(args))}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value >= int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u8 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				return &object.Integer{Value: int64(b.Value[offset.Value])}
			},
		})
	case "write_u8":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("write_u8 expected 2 arguments (offset, value), got %d", len(args))}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value >= int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u8 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				b.Value[offset.Value] = byte(val.Value)
				return b
			},
		})
	case "read_u16":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: "read_u16 expected 1 or 2 arguments (offset, [endian])"}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 2 {
					if e, ok := args[1].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value+2 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u16 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				var val uint16
				if endian == "le" {
					val = binary.LittleEndian.Uint16(b.Value[offset.Value:])
				} else {
					val = binary.BigEndian.Uint16(b.Value[offset.Value:])
				}
				return &object.Integer{Value: int64(val)}
			},
		})
	case "write_u16":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: "write_u16 expected 2 or 3 arguments (offset, value, [endian])"}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 3 {
					if e, ok := args[2].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value+2 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u16 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				if endian == "le" {
					binary.LittleEndian.PutUint16(b.Value[offset.Value:], uint16(val.Value))
				} else {
					binary.BigEndian.PutUint16(b.Value[offset.Value:], uint16(val.Value))
				}
				return b
			},
		})
	case "read_u32":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: "read_u32 expected 1 or 2 arguments (offset, [endian])"}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 2 {
					if e, ok := args[1].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				if offset.Value < 0 || offset.Value+4 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("read_u32 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				var val uint32
				if endian == "le" {
					val = binary.LittleEndian.Uint32(b.Value[offset.Value:])
				} else {
					val = binary.BigEndian.Uint32(b.Value[offset.Value:])
				}
				return &object.Integer{Value: int64(val)}
			},
		})
	case "write_u32":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: "write_u32 expected 2 or 3 arguments (offset, value, [endian])"}
				}
				offset, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "offset must be an INTEGER"}
				}
				val, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "value must be an INTEGER"}
				}
				endian := "be"
				if len(args) == 3 {
					if e, ok := args[2].(*object.String); ok {
						endian = strings.ToLower(e.Value)
					}
				}
				b.Mu.Lock()
				defer b.Mu.Unlock()
				if offset.Value < 0 || offset.Value+4 > int64(len(b.Value)) {
					return &object.Error{Message: fmt.Sprintf("write_u32 offset %d out of bounds (length %d)", offset.Value, len(b.Value))}
				}
				if endian == "le" {
					binary.LittleEndian.PutUint32(b.Value[offset.Value:], uint32(val.Value))
				} else {
					binary.BigEndian.PutUint32(b.Value[offset.Value:], uint32(val.Value))
				}
				return b
			},
		})
	case "slice":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: "slice expected 1 or 2 arguments (start, [end])"}
				}
				start, ok := args[0].(*object.Integer)
				if !ok {
					return &object.Error{Message: "start must be an INTEGER"}
				}
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				length := int64(len(b.Value))
				s := start.Value
				if s < 0 {
					s = length + s
					if s < 0 {
						s = 0
					}
				}
				if s > length {
					s = length
				}
				e := length
				if len(args) == 2 {
					if end, ok := args[1].(*object.Integer); ok {
						e = end.Value
						if e < 0 {
							e = length + e
							if e < 0 {
								e = 0
							}
						}
						if e > length {
							e = length
						}
						if e < s {
							e = s
						}
					}
				}
				sub := make([]byte, e-s)
				copy(sub, b.Value[s:e])
				return &object.Bytes{Value: sub}
			},
		})
	case "to_string":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				return &object.String{Value: string(b.Value)}
			},
		})
	case "to_hex":
		return true, vm.push(&object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				b.Mu.RLock()
				defer b.Mu.RUnlock()
				return &object.String{Value: fmt.Sprintf("%x", b.Value)}
			},
		})
	}
	return false, nil
}
