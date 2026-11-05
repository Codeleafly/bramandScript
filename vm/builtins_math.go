package vm

import (
	"fmt"
	"math"
	"math/big"

	"bramandscript/object"
)

func init() {
	bigIntBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			switch a := args[0].(type) {
			case *object.Integer:
				return &object.BigInt{Value: big.NewInt(a.Value)}
			case *object.String:
				bi := new(big.Int)
				bi, ok := bi.SetString(a.Value, 0)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("invalid string for big_int: %q", a.Value)}
				}
				return &object.BigInt{Value: bi}
			case *object.BigInt:
				return &object.BigInt{Value: new(big.Int).Set(a.Value)}
			default:
				return &object.Error{Message: fmt.Sprintf("cannot convert %s to big_int", args[0].Type())}
			}
		},
	}
	RegisterBuiltinAliases(bigIntBuiltin, "big_int", "bigint")

	bigFloatBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 || len(args) > 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
			}
			prec := uint(128)
			if len(args) == 2 {
				if p, ok := args[1].(*object.Integer); ok && p.Value > 0 {
					prec = uint(p.Value)
				}
			}
			bf := new(big.Float).SetPrec(prec)
			switch a := args[0].(type) {
			case *object.Float:
				bf.SetFloat64(a.Value)
				return &object.BigFloat{Value: bf}
			case *object.Integer:
				bf.SetInt64(a.Value)
				return &object.BigFloat{Value: bf}
			case *object.String:
				bf, ok := bf.SetString(a.Value)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("invalid string for big_float: %q", a.Value)}
				}
				return &object.BigFloat{Value: bf}
			case *object.BigInt:
				bf.SetInt(a.Value)
				return &object.BigFloat{Value: bf}
			case *object.BigFloat:
				return &object.BigFloat{Value: new(big.Float).Copy(a.Value)}
			default:
				return &object.Error{Message: fmt.Sprintf("cannot convert %s to big_float", args[0].Type())}
			}
		},
	}
	RegisterBuiltinAliases(bigFloatBuiltin, "big_float", "bigfloat")

	vecBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var elements []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						v, ok := toFloat(el)
						if !ok {
							return &object.Error{Message: fmt.Sprintf("vector elements must be numbers, got %s", el.Type())}
						}
						elements = append(elements, v)
					}
				} else {
					v, ok := toFloat(args[0])
					if !ok {
						return &object.Error{Message: fmt.Sprintf("vector argument must be array or numbers, got %s", args[0].Type())}
					}
					elements = append(elements, v)
				}
			} else {
				for _, arg := range args {
					v, ok := toFloat(arg)
					if !ok {
						return &object.Error{Message: fmt.Sprintf("vector elements must be numbers, got %s", arg.Type())}
					}
					elements = append(elements, v)
				}
			}
			return &object.Vector{Elements: elements}
		},
	}
	RegisterBuiltinAliases(vecBuiltin, "vec", "vector")

	RegisterBuiltin("matrix", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) == 1 {
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `matrix` must be 2D array, got %s", args[0].Type())}
				}
				rows := len(arr.Elements)
				if rows == 0 {
					return object.NewMatrix(0, 0)
				}
				cols := -1
				var data [][]float64
				for r, rowObj := range arr.Elements {
					rowArr, ok := rowObj.(*object.Array)
					if !ok {
						return &object.Error{Message: fmt.Sprintf("row %d of matrix must be array, got %s", r, rowObj.Type())}
					}
					if cols == -1 {
						cols = len(rowArr.Elements)
					} else if len(rowArr.Elements) != cols {
						return &object.Error{Message: fmt.Sprintf("inconsistent row length in matrix at row %d", r)}
					}
					var rowData []float64
					for _, el := range rowArr.Elements {
						v, ok := toFloat(el)
						if !ok {
							return &object.Error{Message: fmt.Sprintf("matrix elements must be numbers, got %s", el.Type())}
						}
						rowData = append(rowData, v)
					}
					data = append(data, rowData)
				}
				return &object.Matrix{Rows: rows, Cols: cols, Data: data}
			} else if len(args) >= 2 {
				rObj, ok1 := args[0].(*object.Integer)
				cObj, ok2 := args[1].(*object.Integer)
				if !ok1 || !ok2 {
					return &object.Error{Message: "matrix dimensions must be integers"}
				}
				m := object.NewMatrix(int(rObj.Value), int(cObj.Value))
				if len(args) >= 3 {
					if arr, ok := args[2].(*object.Array); ok {
						idx := 0
						for i := 0; i < m.Rows; i++ {
							for j := 0; j < m.Cols; j++ {
								if idx < len(arr.Elements) {
									if f, ok := toFloat(arr.Elements[idx]); ok {
										m.Data[i][j] = f
									}
									idx++
								}
							}
						}
						return m
					}
					fill := 0.0
					if f, ok := toFloat(args[2]); ok {
						fill = f
					}
					for i := 0; i < m.Rows; i++ {
						for j := 0; j < m.Cols; j++ {
							m.Data[i][j] = fill
						}
					}
					return m
				}
				return m
			}
			return &object.Error{Message: "wrong number of arguments to `matrix`"}
		},
	})

	RegisterBuiltin("pi", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.Pi}
		},
	})

	RegisterBuiltin("e", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: math.E}
		},
	})

	RegisterBuiltin("sin", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `sin` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Sin(val)}
		},
	})

	RegisterBuiltin("cos", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `cos` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Cos(val)}
		},
	})

	RegisterBuiltin("tan", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `tan` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Tan(val)}
		},
	})

	RegisterBuiltin("asin", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `asin` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Asin(val)}
		},
	})

	RegisterBuiltin("acos", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `acos` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Acos(val)}
		},
	})

	RegisterBuiltin("atan", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `atan` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Atan(val)}
		},
	})

	RegisterBuiltin("atan2", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
			}
			y, ok1 := toFloat(args[0])
			x, ok2 := toFloat(args[1])
			if !ok1 || !ok2 {
				return &object.Error{Message: "arguments to `atan2` must be numbers"}
			}
			return &object.Float{Value: math.Atan2(y, x)}
		},
	})

	RegisterBuiltin("sinh", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `sinh` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Sinh(val)}
		},
	})

	RegisterBuiltin("cosh", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `cosh` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Cosh(val)}
		},
	})

	RegisterBuiltin("tanh", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `tanh` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Tanh(val)}
		},
	})

	RegisterBuiltin("sqrt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `sqrt` must be number, got %s", args[0].Type())}
			}
			if val < 0 {
				return &object.Error{Message: "square root of negative number"}
			}
			return &object.Float{Value: math.Sqrt(val)}
		},
	})

	RegisterBuiltin("cbrt", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `cbrt` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Cbrt(val)}
		},
	})

	RegisterBuiltin("hypot", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
			}
			a, ok1 := toFloat(args[0])
			b, ok2 := toFloat(args[1])
			if !ok1 || !ok2 {
				return &object.Error{Message: "arguments to `hypot` must be numbers"}
			}
			return &object.Float{Value: math.Hypot(a, b)}
		},
	})

	RegisterBuiltin("log", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `log` must be number, got %s", args[0].Type())}
			}
			if val <= 0 {
				return &object.Error{Message: "logarithm of non-positive number"}
			}
			return &object.Float{Value: math.Log(val)}
		},
	})

	RegisterBuiltin("log10", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `log10` must be number, got %s", args[0].Type())}
			}
			if val <= 0 {
				return &object.Error{Message: "log10 of non-positive number"}
			}
			return &object.Float{Value: math.Log10(val)}
		},
	})

	RegisterBuiltin("log2", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `log2` must be number, got %s", args[0].Type())}
			}
			if val <= 0 {
				return &object.Error{Message: "log2 of non-positive number"}
			}
			return &object.Float{Value: math.Log2(val)}
		},
	})

	RegisterBuiltin("exp", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `exp` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: math.Exp(val)}
		},
	})

	RegisterBuiltin("pow", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
			}
			base, ok1 := toFloat(args[0])
			exp, ok2 := toFloat(args[1])
			if !ok1 || !ok2 {
				return &object.Error{Message: "arguments to `pow` must be numbers"}
			}
			return &object.Float{Value: math.Pow(base, exp)}
		},
	})

	RegisterBuiltin("abs", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			switch a := args[0].(type) {
			case *object.Integer:
				if a.Value < 0 {
					return &object.Integer{Value: -a.Value}
				}
				return a
			case *object.Float:
				return &object.Float{Value: math.Abs(a.Value)}
			case *object.BigInt:
				return a.Abs()
			case *object.BigFloat:
				return a.Abs()
			default:
				return &object.Error{Message: fmt.Sprintf("argument to `abs` must be number, got %s", args[0].Type())}
			}
		},
	})

	RegisterBuiltin("floor", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `floor` must be number, got %s", args[0].Type())}
			}
			return &object.Integer{Value: int64(math.Floor(val))}
		},
	})

	RegisterBuiltin("ceil", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `ceil` must be number, got %s", args[0].Type())}
			}
			return &object.Integer{Value: int64(math.Ceil(val))}
		},
	})

	RegisterBuiltin("round", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `round` must be number, got %s", args[0].Type())}
			}
			return &object.Integer{Value: int64(math.Round(val))}
		},
	})

	RegisterBuiltin("trunc", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `trunc` must be number, got %s", args[0].Type())}
			}
			return &object.Integer{Value: int64(math.Trunc(val))}
		},
	})

	RegisterBuiltin("deg_to_rad", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `deg_to_rad` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: val * math.Pi / 180.0}
		},
	})

	RegisterBuiltin("rad_to_deg", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
			}
			val, ok := toFloat(args[0])
			if !ok {
				return &object.Error{Message: fmt.Sprintf("argument to `rad_to_deg` must be number, got %s", args[0].Type())}
			}
			return &object.Float{Value: val * 180.0 / math.Pi}
		},
	})

	RegisterBuiltin("min", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						v, ok := toFloat(el)
						if !ok {
							return &object.Error{Message: "elements must be numbers"}
						}
						nums = append(nums, v)
					}
				} else if _, ok := toFloat(args[0]); ok {
					return args[0]
				}
			} else if len(args) > 1 {
				for _, el := range args {
					v, ok := toFloat(el)
					if !ok {
						return &object.Error{Message: "arguments must be numbers"}
					}
					nums = append(nums, v)
				}
			}
			if len(nums) == 0 {
				return Null
			}
			minVal := nums[0]
			for _, n := range nums[1:] {
				if n < minVal {
					minVal = n
				}
			}
			if math.Floor(minVal) == minVal {
				return &object.Integer{Value: int64(minVal)}
			}
			return &object.Float{Value: minVal}
		},
	})

	RegisterBuiltin("max", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						v, ok := toFloat(el)
						if !ok {
							return &object.Error{Message: "elements must be numbers"}
						}
						nums = append(nums, v)
					}
				} else if _, ok := toFloat(args[0]); ok {
					return args[0]
				}
			} else if len(args) > 1 {
				for _, el := range args {
					v, ok := toFloat(el)
					if !ok {
						return &object.Error{Message: "arguments must be numbers"}
					}
					nums = append(nums, v)
				}
			}
			if len(nums) == 0 {
				return Null
			}
			maxVal := nums[0]
			for _, n := range nums[1:] {
				if n > maxVal {
					maxVal = n
				}
			}
			if math.Floor(maxVal) == maxVal {
				return &object.Integer{Value: int64(maxVal)}
			}
			return &object.Float{Value: maxVal}
		},
	})

}
