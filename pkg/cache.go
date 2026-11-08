package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CacheInfoResult contains metrics and storage usage of the centralized library directory.
type CacheInfoResult struct {
	BaseDir          string
	TotalSize        int64
	TarballCount     int
	TarballSize      int64
	RegistryCount    int
	RegistrySize     int64
	IntegrityRecords int
	IntegrityDBPath  string
}

// CacheInfo scans the centralized directory and aggregates storage and package metrics.
func CacheInfo(baseDir ...string) (*CacheInfoResult, error) {
	base := resolveBaseDir(baseDir)
	_ = EnsureDirectoryStructure(base)

	res := &CacheInfoResult{
		BaseDir:         base,
		IntegrityDBPath: GetIntegrityDBPath(base),
	}

	// 1. Total storage size
	_ = filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			res.TotalSize += info.Size()
		}
		return nil
	})

	// 2. Tarballs metrics
	tarballsDir := GetCacheTarballsDir(base)
	_ = filepath.Walk(tarballsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			res.TarballCount++
			res.TarballSize += info.Size()
		}
		return nil
	})

	// 3. Registry packages metrics
	registryDir := GetRegistryDir(base)
	_ = filepath.Walk(registryDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if !info.IsDir() {
				res.RegistrySize += info.Size()
			} else if path != registryDir {
				// Check if directory is a package root
				hasEntry := fileExists(filepath.Join(path, "bmdpkg.json")) ||
					fileExists(filepath.Join(path, "main.bmd")) ||
					fileExists(filepath.Join(path, "index.bmd"))
				if hasEntry {
					res.RegistryCount++
				}
			}
		}
		return nil
	})

	// 4. Integrity records count
	db, err := LoadIntegrityDB(res.IntegrityDBPath)
	if err == nil {
		res.IntegrityRecords = len(db.Records)
	}

	return res, nil
}

// CachePruneResult details the results of cache pruning.
type CachePruneResult struct {
	RemovedTarballs   []string
	FreedBytes        int64
	PreservedPackages int
}

// CachePrune removes tarballs from cache/tarballs/ whose corresponding versions
// have already been extracted to registry/, safely freeing disk space while
// keeping active registry package trees intact.
func CachePrune(baseDir ...string) (*CachePruneResult, error) {
	base := resolveBaseDir(baseDir)
	_ = EnsureDirectoryStructure(base)

	tarballsDir := GetCacheTarballsDir(base)
	registryDir := GetRegistryDir(base)
	db, _ := LoadIntegrityDB(GetIntegrityDBPath(base))

	res := &CachePruneResult{}

	entries, err := os.ReadDir(tarballsDir)
	if err != nil {
		return res, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		filePath := filepath.Join(tarballsDir, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Determine if corresponding package exists in registry
		canRemove := false

		// Pattern: github.com_<user>_<repo>_<ver>.tar.gz
		if strings.HasPrefix(name, "github.com_") {
			clean := strings.TrimPrefix(name, "github.com_")
			clean = strings.TrimSuffix(clean, ".tar.gz")
			clean = strings.TrimSuffix(clean, ".tgz")
			parts := strings.Split(clean, "_")
			if len(parts) >= 3 {
				user := parts[0]
				repo := parts[1]
				ver := parts[2]
				candDir := RegistryGithubPath(user, repo, ver, base)
				if isDirWithEntrypoint(candDir) {
					canRemove = true
				}
			}
		} else if strings.HasPrefix(name, "cdn_") {
			// Pattern: cdn_<domainHash>_<ver>.tar.gz
			clean := strings.TrimPrefix(name, "cdn_")
			clean = strings.TrimSuffix(clean, ".tar.gz")
			clean = strings.TrimSuffix(clean, ".tgz")
			parts := strings.Split(clean, "_")
			if len(parts) >= 2 {
				domainHash := parts[0]
				ver := parts[1]
				candDir := RegistryCdnPath(domainHash, ver, base)
				if isDirWithEntrypoint(candDir) {
					canRemove = true
				}
			}
		} else {
			// Other archives: check if any directory in registry matches
			canRemove = false
		}

		if canRemove {
			if err := os.Remove(filePath); err == nil {
				res.RemovedTarballs = append(res.RemovedTarballs, name)
				res.FreedBytes += info.Size()
				if db != nil {
					_ = db.Delete("tarballs/" + name)
				}
			}
		}
	}

	if db != nil {
		_ = db.Save()
	}

	// Count preserved active packages in registry
	_ = filepath.Walk(registryDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != registryDir {
			if isDirWithEntrypoint(path) {
				res.PreservedPackages++
			}
		}
		return nil
	})

	return res, nil
}

func isDirWithEntrypoint(dir string) bool {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false
	}
	return fileExists(filepath.Join(dir, "bmdpkg.json")) ||
		fileExists(filepath.Join(dir, "main.bmd")) ||
		fileExists(filepath.Join(dir, "index.bmd"))
}

