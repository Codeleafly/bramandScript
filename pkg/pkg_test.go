package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureDirectoryStructure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmd_libs_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	err = EnsureDirectoryStructure(tempDir)
	if err != nil {
		t.Fatalf("EnsureDirectoryStructure failed: %v", err)
	}

	expectedDirs := []string{
		filepath.Join(tempDir, "cache", "tarballs"),
		filepath.Join(tempDir, "registry"),
		filepath.Join(tempDir, "locks"),
	}

	for _, d := range expectedDirs {
		fi, err := os.Stat(d)
		if err != nil {
			t.Errorf("expected directory '%s' to exist: %v", d, err)
		} else if !fi.IsDir() {
			t.Errorf("expected '%s' to be a directory", d)
		}
	}
}

func TestIntegrityDB(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmd_integrity_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "locks", "integrity.db")
	db, err := LoadIntegrityDB(dbPath)
	if err != nil {
		t.Fatalf("LoadIntegrityDB failed: %v", err)
	}

	sampleData := []byte("let add = fn(a, b) { return a + b; };")
	chk := ComputeSHA256(sampleData)
	db.Set("github.com/bramand/math@1.0.0", chk, int64(len(sampleData)))

	// Verify in-memory
	ok, err := db.Verify("github.com/bramand/math@1.0.0", sampleData)
	if !ok || err != nil {
		t.Errorf("Verify failed for matching data: ok=%v, err=%v", ok, err)
	}

	// Verify mismatch
	ok, err = db.Verify("github.com/bramand/math@1.0.0", []byte("tampered content"))
	if ok || err == nil {
		t.Errorf("expected verify failure for tampered content, got ok=%v", ok)
	}

	// Save to disk and reload
	if err := db.Save(dbPath); err != nil {
		t.Fatalf("db.Save failed: %v", err)
	}

	reloadedDB, err := LoadIntegrityDB(dbPath)
	if err != nil {
		t.Fatalf("LoadIntegrityDB reload failed: %v", err)
	}

	rec, found := reloadedDB.Get("github.com/bramand/math@1.0.0")
	if !found {
		t.Fatalf("expected key to exist in reloaded DB")
	}
	if rec.Checksum != chk {
		t.Errorf("expected checksum %s, got %s", chk, rec.Checksum)
	}

	// Test RecordFile and VerifyFile
	testFile := filepath.Join(tempDir, "package.tar.gz")
	if err := os.WriteFile(testFile, sampleData, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	fRec, err := reloadedDB.RecordFile("tarballs/package.tar.gz", testFile)
	if err != nil {
		t.Fatalf("RecordFile failed: %v", err)
	}
	if fRec.Checksum != chk {
		t.Errorf("RecordFile checksum mismatch: want %s, got %s", chk, fRec.Checksum)
	}

	vOk, err := reloadedDB.VerifyFile("tarballs/package.tar.gz", testFile)
	if !vOk || err != nil {
		t.Errorf("VerifyFile failed: ok=%v, err=%v", vOk, err)
	}
}

