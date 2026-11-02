package vm

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bramandscript/object"
)

var (
	programArgs     = []string{}
	stdinReader     = bufio.NewReader(os.Stdin)
	stdinReaderLock sync.Mutex
)

func SetProgramArgs(args []string) {
	programArgs = make([]string, len(args))
	copy(programArgs, args)
}

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
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	text, _ := stdinReader.ReadString('\n')
	return strings.TrimRight(text, "\r\n")
}

func ResetStdin() {
	stdinReaderLock.Lock()
	defer stdinReaderLock.Unlock()
	stdinReader = bufio.NewReader(os.Stdin)
}

func getBuiltinByName(name string) *object.Builtin {
	switch name {
	case "len":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				switch arg := args[0].(type) {
				case *object.Array:
					return &object.Integer{Value: int64(len(arg.Elements))}
				case *object.String:
					return &object.Integer{Value: int64(len([]rune(arg.Value)))}
				case *object.Hash:
					return &object.Integer{Value: int64(len(arg.Pairs))}
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `len` not supported, got %s", args[0].Type())}
				}
			},
		}
	case "print":
		return &object.Builtin{
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
		}
	case "println":
		return &object.Builtin{
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
		}
	case "print_raw":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Print(" ")
					}
					fmt.Print(arg.Inspect())
				}
				return Null
			},
		}
	case "type":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: string(args[0].Type())}
			},
		}
	case "push":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				if args[0].Type() != object.ARRAY_OBJ {
					return &object.Error{Message: fmt.Sprintf("first argument to `push` must be ARRAY, got %s", args[0].Type())}
				}
				arr := args[0].(*object.Array)
				arr.Elements = append(arr.Elements, args[1])
				return arr
			},
		}
	case "now":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Integer{Value: time.Now().Unix()}
			},
		}
	case "input":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.String{Value: readLineFromStdin(args)}
			},
		}
	case "int":
		return &object.Builtin{
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
		}
	case "float":
		return &object.Builtin{
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
		}
	case "str":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return &object.String{Value: args[0].Inspect()}
			},
		}
	case "bool":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				return nativeBoolToBooleanObject(isTruthy(args[0]))
			},
		}
	case "keys":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `keys` must be HASH, got %s", args[0].Type())}
				}
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Key)
				}
				return &object.Array{Elements: elements}
			},
		}
	case "values":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				hash, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `values` must be HASH, got %s", args[0].Type())}
				}
				elements := make([]object.Object, 0, len(hash.Pairs))
				for _, pair := range hash.Pairs {
					elements = append(elements, pair.Value)
				}
				return &object.Array{Elements: elements}
			},
		}
	case "slice", "substring":
		return &object.Builtin{
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
						s = 0
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
						s = 0
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
		}
	case "pop":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `pop` must be ARRAY, got %s", args[0].Type())}
				}
				if len(arr.Elements) == 0 {
					return Null
				}
				lastIdx := len(arr.Elements) - 1
				lastElem := arr.Elements[lastIdx]
				arr.Elements = arr.Elements[:lastIdx]
				return lastElem
			},
		}
	case "contains":
		return &object.Builtin{
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
						_, exists := coll.Pairs[key.HashKey()]
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
		}
	case "delete":
		return &object.Builtin{
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
				delete(hash.Pairs, key.HashKey())
				return Null
			},
		}
	case "exit":
		return &object.Builtin{
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
		}
	case "chan":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				capacity := 0
				if len(args) > 0 {
					if intObj, ok := args[0].(*object.Integer); ok {
						if intObj.Value < 0 {
							return &object.Error{Message: "channel capacity cannot be negative"}
						}
						capacity = int(intObj.Value)
					} else {
						return &object.Error{Message: fmt.Sprintf("channel capacity must be INTEGER, got %s", args[0].Type())}
					}
				}
				return &object.Channel{
					Ch:       make(chan object.Object, capacity),
					Capacity: capacity,
				}
			},
		}
	case "close":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				ch, ok := args[0].(*object.Channel)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `close` must be CHANNEL, got %s", args[0].Type())}
				}
				err := ch.Close()
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return Null
			},
		}
	case "mutex":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Mutex{}
			},
		}
	case "wait_group":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.WaitGroup{}
			},
		}
	case "sleep":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				var ms int64
				switch arg := args[0].(type) {
				case *object.Integer:
					ms = arg.Value
				case *object.Float:
					ms = int64(arg.Value)
				default:
					return &object.Error{Message: fmt.Sprintf("argument to `sleep` must be number, got %s", args[0].Type())}
				}
				if ms > 0 {
					time.Sleep(time.Duration(ms) * time.Millisecond)
				}
				return Null
			},
		}
	case "read_file":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `read_file` must be STRING, got %s", args[0].Type())}
				}
				data, err := os.ReadFile(pathObj.Value)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("read_file failed: %s", err)}
				}
				return &object.String{Value: string(data)}
			},
		}
	case "write_file":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `write_file` must be STRING, got %s", args[0].Type())}
				}
				content := args[1].Inspect()
				if strObj, ok := args[1].(*object.String); ok {
					content = strObj.Value
				}
				if dir := filepath.Dir(pathObj.Value); dir != "" && dir != "." {
					_ = os.MkdirAll(dir, 0755)
				}
				err := os.WriteFile(pathObj.Value, []byte(content), 0644)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("write_file failed: %s", err)}
				}
				return True
			},
		}
	case "append_file":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `append_file` must be STRING, got %s", args[0].Type())}
				}
				content := args[1].Inspect()
				if strObj, ok := args[1].(*object.String); ok {
					content = strObj.Value
				}
				if dir := filepath.Dir(pathObj.Value); dir != "" && dir != "." {
					_ = os.MkdirAll(dir, 0755)
				}
				f, err := os.OpenFile(pathObj.Value, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("append_file failed: %s", err)}
				}
				defer f.Close()
				if _, err := f.WriteString(content); err != nil {
					return &object.Error{Message: fmt.Sprintf("append_file failed: %s", err)}
				}
				return True
			},
		}
	case "delete_file":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `delete_file` must be STRING, got %s", args[0].Type())}
				}
				err := os.Remove(pathObj.Value)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("delete_file failed: %s", err)}
				}
				return True
			},
		}
	case "file_exists":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `file_exists` must be STRING, got %s", args[0].Type())}
				}
				_, err := os.Stat(pathObj.Value)
				return nativeBoolToBooleanObject(err == nil)
			},
		}
	case "list_dir":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				dirPath := "."
				if len(args) > 0 {
					if pathObj, ok := args[0].(*object.String); ok {
						dirPath = pathObj.Value
					} else {
						return &object.Error{Message: fmt.Sprintf("argument to `list_dir` must be STRING, got %s", args[0].Type())}
					}
				}
				entries, err := os.ReadDir(dirPath)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("list_dir failed: %s", err)}
				}
				elements := make([]object.Object, len(entries))
				for i, entry := range entries {
					elements[i] = &object.String{Value: entry.Name()}
				}
				return &object.Array{Elements: elements}
			},
		}
	case "chmod":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `chmod` must be STRING, got %s", args[0].Type())}
				}
				modeObj, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("second argument to `chmod` must be INTEGER, got %s", args[1].Type())}
				}
				err := os.Chmod(pathObj.Value, os.FileMode(modeObj.Value))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("chmod failed: %s", err)}
				}
				return True
			},
		}
	case "is_dir":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				pathObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `is_dir` must be STRING, got %s", args[0].Type())}
				}
				fi, err := os.Stat(pathObj.Value)
				if err != nil {
					return False
				}
				return nativeBoolToBooleanObject(fi.IsDir())
			},
		}
	case "env":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) == 0 {
					pairs := make(map[object.HashKey]object.HashPair)
					for _, envStr := range os.Environ() {
						parts := strings.SplitN(envStr, "=", 2)
						key := &object.String{Value: parts[0]}
						val := &object.String{Value: parts[1]}
						pairs[key.HashKey()] = object.HashPair{Key: key, Value: val}
					}
					return &object.Hash{Pairs: pairs}
				}
				keyObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `env` must be STRING, got %s", args[0].Type())}
				}
				val, ok := os.LookupEnv(keyObj.Value)
				if !ok {
					return Null
				}
				return &object.String{Value: val}
			},
		}
	case "set_env":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				keyObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `set_env` must be STRING, got %s", args[0].Type())}
				}
				val := args[1].Inspect()
				if strObj, ok := args[1].(*object.String); ok {
					val = strObj.Value
				}
				err := os.Setenv(keyObj.Value, val)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("set_env failed: %s", err)}
				}
				return True
			},
		}
	case "exec":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) == 0 {
					return &object.Error{Message: "exec requires at least 1 argument"}
				}
				cmdStr := args[0].Inspect()
				if strObj, ok := args[0].(*object.String); ok {
					cmdStr = strObj.Value
				}
				var cmdArgs []string
				if len(args) == 2 {
					if arr, ok := args[1].(*object.Array); ok {
						for _, el := range arr.Elements {
							if s, ok := el.(*object.String); ok {
								cmdArgs = append(cmdArgs, s.Value)
							} else {
								cmdArgs = append(cmdArgs, el.Inspect())
							}
						}
					} else {
						for _, arg := range args[1:] {
							if s, ok := arg.(*object.String); ok {
								cmdArgs = append(cmdArgs, s.Value)
							} else {
								cmdArgs = append(cmdArgs, arg.Inspect())
							}
						}
					}
				} else if len(args) > 2 {
					for _, arg := range args[1:] {
						if s, ok := arg.(*object.String); ok {
							cmdArgs = append(cmdArgs, s.Value)
						} else {
							cmdArgs = append(cmdArgs, arg.Inspect())
						}
					}
				}
				cmd := exec.Command(cmdStr, cmdArgs...)
				var stdoutBuf, stderrBuf bytes.Buffer
				cmd.Stdout = &stdoutBuf
				cmd.Stderr = &stderrBuf
				err := cmd.Run()
				exitCode := int64(0)
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						exitCode = int64(exitErr.ExitCode())
					} else {
						exitCode = -1
						stderrBuf.WriteString(err.Error())
					}
				}
				pairs := make(map[object.HashKey]object.HashPair)
				kStdout := &object.String{Value: "stdout"}
				vStdout := &object.String{Value: stdoutBuf.String()}
				pairs[kStdout.HashKey()] = object.HashPair{Key: kStdout, Value: vStdout}

				kStderr := &object.String{Value: "stderr"}
				vStderr := &object.String{Value: stderrBuf.String()}
				pairs[kStderr.HashKey()] = object.HashPair{Key: kStderr, Value: vStderr}

				kCode := &object.String{Value: "code"}
				vCode := &object.Integer{Value: exitCode}
				pairs[kCode.HashKey()] = object.HashPair{Key: kCode, Value: vCode}

				kExitCode := &object.String{Value: "exit_code"}
				pairs[kExitCode.HashKey()] = object.HashPair{Key: kExitCode, Value: vCode}

				return &object.Hash{Pairs: pairs}
			},
		}
	case "eprint":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Fprint(os.Stderr, " ")
					}
					fmt.Fprint(os.Stderr, arg.Inspect())
				}
				return Null
			},
		}
	case "eprintln":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				for i, arg := range args {
					if i > 0 {
						fmt.Fprint(os.Stderr, " ")
					}
					fmt.Fprint(os.Stderr, arg.Inspect())
				}
				fmt.Fprintln(os.Stderr)
				return Null
			},
		}
	case "read_line":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.String{Value: readLineFromStdin(args)}
			},
		}
	case "color":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				colorName := args[0].Inspect()
				if s, ok := args[0].(*object.String); ok {
					colorName = s.Value
				}
				text := args[1].Inspect()
				if s, ok := args[1].(*object.String); ok {
					text = s.Value
				}
				var code string
				switch strings.ToLower(colorName) {
				case "black":
					code = "30"
				case "red":
					code = "31"
				case "green":
					code = "32"
				case "yellow":
					code = "33"
				case "blue":
					code = "34"
				case "magenta":
					code = "35"
				case "cyan":
					code = "36"
				case "white":
					code = "37"
				case "gray", "grey":
					code = "90"
				default:
					return &object.String{Value: text}
				}
				return &object.String{Value: fmt.Sprintf("\033[%sm%s\033[0m", code, text)}
			},
		}
	case "styled":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				text := args[0].Inspect()
				if s, ok := args[0].(*object.String); ok {
					text = s.Value
				}
				styleName := args[1].Inspect()
				if s, ok := args[1].(*object.String); ok {
					styleName = s.Value
				}
				var code string
				switch strings.ToLower(styleName) {
				case "bold":
					code = "1"
				case "dim":
					code = "2"
				case "italic":
					code = "3"
				case "underline":
					code = "4"
				case "inverse":
					code = "7"
				case "red":
					code = "31"
				case "green":
					code = "32"
				case "yellow":
					code = "33"
				case "blue":
					code = "34"
				case "magenta":
					code = "35"
				case "cyan":
					code = "36"
				default:
					return &object.String{Value: text}
				}
				return &object.String{Value: fmt.Sprintf("\033[%sm%s\033[0m", code, text)}
			},
		}

	// ---------------- Day 5 BigInt & BigFloat ----------------
	case "big_int", "bigint":
		return &object.Builtin{
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

	case "big_float", "bigfloat":
		return &object.Builtin{
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

	// ---------------- Day 5 Vectors & Matrices ----------------
	case "vec", "vector":
		return &object.Builtin{
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

	case "matrix":
		return &object.Builtin{
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
					fill := 0.0
					if len(args) >= 3 {
						if f, ok := toFloat(args[2]); ok {
							fill = f
						}
					}
					for i := 0; i < m.Rows; i++ {
						for j := 0; j < m.Cols; j++ {
							m.Data[i][j] = fill
						}
					}
					return m
				}
				return &object.Error{Message: "wrong number of arguments to `matrix`"}
			},
		}

	// ---------------- Day 5 Math Standard Library ----------------
	case "pi":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Float{Value: math.Pi}
			},
		}

	case "e":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Float{Value: math.E}
			},
		}

	case "sin":
		return &object.Builtin{
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
		}

	case "cos":
		return &object.Builtin{
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
		}

	case "tan":
		return &object.Builtin{
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
		}

	case "asin":
		return &object.Builtin{
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
		}

	case "acos":
		return &object.Builtin{
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
		}

	case "atan":
		return &object.Builtin{
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
		}

	case "atan2":
		return &object.Builtin{
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
		}

	case "sinh":
		return &object.Builtin{
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
		}

	case "cosh":
		return &object.Builtin{
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
		}

	case "tanh":
		return &object.Builtin{
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
		}

	case "sqrt":
		return &object.Builtin{
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
		}

	case "cbrt":
		return &object.Builtin{
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
		}

	case "hypot":
		return &object.Builtin{
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
		}

	case "log":
		return &object.Builtin{
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
		}

	case "log10":
		return &object.Builtin{
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
		}

	case "log2":
		return &object.Builtin{
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
		}

	case "exp":
		return &object.Builtin{
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
		}

	case "pow":
		return &object.Builtin{
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
		}

	case "abs":
		return &object.Builtin{
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
		}

	case "floor":
		return &object.Builtin{
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
		}

	case "ceil":
		return &object.Builtin{
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
		}

	case "round":
		return &object.Builtin{
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
		}

	case "trunc":
		return &object.Builtin{
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
		}

	case "deg_to_rad":
		return &object.Builtin{
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
		}

	case "rad_to_deg":
		return &object.Builtin{
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
		}

	case "min":
		return &object.Builtin{
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
		}

	case "max":
		return &object.Builtin{
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
		}

	// ---------------- Day 5 Statistics ----------------
	case "sum":
		return &object.Builtin{
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
		}

	case "mean", "math_mean":
		return &object.Builtin{
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

	case "median":
		return &object.Builtin{
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
		}

	case "mode":
		return &object.Builtin{
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
		}

	case "variance":
		return &object.Builtin{
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
		}

	case "std_dev":
		return &object.Builtin{
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
		}

	// ---------------- Day 5 JSON Parser & Serializer ----------------
	case "json_parse":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `json_parse` must be STRING, got %s", args[0].Type())}
				}
				var val interface{}
				err := json.Unmarshal([]byte(strObj.Value), &val)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("json_parse error: %s", err)}
				}
				return nativeToBramandObject(val)
			},
		}

	case "json_stringify":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				native := bramandObjectToNative(args[0])
				var data []byte
				var err error
				if len(args) == 2 {
					indentSpaces := 2
					if intObj, ok := args[1].(*object.Integer); ok && intObj.Value >= 0 {
						indentSpaces = int(intObj.Value)
					}
					indent := strings.Repeat(" ", indentSpaces)
					data, err = json.MarshalIndent(native, "", indent)
				} else {
					data, err = json.Marshal(native)
				}
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("json_stringify error: %s", err)}
				}
				return &object.String{Value: string(data)}
			},
		}

	// ---------------- Day 5 CSV & Base64 ----------------
	case "csv_parse":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `csv_parse` must be STRING, got %s", args[0].Type())}
				}
				r := csv.NewReader(strings.NewReader(strObj.Value))
				if len(args) == 2 {
					if delimObj, ok := args[1].(*object.String); ok && len(delimObj.Value) > 0 {
						r.Comma = rune(delimObj.Value[0])
					}
				}
				records, err := r.ReadAll()
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("csv_parse error: %s", err)}
				}
				rows := make([]object.Object, len(records))
				for i, rec := range records {
					cols := make([]object.Object, len(rec))
					for j, col := range rec {
						cols[j] = &object.String{Value: col}
					}
					rows[i] = &object.Array{Elements: cols}
				}
				return &object.Array{Elements: rows}
			},
		}

	case "csv_stringify":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `csv_stringify` must be ARRAY, got %s", args[0].Type())}
				}
				var buf bytes.Buffer
				w := csv.NewWriter(&buf)
				if len(args) == 2 {
					if delimObj, ok := args[1].(*object.String); ok && len(delimObj.Value) > 0 {
						w.Comma = rune(delimObj.Value[0])
					}
				}
				for _, rowObj := range arr.Elements {
					rowArr, ok := rowObj.(*object.Array)
					if !ok {
						return &object.Error{Message: "csv rows must be arrays"}
					}
					var rec []string
					for _, col := range rowArr.Elements {
						rec = append(rec, col.Inspect())
					}
					if err := w.Write(rec); err != nil {
						return &object.Error{Message: fmt.Sprintf("csv_stringify error: %s", err)}
					}
				}
				w.Flush()
				return &object.String{Value: buf.String()}
			},
		}

	case "base64_encode":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				data := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				return &object.String{Value: base64.StdEncoding.EncodeToString(data)}
			},
		}

	case "base64_decode":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `base64_decode` must be STRING, got %s", args[0].Type())}
				}
				data, err := base64.StdEncoding.DecodeString(strObj.Value)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("base64_decode error: %s", err)}
				}
				return &object.String{Value: string(data)}
			},
		}

	case "base64_url_encode":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				data := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				return &object.String{Value: base64.URLEncoding.EncodeToString(data)}
			},
		}

	case "base64_url_decode":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("argument to `base64_url_decode` must be STRING, got %s", args[0].Type())}
				}
				data, err := base64.URLEncoding.DecodeString(strObj.Value)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("base64_url_decode error: %s", err)}
				}
				return &object.String{Value: string(data)}
			},
		}

	// ---------------- Day 5 Cryptography & CSPRNG ----------------
	case "sha256", "crypto_sha256":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				data := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				h := sha256.Sum256(data)
				return &object.String{Value: hex.EncodeToString(h[:])}
			},
		}

	case "sha512":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				data := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				h := sha512.Sum512(data)
				return &object.String{Value: hex.EncodeToString(h[:])}
			},
		}

	case "md5":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				data := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				h := md5.Sum(data)
				return &object.String{Value: hex.EncodeToString(h[:])}
			},
		}

	case "hmac_sha256", "crypto_hmac":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				key := []byte(args[0].Inspect())
				if strObj, ok := args[0].(*object.String); ok {
					key = []byte(strObj.Value)
				}
				data := []byte(args[1].Inspect())
				if strObj, ok := args[1].(*object.String); ok {
					data = []byte(strObj.Value)
				}
				mac := hmac.New(sha256.New, key)
				mac.Write(data)
				return &object.String{Value: hex.EncodeToString(mac.Sum(nil))}
			},
		}

	case "random_bytes":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				count := 16
				if len(args) > 0 {
					if intObj, ok := args[0].(*object.Integer); ok && intObj.Value > 0 {
						count = int(intObj.Value)
					}
				}
				b := make([]byte, count)
				_, err := crand.Read(b)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("random_bytes error: %s", err)}
				}
				return &object.String{Value: hex.EncodeToString(b)}
			},
		}

	case "random_int":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				var minVal, maxVal int64
				if len(args) == 1 {
					minVal = 0
					if intObj, ok := args[0].(*object.Integer); ok {
						maxVal = intObj.Value
					} else {
						return &object.Error{Message: "max argument must be INTEGER"}
					}
				} else if len(args) >= 2 {
					if intObj1, ok := args[0].(*object.Integer); ok {
						minVal = intObj1.Value
					}
					if intObj2, ok := args[1].(*object.Integer); ok {
						maxVal = intObj2.Value
					}
				} else {
					return &object.Error{Message: "wrong number of arguments to `random_int`"}
				}

				if maxVal < minVal {
					minVal, maxVal = maxVal, minVal
				}
				diff := maxVal - minVal + 1
				n, err := crand.Int(crand.Reader, big.NewInt(diff))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("random_int error: %s", err)}
				}
				return &object.Integer{Value: minVal + n.Int64()}
			},
		}

	case "uuid":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				var b [16]byte
				_, err := crand.Read(b[:])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("uuid error: %s", err)}
				}
				// RFC 4122 variant & version 4
				b[6] = (b[6] & 0x0f) | 0x40
				b[8] = (b[8] & 0x3f) | 0x80
				uuidStr := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
					b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
				return &object.String{Value: uuidStr}
			},
		}

	// ============================================================================
	// Day 6 TCP & UDP Sockets
	// ============================================================================
	case "tcp_listen":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpServer{Listener: l}
			},
		}

	case "tcp_connect":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				}
				timeout := 5 * time.Second
				if len(args) == 2 {
					if iObj, ok := args[1].(*object.Integer); ok && iObj.Value > 0 {
						timeout = time.Duration(iObj.Value) * time.Millisecond
					}
				}
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.TcpConn{
					Conn:   conn,
					Reader: bufio.NewReader(conn),
				}
			},
		}

	case "udp_listen":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
				pc, err := net.ListenPacket("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{PacketConn: pc}
			},
		}

	case "udp_connect":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				}
				conn, err := net.Dial("udp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.UdpSocket{Conn: conn}
			},
		}

	// ============================================================================
	// Day 6 HTTP/HTTPS Client
	// ============================================================================
	case "http_get":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				var headers object.Object
				if len(args) == 2 {
					headers = args[1]
				}
				return executeHttpRequest("GET", urlStr, "", headers, 0)
			},
		}

	case "http_post":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				bodyStr := ""
				if sObj, ok := args[1].(*object.String); ok {
					bodyStr = sObj.Value
				} else {
					jsonData, _ := json.Marshal(bramandObjectToNative(args[1]))
					bodyStr = string(jsonData)
				}
				var headers object.Object
				if len(args) == 3 {
					headers = args[2]
				}
				return executeHttpRequest("POST", urlStr, bodyStr, headers, 0)
			},
		}

	case "http_put":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				bodyStr := ""
				if sObj, ok := args[1].(*object.String); ok {
					bodyStr = sObj.Value
				} else {
					jsonData, _ := json.Marshal(bramandObjectToNative(args[1]))
					bodyStr = string(jsonData)
				}
				var headers object.Object
				if len(args) == 3 {
					headers = args[2]
				}
				return executeHttpRequest("PUT", urlStr, bodyStr, headers, 0)
			},
		}

	case "http_delete":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				urlStr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					urlStr = sObj.Value
				}
				var headers object.Object
				if len(args) == 2 {
					headers = args[1]
				}
				return executeHttpRequest("DELETE", urlStr, "", headers, 0)
			},
		}

	case "http_request":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				opts, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: "argument to http_request must be Hash"}
				}
				method := "GET"
				urlStr := ""
				bodyStr := ""
				var headers object.Object
				var timeoutMs int64

				for _, p := range opts.Pairs {
					k := p.Key.Inspect()
					switch k {
					case "method":
						method = strings.ToUpper(p.Value.Inspect())
						if s, ok := p.Value.(*object.String); ok {
							method = strings.ToUpper(s.Value)
						}
					case "url":
						urlStr = p.Value.Inspect()
						if s, ok := p.Value.(*object.String); ok {
							urlStr = s.Value
						}
					case "body":
						if s, ok := p.Value.(*object.String); ok {
							bodyStr = s.Value
						} else {
							b, _ := json.Marshal(bramandObjectToNative(p.Value))
							bodyStr = string(b)
						}
					case "headers":
						headers = p.Value
					case "timeout":
						if i, ok := p.Value.(*object.Integer); ok {
							timeoutMs = i.Value
						}
					}
				}
				if urlStr == "" {
					return &object.Error{Message: "http_request requires 'url' field"}
				}
				return executeHttpRequest(method, urlStr, bodyStr, headers, timeoutMs)
			},
		}

	// ============================================================================
	// Day 6 HTTP Web Server
	// ============================================================================
	case "http_server":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return object.NewHttpServer()
			},
		}

	// ============================================================================
	// Day 6 WebSockets (RFC 6455)
	// ============================================================================
	case "ws_connect":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=1", len(args))}
				}
				u := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					u = sObj.Value
				}
				conn, err := object.WsConnect(u)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return conn
			},
		}

	case "ws_server":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				addr := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					addr = sObj.Value
				} else if iObj, ok := args[0].(*object.Integer); ok {
					addr = fmt.Sprintf("127.0.0.1:%d", iObj.Value)
				}
				path := "/"
				if len(args) == 2 {
					if pObj, ok := args[1].(*object.String); ok {
						path = pObj.Value
					}
				}
				l, err := net.Listen("tcp", addr)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.WsServer{Listener: l, Path: path}
			},
		}

	// ============================================================================
	// Day 6 AI Integrations (HTTP streaming LLM calls, structured JSON outputs)
	// ============================================================================
	case "ai_prompt":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "ai_prompt requires at least 2 arguments (model, prompt)"}
				}
				model := args[0].Inspect()
				if sObj, ok := args[0].(*object.String); ok {
					model = sObj.Value
				}
				prompt := args[1].Inspect()
				if sObj, ok := args[1].(*object.String); ok {
					prompt = sObj.Value
				}

				endpoint := "http://localhost:11434/api/generate"
				apiKey := ""
				var systemPrompt string
				if len(args) >= 3 {
					if opts, ok := args[2].(*object.Hash); ok {
						for _, pair := range opts.Pairs {
							k := pair.Key.Inspect()
							if k == "endpoint" || k == "url" {
								endpoint = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									endpoint = s.Value
								}
							} else if k == "api_key" {
								apiKey = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									apiKey = s.Value
								}
							} else if k == "system" {
								systemPrompt = pair.Value.Inspect()
								if s, ok := pair.Value.(*object.String); ok {
									systemPrompt = s.Value
								}
							}
						}
					}
				}

				isOpenAI := strings.Contains(endpoint, "/v1/") || strings.Contains(endpoint, "chat/completions")
				var reqBody []byte
				if isOpenAI {
					messages := []map[string]string{}
					if systemPrompt != "" {
						messages = append(messages, map[string]string{"role": "system", "content": systemPrompt})
					}
					messages = append(messages, map[string]string{"role": "user", "content": prompt})
					payload := map[string]interface{}{
						"model":    model,
						"messages": messages,
						"stream":   false,
					}
					reqBody, _ = json.Marshal(payload)
				} else {
					payload := map[string]interface{}{
						"model":  model,
						"prompt": prompt,
						"stream": false,
					}
					if systemPrompt != "" {
						payload["system"] = systemPrompt
					}
					reqBody, _ = json.Marshal(payload)
				}

				req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_prompt request creation failed: %s", err)}
				}
				req.Header.Set("Content-Type", "application/json")
				if apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+apiKey)
				}

				client := &http.Client{Timeout: 60 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_prompt request failed: %s", err)}
				}
				defer resp.Body.Close()

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("reading ai_prompt response failed: %s", err)}
				}

				var result map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &result); err != nil {
					return &object.String{Value: string(bodyBytes)}
				}

				if respText, ok := result["response"].(string); ok {
					return &object.String{Value: respText}
				}
				if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
					if c0, ok := choices[0].(map[string]interface{}); ok {
						if msg, ok := c0["message"].(map[string]interface{}); ok {
							if content, ok := msg["content"].(string); ok {
								return &object.String{Value: content}
							}
						}
					}
				}
				return &object.String{Value: string(bodyBytes)}
			},
		}

	case "ai_stream":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: "ai_stream requires 2 arguments (options, chunk_callback)"}
				}
				opts, ok := args[0].(*object.Hash)
				if !ok {
					return &object.Error{Message: "first argument to ai_stream must be options Hash"}
				}
				callback := args[1]

				endpoint := "http://localhost:11434/api/generate"
				var reqBody []byte
				apiKey := ""
				for _, pair := range opts.Pairs {
					k := pair.Key.Inspect()
					if k == "url" || k == "endpoint" {
						endpoint = pair.Value.Inspect()
						if s, ok := pair.Value.(*object.String); ok {
							endpoint = s.Value
						}
					} else if k == "api_key" {
						apiKey = pair.Value.Inspect()
						if s, ok := pair.Value.(*object.String); ok {
							apiKey = s.Value
						}
					} else if k == "body" {
						if s, ok := pair.Value.(*object.String); ok {
							reqBody = []byte(s.Value)
						} else {
							reqBody, _ = json.Marshal(bramandObjectToNative(pair.Value))
						}
					}
				}

				req, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_stream request creation failed: %s", err)}
				}
				req.Header.Set("Content-Type", "application/json")
				if apiKey != "" {
					req.Header.Set("Authorization", "Bearer "+apiKey)
				}

				client := &http.Client{Timeout: 120 * time.Second}
				resp, err := client.Do(req)
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("ai_stream connection failed: %s", err)}
				}
				defer resp.Body.Close()

				scanner := bufio.NewScanner(resp.Body)
				var accumulated strings.Builder
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if line == "" || line == "data: [DONE]" {
						continue
					}
					cleanJSON := line
					if strings.HasPrefix(line, "data: ") {
						cleanJSON = strings.TrimPrefix(line, "data: ")
					}

					var chunkMap map[string]interface{}
					if err := json.Unmarshal([]byte(cleanJSON), &chunkMap); err != nil {
						_, _ = ExecuteCallable(callback, &object.String{Value: cleanJSON})
						accumulated.WriteString(cleanJSON)
						continue
					}

					var chunkText string
					if r, ok := chunkMap["response"].(string); ok {
						chunkText = r
					} else if choices, ok := chunkMap["choices"].([]interface{}); ok && len(choices) > 0 {
						if c0, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := c0["delta"].(map[string]interface{}); ok {
								if c, ok := delta["content"].(string); ok {
									chunkText = c
								}
							}
						}
					}

					if len(chunkText) > 0 {
						accumulated.WriteString(chunkText)
						_, _ = ExecuteCallable(callback, &object.String{Value: chunkText})
					}
				}

				return &object.String{Value: accumulated.String()}
			},
		}

	case "ai_structured":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 {
					return &object.Error{Message: "ai_structured requires 2 arguments (options, schema_or_example)"}
				}
				schemaJSON, _ := json.Marshal(bramandObjectToNative(args[1]))
				schemaInstruction := fmt.Sprintf("\nYou MUST respond ONLY with valid, raw JSON matching this format: %s\nDo not include any conversational preamble or markdown code fences.", string(schemaJSON))

				var model, prompt, endpoint, apiKey string
				endpoint = "http://localhost:11434/api/generate"
				if opts, ok := args[0].(*object.Hash); ok {
					for _, pair := range opts.Pairs {
						k := pair.Key.Inspect()
						if k == "model" {
							model = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								model = s.Value
							}
						} else if k == "prompt" {
							prompt = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								prompt = s.Value
							}
						} else if k == "endpoint" || k == "url" {
							endpoint = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								endpoint = s.Value
							}
						} else if k == "api_key" {
							apiKey = pair.Value.Inspect()
							if s, ok := pair.Value.(*object.String); ok {
								apiKey = s.Value
							}
						}
					}
				}

				fullPrompt := prompt + schemaInstruction
				promptBuiltin := getBuiltinByName("ai_prompt")
				optsPairs := make(map[object.HashKey]object.HashPair)
				setOpt := func(k, v string) {
					kObj := &object.String{Value: k}
					optsPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
				}
				setOpt("endpoint", endpoint)
				if apiKey != "" {
					setOpt("api_key", apiKey)
				}

				rawResp := promptBuiltin.Fn(&object.String{Value: model}, &object.String{Value: fullPrompt}, &object.Hash{Pairs: optsPairs})
				if errObj, ok := rawResp.(*object.Error); ok {
					return errObj
				}

				respStr := rawResp.Inspect()
				if sObj, ok := rawResp.(*object.String); ok {
					respStr = sObj.Value
				}

				cleaned := strings.TrimSpace(respStr)
				if strings.HasPrefix(cleaned, "```json") {
					cleaned = strings.TrimPrefix(cleaned, "```json")
					cleaned = strings.TrimSuffix(cleaned, "```")
				} else if strings.HasPrefix(cleaned, "```") {
					cleaned = strings.TrimPrefix(cleaned, "```")
					cleaned = strings.TrimSuffix(cleaned, "```")
				}
				cleaned = strings.TrimSpace(cleaned)

				var parsed interface{}
				if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
					return &object.Error{Message: fmt.Sprintf("failed to parse structured JSON: %s. Response was: %q", err, cleaned)}
				}
				return nativeToBramandObject(parsed)
			},
		}

	// ============================================================================
	// Day 6 Vector Similarity & AI Embeddings
	// ============================================================================
	case "cosine_similarity":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to cosine_similarity must be vectors or arrays of numbers"}
				}
				sim, err := calcCosineSimilarity(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: sim}
			},
		}

	case "cosine_distance":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to cosine_distance must be vectors or arrays of numbers"}
				}
				sim, err := calcCosineSimilarity(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: 1.0 - sim}
			},
		}

	case "euclidean_distance":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to euclidean_distance must be vectors or arrays of numbers"}
				}
				dist, err := calcEuclideanDistance(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: dist}
			},
		}

	case "manhattan_distance":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want=2", len(args))}
				}
				v1, err1 := extractVectorSlice(args[0])
				v2, err2 := extractVectorSlice(args[1])
				if err1 != nil || err2 != nil {
					return &object.Error{Message: "arguments to manhattan_distance must be vectors or arrays of numbers"}
				}
				dist, err := calcManhattanDistance(v1, v2)
				if err != nil {
					return &object.Error{Message: err.Error()}
				}
				return &object.Float{Value: dist}
			},
		}

	case "vector_search":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 4 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 to 4", len(args))}
				}
				query, err := extractVectorSlice(args[0])
				if err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid query vector: %s", err)}
				}
				candList, ok := args[1].(*object.Array)
				if !ok {
					return &object.Error{Message: "candidates must be an Array"}
				}

				topK := len(candList.Elements)
				if len(args) >= 3 {
					if intObj, ok := args[2].(*object.Integer); ok && intObj.Value > 0 {
						topK = int(intObj.Value)
					}
				}

				metric := "cosine"
				if len(args) >= 4 {
					if sObj, ok := args[3].(*object.String); ok {
						metric = sObj.Value
					}
				}

				type searchResult struct {
					item  object.Object
					score float64
				}
				var results []searchResult

				for _, cand := range candList.Elements {
					var candVec []float64
					if h, ok := cand.(*object.Hash); ok {
						for _, p := range h.Pairs {
							k := p.Key.Inspect()
							if k == "embedding" || k == "vector" || k == "vec" {
								candVec, _ = extractVectorSlice(p.Value)
								break
							}
						}
					}
					if candVec == nil {
						candVec, _ = extractVectorSlice(cand)
					}
					if candVec == nil || len(candVec) != len(query) {
						continue
					}

					var score float64
					switch metric {
					case "euclidean":
						d, _ := calcEuclideanDistance(query, candVec)
						score = d
					case "manhattan":
						d, _ := calcManhattanDistance(query, candVec)
						score = d
					default: // cosine
						s, _ := calcCosineSimilarity(query, candVec)
						score = s
					}
					results = append(results, searchResult{item: cand, score: score})
				}

				if metric == "euclidean" || metric == "manhattan" {
					sort.Slice(results, func(i, j int) bool {
						return results[i].score < results[j].score
					})
				} else {
					sort.Slice(results, func(i, j int) bool {
						return results[i].score > results[j].score
					})
				}

				if len(results) > topK {
					results = results[:topK]
				}

				outArray := make([]object.Object, len(results))
				for i, res := range results {
					pairs := make(map[object.HashKey]object.HashPair)
					itemKey := &object.String{Value: "item"}
					scoreKey := &object.String{Value: "score"}
					pairs[itemKey.HashKey()] = object.HashPair{Key: itemKey, Value: res.item}
					pairs[scoreKey.HashKey()] = object.HashPair{Key: scoreKey, Value: &object.Float{Value: res.score}}
					outArray[i] = &object.Hash{Pairs: pairs}
				}
				return &object.Array{Elements: outArray}
			},
		}

	// Day 7 Testing & Developer Tooling
	case "assert":
		return &object.Builtin{
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
		}

	case "assert_eq":
		return &object.Builtin{
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
		}

	case "assert_ne":
		return &object.Builtin{
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
		}

	case "panic":
		return &object.Builtin{
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
		}

	case "test":
		return &object.Builtin{
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
						fmt.Printf("  \033[31m✗ %s (%v)\033[0m\n", nameStr, elapsed)
						fmt.Printf("    \033[31m--> %s\033[0m\n", err.Error())
					}
				} else {
					ctx.TotalPassed++
					if !silent {
						fmt.Printf("  \033[32m✓ %s (%v)\033[0m\n", nameStr, elapsed)
					}
				}

				ctx.Cases = append(ctx.Cases, res)
				ctx.mu.Unlock()
				return nativeBoolToBooleanObject(err == nil)
			},
		}

	case "describe":
		return &object.Builtin{
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
		}

	case "time_now":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Float{Value: float64(time.Now().UnixNano()) / 1e9}
			},
		}

	case "time_ms":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				return &object.Integer{Value: time.Now().UnixMilli()}
			},
		}

	case "benchmark":
		return &object.Builtin{
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
		}

	case "args", "argv":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				elements := make([]object.Object, len(programArgs))
				for i, a := range programArgs {
					elements[i] = &object.String{Value: a}
				}
				return &object.Array{Elements: elements}
			},
		}

	case "sort":
		return &object.Builtin{
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
		}

	case "reverse":
		return &object.Builtin{
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
		}

	case "split":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2", len(args))}
				}
				strObj, ok1 := args[0].(*object.String)
				sepObj, ok2 := args[1].(*object.String)
				if !ok1 || !ok2 {
					return &object.Error{Message: "arguments to `split` must be (STRING, STRING)"}
				}
				parts := strings.Split(strObj.Value, sepObj.Value)
				elems := make([]object.Object, len(parts))
				for i, p := range parts {
					elems[i] = &object.String{Value: p}
				}
				return &object.Array{Elements: elems}
			},
		}

	case "join":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2", len(args))}
				}
				arr, ok1 := args[0].(*object.Array)
				sepObj, ok2 := args[1].(*object.String)
				if !ok1 || !ok2 {
					return &object.Error{Message: "arguments to `join` must be (ARRAY, STRING)"}
				}
				parts := make([]string, len(arr.Elements))
				for i, el := range arr.Elements {
					if s, ok := el.(*object.String); ok {
						parts[i] = s.Value
					} else {
						parts[i] = el.Inspect()
					}
				}
				return &object.String{Value: strings.Join(parts, sepObj.Value)}
			},
		}

	case "trim":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 1 || len(args) > 2 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 1 or 2", len(args))}
				}
				strObj, ok := args[0].(*object.String)
				if !ok {
					return &object.Error{Message: "first argument to `trim` must be STRING"}
				}
				if len(args) == 2 {
					cutset, ok := args[1].(*object.String)
					if !ok {
						return &object.Error{Message: "second argument to `trim` must be STRING"}
					}
					return &object.String{Value: strings.Trim(strObj.Value, cutset.Value)}
				}
				return &object.String{Value: strings.TrimSpace(strObj.Value)}
			},
		}

	case "to_lower":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: "to_lower requires 1 argument"}
				}
				s := args[0].Inspect()
				if strObj, ok := args[0].(*object.String); ok {
					s = strObj.Value
				}
				return &object.String{Value: strings.ToLower(s)}
			},
		}

	case "to_upper":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 1 {
					return &object.Error{Message: "to_upper requires 1 argument"}
				}
				s := args[0].Inspect()
				if strObj, ok := args[0].(*object.String); ok {
					s = strObj.Value
				}
				return &object.String{Value: strings.ToUpper(s)}
			},
		}

	case "replace":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) != 3 {
					return &object.Error{Message: "replace requires 3 arguments: (str, old, new)"}
				}
				strObj, ok1 := args[0].(*object.String)
				oldObj, ok2 := args[1].(*object.String)
				newObj, ok3 := args[2].(*object.String)
				if !ok1 || !ok2 || !ok3 {
					return &object.Error{Message: "all arguments to `replace` must be strings"}
				}
				return &object.String{Value: strings.ReplaceAll(strObj.Value, oldObj.Value, newObj.Value)}
			},
		}

	case "time_iso":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				t := time.Now().UTC()
				if len(args) > 0 {
					switch v := args[0].(type) {
					case *object.Integer:
						t = time.Unix(v.Value, 0).UTC()
					case *object.Float:
						sec := int64(v.Value)
						nsec := int64((v.Value - float64(sec)) * 1e9)
						t = time.Unix(sec, nsec).UTC()
					}
				}
				return &object.String{Value: t.Format(time.RFC3339)}
			},
		}

	case "date_format":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				t := time.Now()
				layout := "2006-01-02 15:04:05"
				if len(args) >= 1 {
					if i, ok := args[0].(*object.Integer); ok {
						t = time.Unix(i.Value, 0)
					}
				}
				if len(args) >= 2 {
					if s, ok := args[1].(*object.String); ok {
						layout = s.Value
					}
				}
				return &object.String{Value: t.Format(layout)}
			},
		}

	default:
		return nil
	}
}

