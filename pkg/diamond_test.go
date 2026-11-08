package pkg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSemVerCompatibility(t *testing.T) {
	tests := []struct {
		v1         string
		v2         string
		compatible bool
	}{
		{"1.0.0", "1.1.0", true},
		{"1.2.0", "1.2.5", true},
		{"^1.0.0", "1.3.0", true},
		{"~1.2.0", "1.2.4", true},
		{"1.0.0", "2.0.0", false},
		{"0.1.0", "0.2.0", false}, // 0.x breaking minor
		{"0.2.1", "0.2.4", true},
		{"dev", "dev", true},
		{"dev", "1.0.0", false},
	}

	for _, tc := range tests {
		comp := AreVersionsCompatible(tc.v1, tc.v2)
		if comp != tc.compatible {
			t.Errorf("AreVersionsCompatible(%s, %s): expected %v, got %v", tc.v1, tc.v2, tc.compatible, comp)
		}
	}
}

func TestResolveDiamondConflict_Compatible(t *testing.T) {
	pkgName := "pkg-math"
	requested := []string{"1.0.0", "1.1.0"}

	res, err := ResolveDiamondConflict(pkgName, requested)
	if err != nil {
		t.Fatalf("unexpected error resolving diamond conflict: %v", err)
	}

	if !res.IsCompatible {
		t.Fatalf("expected compatible resolution for %v", requested)
	}

	if res.ResolvedVersion != "1.1.0" {
		t.Fatalf("expected resolved version '1.1.0', got '%s'", res.ResolvedVersion)
	}
}

func TestResolveDiamondConflict_IncompatibleMajorVersions(t *testing.T) {
	pkgName := "pkg-db"
	requested := []string{"1.0.0", "2.1.0"}

	res, err := ResolveDiamondConflict(pkgName, requested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.IsCompatible {
		t.Fatalf("expected incompatible resolution for %v", requested)
	}

	if len(res.SideBySide) != 2 {
		t.Fatalf("expected 2 side-by-side versions, got %d", len(res.SideBySide))
	}

	expectedFirst := "2.1.0"
	expectedSecond := "1.0.0"
	if res.SideBySide[0] != expectedFirst || res.SideBySide[1] != expectedSecond {
		t.Fatalf("expected side-by-side [%s, %s], got %v", expectedFirst, expectedSecond, res.SideBySide)
	}
}

func TestGet_DiamondDependencyResolution(t *testing.T) {
	baseDir := t.TempDir()
	_ = EnsureDirectoryStructure(baseDir)

	// Pre-install pkg-x versions 1.0.0 and 1.1.0 in registry
	regX100 := filepath.Join(GetRegistryDir(baseDir), "local", "pkg_x", "1.0.0")
	_ = os.MkdirAll(regX100, 0755)
	_ = os.WriteFile(filepath.Join(regX100, "bmdpkg.json"), []byte(`{"name":"pkg_x","version":"1.0.0","entry":"main.bmd"}`), 0644)
	_ = os.WriteFile(filepath.Join(regX100, "main.bmd"), []byte(`let ver = "1.0.0";`), 0644)

	regX110 := filepath.Join(GetRegistryDir(baseDir), "local", "pkg_x", "1.1.0")
	_ = os.MkdirAll(regX110, 0755)
	_ = os.WriteFile(filepath.Join(regX110, "bmdpkg.json"), []byte(`{"name":"pkg_x","version":"1.1.0","entry":"main.bmd"}`), 0644)
	_ = os.WriteFile(filepath.Join(regX110, "main.bmd"), []byte(`let ver = "1.1.0";`), 0644)

	// Create Package A requiring pkg_x@1.0.0
	pkgADir := filepath.Join(t.TempDir(), "pkg_a")
	_ = os.MkdirAll(pkgADir, 0755)
	_ = os.WriteFile(filepath.Join(pkgADir, "bmdpkg.json"), []byte(`{
		"name": "pkg_a",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"local/pkg_x": "1.0.0"
		}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgADir, "main.bmd"), []byte(`import "local/pkg_x" as x; let a = "a";`), 0644)

	// Create Package B requiring pkg_x@1.1.0
	pkgBDir := filepath.Join(t.TempDir(), "pkg_b")
	_ = os.MkdirAll(pkgBDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgBDir, "bmdpkg.json"), []byte(`{
		"name": "pkg_b",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"local/pkg_x": "1.1.0"
		}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(pkgBDir, "main.bmd"), []byte(`import "local/pkg_x" as x; let b = "b";`), 0644)

	// Install Package A and Package B into registry
	inst := NewInstaller(baseDir)
	_, err := inst.Install(pkgADir)
	if err != nil {
		t.Fatalf("failed to install pkg_a: %v", err)
	}
	_, err = inst.Install(pkgBDir)
	if err != nil {
		t.Fatalf("failed to install pkg_b: %v", err)
	}

	// Create Project MyApp depending on both pkg_a and pkg_b
	projectDir := filepath.Join(t.TempDir(), "my_app")
	_ = os.MkdirAll(projectDir, 0755)
	_ = os.WriteFile(filepath.Join(projectDir, "bmdpkg.json"), []byte(`{
		"name": "my_app",
		"version": "1.0.0",
		"entry": "main.bmd",
		"dependencies": {
			"local/pkg_a": "1.0.0",
			"local/pkg_b": "1.0.0"
		}
	}`), 0644)
	_ = os.WriteFile(filepath.Join(projectDir, "main.bmd"), []byte(`import "local/pkg_a"; import "local/pkg_b";`), 0644)

	// Run Get
	res, err := Get(projectDir, baseDir)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if res.FailedCount > 0 {
		t.Fatalf("unexpected failures in Get: %d failed", res.FailedCount)
	}

	// Verify diamond dependency on pkg_x was discovered and resolved to 1.1.0
	foundDiamond := false
	for _, dr := range res.DiamondResolutions {
		if dr.PackageName == "local/pkg_x" {
			foundDiamond = true
			if !dr.IsCompatible {
				t.Fatalf("expected local/pkg_x diamond to be compatible")
			}
			if dr.ResolvedVersion != "1.1.0" {
				t.Fatalf("expected resolved version '1.1.0', got '%s'", dr.ResolvedVersion)
			}
		}
	}

	if !foundDiamond {
		t.Fatalf("expected diamond resolution for local/pkg_x, but none was recorded: %v", res.DiamondResolutions)
	}
}
