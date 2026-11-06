package vm

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	"bramandscript/object"
)

func (vm *VM) executeMathAndConcurrencyMethod(left object.Object, index object.Object) (bool, error) {
	switch {
	case left.Type() == object.MUTEX_OBJ && index.Type() == object.STRING_OBJ:
		mtx := left.(*object.Mutex)
		prop := index.(*object.String).Value
		switch prop {
		case "lock":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					mtx.Lock()
					return Null
				},
			})
		case "unlock":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					mtx.Unlock()
					return Null
				},
			})
		case "try_lock":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return nativeBoolToBooleanObject(mtx.TryLock())
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on mutex", prop)
		}

	case left.Type() == object.WAIT_GROUP_OBJ && index.Type() == object.STRING_OBJ:
		wg := left.(*object.WaitGroup)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					wg.Done()
					return Null
				},
			})
		case "wait":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					wg.Wait()
					return Null
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on wait_group", prop)
		}

	case left.Type() == object.CHANNEL_OBJ && index.Type() == object.STRING_OBJ:
		ch := left.(*object.Channel)
		prop := index.(*object.String).Value
		switch prop {
		case "send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) < 1 || len(args) > 2 {
						return &object.Error{Message: fmt.Sprintf("send requires 1 or 2 arguments, got %d", len(args))}
					}
					var timeout time.Duration
					if len(args) == 2 {
						if intObj, ok := args[1].(*object.Integer); ok && intObj.Value > 0 {
							timeout = time.Duration(intObj.Value) * time.Millisecond
						}
					}
					err := ch.SendTimeout(args[0], timeout)
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "try_send":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: fmt.Sprintf("try_send requires 1 argument, got %d", len(args))}
					}
					if ch.TrySend(args[0]) {
						return True
					}
					return False
				},
			})
		case "recv", "receive":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) > 1 {
						return &object.Error{Message: fmt.Sprintf("recv requires 0 or 1 arguments, got %d", len(args))}
					}
					var timeout time.Duration
					if len(args) == 1 {
						if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
							timeout = time.Duration(intObj.Value) * time.Millisecond
						}
					}
					val, ok := ch.RecvTimeout(timeout)
					if !ok || val == nil {
						return Null
					}
					return val
				},
			})
		case "try_recv":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 0 {
						return &object.Error{Message: fmt.Sprintf("try_recv requires 0 arguments, got %d", len(args))}
					}
					val, ok := ch.TryRecv()
					if !ok || val == nil {
						return Null
					}
					return val
				},
			})
		case "close":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					err := ch.Close()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return Null
				},
			})
		case "capacity", "cap":
			return true, vm.push(&object.Integer{Value: int64(ch.Capacity)})
		default:
			return true, fmt.Errorf("unknown method '%s' on channel", prop)
		}

	case left.Type() == object.VECTOR_OBJ && index.Type() == object.INTEGER_OBJ:
		v := left.(*object.Vector)
		i := index.(*object.Integer).Value
		if i < 0 || i >= int64(len(v.Elements)) {
			return true, vm.push(Null)
		}
		return true, vm.push(&object.Float{Value: v.Elements[i]})

	case left.Type() == object.VECTOR_OBJ && index.Type() == object.STRING_OBJ:
		v := left.(*object.Vector)
		prop := index.(*object.String).Value
		switch prop {
		case "dim", "len", "length":
			return true, vm.push(&object.Integer{Value: int64(v.Dim())})
		case "elements":
			return true, vm.push(v.ToArray())
		case "norm", "magnitude":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Float{Value: v.Norm()}
				},
			})
		case "normalize":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					res, err := v.Normalize()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "dot":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return v.ToArray()
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on Vector", prop)
		}

	case left.Type() == object.MATRIX_OBJ && index.Type() == object.INTEGER_OBJ:
		m := left.(*object.Matrix)
		i := index.(*object.Integer).Value
		if i < 0 || i >= int64(m.Rows) {
			return true, vm.push(Null)
		}
		cols := make([]object.Object, m.Cols)
		for j := 0; j < m.Cols; j++ {
			cols[j] = &object.Float{Value: m.Data[i][j]}
		}
		return true, vm.push(&object.Array{Elements: cols})

	case left.Type() == object.MATRIX_OBJ && index.Type() == object.STRING_OBJ:
		m := left.(*object.Matrix)
		prop := index.(*object.String).Value
		switch prop {
		case "rows":
			return true, vm.push(&object.Integer{Value: int64(m.Rows)})
		case "cols":
			return true, vm.push(&object.Integer{Value: int64(m.Cols)})
		case "data", "elements":
			return true, vm.push(m.ToArray())
		case "get":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return m.Transpose()
				},
			})
		case "det":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					d, err := m.Det()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return &object.Float{Value: d}
				},
			})
		case "inv":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					inv, err := m.Inv()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return inv
				},
			})
		case "to_array":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return m.ToArray()
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on Matrix", prop)
		}

	case left.Type() == object.INTEGER_OBJ && index.Type() == object.STRING_OBJ:
		iv := left.(*object.Integer).Value
		prop := index.(*object.String).Value
		switch prop {
		case "str", "to_str", "to_string":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: strconv.FormatInt(iv, 10)}
				},
			})
		case "abs":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if iv < 0 {
						if iv == math.MinInt64 {
							return &object.BigInt{Value: new(big.Int).Abs(big.NewInt(iv))}
						}
						return &object.Integer{Value: -iv}
					}
					return &object.Integer{Value: iv}
				},
			})
		case "cmp":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					if len(args) != 1 {
						return &object.Error{Message: "cmp requires 1 argument"}
					}
					r, ok := toBigInt(args[0])
					if !ok {
						return &object.Error{Message: "argument to cmp must be integer or big_int"}
					}
					return &object.Integer{Value: int64(big.NewInt(iv).Cmp(r))}
				},
			})
		case "to_float":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Float{Value: float64(iv)}
				},
			})
		case "to_big":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.BigInt{Value: big.NewInt(iv)}
				},
			})
		case "bit_len":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Integer{Value: int64(big.NewInt(iv).BitLen())}
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on Integer", prop)
		}

	case left.Type() == object.FLOAT_OBJ && index.Type() == object.STRING_OBJ:
		fv := left.(*object.Float).Value
		prop := index.(*object.String).Value
		switch prop {
		case "str", "to_str", "to_string":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: strconv.FormatFloat(fv, 'g', -1, 64)}
				},
			})
		case "abs":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Float{Value: math.Abs(fv)}
				},
			})
		case "to_int":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.Integer{Value: int64(fv)}
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on Float", prop)
		}

	case left.Type() == object.BIG_INT_OBJ && index.Type() == object.STRING_OBJ:
		bi := left.(*object.BigInt)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bi.Abs()
				},
			})
		case "neg":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bi.Neg()
				},
			})
		case "str", "to_str", "to_string":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return &object.String{Value: bi.Value.String()}
				},
			})
		default:
			return true, fmt.Errorf("unknown method '%s' on BigInt", prop)
		}

	case left.Type() == object.BIG_FLOAT_OBJ && index.Type() == object.STRING_OBJ:
		bf := left.(*object.BigFloat)
		prop := index.(*object.String).Value
		switch prop {
		case "add":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					res, err := bf.Sqrt()
					if err != nil {
						return &object.Error{Message: err.Error()}
					}
					return res
				},
			})
		case "cmp":
			return true, vm.push(&object.Builtin{
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
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bf.Abs()
				},
			})
		case "neg":
			return true, vm.push(&object.Builtin{
				Fn: func(args ...object.Object) object.Object {
					return bf.Neg()
				},
			})
		case "str", "to_str", "to_string":
			return true, vm.push(&object.Builtin{
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
			return true, fmt.Errorf("unknown method '%s' on BigFloat", prop)
		}
	}

	return false, nil
}
