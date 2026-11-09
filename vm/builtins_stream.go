package vm

import (
	"sync"
	"time"

	"bramandscript/object"
)

// StreamProducer is a function that emits items downstream until done or cancelled.
// emit returns false if the downstream no longer accepts items (e.g. cancelled).
type StreamProducer func(emit func(val object.Object) bool, done func())

// streamInstance represents an active or pipeline reactive stream.
type streamInstance struct {
	producer StreamProducer
}

func newStreamInstance(producer StreamProducer) *streamInstance {
	return &streamInstance{producer: producer}
}

var streamRegistry sync.Map

// createStreamObject constructs the BramandScript Hash object representing a Stream.
func createStreamObject(s *streamInstance) *object.Hash {
	h := &object.Hash{
		Pairs: make(map[object.HashKey]object.HashPair),
	}
	streamRegistry.Store(h, s)

	setMethod := func(name string, fn func(args ...object.Object) object.Object) {
		k := &object.String{Value: name}
		h.Pairs[k.HashKey()] = object.HashPair{
			Key:   k,
			Value: &object.Builtin{Fn: fn},
		}
	}

	// 1. collect() / to_array()
	collectFn := func(args ...object.Object) object.Object {
		var mu sync.Mutex
		var items []object.Object
		doneCh := make(chan struct{})
		var once sync.Once
		finish := func() {
			once.Do(func() {
				close(doneCh)
			})
		}

		go func() {
			defer finish()
			s.producer(func(val object.Object) bool {
				mu.Lock()
				items = append(items, val)
				mu.Unlock()
				return true
			}, finish)
		}()

		<-doneCh
		return &object.Array{Elements: items}
	}
	setMethod("collect", collectFn)
	setMethod("to_array", collectFn)

	// 2. subscribe(observer) with backpressure
	setMethod("subscribe", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.subscribe expects at least 1 argument (observer or callback)"}
		}

		var nextFn object.Object
		var errFn object.Object
		var compFn object.Object

		if obsHash, ok := args[0].(*object.Hash); ok {
			for _, pair := range obsHash.Pairs {
				k := pair.Key.Inspect()
				if k == "next" {
					nextFn = pair.Value
				} else if k == "error" {
					errFn = pair.Value
				} else if k == "complete" {
					compFn = pair.Value
				}
			}
		} else {
			nextFn = args[0]
			if len(args) >= 2 {
				errFn = args[1]
			}
			if len(args) >= 3 {
				compFn = args[2]
			}
		}

		// Backpressure controller state
		var mu sync.Mutex
		cond := sync.NewCond(&mu)
		paused := false
		cancelled := false
		var requested int64 = -1 // -1 means unconstrained (push mode), >= 0 means pull/backpressure mode

		sub := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
		setSubMethod := func(name string, bFn func(args ...object.Object) object.Object) {
			k := &object.String{Value: name}
			sub.Pairs[k.HashKey()] = object.HashPair{
				Key:   k,
				Value: &object.Builtin{Fn: bFn},
			}
		}

		setSubMethod("request", func(rArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			n := int64(1)
			if len(rArgs) > 0 {
				if intObj, ok := rArgs[0].(*object.Integer); ok {
					n = intObj.Value
				}
			}
			if requested < 0 {
				requested = 0
			}
			requested += n
			cond.Broadcast()
			return True
		})

		setSubMethod("pause", func(pArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			paused = true
			return True
		})

		setSubMethod("resume", func(rArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			paused = false
			cond.Broadcast()
			return True
		})

		setSubMethod("cancel", func(cArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			cancelled = true
			cond.Broadcast()
			return True
		})
		setSubMethod("unsubscribe", func(cArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			cancelled = true
			cond.Broadcast()
			return True
		})

		setSubMethod("is_active", func(aArgs ...object.Object) object.Object {
			mu.Lock()
			defer mu.Unlock()
			if cancelled {
				return False
			}
			return True
		})

		go func() {
			s.producer(func(val object.Object) bool {
				mu.Lock()
				for (paused || (requested == 0)) && !cancelled {
					cond.Wait()
				}
				if cancelled {
					mu.Unlock()
					return false
				}
				if requested > 0 {
					requested--
				}
				mu.Unlock()

				if nextFn != nil {
					res, err := ExecuteCallable(nextFn, val)
					if err != nil && errFn != nil {
						_, _ = ExecuteCallable(errFn, &object.String{Value: err.Error()})
					} else if errObj, ok := res.(*object.Error); ok && errFn != nil {
						_, _ = ExecuteCallable(errFn, errObj)
					}
				}
				return true
			}, func() {
				mu.Lock()
				isCanc := cancelled
				mu.Unlock()
				if !isCanc && compFn != nil {
					_, _ = ExecuteCallable(compFn)
				}
			})
		}()

		return sub
	})

	// 3. pull(n)
	setMethod("pull", func(args ...object.Object) object.Object {
		n := 1
		if len(args) > 0 {
			if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
				n = int(intObj.Value)
			}
		}

		var mu sync.Mutex
		var items []object.Object
		doneCh := make(chan struct{})

		go func() {
			defer close(doneCh)
			count := 0
			s.producer(func(val object.Object) bool {
				mu.Lock()
				items = append(items, val)
				count++
				stop := count >= n
				mu.Unlock()
				return !stop
			}, func() {})
		}()

		<-doneCh
		return &object.Array{Elements: items}
	})

	// 4. map(fn)
	setMethod("map", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.map expects a mapper function"}
		}
		mapper := args[0]
		newProd := func(emit func(val object.Object) bool, done func()) {
			s.producer(func(val object.Object) bool {
				mapped, err := ExecuteCallable(mapper, val)
				if err != nil {
					mapped = &object.Error{Message: err.Error()}
				}
				return emit(mapped)
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 5. filter(fn)
	setMethod("filter", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.filter expects a predicate function"}
		}
		pred := args[0]
		newProd := func(emit func(val object.Object) bool, done func()) {
			s.producer(func(val object.Object) bool {
				res, err := ExecuteCallable(pred, val)
				if err == nil && isTruthy(res) {
					return emit(val)
				}
				return true
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 6. take(n)
	setMethod("take", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.take expects integer n"}
		}
		maxCount := int64(0)
		if intObj, ok := args[0].(*object.Integer); ok {
			maxCount = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var count int64
			s.producer(func(val object.Object) bool {
				if count >= maxCount {
					return false
				}
				count++
				keepGoing := emit(val)
				if count >= maxCount {
					return false
				}
				return keepGoing
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 7. take_while(fn)
	setMethod("take_while", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.take_while expects a predicate function"}
		}
		pred := args[0]
		newProd := func(emit func(val object.Object) bool, done func()) {
			stopped := false
			s.producer(func(val object.Object) bool {
				if stopped {
					return false
				}
				res, err := ExecuteCallable(pred, val)
				if err != nil || !isTruthy(res) {
					stopped = true
					return false
				}
				return emit(val)
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 8. skip(n) / drop(n)
	skipFn := func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.skip expects integer n"}
		}
		skipCount := int64(0)
		if intObj, ok := args[0].(*object.Integer); ok {
			skipCount = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var current int64
			s.producer(func(val object.Object) bool {
				if current < skipCount {
					current++
					return true
				}
				return emit(val)
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	}
	setMethod("skip", skipFn)
	setMethod("drop", skipFn)

	// 9. reduce(fn, initial)
	setMethod("reduce", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.reduce expects at least a reducer function"}
		}
		reducer := args[0]
		var acc object.Object
		hasAcc := false
		if len(args) >= 2 {
			acc = args[1]
			hasAcc = true
		}

		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			s.producer(func(val object.Object) bool {
				if !hasAcc {
					acc = val
					hasAcc = true
					return true
				}
				nextAcc, err := ExecuteCallable(reducer, acc, val)
				if err != nil {
					acc = &object.Error{Message: err.Error()}
					return false
				}
				acc = nextAcc
				return true
			}, func() {})
		}()

		<-doneCh
		if acc == nil {
			return Null
		}
		return acc
	})

	// 10. for_each(fn)
	setMethod("for_each", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "stream.for_each expects a callback function"}
		}
		cb := args[0]
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			s.producer(func(val object.Object) bool {
				_, _ = ExecuteCallable(cb, val)
				return true
			}, func() {})
		}()
		<-doneCh
		return Null
	})

	// 11. count()
	setMethod("count", func(args ...object.Object) object.Object {
		var cnt int64
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			s.producer(func(val object.Object) bool {
				cnt++
				return true
			}, func() {})
		}()
		<-doneCh
		return &object.Integer{Value: cnt}
	})

	// 12. first()
	setMethod("first", func(args ...object.Object) object.Object {
		var firstVal object.Object = Null
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			s.producer(func(val object.Object) bool {
				firstVal = val
				return false
			}, func() {})
		}()
		<-doneCh
		return firstVal
	})

	// 13. last()
	setMethod("last", func(args ...object.Object) object.Object {
		var lastVal object.Object = Null
		doneCh := make(chan struct{})
		go func() {
			defer close(doneCh)
			s.producer(func(val object.Object) bool {
				lastVal = val
				return true
			}, func() {})
		}()
		<-doneCh
		return lastVal
	})

	// 14. pipe(op1, op2, ...)
	setMethod("pipe", func(args ...object.Object) object.Object {
		current := object.Object(h)
		for _, op := range args {
			res, err := ExecuteCallable(op, current)
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			current = res
		}
		return current
	})

	// 15. debounce(ms) (Task 14)
	setMethod("debounce", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "debounce requires delay in milliseconds"}
		}
		ms := int64(100)
		if intObj, ok := args[0].(*object.Integer); ok {
			ms = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var mu sync.Mutex
			var timer *time.Timer
			var lastVal object.Object
			var active bool
			isDone := false
			doneCh := make(chan struct{})
			var once sync.Once
			finish := func() {
				once.Do(func() {
					mu.Lock()
					isDone = true
					if timer != nil {
						timer.Stop()
					}
					if active {
						active = false
						emit(lastVal)
					}
					mu.Unlock()
					done()
					close(doneCh)
				})
			}

			s.producer(func(val object.Object) bool {
				mu.Lock()
				defer mu.Unlock()
				lastVal = val
				active = true
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
					mu.Lock()
					defer mu.Unlock()
					if active && !isDone {
						active = false
						emit(lastVal)
					}
				})
				return true
			}, finish)
			<-doneCh
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 16. throttle(ms) (Task 14)
	setMethod("throttle", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "throttle requires interval in milliseconds"}
		}
		ms := int64(100)
		if intObj, ok := args[0].(*object.Integer); ok {
			ms = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var mu sync.Mutex
			var lastEmitted time.Time
			s.producer(func(val object.Object) bool {
				mu.Lock()
				defer mu.Unlock()
				now := time.Now()
				if lastEmitted.IsZero() || now.Sub(lastEmitted) >= time.Duration(ms)*time.Millisecond {
					lastEmitted = now
					return emit(val)
				}
				return true // suppress emission within window
			}, done)
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 17. window(size) (Task 14)
	setMethod("window", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "window requires size integer"}
		}
		size := int64(1)
		if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
			size = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var mu sync.Mutex
			var buf []object.Object
			s.producer(func(val object.Object) bool {
				mu.Lock()
				buf = append(buf, val)
				if int64(len(buf)) >= size {
					chunk := make([]object.Object, len(buf))
					copy(chunk, buf)
					buf = buf[:0]
					mu.Unlock()
					return emit(&object.Array{Elements: chunk})
				}
				mu.Unlock()
				return true
			}, func() {
				mu.Lock()
				if len(buf) > 0 {
					chunk := make([]object.Object, len(buf))
					copy(chunk, buf)
					buf = nil
					mu.Unlock()
					emit(&object.Array{Elements: chunk})
				} else {
					mu.Unlock()
				}
				done()
			})
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 18. sample(ms) (Task 14)
	setMethod("sample", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "sample requires interval in milliseconds"}
		}
		ms := int64(100)
		if intObj, ok := args[0].(*object.Integer); ok {
			ms = intObj.Value
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var mu sync.Mutex
			var latestVal object.Object
			hasVal := false
			isDone := false
			doneCh := make(chan struct{})

			ticker := time.NewTicker(time.Duration(ms) * time.Millisecond)
			defer ticker.Stop()

			go func() {
				for {
					select {
					case <-ticker.C:
						mu.Lock()
						if hasVal && !isDone {
							valToEmit := latestVal
							hasVal = false
							mu.Unlock()
							if !emit(valToEmit) {
								return
							}
						} else {
							mu.Unlock()
						}
					case <-doneCh:
						return
					}
				}
			}()

			s.producer(func(val object.Object) bool {
				mu.Lock()
				latestVal = val
				hasVal = true
				mu.Unlock()
				return true
			}, func() {
				mu.Lock()
				isDone = true
				mu.Unlock()
				close(doneCh)
				done()
			})
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 19. merge(other) (Task 15)
	setMethod("merge", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "merge requires another stream"}
		}
		otherStream := extractStreamInstance(args[0])
		if otherStream == nil {
			return &object.Error{Message: "argument to merge must be a stream"}
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			var wg sync.WaitGroup
			var mu sync.Mutex
			stopped := false
			wg.Add(2)

			safeEmit := func(val object.Object) bool {
				mu.Lock()
				defer mu.Unlock()
				if stopped {
					return false
				}
				if !emit(val) {
					stopped = true
					return false
				}
				return true
			}

			go func() {
				defer wg.Done()
				s.producer(safeEmit, func() {})
			}()

			go func() {
				defer wg.Done()
				otherStream.producer(safeEmit, func() {})
			}()

			wg.Wait()
			done()
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 20. zip(other) (Task 15)
	setMethod("zip", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "zip requires another stream"}
		}
		otherStream := extractStreamInstance(args[0])
		if otherStream == nil {
			return &object.Error{Message: "argument to zip must be a stream"}
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			q1 := make(chan object.Object, 1024)
			q2 := make(chan object.Object, 1024)
			done1 := make(chan struct{})
			done2 := make(chan struct{})
			stopZip := make(chan struct{})
			var once sync.Once
			closeStop := func() {
				once.Do(func() {
					close(stopZip)
				})
			}
			defer closeStop()

			go func() {
				defer close(done1)
				s.producer(func(val object.Object) bool {
					select {
					case q1 <- val:
						return true
					case <-stopZip:
						return false
					}
				}, func() {})
			}()

			go func() {
				defer close(done2)
				otherStream.producer(func(val object.Object) bool {
					select {
					case q2 <- val:
						return true
					case <-stopZip:
						return false
					}
				}, func() {})
			}()

			for {
				var v1, v2 object.Object
				select {
				case v1 = <-q1:
				case <-done1:
					if len(q1) == 0 {
						done()
						return
					}
					v1 = <-q1
				case <-stopZip:
					return
				}

				select {
				case v2 = <-q2:
				case <-done2:
					if len(q2) == 0 {
						done()
						return
					}
					v2 = <-q2
				case <-stopZip:
					return
				}

				pair := &object.Array{Elements: []object.Object{v1, v2}}
				if !emit(pair) {
					done()
					return
				}
			}
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 21. concat(other) (Task 15)
	setMethod("concat", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "concat requires another stream"}
		}
		otherStream := extractStreamInstance(args[0])
		if otherStream == nil {
			return &object.Error{Message: "argument to concat must be a stream"}
		}
		newProd := func(emit func(val object.Object) bool, done func()) {
			stopped := false
			s.producer(func(val object.Object) bool {
				if !emit(val) {
					stopped = true
					return false
				}
				return true
			}, func() {})

			if !stopped {
				otherStream.producer(func(val object.Object) bool {
					return emit(val)
				}, func() {})
			}
			done()
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	// 22. switch_map(fn) (Task 15)
	setMethod("switch_map", func(args ...object.Object) object.Object {
		if len(args) < 1 {
			return &object.Error{Message: "switch_map requires mapping function that returns a stream"}
		}
		fn := args[0]
		newProd := func(emit func(val object.Object) bool, done func()) {
			var mu sync.Mutex
			var cancelCurrent func()
			var currentWg sync.WaitGroup

			s.producer(func(val object.Object) bool {
				innerObj, err := ExecuteCallable(fn, val)
				if err != nil {
					return emit(&object.Error{Message: err.Error()})
				}
				innerStream := extractStreamInstance(innerObj)
				if innerStream == nil {
					return emit(&object.Error{Message: "switch_map transform must return a stream"})
				}

				mu.Lock()
				if cancelCurrent != nil {
					cancelCurrent()
				}
				innerCancelled := false
				cancelCurrent = func() {
					innerCancelled = true
				}
				mu.Unlock()

				currentWg.Add(1)
				go func() {
					defer currentWg.Done()
					innerStream.producer(func(innerVal object.Object) bool {
						mu.Lock()
						cancelled := innerCancelled
						mu.Unlock()
						if cancelled {
							return false
						}
						return emit(innerVal)
					}, func() {})
				}()
				return true
			}, func() {})

			currentWg.Wait()
			done()
		}
		return createStreamObject(newStreamInstance(newProd))
	})

	return h
}

