package vm

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"bramandscript/object"
)

func extractProcess(arg object.Object) (*object.Process, error) {
	if p, ok := arg.(*object.Process); ok {
		return p, nil
	}
	if h, ok := arg.(*object.Hash); ok {
		rawKey := (&object.String{Value: "raw"}).HashKey()
		h.Mu.RLock()
		pair, found := h.Pairs[rawKey]
		h.Mu.RUnlock()
		if found {
			if p, ok := pair.Value.(*object.Process); ok {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("expected Process object, got %s", arg.Type())
}

func init() {
	RegisterBuiltin("proc_spawn", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if err := CheckPermission(PermRun); err != nil {
				return &object.Error{Message: err.Error(), Fatal: true}
			}
			if len(args) < 1 {
				return &object.Error{Message: "proc_spawn requires at least 1 argument (cmd)"}
			}

			cmdStr := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				cmdStr = s.Value
			}

			var cmdArgs []string
			if len(args) >= 2 {
				switch a := args[1].(type) {
				case *object.Array:
					a.Mu.RLock()
					for _, el := range a.Elements {
						if s, ok := el.(*object.String); ok {
							cmdArgs = append(cmdArgs, s.Value)
						} else {
							cmdArgs = append(cmdArgs, el.Inspect())
						}
					}
					a.Mu.RUnlock()
				case *object.String:
					cmdArgs = append(cmdArgs, a.Value)
				case *object.Null:
					// empty args
				default:
					cmdArgs = append(cmdArgs, a.Inspect())
				}
			}

			if len(cmdArgs) == 0 && strings.Contains(cmdStr, " ") {
				parts := strings.Fields(cmdStr)
				if len(parts) > 0 {
					cmdStr = parts[0]
					cmdArgs = parts[1:]
				}
			}

			cwd := ""
			var customEnv []string
			if len(args) >= 3 {
				if opts, ok := args[2].(*object.Hash); ok {
					opts.Mu.RLock()
					// Check cwd
					kCwd := (&object.String{Value: "cwd"}).HashKey()
					if p, found := opts.Pairs[kCwd]; found {
						if s, ok := p.Value.(*object.String); ok {
							cwd = s.Value
						} else {
							cwd = p.Value.Inspect()
						}
					}

					// Check env
					kEnv := (&object.String{Value: "env"}).HashKey()
					if p, found := opts.Pairs[kEnv]; found {
						if envHash, ok := p.Value.(*object.Hash); ok {
							envHash.Mu.RLock()
							envMap := make(map[string]string)
							for _, e := range os.Environ() {
								parts := strings.SplitN(e, "=", 2)
								if len(parts) == 2 {
									envMap[parts[0]] = parts[1]
								}
							}
							for _, ep := range envHash.Pairs {
								ek := ep.Key.Inspect()
								if s, ok := ep.Key.(*object.String); ok {
									ek = s.Value
								}
								ev := ep.Value.Inspect()
								if s, ok := ep.Value.(*object.String); ok {
									ev = s.Value
								}
								envMap[ek] = ev
							}
							envHash.Mu.RUnlock()

							for k, v := range envMap {
								customEnv = append(customEnv, k+"="+v)
							}
						}
					}
					opts.Mu.RUnlock()
				}
			}

			cmd := exec.Command(cmdStr, cmdArgs...)
			if cwd != "" {
				cmd.Dir = cwd
			}
			if len(customEnv) > 0 {
				cmd.Env = customEnv
			}

			stdinPipe, err := cmd.StdinPipe()
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("proc_spawn stdin pipe failed: %s", err)}
			}

			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				_ = stdinPipe.Close()
				return &object.Error{Message: fmt.Sprintf("proc_spawn stdout pipe failed: %s", err)}
			}

			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				_ = stdinPipe.Close()
				_ = stdoutPipe.Close()
				return &object.Error{Message: fmt.Sprintf("proc_spawn stderr pipe failed: %s", err)}
			}

			err = cmd.Start()
			if err != nil {
				_ = stdinPipe.Close()
				_ = stdoutPipe.Close()
				_ = stderrPipe.Close()
				return &object.Error{Message: fmt.Sprintf("proc_spawn failed: %s", err)}
			}

			return &object.Process{
				Cmd:    cmd,
				Stdin:  stdinPipe,
				Stdout: stdoutPipe,
				Stderr: stderrPipe,
				PID:    cmd.Process.Pid,
			}
		},
	})

	RegisterBuiltin("proc_stdin_write", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "proc_stdin_write requires at least 2 arguments (proc, data)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			var b []byte
			switch v := args[1].(type) {
			case *object.String:
				b = []byte(v.Value)
			case *object.Bytes:
				v.Mu.RLock()
				b = make([]byte, len(v.Value))
				copy(b, v.Value)
				v.Mu.RUnlock()
			default:
				b = []byte(v.Inspect())
			}

			n, writeErr := proc.WriteStdin(b)
			if writeErr != nil {
				return &object.Error{Message: fmt.Sprintf("proc_stdin_write failed: %s", writeErr)}
			}
			return &object.Integer{Value: int64(n)}
		},
	})

	RegisterBuiltin("proc_stdin_close", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_stdin_close requires 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			if closeErr := proc.CloseStdin(); closeErr != nil {
				return &object.Error{Message: fmt.Sprintf("proc_stdin_close failed: %s", closeErr)}
			}
			return True
		},
	})

	RegisterBuiltin("proc_stdout_read", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_stdout_read requires at least 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			maxBytes := 4096
			if len(args) >= 2 {
				if i, ok := args[1].(*object.Integer); ok {
					maxBytes = int(i.Value)
				}
			}
			data, readErr := proc.ReadStdout(maxBytes)
			if readErr == io.EOF && len(data) == 0 {
				return Null
			}
			if readErr != nil && len(data) == 0 {
				return &object.Error{Message: fmt.Sprintf("proc_stdout_read failed: %s", readErr)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("proc_stdout_read_all", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_stdout_read_all requires 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			data, readErr := proc.ReadStdoutAll()
			if readErr != nil {
				return &object.Error{Message: fmt.Sprintf("proc_stdout_read_all failed: %s", readErr)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("proc_stderr_read", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_stderr_read requires at least 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			maxBytes := 4096
			if len(args) >= 2 {
				if i, ok := args[1].(*object.Integer); ok {
					maxBytes = int(i.Value)
				}
			}
			data, readErr := proc.ReadStderr(maxBytes)
			if readErr == io.EOF && len(data) == 0 {
				return Null
			}
			if readErr != nil && len(data) == 0 {
				return &object.Error{Message: fmt.Sprintf("proc_stderr_read failed: %s", readErr)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("proc_stderr_read_all", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_stderr_read_all requires 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			data, readErr := proc.ReadStderrAll()
			if readErr != nil {
				return &object.Error{Message: fmt.Sprintf("proc_stderr_read_all failed: %s", readErr)}
			}
			return &object.String{Value: string(data)}
		},
	})

	RegisterBuiltin("proc_wait", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_wait requires 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			code, pid, _ := proc.Wait()

			pairs := make(map[object.HashKey]object.HashPair)
			kExitCode := &object.String{Value: "exit_code"}
			vExitCode := &object.Integer{Value: int64(code)}
			pairs[kExitCode.HashKey()] = object.HashPair{Key: kExitCode, Value: vExitCode}

			kCode := &object.String{Value: "code"}
			pairs[kCode.HashKey()] = object.HashPair{Key: kCode, Value: vExitCode}

			kPid := &object.String{Value: "pid"}
			vPid := &object.Integer{Value: int64(pid)}
			pairs[kPid.HashKey()] = object.HashPair{Key: kPid, Value: vPid}

			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("proc_kill", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_kill requires at least 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}

			var sig os.Signal = syscall.SIGTERM
			if len(args) >= 2 {
				sigStr := strings.ToUpper(args[1].Inspect())
				if s, ok := args[1].(*object.String); ok {
					sigStr = strings.ToUpper(s.Value)
				}
				switch sigStr {
				case "SIGKILL", "KILL", "9":
					sig = syscall.SIGKILL
				case "SIGINT", "INT", "2":
					sig = syscall.SIGINT
				case "SIGHUP", "HUP", "1":
					sig = syscall.SIGHUP
				case "SIGTERM", "TERM", "15":
					sig = syscall.SIGTERM
				default:
					sig = syscall.SIGTERM
				}
			}

			if killErr := proc.Kill(sig); killErr != nil {
				return &object.Error{Message: fmt.Sprintf("proc_kill failed: %s", killErr)}
			}
			return True
		},
	})

	RegisterBuiltin("proc_pid", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "proc_pid requires 1 argument (proc)"}
			}
			proc, err := extractProcess(args[0])
			if err != nil {
				return &object.Error{Message: err.Error()}
			}
			return &object.Integer{Value: int64(proc.PID)}
		},
	})

	RegisterBuiltin("os_platform", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOOS}
		},
	})

	RegisterBuiltin("os_arch", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.String{Value: runtime.GOARCH}
		},
	})

	RegisterBuiltin("os_cpus", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(runtime.NumCPU())}
		},
	})

	RegisterBuiltin("os_memory_info", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			pairs := make(map[object.HashKey]object.HashPair)
			setMem := func(name string, val uint64) {
				k := &object.String{Value: name}
				v := &object.Integer{Value: int64(val)}
				pairs[k.HashKey()] = object.HashPair{Key: k, Value: v}
			}
			setMem("alloc", m.Alloc)
			setMem("total_alloc", m.TotalAlloc)
			setMem("sys", m.Sys)
			setMem("num_gc", uint64(m.NumGC))
			setMem("heap_alloc", m.HeapAlloc)
			setMem("heap_sys", m.HeapSys)
			setMem("heap_idle", m.HeapIdle)
			setMem("heap_inuse", m.HeapInuse)

			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("os_env_all", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			pairs := make(map[object.HashKey]object.HashPair)
			for _, envStr := range os.Environ() {
				parts := strings.SplitN(envStr, "=", 2)
				key := &object.String{Value: parts[0]}
				val := ""
				if len(parts) == 2 {
					val = parts[1]
				}
				pairs[key.HashKey()] = object.HashPair{Key: key, Value: &object.String{Value: val}}
			}
			return &object.Hash{Pairs: pairs}
		},
	})

	RegisterBuiltin("os_get_env", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 1 {
				return &object.Error{Message: "os_get_env requires at least 1 argument (key)"}
			}
			key := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				key = s.Value
			}
			val, ok := os.LookupEnv(key)
			if !ok {
				if len(args) >= 2 {
					return args[1]
				}
				return Null
			}
			return &object.String{Value: val}
		},
	})

	RegisterBuiltin("os_set_env", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) < 2 {
				return &object.Error{Message: "os_set_env requires at least 2 arguments (key, val)"}
			}
			key := args[0].Inspect()
			if s, ok := args[0].(*object.String); ok {
				key = s.Value
			}
			val := args[1].Inspect()
			if s, ok := args[1].(*object.String); ok {
				val = s.Value
			}
			if err := os.Setenv(key, val); err != nil {
				return &object.Error{Message: fmt.Sprintf("os_set_env failed: %s", err)}
			}
			return True
		},
	})

	RegisterBuiltin("os_cwd", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			dir, err := os.Getwd()
			if err != nil {
				return &object.Error{Message: fmt.Sprintf("os_cwd failed: %s", err)}
			}
			return &object.String{Value: filepath.ToSlash(dir)}
		},
	})

	RegisterBuiltin("os_pid", &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: int64(os.Getpid())}
		},
	})
}
