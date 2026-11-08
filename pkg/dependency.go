package pkg

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// DependencyItem describes the resolution and installation state of a dependency.
type DependencyItem struct {
	Name           string
	Version        string
	Status         string // "CACHED", "FETCHED", "ERROR"
	RegistryDir    string
	SecurityNotice string
	Error          error
}

// GetResult details the overall outcome of `bmd get`.
type GetResult struct {
	ManifestPath   string
	Dependencies   []DependencyItem
	CachedCount    int
	FetchedCount   int
	FailedCount    int
}

// Get inspects the bmdpkg.json file in projectDir, verifies if each dependency
// is already present in the centralized registry, and only downloads missing dependencies.
func Get(projectDir string, baseDir ...string) (*GetResult, error) {
	base := resolveBaseDir(baseDir)
	if err := EnsureDirectoryStructure(base); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(projectDir, "bmdpkg.json")
	if !fileExists(manifestPath) {
		// Fallback check for package.json
		legacyPkg := filepath.Join(projectDir, "package.json")
		if !fileExists(legacyPkg) {
			return nil, fmt.Errorf("no 'bmdpkg.json' manifest found in '%s'", projectDir)
		}
		manifestPath = legacyPkg
	}

	manifest, err := ReadManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read '%s': %w", manifestPath, err)
	}

	res := &GetResult{
		ManifestPath: manifestPath,
	}

	if len(manifest.Dependencies) == 0 {
		return res, nil
	}

	// Sort dependency keys for deterministic processing
	depKeys := make([]string, 0, len(manifest.Dependencies))
	for k := range manifest.Dependencies {
		depKeys = append(depKeys, k)
	}
	sort.Strings(depKeys)

	inst := NewInstaller(base)

	for _, depName := range depKeys {
		depVer := manifest.Dependencies[depName]
		item := DependencyItem{
			Name:    depName,
			Version: depVer,
		}

		// 1. Check if dependency already exists in registry
		isCached, regDir, secNotice := checkRegistryDependency(depName, depVer, projectDir, base)
		if isCached {
			item.Status = "CACHED"
			item.RegistryDir = regDir
			item.SecurityNotice = secNotice
			res.CachedCount++
			res.Dependencies = append(res.Dependencies, item)
			continue
		}

		// 2. Not cached: fetch and install missing dependency
		target := depName
		if strings.HasPrefix(depName, "./") || strings.HasPrefix(depName, "../") {
			target = filepath.Join(projectDir, depName)
		} else if strings.HasPrefix(depName, "github.com/") && !strings.Contains(depName, "@") {
			target = depName + "@" + depVer
		} else if strings.HasPrefix(depName, "local/") && !strings.Contains(depName, "@") {
			target = depName + "@" + depVer
		}

		installRes, err := inst.Install(target, depVer)
		if err != nil {
			item.Status = "ERROR"
			item.Error = err
			res.FailedCount++
		} else {
			item.Status = "FETCHED"
			item.RegistryDir = installRes.RegistryDir
			item.SecurityNotice = installRes.SecurityNotice
			res.FetchedCount++
		}

		res.Dependencies = append(res.Dependencies, item)
	}

	return res, nil
}

func checkRegistryDependency(depName, depVer, projectDir, base string) (bool, string, string) {
	registryDir := GetRegistryDir(base)

	// A. Local relative path: ./something or ../something
	if strings.HasPrefix(depName, "./") || strings.HasPrefix(depName, "../") {
		absPath := filepath.Join(projectDir, depName)
		if m, err := ReadManifest(filepath.Join(absPath, "bmdpkg.json")); err == nil {
			candDir := RegistryLocalPath(m.Name, base)
			if isDirWithEntrypoint(candDir) {
				return true, candDir, inspectSecurityNotice(candDir)
			}
		} else {
			baseName := filepath.Base(absPath)
			candDir := RegistryLocalPath(baseName, base)
			if isDirWithEntrypoint(candDir) {
				return true, candDir, inspectSecurityNotice(candDir)
			}
		}
	}

	// B. GitHub package: github.com/<user>/<repo>
	if strings.HasPrefix(depName, "github.com/") {
		clean := strings.TrimPrefix(depName, "github.com/")
		parts := strings.Split(clean, "/")
		if len(parts) >= 2 {
			user := parts[0]
			repo := parts[1]
			candDir := RegistryGithubPath(user, repo, depVer, base)
			if isDirWithEntrypoint(candDir) {
				return true, candDir, inspectSecurityNotice(candDir)
			}
		}
	}

	// C. Local package: local/<name>
	if strings.HasPrefix(depName, "local/") {
		pkgName := strings.TrimPrefix(depName, "local/")
		candDir := RegistryLocalPath(pkgName, base)
		if isDirWithEntrypoint(candDir) {
			return true, candDir, inspectSecurityNotice(candDir)
		}
		candVer := filepath.Join(registryDir, "local", pkgName, depVer)
		if isDirWithEntrypoint(candVer) {
			return true, candVer, inspectSecurityNotice(candVer)
		}
	}

	// D. CDN package: cdn/<domainHash>/<ver>
	if strings.HasPrefix(depName, "cdn/") {
		parts := strings.Split(strings.TrimPrefix(depName, "cdn/"), "/")
		if len(parts) >= 2 {
			domainHash := parts[0]
			candDir := RegistryCdnPath(domainHash, depVer, base)
			if isDirWithEntrypoint(candDir) {
				return true, candDir, inspectSecurityNotice(candDir)
			}
		}
	}

	// E. Attempt resolution through standard resolver
	candidateQueries := []string{
		depName + "@" + depVer,
		depName + "/" + depVer,
		depName,
	}

	for _, query := range candidateQueries {
		if entrypoint, err := ResolveRegistryPackage(query, base); err == nil && entrypoint != "" {
			pkgDir := filepath.Dir(entrypoint)
			return true, pkgDir, inspectSecurityNotice(pkgDir)
		}
	}

	return false, "", ""
}

func inspectSecurityNotice(dir string) string {
	mfPath := filepath.Join(dir, "bmdpkg.json")
	if m, err := ReadManifest(mfPath); err == nil {
		return m.Permissions.SecurityNotice()
	}
	return ""
}
