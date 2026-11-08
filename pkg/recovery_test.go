package pkg

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createValidTarGz(t *testing.T) ([]byte, string) {
	tempSrc := t.TempDir()
	m := Manifest{
		Name:    "valid_recovery_pkg",
		Version: "1.0.0",
		Entry:   "main.bmd",
	}
	mBytes, _ := m.ToJSON()
	_ = os.WriteFile(filepath.Join(tempSrc, "bmdpkg.json"), mBytes, 0644)
	_ = os.WriteFile(filepath.Join(tempSrc, "main.bmd"), []byte("let msg = 'recovered';\n"), 0644)

	archivePath := filepath.Join(t.TempDir(), "valid.tar.gz")
	if err := CreateTarGz(tempSrc, archivePath); err != nil {
		t.Fatalf("failed to create valid tar.gz: %v", err)
	}

	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("failed to read created archive: %v", err)
	}
	return data, ComputeSHA256(data)
}

func TestCorruptedArchiveRecovery_ZeroByteTarball(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	validData, _ := createValidTarGz(t)
	downloadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadCount++
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(validData)
	}))
	defer server.Close()

	inst := NewInstaller(baseDir)
	inst.HTTPClient = server.Client()

	// Pre-create a corrupted 0-byte tarball in cache/tarballs/
	url := server.URL + "/mypkg@1.0.0.tar.gz"
	uHost := server.Listener.Addr().String()
	domainHash := ComputeSHA256([]byte(uHost))[:12]
	tarballName := fmt.Sprintf("cdn_%s_1.0.0.tar.gz", domainHash)
	cachedTarballPath := filepath.Join(GetCacheTarballsDir(baseDir), tarballName)

	_ = os.WriteFile(cachedTarballPath, []byte(""), 0644)

	// Install should detect 0-byte corrupt tarball, prune it, and re-download
	res, err := inst.Install(url, "1.0.0")
	if err != nil {
		t.Fatalf("install failed to recover from 0-byte tarball: %v", err)
	}

	if downloadCount != 1 {
		t.Fatalf("expected 1 download during recovery, got %d", downloadCount)
	}

	// Verify package is installed in registry
	entryFile := filepath.Join(res.RegistryDir, "main.bmd")
	if !fileExists(entryFile) {
		t.Fatalf("entrypoint 'main.bmd' not found in recovered registry: %s", res.RegistryDir)
	}
}

func TestCorruptedArchiveRecovery_CorruptedGzipHeader(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	validData, _ := createValidTarGz(t)
	downloadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadCount++
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(validData)
	}))
	defer server.Close()

	inst := NewInstaller(baseDir)
	inst.HTTPClient = server.Client()

	url := server.URL + "/mypkg@1.0.0.tar.gz"
	uHost := server.Listener.Addr().String()
	domainHash := ComputeSHA256([]byte(uHost))[:12]
	tarballName := fmt.Sprintf("cdn_%s_1.0.0.tar.gz", domainHash)
	cachedTarballPath := filepath.Join(GetCacheTarballsDir(baseDir), tarballName)

	// Write garbage bytes
	_ = os.WriteFile(cachedTarballPath, []byte("THIS_IS_NOT_A_VALID_GZIP_FILE_RANDOM_GARBAGE"), 0644)

	res, err := inst.Install(url, "1.0.0")
	if err != nil {
		t.Fatalf("install failed to recover from corrupt gzip archive: %v", err)
	}

	if downloadCount != 1 {
		t.Fatalf("expected 1 download during recovery, got %d", downloadCount)
	}

	entryFile := filepath.Join(res.RegistryDir, "main.bmd")
	if !fileExists(entryFile) {
		t.Fatalf("entrypoint 'main.bmd' not found in recovered registry: %s", res.RegistryDir)
	}
}

func TestCorruptedArchiveRecovery_Sha256Mismatch(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	validData, validSHA := createValidTarGz(t)
	downloadCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloadCount++
		w.Header().Set("Content-Type", "application/gzip")
		_, _ = w.Write(validData)
	}))
	defer server.Close()

	inst := NewInstaller(baseDir)
	inst.HTTPClient = server.Client()

	url := server.URL + "/mypkg@1.0.0.tar.gz"
	uHost := server.Listener.Addr().String()
	domainHash := ComputeSHA256([]byte(uHost))[:12]
	tarballName := fmt.Sprintf("cdn_%s_1.0.0.tar.gz", domainHash)
	cachedTarballPath := filepath.Join(GetCacheTarballsDir(baseDir), tarballName)
	integrityKey := fmt.Sprintf("cdn/%s/1.0.0", domainHash)

	// Record legitimate SHA in integrity.db
	_ = UpdateIntegrityDB(baseDir, func(db *IntegrityDB) error {
		db.Set(integrityKey, validSHA, int64(len(validData)))
		return nil
	})

	// Pre-create another valid gzip file whose hash differs from recorded SHA
	otherTempSrc := t.TempDir()
	_ = os.WriteFile(filepath.Join(otherTempSrc, "other.txt"), []byte("tampered content"), 0644)
	_ = CreateTarGz(otherTempSrc, cachedTarballPath)

	// Install must detect SHA mismatch against integrity.db, prune tampered file, and re-download
	res, err := inst.Install(url, "1.0.0")
	if err != nil {
		t.Fatalf("install failed to recover from SHA mismatch: %v", err)
	}

	if downloadCount != 1 {
		t.Fatalf("expected 1 download during recovery, got %d", downloadCount)
	}

	entryFile := filepath.Join(res.RegistryDir, "main.bmd")
	if !fileExists(entryFile) {
		t.Fatalf("entrypoint 'main.bmd' not found in recovered registry: %s", res.RegistryDir)
	}
}
