package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"testing"
)

func runDatetimeSource(t *testing.T, input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	c := compiler.New()
	err := c.Compile(prog)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}
	v := New(c.Bytecode())
	err = v.Run()
	if err != nil {
		t.Fatalf("vm error: %v", err)
	}
	return v.LastPoppedStackElem()
}

func TestDatetimeBuiltins(t *testing.T) {
	input := `
	let now = datetime_now();
	let is_leap = datetime_is_leap_year(2024);
	let not_leap = datetime_is_leap_year(2023);
	let parsed = datetime_parse("2026-11-09 12:30:00", "YYYY-MM-DD HH:mm:ss");
	let y = parsed["year"];
	let m = parsed["month"];
	let d = parsed["day"];
	let formatted = datetime_format(parsed["unix_ms"], "YYYY/MM/DD");
	let added = datetime_add(parsed["unix_ms"], 2, "days");
	let diff_days = datetime_diff(added, parsed["unix_ms"], "d");
	let feb_days = datetime_days_in_month(2024, 2);

	[is_leap, not_leap, y, m, d, formatted, diff_days, feb_days]
	`
	res := runDatetimeSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}

	if arr.Elements[0] != True {
		t.Errorf("2024 should be leap year, got %v", arr.Elements[0])
	}
	if arr.Elements[1] != False {
		t.Errorf("2023 should not be leap year, got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.Integer).Value != 2026 {
		t.Errorf("expected year 2026, got %v", arr.Elements[2])
	}
	if arr.Elements[3].(*object.Integer).Value != 11 {
		t.Errorf("expected month 11, got %v", arr.Elements[3])
	}
	if arr.Elements[4].(*object.Integer).Value != 9 {
		t.Errorf("expected day 9, got %v", arr.Elements[4])
	}
	if arr.Elements[5].(*object.String).Value != "2026/11/09" {
		t.Errorf("expected formatted date '2026/11/09', got %v", arr.Elements[5])
	}
	if arr.Elements[6].(*object.Float).Value != 2.0 {
		t.Errorf("expected diff_days 2.0, got %v", arr.Elements[6])
	}
	if arr.Elements[7].(*object.Integer).Value != 29 {
		t.Errorf("expected feb 2024 days 29, got %v", arr.Elements[7])
	}
}

func TestDatetimeStdModule(t *testing.T) {
	input := `
	import "std/datetime" as dt;

	let d = dt.Date(2026, 11, 9);
	let d2 = d.add_days(5);
	let is_lp = dt.is_leap_year(2024);
	let dur = dt.duration(3600000);

	[d["year"], d2["day"], is_lp, dur["total_hours"]]
	`
	res := runDatetimeSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.Integer).Value != 2026 {
		t.Errorf("expected year 2026, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.Integer).Value != 14 {
		t.Errorf("expected day 14, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected leap year true, got %v", arr.Elements[2])
	}
	if arr.Elements[3].(*object.Integer).Value != 1 && arr.Elements[3].(*object.Float).Value != 1.0 {
		t.Errorf("expected 1 total_hour, got %v", arr.Elements[3])
	}
}
