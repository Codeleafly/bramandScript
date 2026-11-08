package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DependencyItem describes the resolution and installation state of a dependency.
type DependencyItem struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	ResolvedVersion string   `json:"resolved_version,omitempty"`
	Status          string   `json:"status"` // "CACHED", "FETCHED", "ERROR"
	RegistryDir     string   `json:"registry_dir,omitempty"`
	SecurityNotice  string   `json:"security_notice,omitempty"`
	Error           error    `json:"error,omitempty"`
	DiamondResolved bool     `json:"diamond_resolved,omitempty"`
	RequestedFrom   []string `json:"requested_from,omitempty"`
}

// GetResult details the overall outcome of `bmd get`.
type GetResult struct {
	ManifestPath       string              `json:"manifest_path"`
	Dependencies       []DependencyItem    `json:"dependencies"`
	DiamondResolutions []DiamondResolution `json:"diamond_resolutions,omitempty"`
	CachedCount        int                 `json:"cached_count"`
	FetchedCount       int                 `json:"fetched_count"`
	FailedCount        int                 `json:"failed_count"`
}

type depRequestInfo struct {
	name          string
	version       string
	requestedFrom string
}

// Get inspects the bmdpkg.json file in projectDir, verifies if each dependency
// (including transitive dependencies and diamond dependencies) is already present
// in the centralized registry, and only downloads missing dependencies.
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

	inst := NewInstaller(base)
	if os.Getenv("BMD_OFFLINE") == "1" || os.Getenv("BMD_OFFLINE") == "true" {
		inst.Offline = true
	}

	// 1. Collect all dependency requests (direct & transitive)
	allRequests := make(map[string][]depRequestInfo)
	queue := make([]depRequestInfo, 0)

	// Enqueue direct dependencies
	for depName, depVer := range manifest.Dependencies {
		req := depRequestInfo{
			name:          depName,
			version:       depVer,
			requestedFrom: "root",
		}
		queue = append(queue, req)
	}

	visitedDirs := make(map[string]bool)

	// Process queue to discover transitive dependencies
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		allRequests[current.name] = append(allRequests[current.name], current)

		// Check if package is already in registry or local path
		regDir, ok := locateDependencyRegistryDir(current.name, current.version, projectDir, base)
		if ok && regDir != "" && !visitedDirs[regDir] {
			visitedDirs[regDir] = true
			// Inspect transitive dependencies
			subManifestPath := filepath.Join(regDir, "bmdpkg.json")
			if subM, err := ReadManifest(subManifestPath); err == nil && len(subM.Dependencies) > 0 {
				for subName, subVer := range subM.Dependencies {
					queue = append(queue, depRequestInfo{
						name:          subName,
						version:       subVer,
						requestedFrom: current.name,
					})
				}
			}
		}
	}

	// 2. Sort package names for deterministic processing
	var pkgNames []string
	for k := range allRequests {
		pkgNames = append(pkgNames, k)
	}
	sort.Strings(pkgNames)

	// 3. Resolve and install each package
	for _, pkgName := range pkgNames {
		requests := allRequests[pkgName]

		var versionsRequested []string
		var callers []string
		for _, r := range requests {
			versionsRequested = append(versionsRequested, r.version)
			callers = append(callers, r.requestedFrom)
		}
		versionsRequested = uniqueStrings(versionsRequested)
		callers = uniqueStrings(callers)

		if len(versionsRequested) > 1 {
			// Diamond dependency detected!
			diamond, err := ResolveDiamondConflict(pkgName, versionsRequested)
			if err == nil {
				res.DiamondResolutions = append(res.DiamondResolutions, *diamond)

				if diamond.IsCompatible {
					// Install single highest compatible version
					resolvedVer := diamond.ResolvedVersion
					item := inst.installOrCheckDependency(pkgName, resolvedVer, projectDir, base)
					item.DiamondResolved = true
					item.RequestedFrom = callers
					item.ResolvedVersion = resolvedVer

					if item.Status == "CACHED" {
						res.CachedCount++
					} else if item.Status == "FETCHED" {
						res.FetchedCount++
					} else {
						res.FailedCount++
					}
					res.Dependencies = append(res.Dependencies, item)
					continue
				} else {
					// Incompatible: Install side-by-side versions in registry
					for _, ver := range diamond.SideBySide {
						item := inst.installOrCheckDependency(pkgName, ver, projectDir, base)
						item.DiamondResolved = true
						item.RequestedFrom = callers
						item.ResolvedVersion = ver

						if item.Status == "CACHED" {
							res.CachedCount++
						} else if item.Status == "FETCHED" {
							res.FetchedCount++
						} else {
							res.FailedCount++
						}
						res.Dependencies = append(res.Dependencies, item)
					}
					continue
				}
			}
		}

		// Single version requested
		targetVer := versionsRequested[0]
		item := inst.installOrCheckDependency(pkgName, targetVer, projectDir, base)
		item.RequestedFrom = callers
		if item.Status == "CACHED" {
			res.CachedCount++
		} else if item.Status == "FETCHED" {
			res.FetchedCount++
		} else {
			res.FailedCount++
		}
		res.Dependencies = append(res.Dependencies, item)
	}

	return res, nil
}

func (inst *Installer) installOrCheckDependency(depName, depVer, projectDir, base string) DependencyItem {
	item := DependencyItem{
		Name:    depName,
		Version: depVer,
	}

	isCached, regDir, secNotice := checkRegistryDependency(depName, depVer, projectDir, base)
	if isCached {
		item.Status = "CACHED"
		item.RegistryDir = regDir
		item.SecurityNotice = secNotice
		return item
	}

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
	} else {
		item.Status = "FETCHED"
		item.RegistryDir = installRes.RegistryDir
		item.SecurityNotice = installRes.SecurityNotice
	}

	return item
}

func locateDependencyRegistryDir(depName, depVer, projectDir, base string) (string, bool) {
	isCached, regDir, _ := checkRegistryDependency(depName, depVer, projectDir, base)
	if isCached {
		return regDir, true
	}
	return "", false
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