// extractStreamInstance extracts the underlying streamInstance from an object.Hash or returns nil.
func extractStreamInstance(obj object.Object) *streamInstance {
	if val, ok := streamRegistry.Load(obj); ok {
		return val.(*streamInstance)
	}

	if hash, ok := obj.(*object.Hash); ok {
		// If it has subscribe, we can wrap its subscribe method as a streamInstance
		subKey := (&object.String{Value: "subscribe"}).HashKey()
		if pair, exists := hash.Pairs[subKey]; exists {
			subFn := pair.Value
			return newStreamInstance(func(emit func(val object.Object) bool, done func()) {
				doneCh := make(chan struct{})
				observer := &object.Hash{Pairs: make(map[object.HashKey]object.HashPair)}
				setPair := func(k string, fn *object.Builtin) {
					key := &object.String{Value: k}
					observer.Pairs[key.HashKey()] = object.HashPair{Key: key, Value: fn}
				}
				setPair("next", &object.Builtin{
					Fn: func(args ...object.Object) object.Object {
						if len(args) > 0 {
							emit(args[0])
						}
						return Null
					},
				})
				setPair("complete", &object.Builtin{
					Fn: func(args ...object.Object) object.Object {
						select {
						case <-doneCh:
						default:
							close(doneCh)
						}
						return Null
					},
				})
				_, _ = ExecuteCallable(subFn, observer)
				<-doneCh
				done()
			})
		}
	}
	return nil
}

