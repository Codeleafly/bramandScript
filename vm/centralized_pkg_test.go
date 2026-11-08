package vm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"bramandscript/pkg"
)

func runScript(t *testing.T, source string, baseDir ...string) (*VM, object.Object, error) {
	t.Helper()
	l := lexer.New(source)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	comp := compiler.New()
	if len(baseDir) > 0 && baseDir[0] != "" {
		comp.SetBaseDir(baseDir[0])
	}
	err := comp.Compile(prog)
	if err != nil {
		return nil, nil, err
	}

	vm := New(comp.Bytecode())
	err = vm.Run()
	return vm, vm.LastPoppedStackElem(), err
}

func TestCentralizedPackageEngine_Stage1_Stdlib(t *testing.T) {
	script := `
		import "std/math" as m1;
		import "std/math" as m2;
		let same = (m1 == m2);
		let pi = m1.PI;
		same;
	`
	_, res, err := runScript(t, script)
	if err != nil {
		t.Fatalf("runScript failed: %v", err)
	}
	b, ok := res.(*object.Boolean)
	if !ok || !b.Value {
		t.Errorf("expected module singleton pointer equality (m1 == m2), got %v", res)
	}
}

func TestCentralizedPackageEngine_Stage2_RelativePathsThreeTier(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmd_stage2_*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Tier 1: Directory with bmdpkg.json specifying entry
	pkgManifest := filepath.Join(tempDir, "tier1_pkg")
	os.MkdirAll(filepath.Join(pkgManifest, "lib"), 0755)
	os.WriteFile(filepath.Join(pkgManifest, "bmdpkg.json"), []byte(`{
		"name": "tier1",
		"version": "1.0.0",
		"entry": "lib/engine.bmd",
		"permissions": {}
	}`), 0644)
	os.WriteFile(filepath.Join(pkgManifest, "lib", "engine.bmd"), []byte(`let answer = 42;`), 0644)

	script1 := `
		import "./tier1_pkg" as t1;
		t1.answer;
	`
	_, res1, err := runScript(t, script1, tempDir)
	if err != nil {
		t.Fatalf("runScript Tier 1 failed: %v", err)
	}
	if num, ok := res1.(*object.Integer); !ok || num.Value != 42 {
		t.Errorf("expected 42 from Tier 1 bmdpkg.json, got %v", res1)
	}

	// Tier 2: Directory with main.bmd
	pkgMain := filepath.Join(tempDir, "tier2_pkg")
	os.MkdirAll(pkgMain, 0755)
	os.WriteFile(filepath.Join(pkgMain, "main.bmd"), []byte(`let greeting = "hello main";`), 0644)

	script2 := `
		import "./tier2_pkg" as t2;
		t2.greeting;
	`
	_, res2, err := runScript(t, script2, tempDir)
	if err != nil {
		t.Fatalf("runScript Tier 2 failed: %v", err)
	}
	if str, ok := res2.(*object.String); !ok || str.Value != "hello main" {
		t.Errorf("expected 'hello main' from Tier 2 main.bmd, got %v", res2)
	}

	// Tier 3: Directory with index.bmd
	pkgIndex := filepath.Join(tempDir, "tier3_pkg")
	os.MkdirAll(pkgIndex, 0755)
	os.WriteFile(filepath.Join(pkgIndex, "index.bmd"), []byte(`let status = "ok index";`), 0644)

	script3 := `
		import "./tier3_pkg" as t3;
		t3.status;
	`
	_, res3, err := runScript(t, script3, tempDir)
	if err != nil {
		t.Fatalf("runScript Tier 3 failed: %v", err)
	}
	if str, ok := res3.(*object.String); !ok || str.Value != "ok index" {
		t.Errorf("expected 'ok index' from Tier 3 index.bmd, got %v", res3)
	}

	// Broken manifest: specifies missing entrypoint
	pkgBroken := filepath.Join(tempDir, "broken_pkg")
	os.MkdirAll(pkgBroken, 0755)
	os.WriteFile(filepath.Join(pkgBroken, "bmdpkg.json"), []byte(`{
		"name": "broken",
		"version": "1.0.0",
		"entry": "nonexistent.bmd",
		"permissions": {}
	}`), 0644)

	scriptBroken := `import "./broken_pkg" as b;`
	_, _, err = runScript(t, scriptBroken, tempDir)
	if err == nil {
		t.Fatalf("expected BrokenManifestError, but succeeded")
	}
	if !strings.Contains(err.Error(), "BrokenManifestError") {
		t.Errorf("expected BrokenManifestError in error message, got %v", err)
	}

	// Empty directory: no bmdpkg.json, no main.bmd, no index.bmd -> clean diagnostic error
	pkgEmpty := filepath.Join(tempDir, "empty_pkg")
	os.MkdirAll(pkgEmpty, 0755)

	scriptEmpty := `import "./empty_pkg" as e;`
	_, _, err = runScript(t, scriptEmpty, tempDir)
	if err == nil {
		t.Fatalf("expected clean diagnostic error, but succeeded")
	}
	expectedMsg := "No entrypoint found: package must define bmdpkg.json, main.bmd, or index.bmd"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("expected '%s' in error message, got %v", expectedMsg, err)
	}
}

