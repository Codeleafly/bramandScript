package vm

import (
	"flag"
	"fmt"
	"sync"
)

type PermissionType string

const (
	PermRun   PermissionType = "run"   // exec
	PermWrite PermissionType = "write" // write_file, append_file, delete_file
	PermNet   PermissionType = "net"   // tcp_listen, web sockets
	PermRead  PermissionType = "read"  // read_file, list_dir
)

type Permissions struct {
	mu         sync.RWMutex
	AllowAll   bool
	AllowRun   bool
	AllowWrite bool
	AllowNet   bool
	AllowRead  bool
}

var globalPermissions = &Permissions{
	AllowRead: true, // Read allowed by default unless restricted
}

func GetPermissions() *Permissions {
	return globalPermissions
}

func SetPermissions(p *Permissions) {
	if p != nil {
		globalPermissions = p
	}
}

func ResetPermissions() {
	globalPermissions = &Permissions{
		AllowRead: true,
	}
}

func isTestMode() bool {
	if IsTestRunnerActive() {
		return true
	}
	return flag.Lookup("test.v") != nil
}

func CheckPermission(perm PermissionType) error {
	globalPermissions.mu.RLock()
	defer globalPermissions.mu.RUnlock()

	// In test runner mode or go test, all permissions are granted
	if isTestMode() {
		return nil
	}

	if globalPermissions.AllowAll {
		return nil
	}

	switch perm {
	case PermRun:
		if !globalPermissions.AllowRun {
			return fmt.Errorf("PermissionDenied: execution of external processes requires '--allow-run' or '--allow-all'")
		}
	case PermWrite:
		if !globalPermissions.AllowWrite {
			return fmt.Errorf("PermissionDenied: file system write access requires '--allow-write' or '--allow-all'")
		}
	case PermNet:
		if !globalPermissions.AllowNet {
			return fmt.Errorf("PermissionDenied: network listen access requires '--allow-net' or '--allow-all'")
		}
	case PermRead:
		if !globalPermissions.AllowRead {
			return fmt.Errorf("PermissionDenied: file system read access requires '--allow-read' or '--allow-all'")
		}
	}
	return nil
}
