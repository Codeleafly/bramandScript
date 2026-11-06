package vm

import (
	"fmt"
	"sort"
	"strings"

	"bramandscript/object"
)

func getSortedHashKeys(h *object.Hash) []object.Object {
	h.Mu.RLock()
	defer h.Mu.RUnlock()
	keys := make([]object.Object, 0, len(h.Pairs))
	for _, pair := range h.Pairs {
		keys = append(keys, pair.Key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Inspect() < keys[j].Inspect()
	})
	return keys
}

func computeSliceBounds(length int64, startObj, endObj, stepObj object.Object) (int64, int64, int64, error) {
	step := int64(1)
	if stepObj != nil && stepObj != Null {
		intStep, ok := stepObj.(*object.Integer)
		if !ok {
			return 0, 0, 0, fmt.Errorf("slice step must be an integer, got %s", stepObj.Type())
		}
		step = intStep.Value
		if step == 0 {
			return 0, 0, 0, fmt.Errorf("slice step cannot be zero")
		}
	}

	var s, e int64

	if step > 0 {
		// Default start is 0
		if startObj == nil || startObj == Null {
			s = 0
		} else {
			intStart, ok := startObj.(*object.Integer)
			if !ok {
				return 0, 0, 0, fmt.Errorf("slice start must be an integer, got %s", startObj.Type())
			}
			s = intStart.Value
			if s < 0 {
				s += length
			}
			if s < 0 {
				s = 0
			}
			if s > length {
				s = length
			}
		}

		// Default end is length
		if endObj == nil || endObj == Null {
			e = length
		} else {
			intEnd, ok := endObj.(*object.Integer)
			if !ok {
				return 0, 0, 0, fmt.Errorf("slice end must be an integer, got %s", endObj.Type())
			}
			e = intEnd.Value
			if e < 0 {
				e += length
			}
			if e < 0 {
				e = 0
			}
			if e > length {
				e = length
			}
		}
		if s > e {
			s = e
		}
	} else {
		// Negative step: Default start is length - 1
		if startObj == nil || startObj == Null {
			s = length - 1
		} else {
			intStart, ok := startObj.(*object.Integer)
			if !ok {
				return 0, 0, 0, fmt.Errorf("slice start must be an integer, got %s", startObj.Type())
			}
			s = intStart.Value
			if s < 0 {
				s += length
			}
			if s < -1 {
				s = -1
			}
			if s >= length {
				s = length - 1
			}
		}

		// Negative step: Default end is -1 (meaning include index 0)
		if endObj == nil || endObj == Null {
			e = -1
		} else {
			intEnd, ok := endObj.(*object.Integer)
			if !ok {
				return 0, 0, 0, fmt.Errorf("slice end must be an integer, got %s", endObj.Type())
			}
			e = intEnd.Value
			if e < 0 {
				e += length
			}
			if e < -1 {
				e = -1
			}
			if e >= length {
				e = length - 1
			}
		}
		if s < e {
			s = e
		}
	}

	return s, e, step, nil
}

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"__iter_len": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Integer{Value: 0}
				}
				switch iter := args[0].(type) {
				case *object.Array:
					iter.Mu.RLock()
					count := int64(len(iter.Elements))
					iter.Mu.RUnlock()
					return &object.Integer{Value: count}
				case *object.Hash:
					iter.Mu.RLock()
					count := int64(len(iter.Pairs))
					iter.Mu.RUnlock()
					return &object.Integer{Value: count}
				case *object.String:
					return &object.Integer{Value: int64(len([]rune(iter.Value)))}
				case *object.Bytes:
					iter.Mu.RLock()
					count := int64(len(iter.Value))
					iter.Mu.RUnlock()
					return &object.Integer{Value: count}
				default:
					return &object.Integer{Value: 0}
				}
			},
		},
		"__iter_key": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return Null
				}
				idxObj, ok := args[1].(*object.Integer)
				if !ok {
					return Null
				}
				idx := int(idxObj.Value)
				switch iter := args[0].(type) {
				case *object.Hash:
					keys := getSortedHashKeys(iter)
					if idx >= 0 && idx < len(keys) {
						return keys[idx]
					}
					return Null
				default:
					return &object.Integer{Value: int64(idx)}
				}
			},
		},
		"__iter_val": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return Null
				}
				idxObj, ok := args[1].(*object.Integer)
				if !ok {
					return Null
				}
				idx := int(idxObj.Value)
				switch iter := args[0].(type) {
				case *object.Hash:
					keys := getSortedHashKeys(iter)
					if idx >= 0 && idx < len(keys) {
						iter.Mu.RLock()
						defer iter.Mu.RUnlock()
						key := keys[idx]
						if hashable, ok := key.(object.Hashable); ok {
							if pair, ok := iter.Pairs[hashable.HashKey()]; ok {
								return pair.Value
							}
						}
					}
					return Null
				case *object.Array:
					iter.Mu.RLock()
					defer iter.Mu.RUnlock()
					if idx >= 0 && idx < len(iter.Elements) {
						return iter.Elements[idx]
					}
					return Null
				case *object.String:
					runes := []rune(iter.Value)
					if idx >= 0 && idx < len(runes) {
						return &object.String{Value: string(runes[idx])}
					}
					return Null
				case *object.Bytes:
					iter.Mu.RLock()
					defer iter.Mu.RUnlock()
					if idx >= 0 && idx < len(iter.Value) {
						return &object.Integer{Value: int64(iter.Value[idx])}
					}
					return Null
				default:
					return Null
				}
			},
		},
		"__iter_item": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return Null
				}
				idxObj, ok := args[1].(*object.Integer)
				if !ok {
					return Null
				}
				idx := int(idxObj.Value)
				switch iter := args[0].(type) {
				case *object.Array:
					iter.Mu.RLock()
					defer iter.Mu.RUnlock()
					if idx >= 0 && idx < len(iter.Elements) {
						return iter.Elements[idx]
					}
					return Null
				case *object.Hash:
					keys := getSortedHashKeys(iter)
					if idx >= 0 && idx < len(keys) {
						return keys[idx]
					}
					return Null
				case *object.String:
					runes := []rune(iter.Value)
					if idx >= 0 && idx < len(runes) {
						return &object.String{Value: string(runes[idx])}
					}
					return Null
				case *object.Bytes:
					iter.Mu.RLock()
					defer iter.Mu.RUnlock()
					if idx >= 0 && idx < len(iter.Value) {
						return &object.Integer{Value: int64(iter.Value[idx])}
					}
					return Null
				default:
					return Null
				}
			},
		},
		"push": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				if args[0].Type() != object.ARRAY_OBJ {
					return &object.Error{Message: fmt.Sprintf("first argument to `push` must be ARRAY, got %s", args[0].Type())}
				}
				arr := args[0].(*object.Array)
				arr.Mu.Lock()
				arr.Elements = append(arr.Elements, args[1])
				arr.Mu.Unlock()
				return arr
			},
		},
		"append": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				if args[0].Type() != object.ARRAY_OBJ {
					return &object.Error{Message: fmt.Sprintf("first argument to `append` must be ARRAY, got %s", args[0].Type())}
				}
				arr := args[0].(*object.Array)
				arr.Mu.Lock()
				arr.Elements = append(arr.Elements, args[1])
				arr.Mu.Unlock()
				return arr
			},
		},
		"pop": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `pop` must be ARRAY, got %s", args[0].Type())}
				}
				arr.Mu.Lock()
				defer arr.Mu.Unlock()
				if len(arr.Elements) == 0 {
					return Null
				}
				lastIdx := len(arr.Elements) - 1
				lastElem := arr.Elements[lastIdx]
				arr.Elements = arr.Elements[:lastIdx]
				return lastElem
			},
		},
		"keys": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `keys` must be HASH, got %s", args[0].Type())}
				}
				hash.Mu.RLock()
				defer hash.Mu.RUnlock()
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Key)
				}
				return &object.Array{Elements: elements}
			},
		},
		"values": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `values` must be HASH, got %s", args[0].Type())}
				}
				hash.Mu.RLock()
				defer hash.Mu.RUnlock()
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Value)
				}
				return &object.Array{Elements: elements}
			},
		},
		"slice": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 4 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2, 3 or 4", len(args))}
				}

				var startObj, endObj, stepObj object.Object
				if len(args) > 1 {
					startObj = args[1]
				}
				if len(args) > 2 {
					endObj = args[2]
				}
				if len(args) > 3 {
					stepObj = args[3]
				}

				switch target := args[0].(type) {
				case *object.Array:
					target.Mu.RLock()
					defer target.Mu.RUnlock()
					length := int64(len(target.Elements))
					s, e, step, err := computeSliceBounds(length, startObj, endObj, stepObj)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					newElements := make([]object.Object, 0)
					if step > 0 {
						for i := s; i < e; i += step {
							newElements = append(newElements, target.Elements[i])
						}
					} else {
						for i := s; i > e; i += step {
							if i >= 0 && i < length {
								newElements = append(newElements, target.Elements[i])
							}
						}
					}
					return &object.Array{Elements: newElements}

				case *object.String:
					runes := []rune(target.Value)
					length := int64(len(runes))
					s, e, step, err := computeSliceBounds(length, startObj, endObj, stepObj)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					resRunes := make([]rune, 0)
					if step > 0 {
						for i := s; i < e; i += step {
							resRunes = append(resRunes, runes[i])
						}
					} else {
						for i := s; i > e; i += step {
							if i >= 0 && i < length {
								resRunes = append(resRunes, runes[i])
							}
						}
					}
					return &object.String{Value: string(resRunes)}

				case *object.Bytes:
					target.Mu.RLock()
					defer target.Mu.RUnlock()
					length := int64(len(target.Value))
					s, e, step, err := computeSliceBounds(length, startObj, endObj, stepObj)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					resBytes := make([]byte, 0)
					if step > 0 {
						for i := s; i < e; i += step {
							resBytes = append(resBytes, target.Value[i])
						}
					} else {
						for i := s; i > e; i += step {
							if i >= 0 && i < length {
								resBytes = append(resBytes, target.Value[i])
							}
						}
					}
					return &object.Bytes{Value: resBytes}

				default:
					return &object.Error{Message: fmt.Sprintf("first argument to `slice` must be ARRAY, STRING, or BYTES, got %s", args[0].Type())}
				}
			},
		},
		"substring": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				start, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "second argument to `substring` must be INTEGER"}
				}

				switch target := args[0].(type) {
				case *object.String:
					runes := []rune(target.Value)
					length := int64(len(runes))
					s := start.Value
					if s < 0 {
						s = 0
					}
					if s > length {
						s = length
					}

					e := length
					if len(args) == 3 {
						end, ok := args[2].(*object.Integer)
						if !ok {
							return &object.Error{Message: "third argument to `substring` must be INTEGER"}
						}
						e = end.Value
						if e > length {
							e = length
						}
						if e < s {
							e = s
						}
					}

					return &object.String{Value: string(runes[s:e])}
				default:
					return &object.Error{Message: fmt.Sprintf("first argument to `substring` must be STRING, got %s", args[0].Type())}
				}
			},
		},
		"contains": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				switch coll := args[0].(type) {
				case *object.Array:
					coll.Mu.RLock()
					defer coll.Mu.RUnlock()
					for _, elem := range coll.Elements {
						if ObjectsEqual(elem, args[1]) {
							return True
						}
					}
					return False
				case *object.Hash:
					if key, ok := args[1].(object.Hashable); ok {
						coll.Mu.RLock()
						_, exists := coll.Pairs[key.HashKey()]
						coll.Mu.RUnlock()
						return nativeBoolToBooleanObject(exists)
					}
					return False
				case *object.String:
					searchStr := args[1].Inspect()
					if strObj, ok := args[1].(*object.String); ok {
						searchStr = strObj.Value
					}
					return nativeBoolToBooleanObject(strings.Contains(coll.Value, searchStr))
				default:
					return &object.Error{Message: fmt.Sprintf("contains not supported on %s", args[0].Type())}
				}
			},
		},
		"delete": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `delete` must be HASH, got %s", args[0].Type())}
				}
				key, ok := args[1].(object.Hashable)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("unusable as hash key: %s", args[1].Type())}
				}
				hash.Mu.Lock()
				delete(hash.Pairs, key.HashKey())
				hash.Mu.Unlock()
				return Null
			},
		},
		"sort": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `sort` must be ARRAY, got %s", args[0].Type())}
				}
				arr.Mu.RLock()
				newElems := make([]object.Object, len(arr.Elements))
				copy(newElems, arr.Elements)
				arr.Mu.RUnlock()

				typeRank := func(o object.Object) int {
					switch o.(type) {
					case *object.Null:
						return 0
					case *object.Boolean:
						return 1
					case *object.Integer, *object.Float, *object.BigInt, *object.BigFloat:
						return 2
					case *object.String:
						return 3
					case *object.Array:
						return 4
					case *object.Hash:
						return 5
					default:
						return 6
					}
				}
				sort.Slice(newElems, func(i, j int) bool {
					o1 := newElems[i]
					o2 := newElems[j]
					r1, r2 := typeRank(o1), typeRank(o2)
					if r1 != r2 {
						return r1 < r2
					}
					switch v1 := o1.(type) {
					case *object.Null:
						return false
					case *object.Boolean:
						if v2, ok := o2.(*object.Boolean); ok {
							return !v1.Value && v2.Value
						}
					case *object.Integer:
						if v2, ok := o2.(*object.Integer); ok {
							return v1.Value < v2.Value
						}
						if v2, ok := o2.(*object.Float); ok {
							return float64(v1.Value) < v2.Value
						}
					case *object.Float:
						if v2, ok := o2.(*object.Float); ok {
							return v1.Value < v2.Value
						}
						if v2, ok := o2.(*object.Integer); ok {
							return v1.Value < float64(v2.Value)
						}
					case *object.String:
						if v2, ok := o2.(*object.String); ok {
							return v1.Value < v2.Value
						}
					}
					return o1.Inspect() < o2.Inspect()
				})
				return &object.Array{Elements: newElems}
			},
		},
		"reverse": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1", len(args))}
				}
				switch col := args[0].(type) {
				case *object.Array:
					col.Mu.RLock()
					defer col.Mu.RUnlock()
					n := len(col.Elements)
					newElems := make([]object.Object, n)
					for i := 0; i < n; i++ {
						newElems[i] = col.Elements[n-1-i]
					}
					return &object.Array{Elements: newElems}
				case *object.String:
					runes := []rune(col.Value)
					n := len(runes)
					for i := 0; i < n/2; i++ {
						runes[i], runes[n-1-i] = runes[n-1-i], runes[i]
					}
					return &object.String{Value: string(runes)}
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `reverse` must be ARRAY or STRING, got %s", args[0].Type())}
				}
			},
		},
	})
}
