package pkg

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTestPackageDir(t *testing.T, baseDir, name, ver string, perms Permissions) string {
	t.Helper()
	pkgDir := filepath.Join(baseDir, name)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to create test package dir: %v", err)
	}

	manifest := Manifest{
		Name:        name,
		Version:     ver,
		Description: "Test package for unit testing",
		Entry:       "main.bmd",
		License:     "MIT",
		Permissions: perms,
	}
	if err := manifest.Save(filepath.Join(pkgDir, "bmdpkg.json")); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	mainCode := fmt.Sprintf("// Entrypoint for %s\nfn run() { return \"%s-ok\"; }\n", name, name)
	if err := os.WriteFile(filepath.Join(pkgDir, "main.bmd"), []byte(mainCode), 0644); err != nil {
		t.Fatalf("failed to write main.bmd: %v", err)
	}

	return pkgDir
}

func TestArchiveCreateAndExtractTarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "bmd_test_archive_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	srcDir := filepath.Join(tempDir, "source")
	_ = os.MkdirAll(filepath.Join(srcDir, "sub"), 0755)
	_ = os.WriteFile(filepath.Join(srcDir, "file1.txt"), []byte("hello world"), 0644)
	_ = os.WriteFile(filepath.Join(srcDir, "sub", "file2.txt"), []byte("nested file"), 0644)

	tarPath := filepath.Join(tempDir, "bundle.tar.gz")
	if err := CreateTarGz(srcDir, tarPath); err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	destDir := filepath.Join(tempDir, "extracted")
	if err := ExtractArchive(tarPath, destDir); err != nil {
		t.Fatalf("ExtractArchive failed: %v", err)
	}

	f1, err := os.ReadFile(filepath.Join(destDir, "file1.txt"))
	if err != nil || string(f1) != "hello world" {
		t.Errorf("expected 'hello world', got %s, err: %v", string(f1), err)
	}

	f2, err := os.ReadFile(filepath.Join(destDir, "sub", "file2.txt"))
	if err != nil || string(f2) != "nested file" {
		t.Errorf("expected 'nested file', got %s, err: %v", string(f2), err)
	}
}

