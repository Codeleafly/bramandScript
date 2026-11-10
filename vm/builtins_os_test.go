package vm

import (
	"strings"
	"testing"
	"time"

	"bramandscript/object"
)

func TestBuiltinsOSProcessSpawnAndEcho(t *testing.T) {
	spawnFn := GetBuiltin("proc_spawn").Fn
	readAllFn := GetBuiltin("proc_stdout_read_all").Fn
	waitFn := GetBuiltin("proc_wait").Fn
	pidFn := GetBuiltin("proc_pid").Fn

	procObj := spawnFn(&object.String{Value: "echo"}, &object.Array{
		Elements: []object.Object{
			&object.String{Value: "Hello BramandScript IPC"},
		},
	})
	if procObj.Type() == object.ERROR_OBJ {
		t.Fatalf("spawn failed: %s", procObj.Inspect())
	}

	pidVal := pidFn(procObj)
	if intObj, ok := pidVal.(*object.Integer); !ok || intObj.Value <= 0 {
		t.Fatalf("expected valid PID, got %s", pidVal.Inspect())
	}

	outObj := readAllFn(procObj)
	if strObj, ok := outObj.(*object.String); !ok || !strings.Contains(strObj.Value, "Hello BramandScript IPC") {
		t.Fatalf("expected stdout to contain Hello BramandScript IPC, got %s", outObj.Inspect())
	}

	waitObj := waitFn(procObj)
	if hashObj, ok := waitObj.(*object.Hash); !ok {
		t.Fatalf("expected wait result hash, got %s", waitObj.Inspect())
	} else {
		keyExit := (&object.String{Value: "exit_code"}).HashKey()
		if p, ok := hashObj.Pairs[keyExit]; !ok || p.Value.(*object.Integer).Value != 0 {
			t.Fatalf("expected exit code 0, got %s", p.Value.Inspect())
		}
	}
}

func TestBuiltinsOSStdinStreamingPipe(t *testing.T) {
	spawnFn := GetBuiltin("proc_spawn").Fn
	writeFn := GetBuiltin("proc_stdin_write").Fn
	closeStdinFn := GetBuiltin("proc_stdin_close").Fn
	readAllFn := GetBuiltin("proc_stdout_read_all").Fn
	waitFn := GetBuiltin("proc_wait").Fn

	procObj := spawnFn(&object.String{Value: "cat"}, &object.Array{Elements: []object.Object{}})
	if procObj.Type() == object.ERROR_OBJ {
		t.Fatalf("spawn cat failed: %s", procObj.Inspect())
	}

	inputData := "Line 1: Stdin streaming\nLine 2: Pure IPC Stream\n"
	writeFn(procObj, &object.String{Value: inputData})
	closeStdinFn(procObj)

	outObj := readAllFn(procObj)
	if strObj, ok := outObj.(*object.String); !ok || strObj.Value != inputData {
		t.Fatalf("expected stdout %q, got %q", inputData, outObj.Inspect())
	}

	waitObj := waitFn(procObj)
	if hashObj, ok := waitObj.(*object.Hash); ok {
		keyExit := (&object.String{Value: "exit_code"}).HashKey()
		if p, ok := hashObj.Pairs[keyExit]; !ok || p.Value.(*object.Integer).Value != 0 {
			t.Fatalf("expected exit code 0, got %s", p.Value.Inspect())
		}
	}
}

func TestBuiltinsOSProcessKill(t *testing.T) {
	spawnFn := GetBuiltin("proc_spawn").Fn
	killFn := GetBuiltin("proc_kill").Fn
	waitFn := GetBuiltin("proc_wait").Fn

	procObj := spawnFn(&object.String{Value: "sleep"}, &object.Array{
		Elements: []object.Object{&object.String{Value: "10"}},
	})
	if procObj.Type() == object.ERROR_OBJ {
		t.Fatalf("spawn sleep failed: %s", procObj.Inspect())
	}

	time.Sleep(20 * time.Millisecond)
	killRes := killFn(procObj, &object.String{Value: "SIGKILL"})
	if killRes.Type() == object.ERROR_OBJ {
		t.Fatalf("kill failed: %s", killRes.Inspect())
	}

	waitObj := waitFn(procObj)
	if hashObj, ok := waitObj.(*object.Hash); ok {
		keyExit := (&object.String{Value: "exit_code"}).HashKey()
		if p, ok := hashObj.Pairs[keyExit]; ok {
			// Process was killed, exit code non-zero
			if p.Value.(*object.Integer).Value == 0 {
				t.Fatalf("expected non-zero exit code for killed process")
			}
		}
	}
}

func TestBuiltinsOSIntrospection(t *testing.T) {
	platformFn := GetBuiltin("os_platform").Fn
	archFn := GetBuiltin("os_arch").Fn
	cpusFn := GetBuiltin("os_cpus").Fn
	memFn := GetBuiltin("os_memory_info").Fn
	envAllFn := GetBuiltin("os_env_all").Fn
	cwdFn := GetBuiltin("os_cwd").Fn
	pidFn := GetBuiltin("os_pid").Fn

	if p, ok := platformFn().(*object.String); !ok || p.Value == "" {
		t.Fatalf("invalid platform: %s", platformFn().Inspect())
	}
	if a, ok := archFn().(*object.String); !ok || a.Value == "" {
		t.Fatalf("invalid arch: %s", archFn().Inspect())
	}
	if c, ok := cpusFn().(*object.Integer); !ok || c.Value <= 0 {
		t.Fatalf("invalid cpus: %s", cpusFn().Inspect())
	}
	if m, ok := memFn().(*object.Hash); !ok || len(m.Pairs) == 0 {
		t.Fatalf("invalid mem info: %s", memFn().Inspect())
	}
	if e, ok := envAllFn().(*object.Hash); !ok || len(e.Pairs) == 0 {
		t.Fatalf("invalid env: %s", envAllFn().Inspect())
	}
	if cwd, ok := cwdFn().(*object.String); !ok || cwd.Value == "" {
		t.Fatalf("invalid cwd: %s", cwdFn().Inspect())
	}
	if pid, ok := pidFn().(*object.Integer); !ok || pid.Value <= 0 {
		t.Fatalf("invalid pid: %s", pidFn().Inspect())
	}
}
