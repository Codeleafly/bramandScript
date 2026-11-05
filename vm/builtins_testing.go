package vm

import (
	"fmt"
	"time"

	"bramandscript/object"
)

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"assert": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to assert. got=%d, want 1 or 2", len(args)), Fatal: true}
				}
				if !isTruthy(args[0]) {
					msg := "assertion failed"
					if len(args) == 2 {
						msg = args[1].Inspect()
						if s, ok := args[1].(*object.String); ok {
							msg = s.Value
						}
					}
					return &object.Error{Message: fmt.Sprintf("AssertionError: %s", msg), Fatal: true}
				}
				return Null
			},
		},
		"assert_true": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to assert_true. got=%d, want 1 or 2", len(args)), Fatal: true}
				}
				if !isTruthy(args[0]) {
					msg := "assertion failed"
					if len(args) == 2 {
						msg = args[1].Inspect()
						if s, ok := args[1].(*object.String); ok {
							msg = s.Value
						}
					}
					return &object.Error{Message: fmt.Sprintf("AssertionError: %s", msg), Fatal: true}
				}
				return Null
			},
		},
		"assert_false": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to assert_false. got=%d, want 1 or 2", len(args)), Fatal: true}
				}
				if isTruthy(args[0]) {
					msg := "assertion failed"
					if len(args) == 2 {
						msg = args[1].Inspect()
						if s, ok := args[1].(*object.String); ok {
							msg = s.Value
						}
					}
					return &object.Error{Message: fmt.Sprintf("AssertionError: %s", msg), Fatal: true}
				}
				return Null
			},
		},
		"assert_eq": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to assert_eq. got=%d, want 2 or 3", len(args)), Fatal: true}
				}
				actual := args[0]
				expected := args[1]
				if !ObjectsEqual(actual, expected) {
					extra := ""
					if len(args) == 3 {
						extra = fmt.Sprintf("\n  Message:  %s", args[2].Inspect())
						if s, ok := args[2].(*object.String); ok {
							extra = fmt.Sprintf("\n  Message:  %s", s.Value)
						}
					}
					errMsg := fmt.Sprintf("AssertionError: equality check failed\n  Expected: %s\n  Actual:   %s%s",
						expected.Inspect(), actual.Inspect(), extra)
					return &object.Error{Message: errMsg, Fatal: true}
				}
				return Null
			},
		},
		"assert_ne": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments to assert_ne. got=%d, want 2 or 3", len(args)), Fatal: true}
				}
				actual := args[0]
				expected := args[1]
				if ObjectsEqual(actual, expected) {
					extra := ""
					if len(args) == 3 {
						extra = fmt.Sprintf("\n  Message:  %s", args[2].Inspect())
						if s, ok := args[2].(*object.String); ok {
							extra = fmt.Sprintf("\n  Message:  %s", s.Value)
						}
					}
					errMsg := fmt.Sprintf("AssertionError: values should not be equal\n  Value:    %s%s",
						actual.Inspect(), extra)
					return &object.Error{Message: errMsg, Fatal: true}
				}
				return Null
			},
		},
		"panic": {
			Fn: func(args ...object.Object) object.Object {
				msg := "explicit panic"
				if len(args) > 0 {
					msg = args[0].Inspect()
					if s, ok := args[0].(*object.String); ok {
						msg = s.Value
					}
				}
				return &object.Error{Message: fmt.Sprintf("Panic: %s", msg), Fatal: true}
			},
		},
		"test": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "test requires 2 arguments: (name, function)", Fatal: true}
				}
				nameStr := args[0].Inspect()
				if s, ok := args[0].(*object.String); ok {
					nameStr = s.Value
				}
				callable := args[1]

				start := time.Now()
				_, err := ExecuteCallable(callable)
				elapsed := time.Since(start)

				ctx := GetTestContext()
				ctx.mu.Lock()
				suiteName := ctx.ActiveSuite
				silent := ctx.SilentMode

				res := TestCaseResult{
					Name:     nameStr,
					Suite:    suiteName,
					Passed:   err == nil,
					Duration: elapsed,
				}

				if err != nil {
					res.Error = err.Error()
					ctx.TotalFailed++
					if !silent {
						fmt.Printf("  \033[31m[FAIL] %s (%v)\033[0m\n", nameStr, elapsed)
						fmt.Printf("    \033[31m--> %s\033[0m\n", err.Error())
					}
				} else {
					ctx.TotalPassed++
					if !silent {
						fmt.Printf("  \033[32m[OK] %s (%v)\033[0m\n", nameStr, elapsed)
					}
				}

				ctx.Cases = append(ctx.Cases, res)
				ctx.mu.Unlock()
				return nativeBoolToBooleanObject(err == nil)
			},
		},
		"describe": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "describe requires 2 arguments: (suite_name, function)", Fatal: true}
				}
				suiteName := args[0].Inspect()
				if s, ok := args[0].(*object.String); ok {
					suiteName = s.Value
				}
				callable := args[1]

				ctx := GetTestContext()
				ctx.mu.Lock()
				oldSuite := ctx.ActiveSuite
				ctx.ActiveSuite = suiteName
				silent := ctx.SilentMode
				ctx.mu.Unlock()

				if !silent {
					fmt.Printf("\n• %s\n", suiteName)
				}

				_, _ = ExecuteCallable(callable)

				ctx.mu.Lock()
				ctx.ActiveSuite = oldSuite
				ctx.mu.Unlock()

				return Null
			},
		},
		"benchmark": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "benchmark requires at least 2 arguments: (name, function, [iterations])", Fatal: true}
				}
				nameStr := args[0].Inspect()
				if s, ok := args[0].(*object.String); ok {
					nameStr = s.Value
				}
				callable := args[1]
				iters := 1000
				if len(args) >= 3 {
					if i, ok := args[2].(*object.Integer); ok && i.Value > 0 {
						iters = int(i.Value)
					}
				}

				// Warmup
				_, _ = ExecuteCallable(callable)

				start := time.Now()
				for i := 0; i < iters; i++ {
					_, err := ExecuteCallable(callable)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("benchmark %s failed: %s", nameStr, err), Fatal: true}
					}
				}
				elapsed := time.Since(start)
				avgPerOp := elapsed / time.Duration(iters)
				opsPerSec := float64(iters) / elapsed.Seconds()

				fmt.Printf("✦ Benchmark %-28s %8d iters   %10v total   %10v/op   (%8.0f ops/sec)\n",
					nameStr, iters, elapsed, avgPerOp, opsPerSec)

				pairs := make(map[object.HashKey]object.HashPair)
				pairs[(&object.String{Value: "name"}).HashKey()] = object.HashPair{
					Key: &object.String{Value: "name"}, Value: &object.String{Value: nameStr},
				}
				pairs[(&object.String{Value: "iterations"}).HashKey()] = object.HashPair{
					Key: &object.String{Value: "iterations"}, Value: &object.Integer{Value: int64(iters)},
				}
				pairs[(&object.String{Value: "total_ms"}).HashKey()] = object.HashPair{
					Key: &object.String{Value: "total_ms"}, Value: &object.Float{Value: float64(elapsed.Microseconds()) / 1000.0},
				}
				pairs[(&object.String{Value: "ops_per_sec"}).HashKey()] = object.HashPair{
					Key: &object.String{Value: "ops_per_sec"}, Value: &object.Float{Value: opsPerSec},
				}
				return &object.Hash{Pairs: pairs}
			},
		},
	})
}