func TestManifestParsingAndValidation(t *testing.T) {
	validJSON := `{
		"name": "crypto_suite",
		"version": "1.2.0",
		"description": "Cryptographic utilities for BramandScript",
		"entry": "src/lib.bmd",
		"license": "Apache-2.0",
		"authors": ["Bramand Core Team <dev@bramand.org>"],
		"permissions": {
			"network": false,
			"filesystem_read": true,
			"filesystem_write": false,
			"subprocess": false
		},
		"dependencies": {
			"github.com/bramand/hash": "1.0.0"
		},
		"compatibility": {
			"bmd_min": "1.0.0",
			"architecture": ["arm64", "amd64"]
		}
	}`

	m, err := ParseManifest([]byte(validJSON))
	if err != nil {
		t.Fatalf("ParseManifest failed on valid JSON: %v", err)
	}

	if m.Name != "crypto_suite" || m.Version != "1.2.0" {
		t.Errorf("parsed name or version incorrect: %s@%s", m.Name, m.Version)
	}
	if m.Entry != "src/lib.bmd" {
		t.Errorf("parsed entry incorrect: %s", m.Entry)
	}
	if !m.Permissions.FilesystemRead || m.Permissions.Network {
		t.Errorf("permissions incorrectly parsed: %+v", m.Permissions)
	}
	if len(m.Compatibility.Architecture) != 2 {
		t.Errorf("expected 2 architectures, got %v", m.Compatibility.Architecture)
	}

	// Test flexible string architecture
	singleArchJSON := `{
		"name": "fast_math",
		"version": "0.5.0",
		"compatibility": {
			"architecture": "all"
		},
		"permissions": {}
	}`
	m2, err := ParseManifest([]byte(singleArchJSON))
	if err != nil {
		t.Fatalf("ParseManifest failed on single arch JSON: %v", err)
	}
	if len(m2.Compatibility.Architecture) != 1 || m2.Compatibility.Architecture[0] != "all" {
		t.Errorf("expected architecture ['all'], got %v", m2.Compatibility.Architecture)
	}

	// Test validation errors
	invalidCases := []struct {
		name    string
		json    string
		errFrag string
	}{
		{"missing name", `{"version": "1.0.0"}`, "'name' is required"},
		{"invalid name", `{"name": "bad name!", "version": "1.0.0"}`, "invalid package name"},
		{"missing version", `{"name": "valid_pkg"}`, "'version' is required"},
		{"invalid version", `{"name": "valid_pkg", "version": "not-semver"}`, "invalid semver"},
		{"path traversal entry", `{"name": "valid_pkg", "version": "1.0.0", "entry": "../secret.bmd"}`, "cannot point outside package root"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(tc.json))
			if err == nil {
				t.Fatalf("expected error for case '%s', but succeeded", tc.name)
			}
			if !strings.Contains(err.Error(), tc.errFrag) {
				t.Errorf("error '%v' does not contain expected fragment '%s'", err, tc.errFrag)
			}
		})
	}
}

func TestThreeTierEntrypointFallbackEngine(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmd_entrypoint_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: bmdpkg.json with valid entry
	pkg1 := filepath.Join(tempDir, "pkg1")
	os.MkdirAll(filepath.Join(pkg1, "src"), 0755)
	os.WriteFile(filepath.Join(pkg1, "bmdpkg.json"), []byte(`{"name":"pkg1","version":"1.0.0","entry":"src/lib.bmd","permissions":{}}`), 0644)
	os.WriteFile(filepath.Join(pkg1, "src", "lib.bmd"), []byte("let v = 1;"), 0644)

	res1, err := ResolvePackageEntrypoint(pkg1)
	if err != nil {
		t.Fatalf("Step 1 failed: %v", err)
	}
	if !strings.HasSuffix(res1, filepath.Join("src", "lib.bmd")) {
		t.Errorf("Step 1 resolved unexpected entrypoint: %s", res1)
	}

	// Step 1 Failure: bmdpkg.json specifies entry that does NOT exist -> BrokenManifestError
	pkg1Broken := filepath.Join(tempDir, "pkg1_broken")
	os.MkdirAll(pkg1Broken, 0755)
	os.WriteFile(filepath.Join(pkg1Broken, "bmdpkg.json"), []byte(`{"name":"pkg1b","version":"1.0.0","entry":"missing_entry.bmd","permissions":{}}`), 0644)

	_, err = ResolvePackageEntrypoint(pkg1Broken)
	if err == nil {
		t.Fatalf("expected BrokenManifestError, got nil")
	}
	bmErr, ok := err.(*BrokenManifestError)
	if !ok {
		t.Errorf("expected *BrokenManifestError, got %T: %v", err, err)
	} else if bmErr.Entry != "missing_entry.bmd" {
		t.Errorf("expected entry 'missing_entry.bmd', got '%s'", bmErr.Entry)
	}

	// Step 2: No bmdpkg.json, but main.bmd exists
	pkg2 := filepath.Join(tempDir, "pkg2")
	os.MkdirAll(pkg2, 0755)
	os.WriteFile(filepath.Join(pkg2, "main.bmd"), []byte("let v = 2;"), 0644)
	os.WriteFile(filepath.Join(pkg2, "index.bmd"), []byte("let v = 99;"), 0644) // main.bmd takes precedence over index.bmd

	res2, err := ResolvePackageEntrypoint(pkg2)
	if err != nil {
		t.Fatalf("Step 2 failed: %v", err)
	}
	if !strings.HasSuffix(res2, "main.bmd") {
		t.Errorf("Step 2 resolved unexpected entrypoint: %s", res2)
	}

	// Step 3: No bmdpkg.json, no main.bmd, but index.bmd exists
	pkg3 := filepath.Join(tempDir, "pkg3")
	os.MkdirAll(pkg3, 0755)
	os.WriteFile(filepath.Join(pkg3, "index.bmd"), []byte("let v = 3;"), 0644)

	res3, err := ResolvePackageEntrypoint(pkg3)
	if err != nil {
		t.Fatalf("Step 3 failed: %v", err)
	}
	if !strings.HasSuffix(res3, "index.bmd") {
		t.Errorf("Step 3 resolved unexpected entrypoint: %s", res3)
	}

	// Step 4: None exists -> clean diagnostic error
	pkgEmpty := filepath.Join(tempDir, "pkg_empty")
	os.MkdirAll(pkgEmpty, 0755)

	_, err = ResolvePackageEntrypoint(pkgEmpty)
	if err == nil {
		t.Fatalf("expected diagnostic error for empty package, got nil")
	}
	expectedFrag := "No entrypoint found: package must define bmdpkg.json, main.bmd, or index.bmd"
	if !strings.Contains(err.Error(), expectedFrag) {
		t.Errorf("expected diagnostic error containing '%s', got '%s'", expectedFrag, err.Error())
	}
}

