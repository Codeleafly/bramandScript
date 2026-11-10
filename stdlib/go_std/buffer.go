package go_std

import (
	"bramandscript/object"
	"bramandscript/vm"
)

func init() {
	RegisterModule("buffer", "Low-level byte slice operations and binary codecs", map[string]*object.Builtin{
		"buf_alloc":       vm.GetBuiltin("buf_alloc"),
		"buf_from_string": vm.GetBuiltin("buf_from_string"),
		"buf_from_array":  vm.GetBuiltin("buf_from_array"),
		"buf_to_string":   vm.GetBuiltin("buf_to_string"),
		"buf_copy":        vm.GetBuiltin("buf_copy"),
		"buf_fill":        vm.GetBuiltin("buf_fill"),
		"buf_read_u8":     vm.GetBuiltin("buf_read_u8"),
		"buf_write_u8":    vm.GetBuiltin("buf_write_u8"),
		"buf_read_u16_be": vm.GetBuiltin("buf_read_u16_be"),
		"buf_read_u16_le": vm.GetBuiltin("buf_read_u16_le"),
		"buf_write_u16_be": vm.GetBuiltin("buf_write_u16_be"),
		"buf_write_u16_le": vm.GetBuiltin("buf_write_u16_le"),
		"buf_read_u32_be": vm.GetBuiltin("buf_read_u32_be"),
		"buf_read_u32_le": vm.GetBuiltin("buf_read_u32_le"),
		"buf_write_u32_be": vm.GetBuiltin("buf_write_u32_be"),
		"buf_write_u32_le": vm.GetBuiltin("buf_write_u32_le"),
		"buf_read_i32_be": vm.GetBuiltin("buf_read_i32_be"),
		"buf_read_i32_le": vm.GetBuiltin("buf_read_i32_le"),
		"buf_write_i32_be": vm.GetBuiltin("buf_write_i32_be"),
		"buf_write_i32_le": vm.GetBuiltin("buf_write_i32_le"),
		"buf_read_f32_be": vm.GetBuiltin("buf_read_f32_be"),
		"buf_read_f32_le": vm.GetBuiltin("buf_read_f32_le"),
		"buf_write_f32_be": vm.GetBuiltin("buf_write_f32_be"),
		"buf_write_f32_le": vm.GetBuiltin("buf_write_f32_le"),
		"buf_read_f64_be": vm.GetBuiltin("buf_read_f64_be"),
		"buf_read_f64_le": vm.GetBuiltin("buf_read_f64_le"),
		"buf_write_f64_be": vm.GetBuiltin("buf_write_f64_be"),
		"buf_write_f64_le": vm.GetBuiltin("buf_write_f64_le"),
		"buf_compare":     vm.GetBuiltin("buf_compare"),
		"buf_concat":      vm.GetBuiltin("buf_concat"),
		"buf_index_of":    vm.GetBuiltin("buf_index_of"),
	})
}