func TestCentralizedPackageEngine_Stage3_RegistrySemverResolution(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_registry_libs_*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	// Set BMD_LIBS_DIR to mock centralized cache
	t.Setenv(pkg.EnvBmdLibsDir, tempLibs)

	// Create multiple versions of github.com/core/algo: 1.0.0 and 2.1.0
	v1Dir := pkg.RegistryGithubPath("core", "algo", "1.0.0")
	v2Dir := pkg.RegistryGithubPath("core", "algo", "2.1.0")
	os.MkdirAll(v1Dir, 0755)
	os.MkdirAll(v2Dir, 0755)

	os.WriteFile(filepath.Join(v1Dir, "main.bmd"), []byte(`let ver = "1.0.0";`), 0644)
	os.WriteFile(filepath.Join(v2Dir, "main.bmd"), []byte(`let ver = "2.1.0";`), 0644)

	// Create local dev package: local/devtools@dev
	devDir := pkg.RegistryLocalPath("devtools")
	os.MkdirAll(devDir, 0755)
	os.WriteFile(filepath.Join(devDir, "index.bmd"), []byte(`let env = "development";`), 0644)

	// 1. Unversioned import resolves to latest semver (2.1.0)
	script1 := `
		import "github.com/core/algo" as algo;
		algo.ver;
	`
	_, res1, err := runScript(t, script1)
	if err != nil {
		t.Fatalf("runScript latest semver failed: %v", err)
	}
	if s, ok := res1.(*object.String); !ok || s.Value != "2.1.0" {
		t.Errorf("expected '2.1.0' from latest version resolution, got %v", res1)
	}

	// 2. Specific version import via @1.0.0 resolves to 1.0.0
	script2 := `
		import "github.com/core/algo@1.0.0" as algo_old;
		algo_old.ver;
	`
	_, res2, err := runScript(t, script2)
	if err != nil {
		t.Fatalf("runScript specific version failed: %v", err)
	}
	if s, ok := res2.(*object.String); !ok || s.Value != "1.0.0" {
		t.Errorf("expected '1.0.0' from specific version import, got %v", res2)
	}

	// 3. Local package import
	script3 := `
		import "local/devtools@dev" as dt;
		dt.env;
	`
	_, res3, err := runScript(t, script3)
	if err != nil {
		t.Fatalf("runScript local devtools failed: %v", err)
	}
	if s, ok := res3.(*object.String); !ok || s.Value != "development" {
		t.Errorf("expected 'development' from local dev package, got %v", res3)
	}
}

func TestCentralizedPackageEngine_Stage4_ModuleSingletonTable(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_singleton_libs_*")
	if err != nil {
		t.Fatalf("temp dir creation failed: %v", err)
	}
	defer os.RemoveAll(tempLibs)
	t.Setenv(pkg.EnvBmdLibsDir, tempLibs)

	// Create registry package
	mathDir := pkg.RegistryGithubPath("math", "calc", "1.0.0")
	os.MkdirAll(mathDir, 0755)
	os.WriteFile(filepath.Join(mathDir, "main.bmd"), []byte(`
		let state = 100;
		let add = fn(a, b) { return a + b; };
	`), 0644)

	// Script imports same module in global scope and within nested function
	script := `
		import "github.com/math/calc" as c1;

		let get_from_inner = fn() {
			import "github.com/math/calc" as c2;
			return c2;
		};

		let c2 = get_from_inner();
		let same_ptr = (c1 == c2);
		same_ptr;
	`
	vm, res, err := runScript(t, script)
	if err != nil {
		t.Fatalf("runScript module singleton failed: %v", err)
	}
	if b, ok := res.(*object.Boolean); !ok || !b.Value {
		t.Errorf("expected exact pointer equality for module singleton across scopes, got %v", res)
	}

	// Verify VM in-memory module singleton table
	modTable := vm.ModuleTable()
	if modTable == nil {
		t.Fatalf("vm.ModuleTable() returned nil")
	}
	if modTable.Count() == 0 {
		t.Errorf("expected at least 1 registered module in ModuleTable, got %d", modTable.Count())
	}

	// Verify entrypoint resolution registered
	entryPath := filepath.Join(mathDir, "main.bmd")
	modHash, found := vm.GetModule(entryPath)
	if !found || modHash == nil {
		t.Fatalf("expected to find '%s' in ModuleTable", entryPath)
	}
	if !modHash.ReadOnly {
		t.Errorf("expected module export hash to be marked ReadOnly")
	}

	// Verify Immutability: attempting to mutate module export hash must fail with TypeError
	mutateScript := `
		import "github.com/math/calc" as c;
		c.state = 999;
	`
	_, _, err = runScript(t, mutateScript)
	if err == nil {
		t.Fatalf("expected mutation of module namespace to fail, but succeeded")
	}
	if !strings.Contains(err.Error(), "read-only module namespace") {
		t.Errorf("expected read-only module namespace error, got %v", err)
	}
}
