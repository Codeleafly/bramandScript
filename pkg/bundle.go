package pkg

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// BundleMagic is the magic header identifying BramandScript standalone embedded bundles.
	BundleMagic = "BMD_BUNDLE_V2\n"
)

// BundleFile represents a single embedded file within the standalone executable bundle.
type BundleFile struct {
	VirtualPath string    `json:"virtual_path"`
	Content     string    `json:"content"`
	Manifest    *Manifest `json:"manifest,omitempty"`
	PackageName string    `json:"package_name,omitempty"`
}

// Bundle represents a self-contained execution package containing the main program
// and all resolved dependency modules.
type Bundle struct {
	Version    int                   `json:"version"`
	MainFile   string                `json:"main_file"`
	MainSource string                `json:"main_source"`
	Files      map[string]BundleFile `json:"files"`
}

var (
	activeBundle   *Bundle
	activeBundleMu sync.RWMutex
)

// RegisterActiveBundle sets the active embedded bundle for the current process.
func RegisterActiveBundle(b *Bundle) {
	activeBundleMu.Lock()
	defer activeBundleMu.Unlock()
	activeBundle = b
}

// GetActiveBundle retrieves the current active embedded bundle, if any.
func GetActiveBundle() *Bundle {
	activeBundleMu.RLock()
	defer activeBundleMu.RUnlock()
	return activeBundle
}

// ClearActiveBundle clears any active embedded bundle.
func ClearActiveBundle() {
	activeBundleMu.Lock()
	defer activeBundleMu.Unlock()
	activeBundle = nil
}

// HasActiveBundle returns true if an active standalone bundle is loaded in memory.
func HasActiveBundle() bool {
	activeBundleMu.RLock()
	defer activeBundleMu.RUnlock()
	return activeBundle != nil
}

// SerializeBundle serializes a Bundle into compressed bytes with the BundleMagic header.
func SerializeBundle(b *Bundle) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(BundleMagic)

	gw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gw)
	if err := enc.Encode(b); err != nil {
		_ = gw.Close()
		return nil, fmt.Errorf("failed to encode bundle: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize bundle compression: %w", err)
	}
	return buf.Bytes(), nil
}

// DeserializeBundle deserializes bundle bytes into a Bundle struct.
func DeserializeBundle(data []byte) (*Bundle, error) {
	if !bytes.HasPrefix(data, []byte(BundleMagic)) {
		return nil, fmt.Errorf("invalid bundle header: missing BMD_BUNDLE_V2 magic")
	}

	payload := data[len(BundleMagic):]
	gr, err := gzip.NewReader(bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress bundle: %w", err)
	}
	defer gr.Close()

	var b Bundle
	if err := json.NewDecoder(gr).Decode(&b); err != nil {
		return nil, fmt.Errorf("failed to decode bundle JSON: %w", err)
	}
	return &b, nil
}

// IsBundleData checks if the provided byte slice begins with the BundleMagic header.
func IsBundleData(data []byte) bool {
	return bytes.HasPrefix(data, []byte(BundleMagic))
}

// ResolveBundleModule searches the active bundle for an import path.
// It checks exact import keys, relative paths against currentBaseDir, and package entrypoints.
func ResolveBundleModule(importPath string, currentBaseDir string) (string, *BundleFile, bool) {
	activeBundleMu.RLock()
	defer activeBundleMu.RUnlock()

	if activeBundle == nil || len(activeBundle.Files) == 0 {
		return "", nil, false
	}

	normImport := filepath.ToSlash(filepath.Clean(importPath))

	// 1. Direct match on import key
	if f, ok := activeBundle.Files[importPath]; ok {
		return "embed:bundle/" + importPath, &f, true
	}
	if f, ok := activeBundle.Files[normImport]; ok {
		return "embed:bundle/" + normImport, &f, true
	}

	// 2. Relative resolution if base dir is an embedded bundle path
	if strings.HasPrefix(currentBaseDir, "embed:bundle/") {
		base := strings.TrimPrefix(currentBaseDir, "embed:bundle/")
		rel := filepath.ToSlash(filepath.Clean(filepath.Join(base, normImport)))

		// Try exact rel, rel.bmd, and rel/index.bmd, rel/main.bmd
		candidates := []string{
			rel,
			rel + ".bmd",
			filepath.ToSlash(filepath.Join(rel, "main.bmd")),
			filepath.ToSlash(filepath.Join(rel, "index.bmd")),
		}
		for _, cand := range candidates {
			if f, ok := activeBundle.Files[cand]; ok {
				return "embed:bundle/" + cand, &f, true
			}
		}
	}

	// 3. Fallback: match by filename or suffix
	for k, f := range activeBundle.Files {
		cleanK := filepath.ToSlash(filepath.Clean(k))
		if cleanK == normImport ||
			cleanK == normImport+".bmd" ||
			strings.HasSuffix(cleanK, "/"+normImport) ||
			strings.HasSuffix(cleanK, "/"+normImport+".bmd") ||
			strings.HasSuffix(cleanK, "/"+normImport+"/main.bmd") ||
			strings.HasSuffix(cleanK, "/"+normImport+"/index.bmd") {
			return "embed:bundle/" + k, &f, true
		}
	}

	return "", nil, false
}

