package compiler

import (
	"bramandscript/lexer"
	"bramandscript/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTransitiveDependencies(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempBase)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Package C: base dependency
	pkgCDir := filepath.Join(tempBase, "registry", "github.com", "core", "pkgC", "1.0.0")
	_ = os.MkdirAll(pkgCDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgCDir, "bmdpkg.json"), []byte(`{"name":"pkgC","version":"1.0.0","entry":"main.bmd","permissions":{}}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgCDir, "main.bmd"), []byte(`let base_val = 10;`), 0644)

	// Package B: intermediate dependency importing Package C
	pkgBDir := filepath.Join(tempBase, "registry", "github.com", "mid", "pkgB", "1.0.0")
	_ = os.MkdirAll(pkgBDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgBDir, "bmdpkg.json"), []byte(`{
		"name": "pkgB",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"pkgC": "github.com/core/pkgC@1.0.0"
		},
		"permissions": {}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgBDir, "main.bmd"), []byte(`
	import "pkgC" as c;
	let mid_val = c.base_val * 2;
	`), 0644)

	// Package A: top-level dependency importing Package B
	pkgADir := filepath.Join(tempBase, "registry", "github.com", "top", "pkgA", "1.0.0")
	_ = os.MkdirAll(pkgADir, 0755)
	_ = os.WriteFile(filepath.Join(pkgADir, "bmdpkg.json"), []byte(`{
		"name": "pkgA",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"pkgB": "github.com/mid/pkgB@1.0.0"
		},
		"permissions": {}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgADir, "main.bmd"), []byte(`
	import "pkgB" as b;
	let top_val = b.mid_val + 5;
	`), 0644)

	// User script imports Package A
	userCode := `
	import "github.com/top/pkgA@1.0.0" as a;
	let result = a.top_val;
	`
	l := lexer.New(userCode)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	comp := New()
	comp.SetBaseDir(tempBase)
	err := comp.Compile(prog)
	if err != nil {
		t.Fatalf("expected transitive dependency compilation to succeed, got: %s", err)
	}

	// Verify all 3 modules were recorded
	if len(comp.ImportedModulePaths()) < 3 {
		t.Fatalf("expected at least 3 imported modules recorded, got: %d (%v)", len(comp.ImportedModulePaths()), comp.ImportedModulePaths())
	}
}

func TestCircularImportDetection(t *testing.T) {
	tempDir := t.TempDir()

	fileA := filepath.Join(tempDir, "modA.bmd")
	fileB := filepath.Join(tempDir, "modB.bmd")

	// Module A imports Module B
	_ = os.WriteFile(fileA, []byte(`
	import "./modB.bmd" as b;
	let a_val = 1;
	`), 0644)

	// Module B imports Module A (cycle!)
	_ = os.WriteFile(fileB, []byte(`
	import "./modA.bmd" as a;
	let b_val = 2;
	`), 0644)

	src := `import "./modA.bmd" as a;`
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()

	comp := New()
	comp.SetBaseDir(tempDir)
	err := comp.Compile(prog)
	if err == nil {
		t.Fatalf("expected circular import error, got nil")
	}

	if !strings.Contains(err.Error(), "circular import detected") {
		t.Fatalf("expected 'circular import detected' message, got: %s", err)
	}
}

func TestDirectSelfImportDetection(t *testing.T) {
	tempDir := t.TempDir()

	fileSelf := filepath.Join(tempDir, "self.bmd")
	_ = os.WriteFile(fileSelf, []byte(`
	import "./self.bmd" as s;
	let x = 10;
	`), 0644)

	src := `import "./self.bmd" as s;`
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()

	comp := New()
	comp.SetBaseDir(tempDir)
	err := comp.Compile(prog)
	if err == nil {
		t.Fatalf("expected self-import circular error, got nil")
	}

	if !strings.Contains(err.Error(), "circular import detected") {
		t.Fatalf("expected 'circular import detected' message, got: %s", err)
	}
}

func TestDiamondDependencyReuseSingleton(t *testing.T) {
	tempDir := t.TempDir()

	fileD := filepath.Join(tempDir, "modD.bmd")
	fileB := filepath.Join(tempDir, "modB.bmd")
	fileC := filepath.Join(tempDir, "modC.bmd")
	fileA := filepath.Join(tempDir, "modA.bmd")

	// D is the shared leaf dependency
	_ = os.WriteFile(fileD, []byte(`let d_count = 100;`), 0644)

	// B imports D
	_ = os.WriteFile(fileB, []byte(`
	import "./modD.bmd" as d;
	let b_val = d.d_count + 1;
	`), 0644)

	// C imports D
	_ = os.WriteFile(fileC, []byte(`
	import "./modD.bmd" as d;
	let c_val = d.d_count + 2;
	`), 0644)

	// A imports both B and C (diamond graph)
	_ = os.WriteFile(fileA, []byte(`
	import "./modB.bmd" as b;
	import "./modC.bmd" as c;
	let total = b.b_val + c.c_val;
	`), 0644)

	src := `import "./modA.bmd" as a;`
	l := lexer.New(src)
	p := parser.New(l)
	prog := p.ParseProgram()

	comp := New()
	comp.SetBaseDir(tempDir)
	err := comp.Compile(prog)
	if err != nil {
		t.Fatalf("diamond dependency compilation failed unexpectedly: %s", err)
	}

	// Singleton table must contain D exactly once
	canonicalD := filepath.Clean(fileD)
	if _, ok := comp.moduleSingletons[canonicalD]; !ok {
		t.Fatalf("shared module D singleton not found in moduleSingletons")
	}
}
