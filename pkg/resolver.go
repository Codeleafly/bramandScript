package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// BrokenManifestError indicates that a package defines a bmdpkg.json manifest,
// but the manifest itself or its specified entrypoint is missing/invalid.
type BrokenManifestError struct {
	PackageDir string
	Entry      string
	Reason     string
}

func (e *BrokenManifestError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("BrokenManifestError: %s (in package '%s')", e.Reason, e.PackageDir)
	}
	return fmt.Sprintf("BrokenManifestError: manifest 'bmdpkg.json' specifies entry '%s' which does not exist in package '%s'", e.Entry, e.PackageDir)
}

// ResolvePackageEntrypoint executes the Three-Tier Entrypoint Fallback Engine:
// Step 1: Check bmdpkg.json. If found, read "entry" field. If entry file exists -> resolve. If missing -> BrokenManifestError.
// Step 2: Check main.bmd. If exists -> use main.bmd as entrypoint.
// Step 3: Check index.bmd. If exists -> use index.bmd as entrypoint.
// If none found -> emit clean diagnostic error: "No entrypoint found: package must define bmdpkg.json, main.bmd, or index.bmd".
func ResolvePackageEntrypoint(pkgDir string) (string, error) {
	fi, err := os.Stat(pkgDir)
	if err != nil {
		return "", err
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("package path is not a directory: %s", pkgDir)
	}

	manifestPath := filepath.Join(pkgDir, "bmdpkg.json")
	if mfi, err := os.Stat(manifestPath); err == nil && !mfi.IsDir() {
		// Step 1: Check bmdpkg.json
		manifest, err := ReadManifest(manifestPath)
		if err != nil {
			return "", &BrokenManifestError{PackageDir: pkgDir, Reason: fmt.Sprintf("invalid manifest: %s", err)}
		}

		if manifest.Entry != "" {
			entryFile := filepath.Join(pkgDir, manifest.Entry)
			if efi, err := os.Stat(entryFile); err == nil && !efi.IsDir() {
				if abs, err := filepath.Abs(entryFile); err == nil {
					return abs, nil
				}
				return entryFile, nil
			}
			return "", &BrokenManifestError{PackageDir: pkgDir, Entry: manifest.Entry}
		}

		// If bmdpkg.json exists but entry is not set, fallback to main.bmd then index.bmd
		mainCandidate := filepath.Join(pkgDir, "main.bmd")
		if efi, err := os.Stat(mainCandidate); err == nil && !efi.IsDir() {
			if abs, err := filepath.Abs(mainCandidate); err == nil {
				return abs, nil
			}
			return mainCandidate, nil
		}

		indexCandidate := filepath.Join(pkgDir, "index.bmd")
		if efi, err := os.Stat(indexCandidate); err == nil && !efi.IsDir() {
			if abs, err := filepath.Abs(indexCandidate); err == nil {
				return abs, nil
			}
			return indexCandidate, nil
		}

		return "", &BrokenManifestError{PackageDir: pkgDir, Reason: "manifest 'bmdpkg.json' defines no 'entry' and neither main.bmd nor index.bmd exists"}
	}

	// Step 2: Check main.bmd
	mainCandidate := filepath.Join(pkgDir, "main.bmd")
	if efi, err := os.Stat(mainCandidate); err == nil && !efi.IsDir() {
		if abs, err := filepath.Abs(mainCandidate); err == nil {
			return abs, nil
		}
		return mainCandidate, nil
	}

	// Step 3: Check index.bmd
	indexCandidate := filepath.Join(pkgDir, "index.bmd")
	if efi, err := os.Stat(indexCandidate); err == nil && !efi.IsDir() {
		if abs, err := filepath.Abs(indexCandidate); err == nil {
			return abs, nil
		}
		return indexCandidate, nil
	}

	// Clean diagnostic error required by specification
	return "", fmt.Errorf("No entrypoint found: package must define bmdpkg.json, main.bmd, or index.bmd: %s", pkgDir)
}

type semverParts struct {
	major      int
	minor      int
	patch      int
	prerelease string
	raw        string
}