// ReadActiveBundleFile returns the source code bytes for an embed:bundle/ file path.
func ReadActiveBundleFile(path string) ([]byte, error) {
	activeBundleMu.RLock()
	defer activeBundleMu.RUnlock()

	if activeBundle == nil {
		return nil, fmt.Errorf("no active bundle loaded")
	}

	key := strings.TrimPrefix(path, "embed:bundle/")
	key = filepath.ToSlash(filepath.Clean(key))

	if f, ok := activeBundle.Files[key]; ok {
		return []byte(f.Content), nil
	}

	// Try without clean or with suffix match
	for k, f := range activeBundle.Files {
		if filepath.ToSlash(filepath.Clean(k)) == key {
			return []byte(f.Content), nil
		}
	}

	return nil, fmt.Errorf("bundled file not found: %s", path)
}

// GetBundleManifest returns the manifest associated with an embedded bundle file, if any.
func GetBundleManifest(path string) (*Manifest, bool) {
	activeBundleMu.RLock()
	defer activeBundleMu.RUnlock()

	if activeBundle == nil {
		return nil, false
	}

	key := strings.TrimPrefix(path, "embed:bundle/")
	key = filepath.ToSlash(filepath.Clean(key))

	if f, ok := activeBundle.Files[key]; ok && f.Manifest != nil {
		return f.Manifest, true
	}

	for k, f := range activeBundle.Files {
		if filepath.ToSlash(filepath.Clean(k)) == key && f.Manifest != nil {
			return f.Manifest, true
		}
	}

	return nil, false
}

// FindPackageManifest searches upwards from a file path on disk to find the nearest bmdpkg.json manifest.
func FindPackageManifest(filePath string) (*Manifest, string) {
	if strings.HasPrefix(filePath, "embed:bundle/") {
		if m, ok := GetBundleManifest(filePath); ok {
			return m, ""
		}
		return nil, ""
	}
	if strings.HasPrefix(filePath, "embed:stdlib/") {
		return nil, ""
	}

	dir := filepath.Dir(filePath)
	for i := 0; i < 6; i++ {
		manifestPath := filepath.Join(dir, "bmdpkg.json")
		if fi, err := os.Stat(manifestPath); err == nil && !fi.IsDir() {
			if m, err := ReadManifest(manifestPath); err == nil {
				return m, dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir || parent == "" {
			break
		}
		dir = parent
	}
	return nil, ""
}

// BuildBundle constructs a self-contained execution Bundle containing the main program
// and all analyzed dependency modules (relative imports, packages from registry, etc.).
func BuildBundle(
	mainFile string,
	mainSource string,
	importedPaths map[string]string,
	importedSources map[string][]byte,
	importedManifests map[string]*Manifest,
) (*Bundle, []byte, error) {
	bundle := &Bundle{
		Version:    1,
		MainFile:   filepath.Base(mainFile),
		MainSource: mainSource,
		Files:      make(map[string]BundleFile),
	}

	mainDir := filepath.Dir(mainFile)
	if absMain, err := filepath.Abs(mainFile); err == nil {
		mainDir = filepath.Dir(absMain)
	}

	for importPath, resolvedPath := range importedPaths {
		// Skip stdlib imports bundled via embed.FS
		if strings.HasPrefix(resolvedPath, "embed:stdlib/") {
			continue
		}

		var src string
		if data, ok := importedSources[resolvedPath]; ok && len(data) > 0 {
			src = string(data)
		} else {
			data, err := os.ReadFile(resolvedPath)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to read dependency module '%s' (%s): %w", importPath, resolvedPath, err)
			}
			src = string(data)
		}

		manifest := importedManifests[resolvedPath]
		pkgDir := ""
		if manifest == nil {
			manifest, pkgDir = FindPackageManifest(resolvedPath)
		} else {
			_, pkgDir = FindPackageManifest(resolvedPath)
		}

		pkgName := ""
		if manifest != nil {
			pkgName = manifest.Name
		}

		bf := BundleFile{
			VirtualPath: importPath,
			Content:     src,
			Manifest:    manifest,
			PackageName: pkgName,
		}

		// Register under exact import path and normalized import path
		bundle.Files[importPath] = bf
		normImport := filepath.ToSlash(filepath.Clean(importPath))
		bundle.Files[normImport] = bf

		// If relative path from mainDir, register clean relative path
		if rel, err := filepath.Rel(mainDir, resolvedPath); err == nil {
			bundle.Files[filepath.ToSlash(rel)] = bf
			bundle.Files["./"+filepath.ToSlash(rel)] = bf
		}

		// If resolved path is inside a package directory, walk and include all .bmd sibling files
		if pkgDir != "" {
			_ = filepath.Walk(pkgDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				if strings.HasSuffix(path, ".bmd") {
					relToPkg, err := filepath.Rel(pkgDir, path)
					if err == nil {
						relSlash := filepath.ToSlash(relToPkg)
						if _, exists := bundle.Files[relSlash]; !exists {
							content, err := os.ReadFile(path)
							if err == nil {
								bundle.Files[relSlash] = BundleFile{
									VirtualPath: relSlash,
									Content:     string(content),
									Manifest:    manifest,
									PackageName: pkgName,
								}
								// Also register prefixed with importPath
								bundle.Files[filepath.ToSlash(filepath.Join(normImport, relSlash))] = BundleFile{
									VirtualPath: filepath.ToSlash(filepath.Join(normImport, relSlash)),
									Content:     string(content),
									Manifest:    manifest,
									PackageName: pkgName,
								}
							}
						}
					}
				}
				return nil
			})
		}
	}

	bundleBytes, err := SerializeBundle(bundle)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize standalone bundle: %w", err)
	}

	return bundle, bundleBytes, nil
}
