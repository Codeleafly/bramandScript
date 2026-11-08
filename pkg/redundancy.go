package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SharedPackageInfo describes a single package shared across multiple projects.
type SharedPackageInfo struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	RegistryPath    string   `json:"registry_path"`
	ReferencingDirs []string `json:"referencing_dirs"`
}

// RedundancyReport details zero-redundancy verification across projects.
type RedundancyReport struct {
	SharedPackages         []SharedPackageInfo `json:"shared_packages"`
	ProjectCount           int                 `json:"project_count"`
	TotalSharedPackages    int                 `json:"total_shared_packages"`
	RedundantBytes         int64               `json:"redundant_bytes"` // Always 0 for centralized architecture
	ZeroRedundancyVerified bool                `json:"zero_redundancy_verified"`
}

// VerifyZeroRedundancy verifies that multiple projects referencing the same packages
// resolve to the exact same directory in $HOME/.bmd_libs/registry/ with 0 disk redundancy.
func VerifyZeroRedundancy(projectDirs []string, baseDir ...string) (*RedundancyReport, error) {
	base := resolveBaseDir(baseDir)
	registryDir := GetRegistryDir(base)

	report := &RedundancyReport{
		ProjectCount:           len(projectDirs),
		RedundantBytes:         0,
		ZeroRedundancyVerified: true,
	}

	// Map: pkgName@ver -> SharedPackageInfo
	packageMap := make(map[string]*SharedPackageInfo)

	for _, pDir := range projectDirs {
		absProjectDir, err := filepath.Abs(pDir)
		if err != nil {
			absProjectDir = pDir
		}

		// 1. Check no redundant local vendor / node_modules directory exists in project
		redundantDirs := []string{
			filepath.Join(absProjectDir, "node_modules"),
			filepath.Join(absProjectDir, "vendor"),
			filepath.Join(absProjectDir, ".bmd_modules"),
		}
		for _, rd := range redundantDirs {
			if fi, err := os.Stat(rd); err == nil && fi.IsDir() {
				// Found redundant local copies
				var dirSize int64
				_ = filepath.Walk(rd, func(p string, info os.FileInfo, err error) error {
					if err == nil && !info.IsDir() {
						dirSize += info.Size()
					}
					return nil
				})
				report.RedundantBytes += dirSize
				report.ZeroRedundancyVerified = false
			}
		}

		// 2. Read project manifest
		mPath := filepath.Join(absProjectDir, "bmdpkg.json")
		if !fileExists(mPath) {
			continue
		}

		manifest, err := ReadManifest(mPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read manifest in '%s': %w", absProjectDir, err)
		}

		for depName, depVer := range manifest.Dependencies {
			// Resolve package directory in centralized registry
			targetQuery := depName
			if !strings.Contains(depName, "@") && depVer != "" && depVer != "dev" {
				targetQuery = depName + "@" + depVer
			}

			entrypoint, err := ResolveRegistryPackage(targetQuery, base)
			if err != nil {
				// Try direct query
				entrypoint, err = ResolveRegistryPackage(depName, base)
			}

			if err == nil && entrypoint != "" {
				pkgDir := filepath.Dir(entrypoint)
				// Verify resolved directory is inside the centralized registry
				cleanPkgDir := filepath.Clean(pkgDir)
				cleanRegDir := filepath.Clean(registryDir)

				if !strings.HasPrefix(cleanPkgDir, cleanRegDir) {
					// Package was not resolved from centralized registry!
					report.ZeroRedundancyVerified = false
				}

				key := depName + "@" + depVer
				if info, exists := packageMap[key]; exists {
					// Check that paths match exactly
					if info.RegistryPath != cleanPkgDir {
						report.ZeroRedundancyVerified = false
					}
					info.ReferencingDirs = append(info.ReferencingDirs, absProjectDir)
				} else {
					packageMap[key] = &SharedPackageInfo{
						Name:            depName,
						Version:         depVer,
						RegistryPath:    cleanPkgDir,
						ReferencingDirs: []string{absProjectDir},
					}
				}
			}
		}
	}

	for _, info := range packageMap {
		report.SharedPackages = append(report.SharedPackages, *info)
	}

	sort.Slice(report.SharedPackages, func(i, j int) bool {
		return report.SharedPackages[i].Name < report.SharedPackages[j].Name
	})

	report.TotalSharedPackages = len(report.SharedPackages)

	return report, nil
}
