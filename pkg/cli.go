package pkg

import (
	"fmt"
	"os"
	"path/filepath"
)

// InstallCLI handles the `bmd install <target>` CLI command.
func InstallCLI(args []string) int {
	offline := os.Getenv("BMD_OFFLINE") == "1" || os.Getenv("BMD_OFFLINE") == "true"
	var positional []string
	for _, a := range args {
		if a == "--offline" || a == "-offline" {
			offline = true
		} else {
			positional = append(positional, a)
		}
	}

	if len(positional) < 1 {
		fmt.Println("Usage: bmd install [--offline] <target> [version]")
		fmt.Println("       bmd install github.com/user/project[@version]")
		fmt.Println("       bmd install https://cdn.bmd.io/package.tar.gz")
		fmt.Println("       bmd install ./local_dir")
		fmt.Println("       bmd install local/<name>@dev")
		return 1
	}

	target := positional[0]
	explicitVer := ""
	if len(positional) > 1 {
		explicitVer = positional[1]
	}

	inst := NewInstaller()
	inst.Offline = offline
	res, err := inst.Install(target, explicitVer)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Installation failed: %s\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("✦ BramandScript Package Installer")
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("Target:       %s\n", res.Target)
	fmt.Printf("Package:      %s (v%s)\n", res.PackageName, res.Version)
	fmt.Printf("Registry:     %s\n", res.RegistryDir)
	if res.TarballPath != "" {
		fmt.Printf("Tarball:      %s\n", res.TarballPath)
	}
	if res.FromCache {
		fmt.Println("Source:       Centralized Cache (0 bytes downloaded)")
	} else {
		fmt.Println("Source:       Downloaded and extracted")
	}
	if res.SecurityNotice != "" {
		fmt.Println("──────────────────────────────────────────────────")
		fmt.Println(res.SecurityNotice)
	}
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("[OK] Successfully installed '%s@%s'\n\n", res.PackageName, res.Version)
	return 0
}

// GetCLI handles the `bmd get` CLI command.
func GetCLI(args []string) int {
	offline := os.Getenv("BMD_OFFLINE") == "1" || os.Getenv("BMD_OFFLINE") == "true"
	targetDir := "."
	for _, a := range args {
		if a == "--offline" || a == "-offline" {
			offline = true
		} else if targetDir == "." {
			targetDir = a
		}
	}

	if offline {
		_ = os.Setenv("BMD_OFFLINE", "1")
	}

	res, err := Get(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] bmd get failed: %s\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("✦ BramandScript Dependency Manager (bmd get)")
	fmt.Printf("Manifest: %s\n", res.ManifestPath)
	fmt.Println("──────────────────────────────────────────────────")

	if len(res.Dependencies) == 0 {
		fmt.Println("  (no dependencies declared in manifest)")
	} else {
		for _, dep := range res.Dependencies {
			switch dep.Status {
			case "CACHED":
				fmt.Printf("  ❯ %s@%s [CACHED] (0 bytes downloaded)\n", dep.Name, dep.Version)
			case "FETCHED":
				fmt.Printf("  ❯ %s@%s [FETCHED] (installed to registry)\n", dep.Name, dep.Version)
			case "ERROR":
				fmt.Printf("  ❯ %s@%s [ERROR] %s\n", dep.Name, dep.Version, dep.Error)
			}
			if dep.SecurityNotice != "" {
				fmt.Printf("    %s\n", dep.SecurityNotice)
			}
		}
	}

	fmt.Println("──────────────────────────────────────────────────")
	if res.FailedCount > 0 {
		fmt.Fprintf(os.Stderr, "[ERROR] Dependency resolution incomplete: %d cached, %d fetched, %d failed\n\n",
			res.CachedCount, res.FetchedCount, res.FailedCount)
		return 1
	}

	fmt.Printf("[OK] Dependencies up to date: %d cached, %d fetched (0 errors)\n\n",
		res.CachedCount, res.FetchedCount)
	return 0
}

