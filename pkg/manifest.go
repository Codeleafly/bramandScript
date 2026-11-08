package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Package names allow lowercase/uppercase alphanumeric, dashes, dots, underscores, and forward slashes (for namespaces)
	packageNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.\-]+(/[a-zA-Z0-9_.\-]+)*$`)

	// Standard SemVer regex with optional 'v' prefix
	semverRegex = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

// Permissions specifies runtime sandbox permissions declared by the package.
type Permissions struct {
	Network         bool `json:"network"`
	FilesystemRead  bool `json:"filesystem_read"`
	FilesystemWrite bool `json:"filesystem_write"`
	Subprocess      bool `json:"subprocess"`
}

// Compatibility specifies engine and hardware requirements.
type Compatibility struct {
	BmdMin       string   `json:"bmd_min,omitempty"`
	Architecture []string `json:"architecture,omitempty"`
}

// UnmarshalJSON provides flexible decoding for Compatibility, accepting
// either a string or an array of strings for "architecture".
func (c *Compatibility) UnmarshalJSON(data []byte) error {
	type rawCompat struct {
		BmdMin       string          `json:"bmd_min,omitempty"`
		Architecture json.RawMessage `json:"architecture,omitempty"`
	}

	var raw rawCompat
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.BmdMin = raw.BmdMin

	if len(raw.Architecture) > 0 {
		// Try string slice
		var slice []string
		if err := json.Unmarshal(raw.Architecture, &slice); err == nil {
			c.Architecture = slice
			return nil
		}

		// Try single string
		var single string
		if err := json.Unmarshal(raw.Architecture, &single); err == nil {
			if single != "" {
				c.Architecture = []string{single}
			}
			return nil
		}
	}

	return nil
}

// Manifest represents the standard bmdpkg.json package definition.
type Manifest struct {
	Name          string            `json:"name"`
	Version       string            `json:"version"`
	Description   string            `json:"description,omitempty"`
	Entry         string            `json:"entry,omitempty"`
	License       string            `json:"license,omitempty"`
	Authors       []string          `json:"authors,omitempty"`
	Permissions   Permissions       `json:"permissions"`
	Dependencies  map[string]string `json:"dependencies,omitempty"`
	Compatibility Compatibility     `json:"compatibility,omitempty"`
}

// ParseManifest parses and validates a bmdpkg.json JSON byte slice.
func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid bmdpkg.json format: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, err
	}

	return &m, nil
}

// ReadManifest reads and parses a bmdpkg.json file from the specified path.
func ReadManifest(filePath string) (*Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest '%s': %w", filePath, err)
	}
	return ParseManifest(data)
}

// Validate checks that all required fields and invariants in the manifest are met.
func (m *Manifest) Validate() error {
	trimmedName := strings.TrimSpace(m.Name)
	if trimmedName == "" {
		return fmt.Errorf("manifest validation error: 'name' is required")
	}
	if !packageNameRegex.MatchString(trimmedName) {
		return fmt.Errorf("manifest validation error: invalid package name '%s'", trimmedName)
	}

	trimmedVer := strings.TrimSpace(m.Version)
	if trimmedVer == "" {
		return fmt.Errorf("manifest validation error: 'version' is required")
	}
	if !semverRegex.MatchString(trimmedVer) && trimmedVer != "dev" {
		return fmt.Errorf("manifest validation error: invalid semver version '%s'", trimmedVer)
	}

	if m.Entry != "" {
		cleanEntry := filepath.Clean(m.Entry)
		if strings.HasPrefix(cleanEntry, "..") || filepath.IsAbs(m.Entry) {
			return fmt.Errorf("manifest validation error: 'entry' path '%s' cannot point outside package root", m.Entry)
		}
	}

	if m.Compatibility.BmdMin != "" {
		cleanMin := strings.TrimSpace(m.Compatibility.BmdMin)
		if !semverRegex.MatchString(cleanMin) {
			return fmt.Errorf("manifest validation error: invalid bmd_min version '%s'", cleanMin)
		}
	}

	for depPkg, depVer := range m.Dependencies {
		if strings.TrimSpace(depPkg) == "" {
			return fmt.Errorf("manifest validation error: dependency package name cannot be empty")
		}
		if strings.TrimSpace(depVer) == "" {
			return fmt.Errorf("manifest validation error: dependency version constraint for '%s' cannot be empty", depPkg)
		}
	}

	return nil
}

// ToJSON serializes the manifest to compact JSON.
func (m *Manifest) ToJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

// ToJSONIndent serializes the manifest to human-readable indented JSON.
func (m *Manifest) ToJSONIndent() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(m, "", "  ")
}

// Save writes the manifest to the specified file path.
func (m *Manifest) Save(filePath string) error {
	data, err := m.ToJSONIndent()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