// CacheVerifyItem represents the verification status of a single record or file.
type CacheVerifyItem struct {
	Key      string
	Path     string
	Checksum string
	Expected string
	Status   string // "OK", "MISMATCH", "MISSING"
	Error    error
}

// CacheVerifyResult details the integrity validation of the cache and registry.
type CacheVerifyResult struct {
	Items       []CacheVerifyItem
	TotalCount  int
	ValidCount  int
	FailedCount int
}

// CacheVerify validates SHA-256 checksums of cached tarballs and packages
// against records stored in locks/integrity.db.
func CacheVerify(baseDir ...string) (*CacheVerifyResult, error) {
	base := resolveBaseDir(baseDir)
	_ = EnsureDirectoryStructure(base)

	db, err := LoadIntegrityDB(GetIntegrityDBPath(base))
	if err != nil {
		return nil, fmt.Errorf("failed to load integrity database: %w", err)
	}

	res := &CacheVerifyResult{}

	// Verify all records registered in integrity.db
	for _, key := range db.Keys() {
		rec, _ := db.Get(key)
		res.TotalCount++

		item := CacheVerifyItem{
			Key:      key,
			Expected: rec.Checksum,
		}

		// Locate target on disk
		targetPath := locateRecordPath(key, base)
		item.Path = targetPath

		if targetPath == "" || !pathExists(targetPath) {
			item.Status = "MISSING"
			item.Error = fmt.Errorf("cached asset not found on disk")
			res.FailedCount++
			res.Items = append(res.Items, item)
			continue
		}

		fi, err := os.Stat(targetPath)
		if err != nil {
			item.Status = "MISSING"
			item.Error = err
			res.FailedCount++
			res.Items = append(res.Items, item)
			continue
		}

		actualHash := ""
		if fi.IsDir() {
			actualHash = computeDirHash(targetPath)
		} else {
			h, _, err := ComputeFileSHA256(targetPath)
			if err != nil {
				item.Status = "ERROR"
				item.Error = err
				res.FailedCount++
				res.Items = append(res.Items, item)
				continue
			}
			actualHash = h
		}

		item.Checksum = actualHash
		if actualHash == rec.Checksum {
			item.Status = "OK"
			res.ValidCount++
		} else {
			item.Status = "MISMATCH"
			item.Error = fmt.Errorf("checksum mismatch: expected %s, got %s", rec.Checksum, actualHash)
			res.FailedCount++
		}

		res.Items = append(res.Items, item)
	}

	return res, nil
}

func locateRecordPath(key, base string) string {
	tarballsDir := GetCacheTarballsDir(base)
	registryDir := GetRegistryDir(base)

	// Check if key is a tarball path
	if strings.HasPrefix(key, "tarballs/") {
		tarName := strings.TrimPrefix(key, "tarballs/")
		return filepath.Join(tarballsDir, tarName)
	}

	// Check GitHub package format: github.com/<user>/<repo>/<ver>
	if strings.HasPrefix(key, "github.com/") {
		parts := strings.Split(strings.TrimPrefix(key, "github.com/"), "/")
		if len(parts) >= 3 {
			user := parts[0]
			repo := parts[1]
			ver := parts[2]
			// Try tarball first
			tarCand := filepath.Join(tarballsDir, fmt.Sprintf("github.com_%s_%s_%s.tar.gz", user, repo, ver))
			if fileExists(tarCand) {
				return tarCand
			}
			// Fallback to registry directory
			regCand := RegistryGithubPath(user, repo, ver, base)
			if pathExists(regCand) {
				return regCand
			}
		}
	}

	// Check CDN format: cdn/<domainHash>/<ver>
	if strings.HasPrefix(key, "cdn/") {
		parts := strings.Split(strings.TrimPrefix(key, "cdn/"), "/")
		if len(parts) >= 2 {
			domainHash := parts[0]
			ver := parts[1]
			tarCand := filepath.Join(tarballsDir, fmt.Sprintf("cdn_%s_%s.tar.gz", domainHash, ver))
			if fileExists(tarCand) {
				return tarCand
			}
			regCand := RegistryCdnPath(domainHash, ver, base)
			if pathExists(regCand) {
				return regCand
			}
		}
	}

	// Check local format: local/<name>@dev or local/<name>@<ver>
	if strings.HasPrefix(key, "local/") {
		parts := strings.SplitN(strings.TrimPrefix(key, "local/"), "@", 2)
		pkgName := parts[0]
		ver := "dev"
		if len(parts) > 1 {
			ver = parts[1]
		}
		candidates := []string{
			RegistryLocalPath(pkgName, base),
			filepath.Join(registryDir, "local", pkgName, ver),
			filepath.Join(registryDir, "local", pkgName+"@"+ver),
			filepath.Join(registryDir, "local", pkgName),
			filepath.Join(registryDir, pkgName, ver),
			filepath.Join(registryDir, pkgName),
		}
		for _, cand := range candidates {
			if pathExists(cand) {
				return cand
			}
		}
	}

	// Try direct join under registry
	cand := filepath.Join(registryDir, key)
	if pathExists(cand) {
		return cand
	}

	return ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
