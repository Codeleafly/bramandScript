package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOfflineInstaller_CachedTarballInstallsOffline(t *testing.T) {
	tempBase := t.TempDir()
	inst := NewInstaller(tempBase)
	inst.Offline = true

	// Pre-seed a tarball in cache/tarballs/
	tarballName := "github.com_offline_calc_1.0.0.tar.gz"
	tarballPath := filepath.Join(GetCacheTarballsDir(tempBase), tarballName)
	_ = os.MkdirAll(filepath.Dir(tarballPath), 0755)

	pkgSrcDir := filepath.Join(tempBase, "src_calc")
	_ = os.MkdirAll(pkgSrcDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgSrcDir, "bmdpkg.json"), []byte(`{"name":"calc","version":"1.0.0","entry":"main.bmd","permissions":{}}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgSrcDir, "main.bmd"), []byte(`let add = fn(a, b) { a + b };`), 0644)

	err := CreateTarGz(pkgSrcDir, tarballPath)
	if err != nil {
		t.Fatalf("failed to create pre-seeded tarball: %s", err)
	}

	// Install in offline mode
	res, err := inst.Install("github.com/offline/calc@1.0.0")
	if err != nil {
		t.Fatalf("expected offline installation from cached tarball to succeed, got: %s", err)
	}

	if !fileExists(filepath.Join(res.RegistryDir, "main.bmd")) {
		t.Fatalf("package was not extracted into registry: %s", res.RegistryDir)
	}

	// Verify integrity DB was updated
	db, err := LoadIntegrityDB(GetIntegrityDBPath(tempBase))
	if err != nil {
		t.Fatalf("failed to load integrity DB: %s", err)
	}
	if len(db.Records) == 0 {
		t.Fatalf("expected integrity record in integrity.db")
	}
}

func TestOfflineInstaller_MissingTarballFailsFast(t *testing.T) {
	tempBase := t.TempDir()
	inst := NewInstaller(tempBase)
	inst.Offline = true

	// Attempting to install uncached package in offline mode must fail immediately
	_, err := inst.Install("github.com/uncached/tool@2.0.0")
	if err == nil {
		t.Fatalf("expected offline install of uncached package to fail, got nil")
	}

	if !strings.Contains(err.Error(), "offline mode") || !strings.Contains(err.Error(), "tarball not cached") {
		t.Fatalf("expected clear offline diagnostic error, got: %s", err)
	}
}

func TestOfflineGet_CachedDependenciesSucceed(t *testing.T) {
	tempBase := t.TempDir()
	_ = os.Setenv("BMD_OFFLINE", "1")
	defer os.Unsetenv("BMD_OFFLINE")

	// Pre-seed registry with local/math@dev
	regDir := RegistryLocalPath("math", tempBase)
	_ = os.MkdirAll(regDir, 0755)
	_ = os.WriteFile(filepath.Join(regDir, "bmdpkg.json"), []byte(`{"name":"math","version":"dev","entry":"main.bmd","permissions":{}}`), 0644)
	_ = os.WriteFile(filepath.Join(regDir, "main.bmd"), []byte(`let pi = 3.14;`), 0644)

	// Create project with manifest depending on math
	projectDir := filepath.Join(tempBase, "my_project")
	_ = os.MkdirAll(projectDir, 0755)
	manifest := `{
		"name": "my_project",
		"version": "1.0.0",
		"dependencies": {
			"local/math": "dev"
		}
	}`
	_ = os.WriteFile(filepath.Join(projectDir, "bmdpkg.json"), []byte(manifest), 0644)

	res, err := Get(projectDir, tempBase)
	if err != nil {
		t.Fatalf("expected offline Get to succeed for cached dependency, got: %s", err)
	}

	if res.CachedCount != 1 || res.FailedCount != 0 {
		t.Fatalf("expected 1 cached and 0 failed, got: cached=%d, failed=%d", res.CachedCount, res.FailedCount)
	}
}
