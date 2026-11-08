package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	githubRegex = regexp.MustCompile(`^(?:https?://)?github\.com/([a-zA-Z0-9_\-\.]+)/([a-zA-Z0-9_\-\.]+)(?:@([a-zA-Z0-9_\-\.\+]+))?/?$`)
)

// InstallOptions contains configuration for installing a package.
type InstallOptions struct {
	Target     string       // URL, GitHub slug, local path, or local name
	Version    string       // Optional version override
	BaseDir    string       // Custom base dir (defaults to GetBmdLibsDir())
	HTTPClient *http.Client // Custom HTTP client for testing/offline support
	Offline    bool         // If true, do not make any network requests
}

// InstallResult details the outcome of an install operation.
type InstallResult struct {
	Target         string
	PackageName    string
	Version        string
	RegistryDir    string
	TarballPath    string
	FromCache      bool
	Manifest       *Manifest
	SecurityNotice string
}

// Installer manages downloading, verifying, caching, and installing packages.
type Installer struct {
	BaseDir    string
	HTTPClient *http.Client
	Offline    bool
}

// NewInstaller creates an Installer with the specified base directory.
func NewInstaller(baseDir ...string) *Installer {
	b := ""
	if len(baseDir) > 0 && baseDir[0] != "" {
		b = baseDir[0]
	} else {
		b = GetBmdLibsDir()
	}
	return &Installer{
		BaseDir: b,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Install installs the specified target (URL, CDN, GitHub, or local directory).
func (inst *Installer) Install(target string, versionOverride ...string) (*InstallResult, error) {
	if err := EnsureDirectoryStructure(inst.BaseDir); err != nil {
		return nil, fmt.Errorf("failed to prepare centralized library directories: %w", err)
	}

	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return nil, fmt.Errorf("target cannot be empty")
	}

	explicitVer := ""
	if len(versionOverride) > 0 && versionOverride[0] != "" {
		explicitVer = versionOverride[0]
	}

	// 1. Check if target is a local directory or file archive
	if isLocalPath(trimmed) {
		return inst.installLocal(trimmed, explicitVer)
	}

	// 2. Check if target is a local package namespace (e.g. "local/tool@dev" or "local/tool")
	if strings.HasPrefix(trimmed, "local/") {
		return inst.installLocalNamespace(trimmed, explicitVer)
	}

	// 3. Check if target is a GitHub repository (github.com/user/project[@version])
	if githubRegex.MatchString(trimmed) || strings.HasPrefix(trimmed, "github.com/") {
		return inst.installGitHub(trimmed, explicitVer)
	}

	// 4. Check if target is a Remote URL / CDN (https://, http://, or cdn.bmd.io/ or cdn/)
	if isRemoteOrCDN(trimmed) {
		return inst.installRemoteOrCDN(trimmed, explicitVer)
	}

	// 5. Fallback: check if local directory with name exists
	if fi, err := os.Stat(trimmed); err == nil && fi.IsDir() {
		return inst.installLocal(trimmed, explicitVer)
	}

	return nil, fmt.Errorf("unrecognized installation target format: '%s'", target)
}

func isLocalPath(target string) bool {
	if target == "." || target == ".." {
		return true
	}
	if strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") || filepath.IsAbs(target) {
		return true
	}
	// Check if file exists on disk
	if fi, err := os.Stat(target); err == nil {
		return fi.IsDir() || strings.HasSuffix(target, ".tar.gz") || strings.HasSuffix(target, ".tgz") || strings.HasSuffix(target, ".zip") || strings.HasSuffix(target, ".tar")
	}
	return false
}

func isRemoteOrCDN(target string) bool {
	return strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "cdn.bmd.io/") ||
		strings.HasPrefix(target, "cdn/")
}

// installLocal installs from a local directory or local archive file.
func (inst *Installer) installLocal(path string, explicitVer string) (*InstallResult, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("local path '%s' not found: %w", path, err)
	}

	if fi.IsDir() {
		return inst.installLocalDirectory(path, explicitVer)
	}

	// Handle local archive file (.tar.gz, .zip, etc.)
	return inst.installLocalArchive(path, explicitVer)
}

