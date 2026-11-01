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
}
