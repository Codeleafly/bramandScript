// Package go_std provides the Go host implementation of BramandScript standard library builtins.
package go_std

import (
	"bramandscript/object"
)

// ModuleInfo describes a standard library component implemented in Go.
type ModuleInfo struct {
	Name        string
	Description string
	Builtins    map[string]*object.Builtin
}

var modules = make(map[string]*ModuleInfo)

// RegisterModule registers a Go stdlib domain module.
func RegisterModule(name, desc string, builtins map[string]*object.Builtin) {
	modules[name] = &ModuleInfo{
		Name:        name,
		Description: desc,
		Builtins:    builtins,
	}
}

// GetModule returns a Go stdlib module by name.
func GetModule(name string) (*ModuleInfo, bool) {
	m, ok := modules[name]
	return m, ok
}

// AllModules returns all registered Go stdlib modules.
func AllModules() map[string]*ModuleInfo {
	res := make(map[string]*ModuleInfo, len(modules))
	for k, v := range modules {
		res[k] = v
	}
	return res
}
