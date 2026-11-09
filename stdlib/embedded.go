package stdlib

import (
	"embed"
	"io/fs"
	"strings"
)

//go:embed bmd_std/*
var FS embed.FS

// ReadFile loads the bytes of an embedded standard library file.
func ReadFile(path string) ([]byte, error) {
	path = normalizePath(path)

	candidates := []string{
		"bmd_std/" + path + "/index.bmd",
		"bmd_std/" + path + ".bmd",
		"bmd_std/" + path,
		path + "/index.bmd",
		path + ".bmd",
		path,
	}

	for _, cand := range candidates {
		data, err := FS.ReadFile(cand)
		if err == nil {
			return data, nil
		}
	}

	return fs.ReadFile(FS, path)
}

// ResolveEmbeddedPath returns the canonical embedded path if the module exists.
func ResolveEmbeddedPath(path string) (string, bool) {
	path = normalizePath(path)

	candidates := []string{
		"bmd_std/" + path + "/index.bmd",
		"bmd_std/" + path + ".bmd",
		"bmd_std/" + path,
		path + "/index.bmd",
		path + ".bmd",
		path,
	}

	for _, cand := range candidates {
		if _, err := FS.ReadFile(cand); err == nil {
			return cand, true
		}
	}
	return "", false
}

// Exists checks if an embedded standard library module exists.
func Exists(path string) bool {
	_, ok := ResolveEmbeddedPath(path)
	return ok
}

func normalizePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.TrimPrefix(path, "embed:stdlib/bmd_std/")
	path = strings.TrimPrefix(path, "embed:stdlib/")
	path = strings.TrimPrefix(path, "embed:")
	path = strings.TrimPrefix(path, "bmd_std/")
	path = strings.TrimPrefix(path, "std/")
	path = strings.TrimPrefix(path, "stdlib/")
	path = strings.TrimPrefix(path, "/")
	return path
}