func TestSemverAndRegistryResolution(t *testing.T) {
	// Version sorting test
	versions := []string{"1.0.0", "2.1.0", "1.2.0", "0.9.5", "2.1.0-alpha", "2.0.0"}
	sorted := SortVersions(versions)
	expectedOrder := []string{"2.1.0", "2.1.0-alpha", "2.0.0", "1.2.0", "1.0.0", "0.9.5"}

	for i, v := range expectedOrder {
		if sorted[i] != v {
			t.Errorf("version sort mismatch at %d: want %s, got %s", i, v, sorted[i])
		}
	}

	// Setup mock registry
	tempLibs, err := os.MkdirTemp("", "bmd_registry_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	// 1. github.com/author/pkg/1.0.0/ and 1.2.0/
	ghDir1 := RegistryGithubPath("alice", "math_lib", "1.0.0", tempLibs)
	ghDir2 := RegistryGithubPath("alice", "math_lib", "1.2.0", tempLibs)
	os.MkdirAll(ghDir1, 0755)
	os.MkdirAll(ghDir2, 0755)
	os.WriteFile(filepath.Join(ghDir1, "main.bmd"), []byte("let version = 1;"), 0644)
	os.WriteFile(filepath.Join(ghDir2, "main.bmd"), []byte("let version = 2;"), 0644)

	// Resolve unversioned github path -> should choose 1.2.0
	resGh, err := ResolveRegistryPackage("github.com/alice/math_lib", tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve github.com/alice/math_lib: %v", err)
	}
	if !strings.Contains(resGh, "1.2.0") {
		t.Errorf("expected resolution to 1.2.0, got %s", resGh)
	}

	// Resolve specific version via @
	resGhVer, err := ResolveRegistryPackage("github.com/alice/math_lib@1.0.0", tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve github.com/alice/math_lib@1.0.0: %v", err)
	}
	if !strings.Contains(resGhVer, "1.0.0") {
		t.Errorf("expected resolution to 1.0.0, got %s", resGhVer)
	}

	// 2. cdn/<domain_hash>/<version>/
	cdnDir := RegistryCdnPath("cdn_hash_abc", "2.0.0", tempLibs)
	os.MkdirAll(cdnDir, 0755)
	os.WriteFile(filepath.Join(cdnDir, "index.bmd"), []byte("let cdn = true;"), 0644)

	resCdn, err := ResolveRegistryPackage("cdn/cdn_hash_abc/2.0.0", tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve cdn package: %v", err)
	}
	if !strings.HasSuffix(resCdn, "index.bmd") {
		t.Errorf("expected index.bmd for cdn package, got %s", resCdn)
	}

	// 3. local/<name>@dev/
	localDir := RegistryLocalPath("my_tool", tempLibs)
	os.MkdirAll(localDir, 0755)
	os.WriteFile(filepath.Join(localDir, "main.bmd"), []byte("let dev = true;"), 0644)

	resLocal, err := ResolveRegistryPackage("local/my_tool@dev", tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve local package: %v", err)
	}
	if !strings.Contains(resLocal, "my_tool@dev") {
		t.Errorf("expected my_tool@dev resolution, got %s", resLocal)
	}

	// Bare local package name "my_tool"
	resBare, err := ResolveRegistryPackage("my_tool", tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve bare name 'my_tool': %v", err)
	}
	if !strings.Contains(resBare, "my_tool@dev") {
		t.Errorf("expected my_tool to resolve to local/my_tool@dev, got %s", resBare)
	}
}
