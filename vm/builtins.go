package vm

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"bramandscript/object"
)

var (
	builtinRegistry   = make(map[string]*object.Builtin)
	builtinRegistryMu sync.RWMutex

	programArgs     = []string{}
	stdinReader     = bufio.NewReader(os.Stdin)
	stdinReaderLock sync.Mutex
)

// RegisterBuiltin registers a single builtin function into the global VM registry.
func RegisterBuiltin(name string, b *object.Builtin) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	builtinRegistry[name] = b
}

// RegisterBuiltins registers a batch of builtins into the global VM registry.
func RegisterBuiltins(entries map[string]*object.Builtin) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	for k, v := range entries {
		builtinRegistry[k] = v
	}
}

// RegisterBuiltinAliases registers a single builtin under multiple aliases.
func RegisterBuiltinAliases(b *object.Builtin, names ...string) {
	builtinRegistryMu.Lock()
	defer builtinRegistryMu.Unlock()
	for _, name := range names {
		builtinRegistry[name] = b
	}
}

// getBuiltinByName retrieves a builtin object by its identifier name.
func getBuiltinByName(name string) *object.Builtin {
	builtinRegistryMu.RLock()
	b, ok := builtinRegistry[name]
	builtinRegistryMu.RUnlock()
	if ok {
		return b
	}
	return nil
}

// SetProgramArgs configures argv / args accessible from BramandScript.
func SetProgramArgs(args []string) {
	programArgs = make([]string, len(args))
	copy(programArgs, args)
}

// GetProgramArgs retrieves a copy of current program args.
func GetProgramArgs() []string {
	res := make([]string, len(programArgs))
	copy(res, programArgs)
	return res
}

func readLineFromStdin(args []object.Object) string {
	if len(args) > 0 {
		if s, ok := args[0].(*object.String); ok {
			fmt.Print(s.Value)
		} else {
			fmt.Print(args[0].Inspect())
		}
	}
	if IsTermReaderActive() {
		return ReadLineFromTermChan()
	}
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	text, _ := stdinReader.ReadString('\n')
	return strings.TrimRight(text, "\r\n")
}

// ResetStdin resets the standard input buffer (used for testing).
func ResetStdin() {
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	stdinReader = bufio.NewReader(os.Stdin)
}