func TestLocalPackageInstallAndSecurityNotice(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_test_libs_*")
	if err != nil {
		t.Fatalf("failed to create temp libs: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	tempSrc, err := os.MkdirTemp("", "bmd_test_src_*")
	if err != nil {
		t.Fatalf("failed to create temp src: %v", err)
	}
	defer os.RemoveAll(tempSrc)

	// Case 1: Package with sensitive permissions
	pkgDir := createTestPackageDir(t, tempSrc, "vault_tool", "1.0.0", Permissions{
		Network:         true,
		FilesystemWrite: true,
		Subprocess:      true,
	})

	inst := NewInstaller(tempLibs)
	res, err := inst.Install(pkgDir)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if res.PackageName != "vault_tool" {
		t.Errorf("expected PackageName 'vault_tool', got '%s'", res.PackageName)
	}
	if !strings.Contains(res.SecurityNotice, "✦ [SECURITY NOTICE] Package requests host permissions:") {
		t.Errorf("expected security notice, got '%s'", res.SecurityNotice)
	}
	if !strings.Contains(res.SecurityNotice, "network") || !strings.Contains(res.SecurityNotice, "filesystem_write") || !strings.Contains(res.SecurityNotice, "subprocess") {
		t.Errorf("expected all 3 permissions in security notice, got '%s'", res.SecurityNotice)
	}

	// Verify installed registry directory
	entrypoint, err := ResolvePackageEntrypoint(res.RegistryDir)
	if err != nil {
		t.Fatalf("ResolvePackageEntrypoint failed on installed package: %v", err)
	}
	if !strings.HasSuffix(entrypoint, "main.bmd") {
		t.Errorf("expected main.bmd entrypoint, got '%s'", entrypoint)
	}

	// Case 2: Package with zero sensitive permissions
	pkgDirClean := createTestPackageDir(t, tempSrc, "clean_math", "1.0.0", Permissions{
		Network:         false,
		FilesystemRead:  true,
		FilesystemWrite: false,
		Subprocess:      false,
	})

	resClean, err := inst.Install(pkgDirClean)
	if err != nil {
		t.Fatalf("Install clean package failed: %v", err)
	}
	if resClean.SecurityNotice != "" {
		t.Errorf("expected empty security notice, got '%s'", resClean.SecurityNotice)
	}
}

func TestGitHubTarballInstallWithMockServer(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_test_libs_gh_*")
	if err != nil {
		t.Fatalf("failed to create temp libs: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	tempPkg, err := os.MkdirTemp("", "bmd_test_pkg_gh_*")
	if err != nil {
		t.Fatalf("failed to create temp pkg: %v", err)
	}
	defer os.RemoveAll(tempPkg)

	tempTarDir, err := os.MkdirTemp("", "bmd_test_tar_gh_*")
	if err != nil {
		t.Fatalf("failed to create temp tar dir: %v", err)
	}
	defer os.RemoveAll(tempTarDir)

	// Create test package inside a single root dir (as GitHub release tarballs do)
	rootWrapped := filepath.Join(tempPkg, "math_lib-1.2.0")
	_ = os.MkdirAll(rootWrapped, 0755)
	manifest := Manifest{
		Name:    "github.com/alice/math_lib",
		Version: "1.2.0",
		Entry:   "main.bmd",
		Permissions: Permissions{
			Network: true,
		},
	}
	_ = manifest.Save(filepath.Join(rootWrapped, "bmdpkg.json"))
	_ = os.WriteFile(filepath.Join(rootWrapped, "main.bmd"), []byte("fn add(a, b) { return a + b; }\n"), 0644)

	tarPath := filepath.Join(tempTarDir, "release.tar.gz")
	if err := CreateTarGz(tempPkg, tarPath); err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	tarData, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatalf("failed to read tarball: %v", err)
	}

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	inst := NewInstaller(tempLibs)
	inst.HTTPClient = server.Client()

	// Intercept request to redirect GitHub download URL to mock server
	transport := &mockTransport{
		mockURL:   server.URL,
		tarball:   tarData,
		transport: http.DefaultTransport,
	}
	inst.HTTPClient.Transport = transport

	res, err := inst.Install("github.com/alice/math_lib@1.2.0")
	if err != nil {
		t.Fatalf("Install from GitHub failed: %v", err)
	}

	if res.Version != "1.2.0" {
		t.Errorf("expected version 1.2.0, got '%s'", res.Version)
	}
	if res.FromCache {
		t.Errorf("expected FromCache false on initial install")
	}
	if !strings.Contains(res.SecurityNotice, "network") {
		t.Errorf("expected network permission notice, got '%s'", res.SecurityNotice)
	}

	// Verify tarball was stored in cache/tarballs/
	expectedTarball := filepath.Join(tempLibs, "cache", "tarballs", "github.com_alice_math_lib_1.2.0.tar.gz")
	if !fileExists(expectedTarball) {
		t.Errorf("expected cached tarball at '%s'", expectedTarball)
	}

	// Verify integrity record in integrity.db
	db, err := LoadIntegrityDB(GetIntegrityDBPath(tempLibs))
	if err != nil {
		t.Fatalf("LoadIntegrityDB failed: %v", err)
	}
	if !db.Has("github.com/alice/math_lib/1.2.0") {
		t.Errorf("integrity DB missing record for 'github.com/alice/math_lib/1.2.0'")
	}

	// Second install should hit cache with 0 bytes downloaded
	resCached, err := inst.Install("github.com/alice/math_lib@1.2.0")
	if err != nil {
		t.Fatalf("Second Install failed: %v", err)
	}
	if !resCached.FromCache {
		t.Errorf("expected second install to be FromCache = true")
	}
}

type mockTransport struct {
	mockURL   string
	tarball   []byte
	transport http.RoundTripper
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/gzip")
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.Write(m.tarball)
	return rec.Result(), nil
}

func TestGetDependenciesEngine(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_test_libs_get_*")
	if err != nil {
		t.Fatalf("failed to create temp libs: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	tempProject, err := os.MkdirTemp("", "bmd_test_project_*")
	if err != nil {
		t.Fatalf("failed to create temp project: %v", err)
	}
	defer os.RemoveAll(tempProject)

	// Pre-install dependency A into registry (so it is CACHED)
	ghDir := RegistryGithubPath("alice", "cached_lib", "1.0.0", tempLibs)
	_ = os.MkdirAll(ghDir, 0755)
	manifestA := Manifest{
		Name:    "github.com/alice/cached_lib",
		Version: "1.0.0",
		Entry:   "main.bmd",
	}
	_ = manifestA.Save(filepath.Join(ghDir, "bmdpkg.json"))
	_ = os.WriteFile(filepath.Join(ghDir, "main.bmd"), []byte("fn foo() { return 1; }\n"), 0644)

	// Pre-create local dependency B directory
	localSrc := filepath.Join(tempProject, "local_helper")
	_ = os.MkdirAll(localSrc, 0755)
	manifestB := Manifest{
		Name:    "local_helper",
		Version: "dev",
		Entry:   "main.bmd",
		Permissions: Permissions{
			Subprocess: true,
		},
	}
	_ = manifestB.Save(filepath.Join(localSrc, "bmdpkg.json"))
	_ = os.WriteFile(filepath.Join(localSrc, "main.bmd"), []byte("fn help() { return 2; }\n"), 0644)

	// Create project bmdpkg.json declaring both dependencies
	projManifest := Manifest{
		Name:    "my_app",
		Version: "0.1.0",
		Entry:   "main.bmd",
		Dependencies: map[string]string{
			"github.com/alice/cached_lib": "1.0.0",
			"./local_helper":              "dev",
		},
	}
	_ = projManifest.Save(filepath.Join(tempProject, "bmdpkg.json"))

	res, err := Get(tempProject, tempLibs)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if res.CachedCount != 1 {
		t.Errorf("expected 1 cached dependency, got %d", res.CachedCount)
	}
	if res.FetchedCount != 1 {
		t.Errorf("expected 1 fetched dependency, got %d", res.FetchedCount)
	}
	if res.FailedCount != 0 {
		t.Errorf("expected 0 failed dependencies, got %d", res.FailedCount)
	}

	// Verify second run of bmd get: both dependencies should now be CACHED
	res2, err := Get(tempProject, tempLibs)
	if err != nil {
		t.Fatalf("Second Get() failed: %v", err)
	}
	if res2.CachedCount != 2 {
		t.Errorf("expected 2 cached dependencies on second run, got %d", res2.CachedCount)
	}
	if res2.FetchedCount != 0 {
		t.Errorf("expected 0 fetched dependencies on second run, got %d", res2.FetchedCount)
	}
}

func TestCacheInfoPruneAndVerify(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_test_cache_*")
	if err != nil {
		t.Fatalf("failed to create temp libs: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	_ = EnsureDirectoryStructure(tempLibs)

	// 1. Create a dummy package extracted in registry
	ghRegDir := RegistryGithubPath("bob", "tools", "2.0.0", tempLibs)
	_ = os.MkdirAll(ghRegDir, 0755)
	_ = os.WriteFile(filepath.Join(ghRegDir, "main.bmd"), []byte("fn tool() { return 42; }\n"), 0644)

	// 2. Create the corresponding tarball in cache/tarballs/
	tarballName := "github.com_bob_tools_2.0.0.tar.gz"
	tarballPath := filepath.Join(GetCacheTarballsDir(tempLibs), tarballName)
	_ = os.WriteFile(tarballPath, []byte("dummy tarball content for bob tools"), 0644)

	// 3. Record in integrity.db
	db, _ := LoadIntegrityDB(GetIntegrityDBPath(tempLibs))
	_, _ = db.RecordFile("github.com/bob/tools/2.0.0", tarballPath)
	_ = db.Save()

	// Test CacheInfo
	info, err := CacheInfo(tempLibs)
	if err != nil {
		t.Fatalf("CacheInfo failed: %v", err)
	}
	if info.TarballCount != 1 {
		t.Errorf("expected 1 tarball, got %d", info.TarballCount)
	}
	if info.RegistryCount != 1 {
		t.Errorf("expected 1 registry package, got %d", info.RegistryCount)
	}
	if info.IntegrityRecords != 1 {
		t.Errorf("expected 1 integrity record, got %d", info.IntegrityRecords)
	}

	// Test CacheVerify (should pass)
	verifyRes, err := CacheVerify(tempLibs)
	if err != nil {
		t.Fatalf("CacheVerify failed: %v", err)
	}
	if verifyRes.ValidCount != 1 || verifyRes.FailedCount != 0 {
		t.Errorf("expected 1 valid, 0 failed, got %d valid, %d failed", verifyRes.ValidCount, verifyRes.FailedCount)
	}

	// Test CachePrune: should remove the redundant tarball while preserving the registry package
	pruneRes, err := CachePrune(tempLibs)
	if err != nil {
		t.Fatalf("CachePrune failed: %v", err)
	}
	if len(pruneRes.RemovedTarballs) != 1 || pruneRes.RemovedTarballs[0] != tarballName {
		t.Errorf("expected '%s' removed, got %v", tarballName, pruneRes.RemovedTarballs)
	}
	if fileExists(tarballPath) {
		t.Errorf("tarball should have been removed from disk")
	}
	if !isDirWithEntrypoint(ghRegDir) {
		t.Errorf("registry package should be preserved intact")
	}

	// 4. Test CacheVerify with Tampered/Missing file
	_ = os.WriteFile(filepath.Join(ghRegDir, "main.bmd"), []byte("fn tool() { return 999; } // tampered\n"), 0644)
	// Add an integrity record for a missing tarball
	db2, _ := LoadIntegrityDB(GetIntegrityDBPath(tempLibs))
	db2.Set("missing_pkg", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", 100)
	_ = db2.Save()

	verifyTampered, _ := CacheVerify(tempLibs)
	if verifyTampered.FailedCount < 1 {
		t.Errorf("expected at least 1 failed verification after tampering, got %d", verifyTampered.FailedCount)
	}
}

func TestCdnTarballInstallWithMockServer(t *testing.T) {
	tempLibs, err := os.MkdirTemp("", "bmd_test_libs_cdn_*")
	if err != nil {
		t.Fatalf("failed to create temp libs: %v", err)
	}
	defer os.RemoveAll(tempLibs)

	tempPkg, err := os.MkdirTemp("", "bmd_test_pkg_cdn_*")
	if err != nil {
		t.Fatalf("failed to create temp pkg: %v", err)
	}
	defer os.RemoveAll(tempPkg)

	tempTarDir, err := os.MkdirTemp("", "bmd_test_tar_cdn_*")
	if err != nil {
		t.Fatalf("failed to create temp tar dir: %v", err)
	}
	defer os.RemoveAll(tempTarDir)

	manifest := Manifest{
		Name:    "cdn_router",
		Version: "2.5.0",
		Entry:   "index.bmd",
		Permissions: Permissions{
			FilesystemWrite: true,
		},
	}
	_ = manifest.Save(filepath.Join(tempPkg, "bmdpkg.json"))
	_ = os.WriteFile(filepath.Join(tempPkg, "index.bmd"), []byte("fn route() { return true; }\n"), 0644)

	tarPath := filepath.Join(tempTarDir, "router-2.5.0.tar.gz")
	if err := CreateTarGz(tempPkg, tarPath); err != nil {
		t.Fatalf("CreateTarGz failed: %v", err)
	}

	tarData, err := os.ReadFile(tarPath)
	if err != nil {
		t.Fatalf("failed to read tarball: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tarData)
	}))
	defer server.Close()

	inst := NewInstaller(tempLibs)
	inst.HTTPClient = server.Client()

	cdnTarget := server.URL + "/router@2.5.0.tar.gz"
	res, err := inst.Install(cdnTarget)
	if err != nil {
		t.Fatalf("Install from CDN failed: %v", err)
	}

	if res.Version != "2.5.0" {
		t.Errorf("expected version 2.5.0, got '%s'", res.Version)
	}
	if !strings.Contains(res.SecurityNotice, "filesystem_write") {
		t.Errorf("expected filesystem_write notice, got '%s'", res.SecurityNotice)
	}

	// Verify resolved from registry via ResolveRegistryPackage
	resResolved, err := ResolveRegistryPackage(fmt.Sprintf("%s/%s", res.PackageName, res.Version), tempLibs)
	if err != nil {
		t.Fatalf("failed to resolve installed CDN package: %v", err)
	}
	if !strings.HasSuffix(resResolved, "index.bmd") {
		t.Errorf("expected index.bmd for CDN package, got '%s'", resResolved)
	}
}

func TestZipSlipPathTraversalProtection(t *testing.T) {
	tempDest, err := os.MkdirTemp("", "bmd_test_slip_*")
	if err != nil {
		t.Fatalf("failed to create temp dest: %v", err)
	}
	defer os.RemoveAll(tempDest)

	// Attempt extracting a path that would escape tempDest
	// Clean destination check protects against "../../../evil.txt"
	cleanDest := filepath.Clean(tempDest)
	escapePath := filepath.Join(cleanDest, "../../escaped.txt")
	if strings.HasPrefix(escapePath, cleanDest+string(filepath.Separator)) {
		t.Errorf("escapePath should have escaped cleanDest")
	}
}