// CacheInfoCLI handles the `bmd cache info` CLI command.
func CacheInfoCLI(args []string) int {
	info, err := CacheInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to query cache info: %s\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("✦ BramandScript Centralized Cache & Registry Overview")
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("Location:            %s\n", info.BaseDir)
	fmt.Printf("Total Disk Usage:    %s (%d bytes)\n", formatBytes(info.TotalSize), info.TotalSize)
	fmt.Printf("Cached Tarballs:     %d archive(s) (%s)\n", info.TarballCount, formatBytes(info.TarballSize))
	fmt.Printf("Registry Packages:   %d package(s) (%s)\n", info.RegistryCount, formatBytes(info.RegistrySize))
	fmt.Printf("Integrity Records:   %d record(s)\n", info.IntegrityRecords)
	fmt.Printf("Integrity DB:        %s\n", info.IntegrityDBPath)
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Println("[OK] Cache status healthy")
	fmt.Println()
	return 0
}

// CachePruneCLI handles the `bmd cache prune` CLI command.
func CachePruneCLI(args []string) int {
	pruneRes, err := CachePrune()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Cache prune failed: %s\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("✦ BramandScript Centralized Cache Prune")
	fmt.Println("──────────────────────────────────────────────────")
	if len(pruneRes.RemovedTarballs) > 0 {
		for _, name := range pruneRes.RemovedTarballs {
			fmt.Printf("  ❯ Removed orphaned archive: %s\n", name)
		}
	} else {
		fmt.Println("  (no orphaned archives found)")
	}
	fmt.Printf("Space Reclaimed:     %s (%d bytes)\n", formatBytes(pruneRes.FreedBytes), pruneRes.FreedBytes)
	fmt.Printf("Preserved Packages:  %d active registry package(s)\n", pruneRes.PreservedPackages)
	fmt.Println("──────────────────────────────────────────────────")
	fmt.Println("[OK] Cache pruned successfully")
	fmt.Println()
	return 0
}

// CacheVerifyCLI handles the `bmd cache verify` CLI command.
func CacheVerifyCLI(args []string) int {
	verifyRes, err := CacheVerify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Cache verify failed: %s\n", err)
		return 1
	}

	fmt.Println()
	fmt.Println("✦ BramandScript Package Integrity Verification")
	fmt.Println("──────────────────────────────────────────────────")
	if len(verifyRes.Items) == 0 {
		fmt.Println("  (no integrity records registered)")
	} else {
		for _, item := range verifyRes.Items {
			switch item.Status {
			case "OK":
				shortHash := item.Checksum
				if len(shortHash) > 16 {
					shortHash = shortHash[:16] + "..."
				}
				fmt.Printf("  [OK] %s (sha256: %s)\n", item.Key, shortHash)
			case "MISMATCH":
				fmt.Printf("  [MISMATCH] %s (expected %s, got %s)\n", item.Key, item.Expected, item.Checksum)
			case "MISSING":
				fmt.Printf("  [MISSING] %s (file or directory not found on disk)\n", item.Key)
			default:
				fmt.Printf("  [ERROR] %s: %s\n", item.Key, item.Error)
			}
		}
	}

	fmt.Println("──────────────────────────────────────────────────")
	fmt.Printf("Integrity: %d total, %d passed, %d failed\n",
		verifyRes.TotalCount, verifyRes.ValidCount, verifyRes.FailedCount)

	if verifyRes.FailedCount > 0 {
		fmt.Fprintf(os.Stderr, "[ERROR] Integrity verification failed: %d mismatch(es) detected\n\n", verifyRes.FailedCount)
		return 1
	}

	fmt.Println("[OK] All package checksums verified successfully")
	fmt.Println()
	return 0
}

