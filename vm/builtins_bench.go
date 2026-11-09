package vm

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"bramandscript/object"
)

// BenchmarkController is passed to the benchmark function fn(b)
type benchmarkController struct {
	n         int64
	startTime time.Time
	duration  time.Duration
	running   bool
}

func newBenchmarkController(n int64) *object.Hash {
	h := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}

	setField := func(k string, v object.Object) {
		key := &object.String{Value: k}
		h.Pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
	}

	setField("n", &object.Integer{Value: n})

	var start time.Time
	var totalDuration time.Duration

	setMethod := func(name string, fn func(args ...object.Object) object.Object) {
		key := &object.String{Value: name}
		h.Pairs[key.HashKey()] = object.HashPair{
			Key:   key,
			Value: &object.Builtin{Fn: fn},
		}
	}

	setMethod("start_timer", func(args ...object.Object) object.Object {
		start = time.Now()
		return Null
	})

	setMethod("stop_timer", func(args ...object.Object) object.Object {
		if !start.IsZero() {
			totalDuration += time.Since(start)
			start = time.Time{}
		}
		return Null
	})

	setMethod("reset_timer", func(args ...object.Object) object.Object {
		start = time.Now()
		totalDuration = 0
		return Null
	})

	setMethod("running", func(args ...object.Object) object.Object {
		return True
	})

	return h
}

// RunBenchmark executes a benchmark function and returns the results map.
func RunBenchmark(name string, callable object.Object, targetDuration time.Duration) (map[string]interface{}, error) {
	if targetDuration <= 0 {
		targetDuration = 500 * time.Millisecond
	}

	// Warmup / calibration
	n := int64(1)
	bController := newBenchmarkController(n)

	var startMem, endMem runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&startMem)
	startTime := time.Now()

	_, err := ExecuteCallable(callable, bController)
	if err != nil {
		return nil, fmt.Errorf("benchmark execution error: %w", err)
	}

	elapsed := time.Since(startTime)
	if elapsed < targetDuration {
		// Estimate required iterations to run for target duration
		rate := float64(elapsed) / float64(n)
		if rate <= 0 {
			rate = 1
		}
		predictedN := int64(float64(targetDuration) / rate)
		if predictedN < 100 {
			predictedN = 100
		}
		if predictedN > 10000000 {
			predictedN = 10000000
		}
		n = predictedN

		bController = newBenchmarkController(n)
		runtime.GC()
		runtime.ReadMemStats(&startMem)
		startTime = time.Now()

		_, err = ExecuteCallable(callable, bController)
		if err != nil {
			return nil, fmt.Errorf("benchmark execution error: %w", err)
		}
		elapsed = time.Since(startTime)
	}

	runtime.ReadMemStats(&endMem)

	totalNs := elapsed.Nanoseconds()
	if n <= 0 {
		n = 1
	}
	nsPerOp := totalNs / n

	totalAllocs := int64(0)
	if endMem.Mallocs >= startMem.Mallocs {
		totalAllocs = int64(endMem.Mallocs - startMem.Mallocs)
	}
	allocsPerOp := totalAllocs / n

	totalBytes := int64(0)
	if endMem.TotalAlloc >= startMem.TotalAlloc {
		totalBytes = int64(endMem.TotalAlloc - startMem.TotalAlloc)
	}
	bytesPerOp := totalBytes / n

	opsPerSec := float64(0)
	if elapsed.Seconds() > 0 {
		opsPerSec = float64(n) / elapsed.Seconds()
	}

	return map[string]interface{}{
		"name":          name,
		"n":             n,
		"duration_ms":   elapsed.Seconds() * 1000.0,
		"ns_per_op":     nsPerOp,
		"ops_per_sec":   opsPerSec,
		"bytes_per_op":  bytesPerOp,
		"allocs_per_op": allocsPerOp,
	}, nil
}

func formatBenchmarkResult(res map[string]interface{}) string {
	name := fmt.Sprintf("%-28s", res["name"])
	n := fmt.Sprintf("%8d", res["n"])
	ns := fmt.Sprintf("%10d ns/op", res["ns_per_op"])
	bytes := fmt.Sprintf("%8d B/op", res["bytes_per_op"])
	allocs := fmt.Sprintf("%6d allocs/op", res["allocs_per_op"])
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", name, n, ns, bytes, allocs)
}

func init() {
	RegisterBuiltin("bench", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "bench requires at least 2 arguments: bench(name, fn)"}
			}

			name := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				name = s.Value
			}
			fn := args[1]

			duration := 200 * time.Millisecond
			if len(args) >= 3 {
				if dInt, ok := args[2].(*object.Integer); ok && dInt.Value > 0 {
					duration = time.Duration(dInt.Value) * time.Millisecond
				}
			}

			res, err := RunBenchmark(name, fn, duration)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			formatted := formatBenchmarkResult(res)
			fmt.Println(formatted)

			h := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
			setPair := func(k string, v object.Object) {
				key := &object.String{Value: k}
				h.Pairs[key.HashKey()] = object.HashPair{Key: key, Value: v}
			}
			setPair("name", &object.String{Value: name})
			setPair("n", &object.Integer{Value: res["n"].(int64)})
			setPair("ns_per_op", &object.Integer{Value: res["ns_per_op"].(int64)})
			setPair("duration_ms", &object.Float{Value: res["duration_ms"].(float64)})
			setPair("ops_per_sec", &object.Float{Value: res["ops_per_sec"].(float64)})
			setPair("bytes_per_op", &object.Integer{Value: res["bytes_per_op"].(int64)})
			setPair("allocs_per_op", &object.Integer{Value: res["allocs_per_op"].(int64)})
			setPair("formatted", &object.String{Value: strings.TrimSpace(formatted)})

			return h
		},
	})
}
