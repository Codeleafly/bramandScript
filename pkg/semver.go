package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a structured Semantic Versioning 2.0.0 data structure.
type SemVer struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
	Raw        string
}

// ParseSemVer parses a SemVer string into a SemVer struct.
func ParseSemVer(v string) (SemVer, error) {
	raw := strings.TrimSpace(v)
	if raw == "dev" {
		return SemVer{Raw: "dev"}, nil
	}

	cleaned := strings.TrimPrefix(raw, "v")
	build := ""
	if idx := strings.Index(cleaned, "+"); idx != -1 {
		build = cleaned[idx+1:]
		cleaned = cleaned[:idx]
	}

	prerelease := ""
	if idx := strings.Index(cleaned, "-"); idx != -1 {
		prerelease = cleaned[idx+1:]
		cleaned = cleaned[:idx]
	}

	parts := strings.Split(cleaned, ".")
	if len(parts) == 0 || parts[0] == "" {
		return SemVer{}, fmt.Errorf("invalid semver: %s", v)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return SemVer{}, fmt.Errorf("invalid semver major version '%s': %w", parts[0], err)
	}

	minor := 0
	if len(parts) > 1 {
		minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid semver minor version '%s': %w", parts[1], err)
		}
	}

	patch := 0
	if len(parts) > 2 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return SemVer{}, fmt.Errorf("invalid semver patch version '%s': %w", parts[2], err)
		}
	}

	return SemVer{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
		Build:      build,
		Raw:        raw,
	}, nil
}

// String returns the canonical string representation of SemVer.
func (s SemVer) String() string {
	if s.Raw == "dev" {
		return "dev"
	}
	res := fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
	if s.Prerelease != "" {
		res += "-" + s.Prerelease
	}
	if s.Build != "" {
		res += "+" + s.Build
	}
	return res
}

// Compare compares this SemVer with other: 1 if s > other, -1 if s < other, 0 if equal.
func (s SemVer) Compare(other SemVer) int {
	if s.Raw == "dev" && other.Raw == "dev" {
		return 0
	}
	if s.Raw == "dev" {
		return -1
	}
	if other.Raw == "dev" {
		return 1
	}

	if s.Major != other.Major {
		if s.Major > other.Major {
			return 1
		}
		return -1
	}
	if s.Minor != other.Minor {
		if s.Minor > other.Minor {
			return 1
		}
		return -1
	}
	if s.Patch != other.Patch {
		if s.Patch > other.Patch {
			return 1
		}
		return -1
	}

	// Release > Pre-release
	if s.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if s.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if s.Prerelease != other.Prerelease {
		if s.Prerelease > other.Prerelease {
			return 1
		}
		return -1
	}

	return 0
}

// AreVersionsCompatible checks whether two versions are mutually backwards-compatible according to SemVer rules:
// - Versions with the same non-zero major version are compatible (e.g. 1.0.0 and 1.1.0).
// - For 0.x.x, versions with the same minor version are compatible (e.g. 0.2.1 and 0.2.4).
// - "dev" is only compatible with "dev".
func AreVersionsCompatible(v1, v2 string) bool {
	clean1 := CleanConstraintPrefix(v1)
	clean2 := CleanConstraintPrefix(v2)

	if clean1 == "dev" || clean2 == "dev" {
		return clean1 == clean2
	}

	s1, err1 := ParseSemVer(clean1)
	s2, err2 := ParseSemVer(clean2)
	if err1 != nil || err2 != nil {
		return clean1 == clean2
	}

	if s1.Major == 0 && s2.Major == 0 {
		return s1.Minor == s2.Minor
	}

	return s1.Major == s2.Major
}

// CleanConstraintPrefix strips constraint operators (^, ~, >=, <=, =, v) from a version string.
func CleanConstraintPrefix(v string) string {
	clean := strings.TrimSpace(v)
	clean = strings.TrimPrefix(clean, "^")
	clean = strings.TrimPrefix(clean, "~")
	clean = strings.TrimPrefix(clean, ">=")
	clean = strings.TrimPrefix(clean, "<=")
	clean = strings.TrimPrefix(clean, "=")
	clean = strings.TrimPrefix(clean, ">")
	clean = strings.TrimPrefix(clean, "<")
	clean = strings.TrimPrefix(clean, "v")
	return strings.TrimSpace(clean)
}

// ResolveHighestCompatible takes a list of requested version strings for the same package.
// If all versions are mutually compatible according to SemVer, it returns the highest version.
// If any pair is incompatible (e.g. major version conflict 1.0.0 vs 2.0.0), it returns an error.
func ResolveHighestCompatible(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions provided")
	}
	if len(versions) == 1 {
		return CleanConstraintPrefix(versions[0]), nil
	}

	baseClean := CleanConstraintPrefix(versions[0])
	highest := baseClean

	for i := 1; i < len(versions); i++ {
		candClean := CleanConstraintPrefix(versions[i])
		if !AreVersionsCompatible(highest, candClean) {
			return "", fmt.Errorf("incompatible versions: '%s' and '%s' have conflicting major versions", highest, candClean)
		}
		if CompareVersions(candClean, highest) > 0 {
			highest = candClean
		}
	}

	return highest, nil
}

// DiamondResolution represents the outcome of diamond dependency conflict resolution.
type DiamondResolution struct {
	PackageName     string   `json:"package_name"`
	Requested       []string `json:"requested"`
	ResolvedVersion string   `json:"resolved_version,omitempty"`
	SideBySide      []string `json:"side_by_side,omitempty"`
	IsCompatible    bool     `json:"is_compatible"`
}

// ResolveDiamondConflict resolves a diamond dependency where multiple modules require pkgName at various versions.
// If compatible (e.g., 1.0.0 and 1.1.0), it resolves to the highest compatible SemVer.
// If incompatible (e.g., 1.0.0 and 2.0.0), it determines side-by-side versions to install in the registry.
func ResolveDiamondConflict(pkgName string, requested []string) (*DiamondResolution, error) {
	res := &DiamondResolution{
		PackageName: pkgName,
		Requested:   requested,
	}

	if len(requested) == 0 {
		return res, fmt.Errorf("no versions requested for '%s'", pkgName)
	}

	// Try resolving as highest compatible
	highest, err := ResolveHighestCompatible(requested)
	if err == nil {
		res.ResolvedVersion = highest
		res.IsCompatible = true
		return res, nil
	}

	// Incompatible major versions -> Partition into side-by-side version groups
	res.IsCompatible = false
	groups := make(map[int][]string)

	for _, req := range requested {
		clean := CleanConstraintPrefix(req)
		s, err := ParseSemVer(clean)
		if err != nil {
			groups[-1] = append(groups[-1], clean)
			continue
		}
		groups[s.Major] = append(groups[s.Major], clean)
	}

	for _, vers := range groups {
		h, err := ResolveHighestCompatible(vers)
		if err == nil {
			res.SideBySide = append(res.SideBySide, h)
		} else {
			res.SideBySide = append(res.SideBySide, vers...)
		}
	}

	res.SideBySide = SortVersions(uniqueStrings(res.SideBySide))

	return res, nil
}

func uniqueStrings(s []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
