package vm

import (
	"bufio"
	"bytes"
	"bramandscript/object"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
	default:
		return nil
	}
}
