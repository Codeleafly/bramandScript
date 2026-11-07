package vm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bramandscript/object"
)

func init() {
	RegisterBuiltin("read_file", &object.Builtin{
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
	})

	RegisterBuiltin("write_file", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermWrite); err != nil {
				return &object.Error{Message: err.Error()}
			}
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
	})

	RegisterBuiltin("append_file", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermWrite); err != nil {
				return &object.Error{Message: err.Error()}
			}
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
	})

	RegisterBuiltin("delete_file", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermWrite); err != nil {
				return &object.Error{Message: err.Error()}
			}
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
	})

	RegisterBuiltin("file_exists", &object.Builtin{
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
	})

	RegisterBuiltin("list_dir", &object.Builtin{
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
	})

	RegisterBuiltin("chmod", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermWrite); err != nil {
				return &object.Error{Message: err.Error()}
			}
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
	})

	RegisterBuiltin("is_dir", &object.Builtin{
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
	})

	RegisterBuiltin("env", &object.Builtin{
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
	})

	RegisterBuiltin("set_env", &object.Builtin{
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
	})

	RegisterBuiltin("exec", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermRun); err != nil {
				return &object.Error{Message: err.Error()}
			}
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
			} else if len(args) == 1 && strings.Contains(cmdStr, " ") {
				parts := strings.Fields(cmdStr)
				if len(parts) > 0 {
					cmdStr = parts[0]
					cmdArgs = parts[1:]
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
	})

	RegisterBuiltin("eprint", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for i, arg := range args {
				if i > 0 {
					fmt.Fprint(os.Stderr, " ")
				}
				fmt.Fprint(os.Stderr, arg.Inspect())
			}
			return Null
		},
	})

	RegisterBuiltin("eprintln", &object.Builtin{
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
	})

	RegisterBuiltin("read_line", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readLineFromStdin(args)
		},
	})

	RegisterBuiltin("input", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return readLineFromStdin(args)
		},
	})

	RegisterBuiltin("color", &object.Builtin{
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
	})

	RegisterBuiltin("styled", &object.Builtin{
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
	})

	RegisterBuiltin("now", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: time.Now().Unix()}
		},
	})

	RegisterBuiltin("time_now", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Float{Value: float64(time.Now().UnixNano()) / 1e9}
		},
	})

	RegisterBuiltin("time_ms", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: time.Now().UnixMilli()}
		},
	})

	RegisterBuiltin("time_iso", &object.Builtin{
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
	})

	RegisterBuiltin("date_format", &object.Builtin{
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
	})
}
