package pkg

import (
	"os"
	"path/filepath"
)

const (
	// EnvBmdLibsDir is the primary environment variable overriding the default ~/.bmd_libs location.
	EnvBmdLibsDir = "BMD_LIBS_DIR"

	// EnvLegacyBramandLibsDir is the legacy environment variable for backwards compatibility.
	EnvLegacyBramandLibsDir = "BRAMAND_LIBS_DIR"
)

// GetBmdLibsDir returns the root centralized library directory.
// Priority:
// 1. BMD_LIBS_DIR environment variable
// 2. BRAMAND_LIBS_DIR environment variable
// 3. $HOME/.bmd_libs
// 4. Current working directory fallback (.bmd_libs)
func GetBmdLibsDir() string {
	if dir := os.Getenv(EnvBmdLibsDir); dir != "" {
		return dir
	}
	if dir := os.Getenv(EnvLegacyBramandLibsDir); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".bmd_libs")
	}
	return ".bmd_libs"
}

func resolveBaseDir(baseDir []string) string {
	if len(baseDir) > 0 && baseDir[0] != "" {
		return baseDir[0]
	}
	return GetBmdLibsDir()
}

// GetCacheDir returns the cache root directory ($HOME/.bmd_libs/cache).
func GetCacheDir(baseDir ...string) string {
	return filepath.Join(resolveBaseDir(baseDir), "cache")
}

// GetCacheTarballsDir returns the tarballs cache directory ($HOME/.bmd_libs/cache/tarballs).
func GetCacheTarballsDir(baseDir ...string) string {
	return filepath.Join(resolveBaseDir(baseDir), "cache", "tarballs")
}

// GetRegistryDir returns the package registry directory ($HOME/.bmd_libs/registry).
func GetRegistryDir(baseDir ...string) string {
	return filepath.Join(resolveBaseDir(baseDir), "registry")
}

// GetLocksDir returns the locks directory ($HOME/.bmd_libs/locks).
func GetLocksDir(baseDir ...string) string {
	return filepath.Join(resolveBaseDir(baseDir), "locks")
}

// GetIntegrityDBPath returns the path to integrity.db ($HOME/.bmd_libs/locks/integrity.db).
func GetIntegrityDBPath(baseDir ...string) string {
	return filepath.Join(resolveBaseDir(baseDir), "locks", "integrity.db")
}

// EnsureDirectoryStructure creates the required centralized directory layout:
// - cache/tarballs/
// - registry/
// - locks/
func EnsureDirectoryStructure(baseDir ...string) error {
	base := resolveBaseDir(baseDir)
	dirs := []string{
		filepath.Join(base, "cache", "tarballs"),
		filepath.Join(base, "registry"),
		filepath.Join(base, "locks"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// RegistryGithubPath returns the target path for a GitHub package version:
// registry/github.com/<author>/<pkg>/<version>/
func RegistryGithubPath(author, pkg, version string, baseDir ...string) string {
	return filepath.Join(GetRegistryDir(baseDir...), "github.com", author, pkg, version)
}

// RegistryCdnPath returns the target path for a CDN package version:
// registry/cdn/<domain_hash>/<version>/
func RegistryCdnPath(domainHash, version string, baseDir ...string) string {
	return filepath.Join(GetRegistryDir(baseDir...), "cdn", domainHash, version)
}

// RegistryLocalPath returns the target path for a local development package:
// registry/local/<name>@dev/
func RegistryLocalPath(name string, baseDir ...string) string {
	return filepath.Join(GetRegistryDir(baseDir...), "local", name+"@dev")
}
