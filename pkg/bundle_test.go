package pkg

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleSerialization(t *testing.T) {
	orig := &Bundle{
		Version:    1,
		MainFile:   "main.bmd",
		MainSource: "let x = 42;",
		Files: map[string]BundleFile{
			"github.com/alice/math": {
				VirtualPath: "github.com/alice/math",
				Content:     "let add = fn(a, b) { a + b };",
				PackageName: "math",
				Manifest: &Manifest{
					Name:    "math",
					Version: "1.0.0",
					Permissions: Permissions{
						Network: false,
					},
				},
			},
		},
	}

	data, err := SerializeBundle(orig)
	if err != nil {
		t.Fatalf("SerializeBundle failed: %s", err)
	}

	if !IsBundleData(data) {
		t.Fatalf("IsBundleData returned false for serialized bundle")
	}

	decoded, err := DeserializeBundle(data)
	if err != nil {
		t.Fatalf("DeserializeBundle failed: %s", err)
	}

	if decoded.MainFile != orig.MainFile || decoded.MainSource != orig.MainSource {
		t.Fatalf("decoded bundle mismatch: got MainFile=%s, MainSource=%s", decoded.MainFile, decoded.MainSource)
	}

	bf, ok := decoded.Files["github.com/alice/math"]
	if !ok {
		t.Fatalf("module not found in decoded bundle")
	}
	if bf.PackageName != "math" || bf.Manifest == nil || bf.Manifest.Permissions.Network {
		t.Fatalf("module metadata mismatch: %+v", bf)
	}
}

func TestActiveBundleResolution(t *testing.T) {
	ClearActiveBundle()
	defer ClearActiveBundle()

	if HasActiveBundle() {
		t.Fatalf("expected HasActiveBundle false initially")
	}

	bundle := &Bundle{
		Version:    1,
		MainFile:   "app.bmd",
		MainSource: `import "github.com/tools/helper" as h; let res = h.val;`,
		Files: map[string]BundleFile{
			"github.com/tools/helper": {
				VirtualPath: "github.com/tools/helper",
				Content:     "let val = 100;",
				PackageName: "helper",
			},
			"github.com/tools/sub/util.bmd": {
				VirtualPath: "github.com/tools/sub/util.bmd",
				Content:     "let subVal = 200;",
				PackageName: "helper",
			},
		},
	}

	RegisterActiveBundle(bundle)
	if !HasActiveBundle() {
		t.Fatalf("expected HasActiveBundle true after registration")
	}

	// Direct resolution
	vPath, file, ok := ResolveBundleModule("github.com/tools/helper", "")
	if !ok || vPath != "embed:bundle/github.com/tools/helper" || file == nil {
		t.Fatalf("direct bundle resolution failed: vPath=%s, ok=%t", vPath, ok)
	}

	contentBytes, err := ReadActiveBundleFile(vPath)
	if err != nil || string(contentBytes) != "let val = 100;" {
		t.Fatalf("ReadActiveBundleFile failed: content=%s, err=%v", string(contentBytes), err)
	}

	// Relative resolution inside embedded bundle
	subPath, _, ok := ResolveBundleModule("./util.bmd", "embed:bundle/github.com/tools/sub")
	if !ok || !strings.Contains(subPath, "util.bmd") {
		t.Fatalf("relative bundle resolution failed: subPath=%s, ok=%t", subPath, ok)
	}
}

