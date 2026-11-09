package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"testing"
)

func runMonadSource(t *testing.T, input string) object.Object {
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

func TestResultMethods(t *testing.T) {
	input := `
	let ok_val = Ok(42);
	let err_val = Err("failed to parse");

	let u_ok = ok_val.unwrap();
	let u_err_or = err_val.unwrap_or(99);
	let is_o = ok_val.is_ok();
	let is_e = err_val.is_err();

	let mapped = ok_val.map(fn(x) { return x * 2; });
	let flat_mapped = ok_val.flat_map(fn(x) { return Ok(x + 10); });

	[u_ok, u_err_or, is_o, is_e, mapped.unwrap(), flat_mapped.unwrap()]
	`
	res := runMonadSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.Integer).Value != 42 {
		t.Errorf("expected 42, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.Integer).Value != 99 {
		t.Errorf("expected 99, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected is_ok true, got %v", arr.Elements[2])
	}
	if arr.Elements[3] != True {
		t.Errorf("expected is_err true, got %v", arr.Elements[3])
	}
	if arr.Elements[4].(*object.Integer).Value != 84 {
		t.Errorf("expected mapped 84, got %v", arr.Elements[4])
	}
	if arr.Elements[5].(*object.Integer).Value != 52 {
		t.Errorf("expected flat_mapped 52, got %v", arr.Elements[5])
	}
}

func TestOptionMethods(t *testing.T) {
	input := `
	let s = Some("Bramand");
	let n = None();

	let s_val = s.unwrap();
	let n_val = n.unwrap_or("Default");
	let is_s = s.is_some();
	let is_n = n.is_none();

	let mapped = s.map(fn(x) { return x + "Script"; });

	[s_val, n_val, is_s, is_n, mapped.unwrap()]
	`
	res := runMonadSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0].(*object.String).Value != "Bramand" {
		t.Errorf("expected Bramand, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "Default" {
		t.Errorf("expected Default, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected is_some true, got %v", arr.Elements[2])
	}
	if arr.Elements[3] != True {
		t.Errorf("expected is_none true, got %v", arr.Elements[3])
	}
	if arr.Elements[4].(*object.String).Value != "BramandScript" {
		t.Errorf("expected BramandScript, got %v", arr.Elements[4])
	}
}

func TestTryPropagationOperator(t *testing.T) {
	input := `
	let safe_div = fn(a, b) {
		if b == 0 {
			return Err("division by zero");
		}
		return Ok(a / b);
	};

	let calculate = fn(a, b, c) {
		let step1 = safe_div(a, b)?;
		let step2 = safe_div(step1, c)?;
		return Ok(step2);
	};

	let success = calculate(100, 2, 5);
	let failure = calculate(100, 0, 5);
	let failure2 = calculate(100, 2, 0);

	[success.is_ok(), success.unwrap(), failure.is_err(), failure2.is_err()]
	`
	res := runMonadSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0] != True {
		t.Errorf("expected success is_ok true, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.Integer).Value != 10 {
		t.Errorf("expected calculate(100, 2, 5) == 10, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected failure is_err true, got %v", arr.Elements[2])
	}
	if arr.Elements[3] != True {
		t.Errorf("expected failure2 is_err true, got %v", arr.Elements[3])
	}
}

func TestTryPropagationWithOption(t *testing.T) {
	input := `
	let find_user = fn(id) {
		if id == 1 {
			return Some("Alice");
		}
		return None();
	};

	let get_greeting = fn(id) {
		let user = find_user(id)?;
		return Some("Hello, " + user);
	};

	let g1 = get_greeting(1);
	let g2 = get_greeting(99);

	[g1.is_some(), g1.unwrap(), g2.is_none()]
	`
	res := runMonadSource(t, input)
	arr, ok := res.(*object.Array)
	if !ok {
		t.Fatalf("expected Array, got %s (%+v)", res.Type(), res)
	}
	if arr.Elements[0] != True {
		t.Errorf("expected g1.is_some true, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.String).Value != "Hello, Alice" {
		t.Errorf("expected 'Hello, Alice', got %v", arr.Elements[1])
	}
	if arr.Elements[2] != True {
		t.Errorf("expected g2.is_none true, got %v", arr.Elements[2])
	}
}