// NativeStreamBuiltin implements the stream(source) builtin function.
func NativeStreamBuiltin(args ...object.Object) object.Object {
	if len(args) == 0 || args[0] == Null {
		// Empty stream
		return createStreamObject(newStreamInstance(func(emit func(val object.Object) bool, done func()) {
			done()
		}))
	}

	arg := args[0]
	switch src := arg.(type) {
	case *object.Array:
		prod := func(emit func(val object.Object) bool, done func()) {
			src.Mu.RLock()
			elements := make([]object.Object, len(src.Elements))
			copy(elements, src.Elements)
			src.Mu.RUnlock()

			for _, el := range elements {
				if !emit(el) {
					break
				}
			}
			done()
		}
		return createStreamObject(newStreamInstance(prod))

	case *object.Channel:
		prod := func(emit func(val object.Object) bool, done func()) {
			for {
				val, ok := src.TryRecv()
				if !ok || val == nil {
					break
				}
				if !emit(val) {
					break
				}
			}
			done()
		}
		return createStreamObject(newStreamInstance(prod))

	default:
		// Callable producer: fn(emit, done)
		prod := func(emit func(val object.Object) bool, done func()) {
			emitBuiltin := &object.Builtin{
				Fn: func(eArgs ...object.Object) object.Object {
					if len(eArgs) > 0 {
						if emit(eArgs[0]) {
							return True
						}
						return False
					}
					return False
				},
			}
			doneBuiltin := &object.Builtin{
				Fn: func(dArgs ...object.Object) object.Object {
					done()
					return Null
				},
			}
			_, _ = ExecuteCallable(src, emitBuiltin, doneBuiltin)
		}
		return createStreamObject(newStreamInstance(prod))
	}
}

func init() {
	RegisterBuiltin("stream", &object.Builtin{
		Fn: NativeStreamBuiltin,
	})
}
