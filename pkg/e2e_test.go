package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_CompleteLifecycle tests the complete centralized package lifecycle in Go.
func TestE2E_CompleteLifecycle(t *testing.T) {
	tempBase := t.TempDir()

	// 1. Install a local package
	pkgDir := filepath.Join(tempBase, "pkg_source")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create pkg dir: %v", err)
	}
	manifestJSON := `{
		"name": "e2e_core",
		"version": "1.0.0",
		"entry": "main.bmd",
		"permissions": {
			"network": false,
			"filesystem_write": false,
			"subprocess": false
		}
	}`
	if err := os.WriteFile(filepath.Join(pkgDir, "bmdpkg.json"), []byte(manifestJSON), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte("let multiply = fn(a, b) { a * b };"), 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	inst := NewInstaller(tempBase)
	res, err := inst.Install(pkgDir)
	if err != nil {
		t.Fatalf("inst.Install failed: %v", err)
	}
	if res.PackageName != "e2e_core" || res.Version != "1.0.0" {
		t.Fatalf("unexpected install result: %+v", res)
	}

	// 2. Query cache info
	info, err := CacheInfo(tempBase)
	if err != nil {
		t.Fatalf("CacheInfo failed: %v", err)
	}
	if info.RegistryCount < 1 {
		t.Fatalf("expected at least 1 registry package, got %d", info.RegistryCount)
	}
	if info.IntegrityRecords < 1 {
		t.Fatalf("expected at least 1 integrity record, got %d", info.IntegrityRecords)
	}

	// 3. Verify cache integrity
	verifyRes, err := CacheVerify(tempBase)
	if err != nil {
		t.Fatalf("CacheVerify failed: %v", err)
	}
	if verifyRes.FailedCount > 0 {
		t.Fatalf("expected 0 verification failures, got %d", verifyRes.FailedCount)
	}

	// 4. Test offline determinism
	instOffline := NewInstaller(tempBase)
	instOffline.Offline = true

	// Missing remote package must fail fast with offline error
	_, err = instOffline.Install("github.com/missing/pkg@1.0.0")
	if err == nil {
		t.Fatalf("expected error installing missing package in offline mode")
	}
	if !strings.Contains(err.Error(), "offline mode") {
		t.Fatalf("expected offline mode error message, got: %v", err)
	}

	// 5. Test bmd get on a project manifest
	projectDir := filepath.Join(tempBase, "consumer_project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("failed to create consumer project: %v", err)
	}
	consumerManifest := `{
		"name": "consumer_app",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"local/e2e_core": "1.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(projectDir, "bmdpkg.json"), []byte(consumerManifest), 0644); err != nil {
		t.Fatalf("failed to write consumer manifest: %v", err)
	}

	getRes, err := Get(projectDir, tempBase)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if getRes.FailedCount > 0 {
		t.Fatalf("expected 0 get failures, got %d", getRes.FailedCount)
	}
	if getRes.CachedCount < 1 {
		t.Fatalf("expected at least 1 cached dependency, got %d", getRes.CachedCount)
	}

	// 6. Test bundle creation
	mainSrc := "import \"local/e2e_core\" as core;\nlet ans = core.multiply(6, 7);\n"
	resolvedPath := filepath.Join(res.RegistryDir, "main.bmd")
	bundle, bundleBytes, err := BuildBundle(
		filepath.Join(projectDir, "main.bmd"),
		mainSrc,
		map[string]string{"local/e2e_core": resolvedPath},
		map[string][]byte{"local/e2e_core": []byte("let multiply = fn(a, b) { a * b };")},
		map[string]*Manifest{"local/e2e_core": res.Manifest},
	)
	if err != nil {
		t.Fatalf("BuildBundle failed: %v", err)
	}
	if len(bundleBytes) == 0 {
		t.Fatalf("bundleBytes empty")
	}
	if !IsBundleData(bundleBytes) {
		t.Fatalf("bundleBytes signature check failed")
	}
	deserialized, err := DeserializeBundle(bundleBytes)
	if err != nil {
		t.Fatalf("DeserializeBundle failed: %v", err)
	}
	if len(deserialized.Files) != len(bundle.Files) {
		t.Fatalf("bundle files mismatch: expected %d, got %d", len(bundle.Files), len(deserialized.Files))
	}
}
