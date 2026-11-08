package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyZeroRedundancy_CleanSharing(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	// 1. Install a shared package into the centralized registry
	sharedPkgDir := filepath.Join(t.TempDir(), "shared_calc")
	_ = os.MkdirAll(sharedPkgDir, 0755)
	_ = os.WriteFile(filepath.Join(sharedPkgDir, "bmdpkg.json"), []byte(`{
		"name": "shared_calc",
		"version": "1.0.0",
		"entry": "main.bmd"
	}`), 0644)
	_ = os.WriteFile(filepath.Join(sharedPkgDir, "main.bmd"), []byte(`fn add(a, b) { return a + b; }`), 0644)

	inst := NewInstaller(baseDir)
	res, err := inst.Install(sharedPkgDir, "1.0.0")
	if err != nil {
		t.Fatalf("failed to install shared package: %v", err)
	}

	// 2. Project Alpha referencing shared_calc
	projectAlpha := filepath.Join(t.TempDir(), "project_alpha")
	_ = os.MkdirAll(projectAlpha, 0755)
	_ = os.WriteFile(filepath.Join(projectAlpha, "bmdpkg.json"), []byte(`{
		"name": "project_alpha",
		"version": "1.0.0",
		"dependencies": {
			"local/shared_calc": "1.0.0"
		}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(projectAlpha, "main.bmd"), []byte(`import "local/shared_calc" as calc;`), 0644)

	// 3. Project Beta referencing shared_calc
	projectBeta := filepath.Join(t.TempDir(), "project_beta")
	_ = os.MkdirAll(projectBeta, 0755)
	_ = os.WriteFile(filepath.Join(projectBeta, "bmdpkg.json"), []byte(`{
		"name": "project_beta",
		"version": "1.0.0",
		"dependencies": {
			"local/shared_calc": "1.0.0"
		}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(projectBeta, "main.bmd"), []byte(`import "local/shared_calc" as calc;`), 0644)

	// 4. Verify Zero Redundancy
	report, err := VerifyZeroRedundancy([]string{projectAlpha, projectBeta}, baseDir)
	if err != nil {
		t.Fatalf("VerifyZeroRedundancy failed: %v", err)
	}

	if !report.ZeroRedundancyVerified {
		t.Fatalf("expected ZeroRedundancyVerified to be true, got false")
	}

	if report.RedundantBytes != 0 {
		t.Fatalf("expected 0 redundant bytes, got %d", report.RedundantBytes)
	}

	if report.TotalSharedPackages != 1 {
		t.Fatalf("expected 1 shared package, got %d", report.TotalSharedPackages)
	}

	sharedInfo := report.SharedPackages[0]
	if sharedInfo.RegistryPath != res.RegistryDir {
		t.Fatalf("expected registry path '%s', got '%s'", res.RegistryDir, sharedInfo.RegistryPath)
	}

	if len(sharedInfo.ReferencingDirs) != 2 {
		t.Fatalf("expected 2 referencing directories, got %d", len(sharedInfo.ReferencingDirs))
	}
}

func TestVerifyZeroRedundancy_DetectsRedundantVendor(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	projectDir := filepath.Join(t.TempDir(), "project_with_vendor")
	_ = os.MkdirAll(projectDir, 0755)
	_ = os.WriteFile(filepath.Join(projectDir, "bmdpkg.json"), []byte(`{
		"name": "project_with_vendor",
		"version": "1.0.0"
	}`), 0644)

	// Intentionally create a redundant local vendor directory
	vendorDir := filepath.Join(projectDir, "vendor")
	_ = os.MkdirAll(vendorDir, 0755)
	_ = os.WriteFile(filepath.Join(vendorDir, "dup.bmd"), []byte("redundant code bytes here"), 0644)

	report, err := VerifyZeroRedundancy([]string{projectDir}, baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.ZeroRedundancyVerified {
		t.Fatalf("expected ZeroRedundancyVerified to be false due to local vendor dir")
	}

	if report.RedundantBytes <= 0 {
		t.Fatalf("expected redundant bytes > 0, got %d", report.RedundantBytes)
	}
}