func TestBuildBundleFromImports(t *testing.T) {
	tempDir := t.TempDir()
	mainFile := filepath.Join(tempDir, "main.bmd")
	helperFile := filepath.Join(tempDir, "helpers", "utils.bmd")
	_ = os.MkdirAll(filepath.Dir(helperFile), 0755)

	mainSrc := `import "./helpers/utils.bmd" as u; let out = u.answer;`
	helperSrc := `let answer = 42;`

	_ = os.WriteFile(mainFile, []byte(mainSrc), 0644)
	_ = os.WriteFile(helperFile, []byte(helperSrc), 0644)

	importedPaths := map[string]string{
		"./helpers/utils.bmd": helperFile,
	}
	importedSources := map[string][]byte{
		helperFile: []byte(helperSrc),
	}
	importedManifests := map[string]*Manifest{}

	bundle, data, err := BuildBundle(mainFile, mainSrc, importedPaths, importedSources, importedManifests)
	if err != nil {
		t.Fatalf("BuildBundle failed: %s", err)
	}

	if len(bundle.Files) == 0 {
		t.Fatalf("expected bundled files, got 0")
	}

	if !IsBundleData(data) {
		t.Fatalf("expected valid bundle binary payload")
	}

	// Verify the helper module can be resolved from the bundle
	RegisterActiveBundle(bundle)
	defer ClearActiveBundle()

	vPath, f, ok := ResolveBundleModule("./helpers/utils.bmd", tempDir)
	if !ok || f == nil || !strings.Contains(f.Content, "let answer = 42;") {
		t.Fatalf("bundled helper resolution failed: vPath=%s, ok=%t", vPath, ok)
	}
}

func TestStandaloneBinaryBuildAndRun(t *testing.T) {
	// Locate bramand executable in workspace
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %s", err)
	}
	bmdBin := filepath.Join(cwd, "..", "bramand")
	if fi, err := os.Stat(bmdBin); err != nil || fi.IsDir() {
		// Try cwd
		bmdBin = filepath.Join(cwd, "bramand")
		if fi, err := os.Stat(bmdBin); err != nil || fi.IsDir() {
			t.Skip("bramand binary not compiled yet, skipping binary test")
		}
	}

	tempLibs := t.TempDir()
	_ = os.Setenv("BMD_LIBS_DIR", tempLibs)
	defer os.Unsetenv("BMD_LIBS_DIR")

	// Pre-seed a registry package
	pkgDir := filepath.Join(tempLibs, "registry", "github.com", "bundled", "mathlib", "1.0.0")
	_ = os.MkdirAll(pkgDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(`{"name":"mathlib","version":"1.0.0","entry":"main.bmd","permissions":{}}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(`let add = fn(a, b) { a + b };`), 0644)

	// Create a project importing this package and a relative helper
	projectDir := t.TempDir()
	mainFile := filepath.Join(projectDir, "main.bmd")
	helperFile := filepath.Join(projectDir, "helper.bmd")

	_ = os.WriteFile(helperFile, []byte(`let multiply = fn(a, b) { a * b };`), 0644)
	_ = os.WriteFile(mainFile, []byte(`
	import "github.com/bundled/mathlib@1.0.0" as m;
	import "./helper.bmd" as h;
	let sum = m.add(10, 20);
	let product = h.multiply(sum, 2);
	println("STANDALONE_SUCCESS:" + str(product));
	`), 0644)

	// Build standalone executable
	outputExe := filepath.Join(projectDir, "standalone_app")
	buildCmd := execCommand(bmdBin, "build", mainFile, "-o", outputExe)
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bmd build failed: %s\nOutput:\n%s", err, string(buildOut))
	}

	// Verify build output contains bundle information
	if !strings.Contains(string(buildOut), "Standalone Executable Builder") {
		t.Fatalf("expected builder banner in output: %s", string(buildOut))
	}

	// Now run the built binary in an isolated environment with NO libs and from /tmp!
	emptyLibs := t.TempDir()
	runCmd := execCommand(outputExe)
	runCmd.Dir = emptyLibs
	runCmd.Env = append(os.Environ(), "BMD_LIBS_DIR="+emptyLibs)
	runOut, err := runCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("running standalone binary failed: %s\nOutput:\n%s", err, string(runOut))
	}

	if !strings.Contains(string(runOut), "STANDALONE_SUCCESS:60") {
		t.Fatalf("expected 'STANDALONE_SUCCESS:60' in output, got:\n%s", string(runOut))
	}
}

func execCommand(name string, args ...string) *execCmd {
	return &execCmd{name: name, args: args}
}

type execCmd struct {
	name string
	args []string
	Dir  string
	Env  []string
}

func (c *execCmd) CombinedOutput() ([]byte, error) {
	cmd := exec.Command(c.name, c.args...)
	if c.Dir != "" {
		cmd.Dir = c.Dir
	}
	if len(c.Env) > 0 {
		cmd.Env = c.Env
	}
	return cmd.CombinedOutput()
}