// ---------------- Serialization & Math Conversion Helpers ----------------

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

// ---------------- Networking & Vector Similarity Helpers ----------------

func executeHttpRequest(method, urlStr, bodyStr string, headersObj object.Object, timeoutMs int64) object.Object {
	var bodyReader io.Reader
	if len(bodyStr) > 0 {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("http request error: %s", err)}
	}

	if hObj, ok := headersObj.(*object.Hash); ok {
		for _, pair := range hObj.Pairs {
			req.Header.Set(pair.Key.Inspect(), pair.Value.Inspect())
		}
	}

	timeout := 15 * time.Second
	if timeoutMs > 0 {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}
	client := &http.Client{Timeout: timeout}

	resp, err := client.Do(req)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("http request failed: %s", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &object.Error{Message: fmt.Sprintf("reading response body failed: %s", err)}
	}
	respBody := string(respBytes)

	pairs := make(map[object.HashKey]object.HashPair)
	setStr := func(k, v string) {
		kObj := &object.String{Value: k}
		pairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: v}}
	}
	statusKey := &object.String{Value: "status"}
	pairs[statusKey.HashKey()] = object.HashPair{Key: statusKey, Value: &object.Integer{Value: int64(resp.StatusCode)}}
	statusCodeKey := &object.String{Value: "status_code"}
	pairs[statusCodeKey.HashKey()] = object.HashPair{Key: statusCodeKey, Value: &object.Integer{Value: int64(resp.StatusCode)}}
	setStr("status_text", resp.Status)
	setStr("body", respBody)

	hdrPairs := make(map[object.HashKey]object.HashPair)
	for hk, hvs := range resp.Header {
		if len(hvs) > 0 {
			kObj := &object.String{Value: hk}
			hdrPairs[kObj.HashKey()] = object.HashPair{Key: kObj, Value: &object.String{Value: hvs[0]}}
		}
	}
	hdrKey := &object.String{Value: "headers"}
	pairs[hdrKey.HashKey()] = object.HashPair{Key: hdrKey, Value: &object.Hash{Pairs: hdrPairs}}

	jsonKey := &object.String{Value: "json"}
	pairs[jsonKey.HashKey()] = object.HashPair{
		Key: jsonKey,
		Value: &object.Builtin{
			Fn: func(a ...object.Object) object.Object {
				var v interface{}
				if err := json.Unmarshal([]byte(respBody), &v); err != nil {
					return &object.Error{Message: fmt.Sprintf("invalid JSON: %s", err)}
				}
				return nativeToBramandObject(v)
			},
		},
	}

	return &object.Hash{Pairs: pairs}
}