func init() {
	RegisterBuiltins(map[string]*object.Builtin{
		"len": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Array:
					arg.Mu.RLock()
					count := int64(len(arg.Elements))
					arg.Mu.RUnlock()
					return &object.Integer{Value: count}
				case *object.String:
					return &object.Integer{Value: int64(len([]rune(arg.Value)))}
				case *object.Hash:
					arg.Mu.RLock()
					count := int64(len(arg.Pairs))
					arg.Mu.RUnlock()
					return &object.Integer{Value: count}
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Type())}
				}
			},
		},
		"print": {
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					if s, ok := arg.(*object.String); ok {
						fmt.Print(s.Value)
					} else {
						fmt.Print(arg.Inspect())
					}
				}
				return Null
			},
		},
		"println": {
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(arg.Inspect())
				}
				fmt.Println()
				return Null
			},
		},
		"print_raw": {
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(arg.Inspect())
				}
				return Null
			},
		},
		"type": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: string(args[0].Type())}
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
		"int": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Integer:
					return arg
				case *object.Float:
					return &object.Integer{Value: int64(arg.Value)}
				case *object.String:
					v, err := strconv.ParseInt(arg.Value, 0, 64)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to integer", arg.Value)}
					}
					return &object.Integer{Value: v}
				case *object.Boolean:
					if arg.Value {
						return &object.Integer{Value: 1}
					}
					return &object.Integer{Value: 0}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to integer", arg.Type())}
				}
			},
		},
		"float": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Float:
					return arg
				case *object.Integer:
					return &object.Float{Value: float64(arg.Value)}
				case *object.String:
					v, err := strconv.ParseFloat(arg.Value, 64)
					if err != nil {
						return &object.Error{Message: fmt.Sprintf("cannot convert %q to float", arg.Value)}
					}
					return &object.Float{Value: v}
				default:
					return &object.Error{Message: fmt.Sprintf("cannot convert %s to float", arg.Type())}
				}
			},
		},
		"str": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: args[0].Inspect()}
			},
		},
		"bool": {
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return nativeBoolToBooleanObject(isTruthy(args[0]))
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
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				start, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "second argument to `slice` must be INTEGER"}
				}

				switch target := args[0].(type) {
				case *object.Array:
					length := int64(len(target.Elements))
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
					if len(args) == 3 {
						end, ok := args[2].(*object.Integer)
						if !ok {
							return &object.Error{Message: "third argument to `slice` must be INTEGER"}
						}
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

					newElements := make([]object.Object, e-s)
					copy(newElements, target.Elements[s:e])
					return &object.Array{Elements: newElements}

				case *object.String:
					runes := []rune(target.Value)
					length := int64(len(runes))
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
					if len(args) == 3 {
						end, ok := args[2].(*object.Integer)
						if !ok {
							return &object.Error{Message: "third argument to `slice` must be INTEGER"}
						}
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

					return &object.String{Value: string(runes[s:e])}

				default:
					return &object.Error{Message: fmt.Sprintf("first argument to `slice` must be ARRAY or STRING, got %s", args[0].Type())}
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
					for _, elem := range coll.Elements {
						if elem.Inspect() == args[1].Inspect() {
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
		"exit": {
			Fn: func(args ...object.Object) object.Object {
				code := 0
				if len(args) > 0 {
					if intObj, ok := args[0].(*object.Integer); ok {
						code = int(intObj.Value)
					}
				}
				os.Exit(code)
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
				newElems := make([]object.Object, len(arr.Elements))
				copy(newElems, arr.Elements)
				sort.Slice(newElems, func(i, j int) bool {
					if i1, ok1 := newElems[i].(*object.Integer); ok1 {
						if i2, ok2 := newElems[j].(*object.Integer); ok2 {
							return i1.Value < i2.Value
						}
						if f2, ok2 := newElems[j].(*object.Float); ok2 {
							return float64(i1.Value) < f2.Value
						}
					}
					if f1, ok1 := newElems[i].(*object.Float); ok1 {
						if f2, ok2 := newElems[j].(*object.Float); ok2 {
							return f1.Value < f2.Value
						}
						if i2, ok2 := newElems[j].(*object.Integer); ok2 {
							return f1.Value < float64(i2.Value)
						}
					}
					if s1, ok1 := newElems[i].(*object.String); ok1 {
						if s2, ok2 := newElems[j].(*object.String); ok2 {
							return s1.Value < s2.Value
						}
					}
					return newElems[i].Inspect() < newElems[j].Inspect()
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
		"args": {
			Fn: func(args ...object.Object) object.Object {
				elements := make([]object.Object, len(programArgs))
				for i, a := range programArgs {
					elements[i] = &object.String{Value: a}
				}
				return &object.Array{Elements: elements}
			},
		},
		"argv": {
			Fn: func(args ...object.Object) object.Object {
				elements := make([]object.Object, len(programArgs))
				for i, a := range programArgs {
					elements[i] = &object.String{Value: a}
				}
				return &object.Array{Elements: elements}
			},
		},
	})
}

func toFloat(obj object.Object) (float64, bool) {
	switch o := obj.(type) {
	case *object.Float:
		return o.Value, true
	case *object.Integer:
		return float64(o.Value), true
	default:
		return 0, false
	}
}

func nativeToBramandObject(val interface{}) object.Object {
	if val == nil {
		return Null
	}
	switch v := val.(type) {
	case bool:
		return nativeBoolToBooleanObject(v)
	case float64:
		if math.Floor(v) == v && !math.IsNaN(v) && !math.IsInf(v, 0) && v >= -9223372036854775808 && v <= 9223372036854775807 {
			return &object.Integer{Value: int64(v)}
		}
		return &object.Float{Value: v}
	case int64:
		return &object.Integer{Value: v}
	case int:
		return &object.Integer{Value: int64(v)}
	case string:
		return &object.String{Value: v}
	case []interface{}:
		elements := make([]object.Object, len(v))
		for i, el := range v {
			elements[i] = nativeToBramandObject(el)
		}
		return &object.Array{Elements: elements}
	case map[string]interface{}:
		pairs := make(map[object.HashKey]object.HashPair)
		for k, val := range v {
			keyObj := &object.String{Value: k}
			valObj := nativeToBramandObject(val)
			pairs[keyObj.HashKey()] = object.HashPair{Key: keyObj, Value: valObj}
		}
		return &object.Hash{Pairs: pairs}
	default:
		return &object.String{Value: fmt.Sprintf("%v", v)}
	}
}

func bramandObjectToNative(obj object.Object) interface{} {
	switch o := obj.(type) {
	case *object.Integer:
		return o.Value
	case *object.Float:
		return o.Value
	case *object.Boolean:
		return o.Value
	case *object.String:
		return o.Value
	case *object.Null:
		return nil
	case *object.Array:
		res := make([]interface{}, len(o.Elements))
		for i, el := range o.Elements {
			res[i] = bramandObjectToNative(el)
		}
		return res
	case *object.Hash:
		o.Mu.RLock()
		defer o.Mu.RUnlock()
		res := make(map[string]interface{})
		for _, pair := range o.Pairs {
			res[pair.Key.Inspect()] = bramandObjectToNative(pair.Value)
		}
		return res
	case *object.Vector:
		res := make([]float64, len(o.Elements))
		copy(res, o.Elements)
		return res
	case *object.Matrix:
		return o.Data
	case *object.BigInt:
		return o.Value.String()
	case *object.BigFloat:
		return o.Value.Text('g', -1)
	default:
		return o.Inspect()
	}
}
