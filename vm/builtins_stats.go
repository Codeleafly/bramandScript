package vm

import (
	"math"
	"sort"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("sum", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			var sum float64
			for _, n := range nums {
				sum += n
			}
			if math.Floor(sum) == sum {
				return &object.Integer{Value: int64(sum)}
			}
			return &object.Float{Value: sum}
		},
	})

	meanBuiltin := &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			if len(nums) == 0 {
				return Null
			}
			var sum float64
			for _, n := range nums {
				sum += n
			}
			return &object.Float{Value: sum / float64(len(nums))}
		},
	}
	RegisterBuiltinAliases(meanBuiltin, "mean", "math_mean")

	RegisterBuiltin("median", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			if len(nums) == 0 {
				return Null
			}
			sort.Float64s(nums)
			n := len(nums)
			if n%2 == 1 {
				return &object.Float{Value: nums[n/2]}
			}
			return &object.Float{Value: (nums[n/2-1] + nums[n/2]) / 2.0}
		},
	})

	RegisterBuiltin("mode", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			if len(nums) == 0 {
				return Null
			}
			counts := make(map[float64]int)
			maxCount := 0
			var modeVal float64
			for _, n := range nums {
				counts[n]++
				if counts[n] > maxCount {
					maxCount = counts[n]
					modeVal = n
				}
			}
			if math.Floor(modeVal) == modeVal {
				return &object.Integer{Value: int64(modeVal)}
			}
			return &object.Float{Value: modeVal}
		},
	})

	RegisterBuiltin("variance", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			if len(nums) == 0 {
				return Null
			}
			var sum float64
			for _, n := range nums {
				sum += n
			}
			mean := sum / float64(len(nums))
			var sumSq float64
			for _, n := range nums {
				d := n - mean
				sumSq += d * d
			}
			return &object.Float{Value: sumSq / float64(len(nums))}
		},
	})

	RegisterBuiltin("std_dev", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var nums []float64
			if len(args) == 1 {
				if arr, ok := args[0].(*object.Array); ok {
					for _, el := range arr.Elements {
						if v, ok := toFloat(el); ok {
							nums = append(nums, v)
						}
					}
				}
			} else {
				for _, arg := range args {
					if v, ok := toFloat(arg); ok {
						nums = append(nums, v)
					}
				}
			}
			if len(nums) == 0 {
				return Null
			}
			var sum float64
			for _, n := range nums {
				sum += n
			}
			mean := sum / float64(len(nums))
			var sumSq float64
			for _, n := range nums {
				d := n - mean
				sumSq += d * d
			}
			return &object.Float{Value: math.Sqrt(sumSq / float64(len(nums)))}
		},
	})
}