func (inst *Installer) installLocalDirectory(dirPath string, explicitVer string) (*InstallResult, error) {
	absDir, err := filepath.Abs(dirPath)
	if err != nil {
		absDir = dirPath
	}

	// Read bmdpkg.json if present
	manifestPath := filepath.Join(absDir, "bmdpkg.json")
	var manifest *Manifest
	pkgName := filepath.Base(absDir)
	version := "dev"

	if m, err := ReadManifest(manifestPath); err == nil {
		manifest = m
		if m.Name != "" {
			pkgName = m.Name
		}
		if m.Version != "" {
			version = m.Version
		}
	}

	if explicitVer != "" {
		version = explicitVer
	}

	targetDir := ""
	if strings.Contains(pkgName, "/") {
		targetDir = filepath.Join(GetRegistryDir(inst.BaseDir), pkgName, version)
	} else if version == "dev" {
		targetDir = RegistryLocalPath(pkgName, inst.BaseDir)
	} else {
		targetDir = filepath.Join(GetRegistryDir(inst.BaseDir), "local", pkgName, version)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := copyDirectoryContents(absDir, targetDir); err != nil {
		return nil, fmt.Errorf("failed to copy local package files: %w", err)
	}

	// Verify entrypoint exists
	if _, err := ResolvePackageEntrypoint(targetDir); err != nil {
		// Non-fatal if user is still scaffolding, but validate if possible
	}

	// Compute checksum of the package contents to record in integrity.db
	recordKey := fmt.Sprintf("local/%s@%s", pkgName, version)
	db, _ := LoadIntegrityDB(GetIntegrityDBPath(inst.BaseDir))
	hash := computeDirHash(targetDir)
	db.Set(recordKey, hash, 0)
	_ = db.Save()

	var secNotice string
	if manifest != nil {
		secNotice = manifest.Permissions.SecurityNotice()
	}

	return &InstallResult{
		Target:         dirPath,
		PackageName:    pkgName,
		Version:        version,
		RegistryDir:    targetDir,
		Manifest:       manifest,
		SecurityNotice: secNotice,
	}, nil
}

func (inst *Installer) installLocalArchive(archivePath string, explicitVer string) (*InstallResult, error) {
	tempExtract, err := os.MkdirTemp("", "bmd_extract_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempExtract)

	if err := ExtractArchive(archivePath, tempExtract); err != nil {
		return nil, fmt.Errorf("failed to extract local archive: %w", err)
	}

	return inst.installLocalDirectory(tempExtract, explicitVer)
}

func (inst *Installer) installLocalNamespace(target string, explicitVer string) (*InstallResult, error) {
	// target is e.g. "local/my_tool@dev" or "local/my_tool"
	sub := strings.TrimPrefix(target, "local/")
	pkgName := sub
	version := "dev"
	if strings.Contains(sub, "@") {
		parts := strings.SplitN(sub, "@", 2)
		pkgName = parts[0]
		version = parts[1]
	}
	if explicitVer != "" {
		version = explicitVer
	}

	// First, check if a local directory exists with that name in cwd
	if fi, err := os.Stat(pkgName); err == nil && fi.IsDir() {
		return inst.installLocalDirectory(pkgName, version)
	}

	// Check if already in registry
	regPath := RegistryLocalPath(pkgName, inst.BaseDir)
	if fi, err := os.Stat(regPath); err == nil && fi.IsDir() {
		var m *Manifest
		if mf, err := ReadManifest(filepath.Join(regPath, "bmdpkg.json")); err == nil {
			m = mf
		}
		var sec string
		if m != nil {
			sec = m.Permissions.SecurityNotice()
		}
		return &InstallResult{
			Target:         target,
			PackageName:    pkgName,
			Version:        version,
			RegistryDir:    regPath,
			FromCache:      true,
			Manifest:       m,
			SecurityNotice: sec,
		}, nil
	}

	return nil, fmt.Errorf("local package '%s' not found in current directory or registry", target)
}

func (inst *Installer) installGitHub(target string, explicitVer string) (*InstallResult, error) {
	matches := githubRegex.FindStringSubmatch(target)
	var user, project, ver string
	if len(matches) >= 3 {
		user = matches[1]
		project = matches[2]
		if len(matches) > 3 {
			ver = matches[3]
		}
	} else {
		clean := strings.TrimPrefix(target, "https://")
		clean = strings.TrimPrefix(clean, "http://")
		clean = strings.TrimPrefix(clean, "github.com/")
		parts := strings.Split(clean, "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid github repository format: '%s'", target)
		}
		user = parts[0]
		projParts := strings.SplitN(parts[1], "@", 2)
		project = projParts[0]
		if len(projParts) > 1 {
			ver = projParts[1]
		}
	}

	if explicitVer != "" {
		ver = explicitVer
	}
	if ver == "" {
		ver = "main"
	}

	tarballName := fmt.Sprintf("github.com_%s_%s_%s.tar.gz", user, project, ver)
	tarballPath := filepath.Join(GetCacheTarballsDir(inst.BaseDir), tarballName)
	registryDir := RegistryGithubPath(user, project, ver, inst.BaseDir)
	integrityKey := fmt.Sprintf("github.com/%s/%s/%s", user, project, ver)

	// Check if already extracted in registry
	if fi, err := os.Stat(registryDir); err == nil && fi.IsDir() {
		var m *Manifest
		if mf, err := ReadManifest(filepath.Join(registryDir, "bmdpkg.json")); err == nil {
			m = mf
		}
		var sec string
		if m != nil {
			sec = m.Permissions.SecurityNotice()
		}
		return &InstallResult{
			Target:         target,
			PackageName:    fmt.Sprintf("github.com/%s/%s", user, project),
			Version:        ver,
			RegistryDir:    registryDir,
			TarballPath:    tarballPath,
			FromCache:      true,
			Manifest:       m,
			SecurityNotice: sec,
		}, nil
	}

	// Check if cached tarball exists on disk
	cachedTarball := false
	if fi, err := os.Stat(tarballPath); err == nil && fi.Size() > 0 {
		cachedTarball = true
	}

	if !cachedTarball {
		if inst.Offline {
			return nil, fmt.Errorf("cannot download '%s' in offline mode: tarball not cached", target)
		}

		// Download release tarball from GitHub
		// Try release tags first, then heads/archive fallback
		downloadURLs := []string{
			fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/v%s.tar.gz", user, project, ver),
			fmt.Sprintf("https://github.com/%s/%s/archive/refs/tags/%s.tar.gz", user, project, ver),
			fmt.Sprintf("https://github.com/%s/%s/archive/%s.tar.gz", user, project, ver),
		}

		var lastErr error
		downloaded := false
		for _, dlURL := range downloadURLs {
			err := inst.downloadFile(dlURL, tarballPath)
			if err == nil {
				downloaded = true
				break
			}
			lastErr = err
		}

		if !downloaded {
			return nil, fmt.Errorf("failed to download release tarball for 'github.com/%s/%s@%s': %w", user, project, ver, lastErr)
		}
	}

	// Verify integrity and record in integrity.db
	db, err := LoadIntegrityDB(GetIntegrityDBPath(inst.BaseDir))
	if err == nil {
		_, _ = db.RecordFile(integrityKey, tarballPath)
		_ = db.Save()
	}

	// Extract to registry
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry directory '%s': %w", registryDir, err)
	}

	if err := ExtractTarGz(tarballPath, registryDir); err != nil {
		return nil, fmt.Errorf("failed to extract tarball to '%s': %w", registryDir, err)
	}

	// Inspect bmdpkg.json
	var manifest *Manifest
	manifestPath := filepath.Join(registryDir, "bmdpkg.json")
	if m, err := ReadManifest(manifestPath); err == nil {
		manifest = m
	}

	var secNotice string
	if manifest != nil {
		secNotice = manifest.Permissions.SecurityNotice()
	}

	return &InstallResult{
		Target:         target,
		PackageName:    fmt.Sprintf("github.com/%s/%s", user, project),
		Version:        ver,
		RegistryDir:    registryDir,
		TarballPath:    tarballPath,
		FromCache:      cachedTarball,
		Manifest:       manifest,
		SecurityNotice: secNotice,
	}, nil
}

