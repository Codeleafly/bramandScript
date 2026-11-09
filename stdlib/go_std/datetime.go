package go_std

import (
	"bramandscript/object"
	"bramandscript/vm"
)

func init() {
	// Register module in the Go stdlib registry
	RegisterModule("datetime", "Date, Time, Duration & Calendar arithmetic engine", map[string]*object.Builtin{
		"now":             vm.GetBuiltin("datetime_now"),
		"now_iso":         vm.GetBuiltin("datetime_now_iso"),
		"parse":           vm.GetBuiltin("datetime_parse"),
		"format":          vm.GetBuiltin("datetime_format"),
		"parts":           vm.GetBuiltin("datetime_parts"),
		"is_leap_year":    vm.GetBuiltin("datetime_is_leap_year"),
		"days_in_month":   vm.GetBuiltin("datetime_days_in_month"),
		"add":             vm.GetBuiltin("datetime_add"),
		"diff":            vm.GetBuiltin("datetime_diff"),
		"timezone_offset": vm.GetBuiltin("datetime_timezone_offset"),
	})
}
