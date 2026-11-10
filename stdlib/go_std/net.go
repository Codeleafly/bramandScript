package go_std

import (
	"bramandscript/object"
	"bramandscript/vm"
)

func init() {
	RegisterModule("net", "Low-level raw TCP, UDP, and TLS networking engine", map[string]*object.Builtin{
		"listen_tcp":     vm.GetBuiltin("net_listen_tcp"),
		"accept":         vm.GetBuiltin("tcp_accept"),
		"dial_tcp":       vm.GetBuiltin("net_dial_tcp"),
		"read":           vm.GetBuiltin("tcp_read"),
		"read_line":      vm.GetBuiltin("tcp_read_line"),
		"write":          vm.GetBuiltin("tcp_write"),
		"close":          vm.GetBuiltin("tcp_close"),
		"set_timeout":    vm.GetBuiltin("tcp_set_timeout"),
		"remote_addr":    vm.GetBuiltin("tcp_remote_addr"),
		"local_addr":     vm.GetBuiltin("tcp_local_addr"),
		"listen_udp":     vm.GetBuiltin("net_listen_udp"),
		"send_to":        vm.GetBuiltin("udp_send_to"),
		"recv_from":      vm.GetBuiltin("udp_recv_from"),
		"udp_close":      vm.GetBuiltin("udp_close"),
		"udp_timeout":    vm.GetBuiltin("udp_set_timeout"),
		"dial_tls":       vm.GetBuiltin("tls_dial"),
		"listen_tls":     vm.GetBuiltin("tls_listen"),
	})
}