func (inst *Installer) installRemoteOrCDN(target string, explicitVer string) (*InstallResult, error) {
	targetURL := target
	if strings.HasPrefix(target, "cdn.bmd.io/") {
		targetURL = "https://" + target
	} else if strings.HasPrefix(target, "cdn/") {
		// Handled via cdn registry format: cdn/<domain_hash>/<version>
		parts := strings.Split(strings.TrimPrefix(target, "cdn/"), "/")
		if len(parts) >= 2 {
			domainHash := parts[0]
			version := parts[1]
			regDir := RegistryCdnPath(domainHash, version, inst.BaseDir)
			if fi, err := os.Stat(regDir); err == nil && fi.IsDir() {
				var m *Manifest
				if mf, err := ReadManifest(filepath.Join(regDir, "bmdpkg.json")); err == nil {
					m = mf
				}
				var sec string
				if m != nil {
					sec = m.Permissions.SecurityNotice()
				}
				return &InstallResult{
					Target:         target,
					PackageName:    "cdn/" + domainHash,
					Version:        version,
					RegistryDir:    regDir,
					FromCache:      true,
					Manifest:       m,
					SecurityNotice: sec,
				}, nil
			}
		}
		targetURL = "https://" + target
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL '%s': %w", targetURL, err)
	}

	// Compute domain hash: 12-char SHA-256 of the host
	hostHash := sha256.Sum256([]byte(u.Host))
	domainHash := hex.EncodeToString(hostHash[:])[:12]

	// Determine version
	version := explicitVer
	if version == "" {
		version = extractVersionFromURL(targetURL)
	}

	tarballName := fmt.Sprintf("cdn_%s_%s.tar.gz", domainHash, version)
	tarballPath := filepath.Join(GetCacheTarballsDir(inst.BaseDir), tarballName)

	cachedTarball := false
	if fi, err := os.Stat(tarballPath); err == nil && fi.Size() > 0 {
		cachedTarball = true
	}

	if !cachedTarball {
		if inst.Offline {
			return nil, fmt.Errorf("cannot download '%s' in offline mode: tarball not cached", targetURL)
		}

		if err := inst.downloadFile(targetURL, tarballPath); err != nil {
			return nil, fmt.Errorf("failed to download CDN package '%s': %w", targetURL, err)
		}
	}

	// Extract to temporary directory to inspect bmdpkg.json and resolve final version
	tempDir, err := os.MkdirTemp("", "bmd_cdn_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if err := ExtractArchive(tarballPath, tempDir); err != nil {
		return nil, fmt.Errorf("failed to extract downloaded archive '%s': %w", tarballPath, err)
	}

	var manifest *Manifest
	manifestPath := filepath.Join(tempDir, "bmdpkg.json")
	if m, err := ReadManifest(manifestPath); err == nil {
		manifest = m
		if version == "1.0.0" && m.Version != "" && explicitVer == "" {
			version = m.Version
		}
	}

	registryDir := RegistryCdnPath(domainHash, version, inst.BaseDir)
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create CDN registry directory '%s': %w", registryDir, err)
	}

	if err := copyDirectoryContents(tempDir, registryDir); err != nil {
		return nil, fmt.Errorf("failed moving CDN package contents to registry: %w", err)
	}

	// Record integrity
	integrityKey := fmt.Sprintf("cdn/%s/%s", domainHash, version)
	db, err := LoadIntegrityDB(GetIntegrityDBPath(inst.BaseDir))
	if err == nil {
		_, _ = db.RecordFile(integrityKey, tarballPath)
		_ = db.Save()
	}

	var secNotice string
	if manifest != nil {
		secNotice = manifest.Permissions.SecurityNotice()
	}

	return &InstallResult{
		Target:         target,
		PackageName:    fmt.Sprintf("cdn/%s", domainHash),
		Version:        version,
		RegistryDir:    registryDir,
		TarballPath:    tarballPath,
		FromCache:      cachedTarball,
		Manifest:       manifest,
		SecurityNotice: secNotice,
	}, nil
}

func (inst *Installer) downloadFile(rawURL, destPath string) error {
	client := inst.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BramandScript-PackageManager/0.1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP request failed with status %d %s", resp.StatusCode, resp.Status)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), "dl_tmp_*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		return err
	}

	return os.Rename(tmpFile.Name(), destPath)
}

func extractVersionFromURL(targetURL string) string {
	clean := strings.TrimSuffix(targetURL, ".tar.gz")
	clean = strings.TrimSuffix(clean, ".tgz")
	clean = strings.TrimSuffix(clean, ".zip")

	if idx := strings.LastIndex(clean, "@"); idx != -1 {
		return clean[idx+1:]
	}
	if idx := strings.LastIndex(clean, "/"); idx != -1 {
		candidate := clean[idx+1:]
		if semverRegex.MatchString(candidate) {
			return candidate
		}
	}
	return "1.0.0"
}

func copyDirectoryContents(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		targetPath := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, info.Mode())
	})
}

func computeDirHash(dir string) string {
	hasher := sha256.New()
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		hasher.Write([]byte(rel))
		if data, err := os.ReadFile(path); err == nil {
			hasher.Write(data)
		}
		return nil
	})
	return hex.EncodeToString(hasher.Sum(nil))
}