func parseSemver(v string) semverParts {
	raw := v
	cleaned := strings.TrimPrefix(v, "v")
	var prerelease string
	if idx := strings.Index(cleaned, "-"); idx != -1 {
		prerelease = cleaned[idx+1:]
		cleaned = cleaned[:idx]
	}
	if idx := strings.Index(cleaned, "+"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	parts := strings.Split(cleaned, ".")
	major, minor, patch := 0, 0, 0
	if len(parts) > 0 {
		major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		patch, _ = strconv.Atoi(parts[2])
	}

	return semverParts{
		major:      major,
		minor:      minor,
		patch:      patch,
		prerelease: prerelease,
		raw:        raw,
	}
}

// CompareVersions compares two semver strings:
// returns 1 if v1 > v2, -1 if v1 < v2, and 0 if equal.
func CompareVersions(v1, v2 string) int {
	p1 := parseSemver(v1)
	p2 := parseSemver(v2)

	if p1.major != p2.major {
		if p1.major > p2.major {
			return 1
		}
		return -1
	}
	if p1.minor != p2.minor {
		if p1.minor > p2.minor {
			return 1
		}
		return -1
	}
	if p1.patch != p2.patch {
		if p1.patch > p2.patch {
			return 1
		}
		return -1
	}

	// Normal release is higher precedence than pre-release
	if p1.prerelease == "" && p2.prerelease != "" {
		return 1
	}
	if p1.prerelease != "" && p2.prerelease == "" {
		return -1
	}
	if p1.prerelease != p2.prerelease {
		if p1.prerelease > p2.prerelease {
			return 1
		}
		return -1
	}

	return 0
}

// SortVersions sorts a slice of version strings descending (highest version first).
func SortVersions(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		return CompareVersions(sorted[i], sorted[j]) > 0
	})
	return sorted
}

// FindLatestVersion scans a package directory containing version subdirectories
// and returns the highest version string according to semver.
func FindLatestVersion(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			if semverRegex.MatchString(name) || name == "dev" {
				versions = append(versions, name)
			}
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no version directories found in '%s'", dir)
	}

	sorted := SortVersions(versions)
	return sorted[0], nil
}

// ResolveRegistryPackage resolves a package import path against the centralized
// registry ($HOME/.bmd_libs/registry/<import_path>/<resolved_version>/)
// using the Three-Tier Fallback Engine.
func ResolveRegistryPackage(importPath string, baseDir ...string) (string, error) {
	registryDir := GetRegistryDir(baseDir...)

	// Normalize import path
	normPath := filepath.Clean(importPath)

	candidatePkgDirs := []string{}

	// Case 1: Import path contains version via '@', e.g. "github.com/user/pkg@1.2.0" or "local/tool@dev"
	if strings.Contains(normPath, "@") {
		parts := strings.SplitN(normPath, "@", 2)
		pkgName := parts[0]
		ver := parts[1]

		candidatePkgDirs = append(candidatePkgDirs,
			filepath.Join(registryDir, pkgName, ver),
			filepath.Join(registryDir, normPath),
		)
		if !strings.Contains(pkgName, "/") {
			candidatePkgDirs = append(candidatePkgDirs,
				filepath.Join(registryDir, "local", pkgName+"@"+ver),
				filepath.Join(registryDir, "local", pkgName, ver),
			)
		}
	} else {
		// Case 2: Exact directory path in registry (e.g. github.com/user/pkg/1.0.0, cdn/hash/1.0.0, local/name@dev)
		exactDir := filepath.Join(registryDir, normPath)
		candidatePkgDirs = append(candidatePkgDirs, exactDir)

		// Case 3: Local namespace candidate (e.g. "local/name@dev" or "local/name")
		if !strings.Contains(normPath, "/") {
			candidatePkgDirs = append(candidatePkgDirs,
				filepath.Join(registryDir, "local", normPath+"@dev"),
				filepath.Join(registryDir, "local", normPath),
			)
		}
	}

	// Try each candidate directory
	for _, candidate := range candidatePkgDirs {
		fi, err := os.Stat(candidate)
		if err != nil || !fi.IsDir() {
			continue
		}

		// Check if candidate itself is a package root (has bmdpkg.json, main.bmd, or index.bmd)
		hasDirectEntrypoint := fileExists(filepath.Join(candidate, "bmdpkg.json")) ||
			fileExists(filepath.Join(candidate, "main.bmd")) ||
			fileExists(filepath.Join(candidate, "index.bmd"))

		if hasDirectEntrypoint {
			return ResolvePackageEntrypoint(candidate)
		}

		// Otherwise, candidate may contain version subdirectories (e.g., 1.0.0/, 1.1.0/)
		latestVer, err := FindLatestVersion(candidate)
		if err == nil {
			versionedDir := filepath.Join(candidate, latestVer)
			return ResolvePackageEntrypoint(versionedDir)
		}
	}

	// Case 4: Package might be at github.com/<author>/<pkg> without version specified
	// Scan candidate subdirectories under registry
	candidateBase := filepath.Join(registryDir, normPath)
	if fi, err := os.Stat(candidateBase); err == nil && fi.IsDir() {
		latestVer, err := FindLatestVersion(candidateBase)
		if err == nil {
			versionedDir := filepath.Join(candidateBase, latestVer)
			return ResolvePackageEntrypoint(versionedDir)
		}
		// If no version subdirectories, attempt resolution on candidateBase directly
		return ResolvePackageEntrypoint(candidateBase)
	}

	return "", fmt.Errorf("package '%s' not found in registry '%s'", importPath, registryDir)
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
