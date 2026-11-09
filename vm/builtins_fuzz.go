package vm

import (
	"fmt"
	"math/rand"
	"time"

	"bramandscript/object"
)

// MutateInput generates a mutated version of a seed input.
func MutateInput(seed string, rng *rand.Rand) string {
	if len(seed) == 0 {
		seeds := []string{"", "a", "0", "\x00", "\n", "999999999", "{\"key\":1}", "null", "true", "false", "-1", "NaN"}
		return seeds[rng.Intn(len(seeds))]
	}

	mutationType := rng.Intn(6)
	b := []byte(seed)

	switch mutationType {
	case 0: // Bit / Byte flip
		idx := rng.Intn(len(b))
		b[idx] ^= byte(1 << rng.Intn(8))
		return string(b)

	case 1: // Random Byte insertion
		idx := rng.Intn(len(b) + 1)
		randomByte := byte(rng.Intn(256))
		newBytes := make([]byte, 0, len(b)+1)
		newBytes = append(newBytes, b[:idx]...)
		newBytes = append(newBytes, randomByte)
		newBytes = append(newBytes, b[idx:]...)
		return string(newBytes)

	case 2: // Random Byte deletion
		if len(b) > 1 {
			idx := rng.Intn(len(b))
			newBytes := append(b[:idx], b[idx+1:]...)
			return string(newBytes)
		}
		return ""

	case 3: // Edge-case injection
		edges := []string{"\x00", "%s%d%n", "undefined", "NaN", "Infinity", "-1", "9223372036854775807", "-9223372036854775808", "''", "{\"a\":null}"}
		chosen := edges[rng.Intn(len(edges))]
		idx := rng.Intn(len(b) + 1)
		return string(b[:idx]) + chosen + string(b[idx:])

	case 4: // Repetition / Length expansion
		if len(seed) < 2048 {
			return seed + seed
		}
		return seed[:len(seed)/2]

	default: // Random swap
		if len(b) > 1 {
			i := rng.Intn(len(b))
			j := rng.Intn(len(b))
			b[i], b[j] = b[j], b[i]
			return string(b)
		}
		return seed
	}
}

// RunFuzz executes automated fuzz testing on a callable function.
func RunFuzz(targetFn object.Object, iterations int, initialSeeds []string) map[string]interface{} {
	if iterations <= 0 {
		iterations = 500
	}

	seeds := []string{"hello", "123", "{\"status\": \"ok\"}", "[1, 2, 3]", "http://localhost", "alpha_beta"}
	if len(initialSeeds) > 0 {
		seeds = initialSeeds
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	var crashed bool
	var crashInput string
	var crashError string
	var crashIteration int

	for i := 1; i <= iterations; i++ {
		seed := seeds[rng.Intn(len(seeds))]
		input := MutateInput(seed, rng)

		inputObj := &object.String{Value: input}

		func() {
			defer func() {
				if r := recover(); r != nil {
					crashed = true
					crashInput = input
					crashError = fmt.Sprintf("panic: %v", r)
					crashIteration = i
				}
			}()

			res, err := ExecuteCallable(targetFn, inputObj)
			if err != nil {
				crashed = true
				crashInput = input
				crashError = err.Error()
				crashIteration = i
			} else if errObj, ok := res.(*object.Error); ok {
				// If function explicitly returns unhandled error during fuzz
				crashed = true
				crashInput = input
				crashError = errObj.Message
				crashIteration = i
			}
		}()

		if crashed {
			break
		}
	}

	return map[string]interface{}{
		"crashed":         crashed,
		"iterations":      iterations,
		"crash_input":     crashInput,
		"crash_error":     crashError,
		"crash_iteration": crashIteration,
	}
}

func init() {
	RegisterBuiltin("fuzz", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "fuzz requires at least 1 callable argument"}
			}

			targetFn := args[0]
			iterations := 250
			var seeds []string

			if len(args) >= 2 {
				if intObj, ok := args[1].(*object.Integer); ok && intObj.Value > 0 {
					iterations = int(intObj.Value)
				} else if optHash, ok := args[1].(*object.Hash); ok {
					for _, pair := range optHash.Pairs {
						k := pair.Key.Inspect()
						if k == "iterations" || k == "count" {
							if intO, ok := pair.Value.(*object.Integer); ok {
								iterations = int(intO.Value)
							}
						} else if k == "corpus" || k == "seeds" {
							if arrO, ok := pair.Value.(*object.Array); ok {
								for _, el := range arrO.Elements {
									seeds = append(seeds, el.Inspect())
								}
							}
						}
					}
				}
			}

			res := RunFuzz(targetFn, iterations, seeds)

			h := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			setPair := func(k string, v object.Object) {
				key := &object.String{Value: k}
				h.Pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
			}

			crashed := res["crashed"].(bool)
			if crashed {
				setPair("crashed", True)
				setPair("crash_input", &object.String{Value: res["crash_input"].(string)})
				setPair("crash_error", &object.String{Value: res["crash_error"].(string)})
				setPair("crash_iteration", &object.Integer{Value: int64(res["crash_iteration"].(int))})
			} else {
				setPair("crashed", False)
				setPair("crash_input", Null)
				setPair("crash_error", Null)
				setPair("crash_iteration", &object.Integer{Value: 0})
			}
			setPair("iterations", &object.Integer{Value: int64(res["iterations"].(int))})

			return h
		},
	})
}
