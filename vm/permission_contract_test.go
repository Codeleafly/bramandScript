package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compileAndRunScript(t *testing.T, src string, baseDir string) (object.Object, error) {
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser error: %v", p.Errors())
	}

	comp := compiler.New()
	comp.SetBaseDir(baseDir)
	err := comp.Compile(prog)
	if err != nil {
		return nil, err
	}

	machine := New(comp.Bytecode())
	SetActiveVM(machine)
	defer SetActiveVM(nil)

	runErr := machine.Run()
	if runErr != nil {
		return nil, runErr
	}

	return machine.LastPoppedStackElem(), nil
}

func TestSupplyChainPermissionContract_SubprocessForbidden(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Create restricted package with subprocess: false
	pkgDir := filepath.Join(tempBase, "registry", "github.com", "restricted", "sh_tool", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)

	manifestJSON := `{
		"name": "sh_tool",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": false
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644)

	pkgCode := `
	let run_bad = fn() {
		return exec("echo pwned");
	};
	`
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(pkgCode), 0644)

	// User main program imports the package and calls run_bad()
	userCode := `
	import "github.com/restricted/sh_tool@1.0.0" as tool;
	let res = tool.run_bad();
	`
	_, err := compileAndRunScript(t, userCode, tempBase)
	if err == nil {
		t.Fatalf("expected SecurityError when restricted library executes subprocess, but got nil")
	}

	if !strings.Contains(err.Error(), "SecurityError") || !strings.Contains(err.Error(), "subprocess: false") {
		t.Fatalf("expected clear diagnostic SecurityError mentioning subprocess: false, got: %s", err)
	}
}

func TestSupplyChainPermissionContract_NetworkForbidden(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Create restricted package with network: false
	pkgDir := filepath.Join(tempBase, "registry", "github.com", "restricted", "net_tool", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)

	manifestJSON := `{
		"name": "net_tool",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": false
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644)

	pkgCode := `
	let start_listener = fn() {
		return tcp_listen("127.0.0.1:9999");
	};
	`
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(pkgCode), 0644)

	userCode := `
	import "github.com/restricted/net_tool@1.0.0" as net_tool;
	let res = net_tool.start_listener();
	`
	_, err := compileAndRunScript(t, userCode, tempBase)
	if err == nil {
		t.Fatalf("expected SecurityError when restricted library opens network socket, but got nil")
	}

	if !strings.Contains(err.Error(), "SecurityError") || !strings.Contains(err.Error(), "network: false") {
		t.Fatalf("expected clear diagnostic SecurityError mentioning network: false, got: %s", err)
	}
}

func TestSupplyChainPermissionContract_FilesystemWriteForbidden(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	pkgDir := filepath.Join(tempBase, "registry", "github.com", "restricted", "fs_tool", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)

	manifestJSON := `{
		"name": "fs_tool",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": false
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644)

	pkgCode := `
	let try_write = fn(target) {
		return write_file(target, "malicious data");
	};
	`
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(pkgCode), 0644)

	userCode := `
	import "github.com/restricted/fs_tool@1.0.0" as fs_tool;
	let res = fs_tool.try_write("/tmp/bmd_hack.txt");
	`
	_, err := compileAndRunScript(t, userCode, tempBase)
	if err == nil {
		t.Fatalf("expected SecurityError when restricted library writes file, but got nil")
	}

	if !strings.Contains(err.Error(), "SecurityError") || !strings.Contains(err.Error(), "filesystem_write: false") {
		t.Fatalf("expected clear diagnostic SecurityError mentioning filesystem_write: false, got: %s", err)
	}
}

func TestSupplyChainPermissionContract_TopLevelImportGuard(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Create library that tries to execute subprocess during top-level module load
	pkgDir := filepath.Join(tempBase, "registry", "github.com", "restricted", "sneaky", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)

	manifestJSON := `{
		"name": "sneaky",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": false
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644)

	// Top level statement runs exec directly
	pkgCode := `
	let initial_payload = exec("whoami");
	let safe_val = 123;
	`
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(pkgCode), 0644)

	userCode := `
	import "github.com/restricted/sneaky@1.0.0" as s;
	`
	_, err := compileAndRunScript(t, userCode, tempBase)
	if err == nil {
		t.Fatalf("expected SecurityError on sneaky top-level import, got nil")
	}

	if !strings.Contains(err.Error(), "SecurityError") || !strings.Contains(err.Error(), "subprocess: false") {
		t.Fatalf("expected SecurityError for top-level module execution, got: %s", err)
	}
}

func TestSupplyChainPermissionContract_PermittedPasses(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Create package declaring subprocess: true
	pkgDir := filepath.Join(tempBase, "registry", "github.com", "allowed", "worker", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)

	manifestJSON := `{
		"name": "worker",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": true
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644)

	pkgCode := `
	let run_echo = fn() {
		let res = exec("echo OK");
		return res["stdout"];
	};
	`
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(pkgCode), 0644)

	userCode := `
	import "github.com/allowed/worker@1.0.0" as w;
	let res = w.run_echo();
	`
	// In test mode, host permissions pass, and package declared subprocess: true so it must succeed
	val, err := compileAndRunScript(t, userCode, tempBase)
	if err != nil {
		t.Fatalf("expected permitted action to succeed, got error: %s", err)
	}

	if strObj, ok := val.(*object.String); !ok || !strings.Contains(strObj.Value, "OK") {
		t.Fatalf("expected echo OK output, got: %v", val)
	}
}

func TestSupplyChainPermissionContract_TransitiveCallStackForbidden(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Package B: unrestricted helper library
	pkgBDir := filepath.Join(tempBase, "registry", "github.com", "helpers", "runner", "1.0.0")
	_ = os.MkdirAll(pkgBDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgBDir, "main.bmd"), []byte(`
	let do_exec = fn() {
		return exec("echo helper");
	};
	`), 0644)

	// Package A: restricted library with subprocess: false
	pkgADir := filepath.Join(tempBase, "registry", "github.com", "caller", "wrapper", "1.0.0")
	_ = os.MkdirAll(pkgADir, 0755)
	manifestA := `{
		"name": "wrapper",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"subprocess": false
		}
	}`
	_ = os.WriteFile(filepath.Join(pkgADir, "bmdpkg.json"), []byte(manifestA), 0644)
	_ = os.WriteFile(filepath.Join(pkgADir, "main.bmd"), []byte(`
	import "github.com/helpers/runner@1.0.0" as runner;
	let delegate_exec = fn() {
		return runner.do_exec();
	};
	`), 0644)

	// User calls Package A which delegates to Package B
	userCode := `
	import "github.com/caller/wrapper@1.0.0" as wrap;
	let res = wrap.delegate_exec();
	`
	_, err := compileAndRunScript(t, userCode, tempBase)
	if err == nil {
		t.Fatalf("expected transitive delegation to fail due to caller permission contract, got nil")
	}

	if !strings.Contains(err.Error(), "SecurityError") || !strings.Contains(err.Error(), "wrapper") {
		t.Fatalf("expected SecurityError blaming wrapper package contract on call stack, got: %s", err)
	}
}
