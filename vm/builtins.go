package vm

import (
	"bufio"
	"bytes"
	crand "crypto/rand"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"bramandscript/object"
)

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
					return &object.Integer{Value: int64(len(arg.Value))}
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
					fmt.Print(arg.Inspect())
				}
				fmt.Println()
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
				length := len(arr.Elements)
				newElements := make([]object.Object, length+1)
				copy(newElements, arr.Elements)
				newElements[length] = args[1]
				return &object.Array{Elements: newElements}
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
				if len(args) > 0 {
					fmt.Print(args[0].Inspect())
				}
				reader := bufio.NewReader(os.Stdin)
				text, _ := reader.ReadString('\n')
				return &object.String{Value: strings.TrimRight(text, "\r\n")}
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
	case "slice":
		return &object.Builtin{
			Fn: func(args ...object.Object) object.Object {
				if len(args) < 2 || len(args) > 3 {
					return &object.Error{Message: fmt.Sprintf("wrong number of arguments. got=%d, want 2 or 3", len(args))}
				}
				arr, ok := args[0].(*object.Array)
				if !ok {
					return &object.Error{Message: fmt.Sprintf("first argument to `slice` must be ARRAY, got %s", args[0].Type())}
				}
				start, ok := args[1].(*object.Integer)
				if !ok {
					return &object.Error{Message: "second argument to `slice` must be INTEGER"}
				}

				length := int64(len(arr.Elements))
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
				copy(newElements, arr.Elements[s:e])
				return &object.Array{Elements: newElements}
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
				if len(args) > 0 {
					fmt.Print(args[0].Inspect())
				}
				reader := bufio.NewReader(os.Stdin)
				text, _ := reader.ReadString('\n')
				return &object.String{Value: strings.TrimRight(text, "\r\n")}
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
	case "big_int":
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

	case "big_float":
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
	case "vec":
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

	case "mean":
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
	case "sha256":
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

	case "hmac_sha256":
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
