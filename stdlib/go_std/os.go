package go_std

import (
	"bramandscript/object"
	"bramandscript/vm"
)

func init() {
	RegisterModule("os", "Low-level OS, Process spawning, IPC streaming and system metrics", map[string]*object.Builtin{
		"proc_spawn":           vm.GetBuiltin("proc_spawn"),
		"proc_stdin_write":     vm.GetBuiltin("proc_stdin_write"),
		"proc_stdin_close":     vm.GetBuiltin("proc_stdin_close"),
		"proc_stdout_read":     vm.GetBuiltin("proc_stdout_read"),
		"proc_stdout_read_all": vm.GetBuiltin("proc_stdout_read_all"),
		"proc_stderr_read":     vm.GetBuiltin("proc_stderr_read"),
		"proc_stderr_read_all": vm.GetBuiltin("proc_stderr_read_all"),
		"proc_wait":            vm.GetBuiltin("proc_wait"),
		"proc_kill":            vm.GetBuiltin("proc_kill"),
		"proc_pid":             vm.GetBuiltin("proc_pid"),
		"os_platform":          vm.GetBuiltin("os_platform"),
		"os_arch":              vm.GetBuiltin("os_arch"),
		"os_cpus":              vm.GetBuiltin("os_cpus"),
		"os_memory_info":       vm.GetBuiltin("os_memory_info"),
		"os_env_all":           vm.GetBuiltin("os_env_all"),
		"os_get_env":           vm.GetBuiltin("os_get_env"),
		"os_set_env":           vm.GetBuiltin("os_set_env"),
		"os_cwd":               vm.GetBuiltin("os_cwd"),
		"os_pid":               vm.GetBuiltin("os_pid"),
	})
}
