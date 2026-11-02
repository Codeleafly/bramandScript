package compiler

type BuiltinDef struct {
	Name string
}

var Builtins = []BuiltinDef{
	{Name: "len"},
	{Name: "print"},
	{Name: "println"},
	{Name: "print_raw"},
	{Name: "type"},
	{Name: "push"},
	{Name: "now"},
	{Name: "input"},
	{Name: "int"},
	{Name: "float"},
	{Name: "str"},
	{Name: "bool"},
	{Name: "keys"},
	{Name: "values"},
	{Name: "slice"},
	{Name: "pop"},
	{Name: "contains"},
	{Name: "delete"},
	{Name: "exit"},

	// Day 4 Concurrency, Channels, Mutex
	{Name: "chan"},
	{Name: "close"},
	{Name: "mutex"},
	{Name: "wait_group"},
	{Name: "sleep"},

	// Day 4 File I/O
	{Name: "read_file"},
	{Name: "write_file"},
	{Name: "append_file"},
	{Name: "delete_file"},
	{Name: "file_exists"},
	{Name: "list_dir"},
	{Name: "chmod"},
	{Name: "is_dir"},

	// Day 4 Process Execution & Environment
	{Name: "env"},
	{Name: "set_env"},
	{Name: "exec"},

	// Day 4 Standard I/O & Formatting
	{Name: "eprint"},
	{Name: "eprintln"},
	{Name: "color"},
	{Name: "styled"},
	{Name: "read_line"},
}