func extractVectorSlice(obj object.Object) ([]float64, error) {
	switch o := obj.(type) {
	case *object.Vector:
		res := make([]float64, len(o.Elements))
		copy(res, o.Elements)
		return res, nil
	case *object.Array:
		res := make([]float64, len(o.Elements))
		for i, el := range o.Elements {
			v, ok := toFloat(el)
			if !ok {
				return nil, fmt.Errorf("element at index %d is not a number", i)
			}
			res[i] = v
		}
		return res, nil
	default:
		return nil, fmt.Errorf("expected Vector or Array of numbers, got %s", obj.Type())
	}
}

func calcCosineSimilarity(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0, nil
	}
	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	if math.Abs(sim-1.0) < 1e-12 {
		return 1.0, nil
	}
	if math.Abs(sim-(-1.0)) < 1e-12 {
		return -1.0, nil
	}
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}
	return sim, nil
}

func calcEuclideanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var sum float64
	for i := range a {
		d := a[i] - b[i]
		sum += d * d
	}
	return math.Sqrt(sum), nil
}

func calcManhattanDistance(a, b []float64) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimensions mismatch: %d != %d", len(a), len(b))
	}
	var sum float64
	for i := range a {
		sum += math.Abs(a[i] - b[i])
	}
	return sum, nil
}