// CacheCLI routes cache subcommands: info, prune, verify.
func CacheCLI(args []string) int {
	if len(args) == 0 {
		fmt.Println("Usage: bmd cache <command>")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  bmd cache info       Display storage and package metrics")
		fmt.Println("  bmd cache prune      Remove orphaned tarballs while preserving registry trees")
		fmt.Println("  bmd cache verify     Validate SHA-256 checksums against integrity.db")
		return 1
	}

	sub := args[0]
	switch sub {
	case "info":
		return CacheInfoCLI(args[1:])
	case "prune":
		return CachePruneCLI(args[1:])
	case "verify":
		return CacheVerifyCLI(args[1:])
	default:
		fmt.Printf("Unknown cache command: %s\n", sub)
		fmt.Println("Available commands: info, prune, verify")
		return 1
	}
}

// PkgListCLI lists dependencies declared in local manifest and installed packages in centralized registry.
func PkgListCLI(args []string) int {
	// 1. Check local manifest
	manifestPath := "bmdpkg.json"
	if data, err := os.ReadFile(manifestPath); err == nil {
		fmt.Println("✦ Local Project Dependencies (bmdpkg.json)")
		if m, err := ParseManifest(data); err == nil {
			fmt.Printf("Package: %s (v%s)\n", m.Name, m.Version)
			if len(m.Dependencies) > 0 {
				for dep, ver := range m.Dependencies {
					fmt.Printf("  ❯ %s: %s\n", dep, ver)
				}
			} else {
				fmt.Println("  (no dependencies declared)")
			}
		}
		fmt.Println("──────────────────────────────────────────────────")
	}

	// 2. Centralized Registry
	libsDir := GetRegistryDir()
	fmt.Println("✦ BramandScript Centralized Package Registry")
	fmt.Printf("Location: %s\n", libsDir)
	fmt.Println("──────────────────────────────────────────────────")

	count := 0
	_ = filepath.Walk(libsDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() && path != libsDir {
			if isDirWithEntrypoint(path) {
				rel, _ := filepath.Rel(libsDir, path)
				fmt.Printf("  ❯ %s\n", rel)
				count++
			}
		}
		return nil
	})

	if count == 0 {
		fmt.Println("  (no packages currently installed)")
	}
	fmt.Println("──────────────────────────────────────────────────")
	return 0
}

// PkgInitCLI initializes a new bmdpkg.json and entrypoint file.
func PkgInitCLI(args []string) int {
	pkgName := "my-package"
	targetDir := "."

	if len(args) >= 1 && args[0] != "" && args[0] != "." {
		pkgName = args[0]
		targetDir = pkgName
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Could not create directory '%s': %s\n", targetDir, err)
			return 1
		}
	} else {
		if cwd, err := os.Getwd(); err == nil {
			base := filepath.Base(cwd)
			if base != "" && base != "/" && base != "." {
				pkgName = base
			}
		}
	}

	manifestPath := filepath.Join(targetDir, "bmdpkg.json")
	if !fileExists(manifestPath) {
		m := Manifest{
			Name:        pkgName,
			Version:     "0.1.0",
			Description: "BramandScript package",
			Entry:       "main.bmd",
			License:     "MIT",
			Permissions: Permissions{
				Network:         false,
				FilesystemRead:  false,
				FilesystemWrite: false,
				Subprocess:      false,
			},
			Dependencies: make(map[string]string),
		}
		if err := m.Save(manifestPath); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed writing bmdpkg.json: %s\n", err)
			return 1
		}
	}

	mainFile := filepath.Join(targetDir, "main.bmd")
	if !fileExists(mainFile) {
		mainContent := fmt.Sprintf("// BramandScript Package: %s\n\nfn info() {\n    return \"Package %s (v0.1.0)\";\n}\n", pkgName, pkgName)
		if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed writing main.bmd: %s\n", err)
			return 1
		}
	}

	fmt.Printf("[OK] Initialized package '%s' (bmdpkg.json, main.bmd)\n", pkgName)
	return 0
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
