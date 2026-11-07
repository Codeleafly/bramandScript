package vm

import (
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"testing"
)

func runVM(t *testing.T, input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors for %q: %v", input, p.Errors())
	}

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compiler error for %q: %s", input, err)
	}

	vm := New(comp.Bytecode())
	err = vm.Run()
	if err != nil {
		t.Fatalf("vm runtime error for %q: %s", input, err)
	}

	return vm.LastPoppedStackElem()
}

func TestTernaryExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"true ? 10 : 20;", int64(10)},
		{"false ? 10 : 20;", int64(20)},
		{"5 > 3 ? \"yes\" : \"no\";", "yes"},
		{"5 < 3 ? \"yes\" : \"no\";", "no"},
		// Right-associative nested ternary: a ? b : c ? d : e
		{"false ? 1 : true ? 2 : 3;", int64(2)},
		{"false ? 1 : false ? 2 : 3;", int64(3)},
		// Short-circuiting verification: right branch has invalid division by zero, shouldn't execute
		{"true ? 42 : (1 / 0);", int64(42)},
	}

	for _, tt := range tests {
		res := runVM(t, tt.input)
		switch expected := tt.expected.(type) {
		case int64:
			intObj, ok := res.(*object.Integer)
			if !ok || intObj.Value != expected {
				t.Fatalf("for %q: expected %d, got %v", tt.input, expected, res)
			}
		case string:
			strObj, ok := res.(*object.String)
			if !ok || strObj.Value != expected {
				t.Fatalf("for %q: expected %q, got %v", tt.input, expected, res)
			}
		}
	}
}

func TestNullCoalescingOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"null ?? 42;", int64(42)},
		{"100 ?? 42;", int64(100)},
		{"0 ?? 42;", int64(0)},                     // 0 is falsey in BramandScript, but NOT null
		{"false ?? true;", false},                  // false is falsey, but NOT null
		{"\"\" ?? \"default\";", ""},               // empty string is NOT null
		{"null ?? null ?? 999;", int64(999)},
		// Short-circuiting verification
		{"50 ?? (1 / 0);", int64(50)},
	}

	for _, tt := range tests {
		res := runVM(t, tt.input)
		switch expected := tt.expected.(type) {
		case int64:
			intObj, ok := res.(*object.Integer)
			if !ok || intObj.Value != expected {
				t.Fatalf("for %q: expected %d, got %v", tt.input, expected, res)
			}
		case bool:
			boolObj, ok := res.(*object.Boolean)
			if !ok || boolObj.Value != expected {
				t.Fatalf("for %q: expected %v, got %v", tt.input, expected, res)
			}
		case string:
			strObj, ok := res.(*object.String)
			if !ok || strObj.Value != expected {
				t.Fatalf("for %q: expected %q, got %v", tt.input, expected, res)
			}
		}
	}
}

func TestNullishAssignment(t *testing.T) {
	inputVar := `
	let a = null;
	a ??= 100;
	let b = 50;
	b ??= 999;
	let c = 0;
	c ??= 42;
	[a, b, c];
	`
	res := runVM(t, inputVar)
	arr, ok := res.(*object.Array)
	if !ok || len(arr.Elements) != 3 {
		t.Fatalf("expected array of 3 elements, got %v", res)
	}
	if arr.Elements[0].(*object.Integer).Value != 100 {
		t.Fatalf("expected a=100, got %v", arr.Elements[0])
	}
	if arr.Elements[1].(*object.Integer).Value != 50 {
		t.Fatalf("expected b=50, got %v", arr.Elements[1])
	}
	if arr.Elements[2].(*object.Integer).Value != 0 {
		t.Fatalf("expected c=0, got %v", arr.Elements[2])
	}

	inputIndex := `
	let dict = {"port": null, "host": "127.0.0.1"};
	dict["port"] ??= 8080;
	dict["host"] ??= "localhost";
	dict["missing"] ??= "present";
	dict;
	`
	resIndex := runVM(t, inputIndex)
	hash, ok := resIndex.(*object.Hash)
	if !ok {
		t.Fatalf("expected hash, got %v", resIndex)
	}
	portKey := (&object.String{Value: "port"}).HashKey()
	hostKey := (&object.String{Value: "host"}).HashKey()
	missKey := (&object.String{Value: "missing"}).HashKey()

	if hash.Pairs[portKey].Value.(*object.Integer).Value != 8080 {
		t.Fatalf("expected dict.port=8080, got %v", hash.Pairs[portKey].Value)
	}
	if hash.Pairs[hostKey].Value.(*object.String).Value != "127.0.0.1" {
		t.Fatalf("expected dict.host=127.0.0.1, got %v", hash.Pairs[hostKey].Value)
	}
	if hash.Pairs[missKey].Value.(*object.String).Value != "present" {
		t.Fatalf("expected dict.missing=present, got %v", hash.Pairs[missKey].Value)
	}
}

func TestOptionalChaining(t *testing.T) {
	inputProp := `
	let user = {"profile": {"name": "Antigravity", "meta": null}};
	let empty_user = null;

	let n1 = user?.profile?.name;
	let n2 = empty_user?.profile?.name;
	let n3 = user?.profile?.meta?.tags;

	[n1, n2, n3];
	`
	res := runVM(t, inputProp)
	arr := res.(*object.Array)
	if arr.Elements[0].(*object.String).Value != "Antigravity" {
		t.Fatalf("expected 'Antigravity', got %v", arr.Elements[0])
	}
	if arr.Elements[1] != Null {
		t.Fatalf("expected null, got %v", arr.Elements[1])
	}
	if arr.Elements[2] != Null {
		t.Fatalf("expected null, got %v", arr.Elements[2])
	}

	inputIndex := `
	let matrix = [[1, 2], null, [5, 6]];
	let empty_matrix = null;

	let i1 = matrix?.[0];
	let i2 = empty_matrix?.[0];
	let i3 = matrix?.[1]?.[0];

	[i1, i2, i3];
	`
	resIndex := runVM(t, inputIndex)
	arrIndex := resIndex.(*object.Array)
	if len(arrIndex.Elements[0].(*object.Array).Elements) != 2 {
		t.Fatalf("expected [1, 2], got %v", arrIndex.Elements[0])
	}
	if arrIndex.Elements[1] != Null {
		t.Fatalf("expected null, got %v", arrIndex.Elements[1])
	}
	if arrIndex.Elements[2] != Null {
		t.Fatalf("expected null, got %v", arrIndex.Elements[2])
	}

	inputCall := `
	let service = {
		"greet": fn(name) { return "Hello, " + name; },
		"noop": null
	};
	let empty_svc = null;

	let g1 = service?.greet("World");
	let g2 = empty_svc?.greet("World");
	let g3 = service?.noop?.("World");

	[g1, g2, g3];
	`
	resCall := runVM(t, inputCall)
	arrCall := resCall.(*object.Array)
	if arrCall.Elements[0].(*object.String).Value != "Hello, World" {
		t.Fatalf("expected 'Hello, World', got %v", arrCall.Elements[0])
	}
	if arrCall.Elements[1] != Null {
		t.Fatalf("expected null, got %v", arrCall.Elements[1])
	}
	if arrCall.Elements[2] != Null {
		t.Fatalf("expected null, got %v", arrCall.Elements[2])
	}
}

func TestStringInterpolation(t *testing.T) {
	input := `
	let name = "Bramand";
	let version = 1;
	let active = true;

	let s1 = f"Welcome to {name} v{version}!";
	let s2 = f"Active: {active}, Double brace: {{literal}}";
	let s3 = ` + "`" + `Template: ${name} (v${version})` + "`" + `;

	[s1, s2, s3];
	`
	res := runVM(t, input)
	arr := res.(*object.Array)
	if arr.Elements[0].(*object.String).Value != "Welcome to Bramand v1!" {
		t.Fatalf("expected 'Welcome to Bramand v1!', got %q", arr.Elements[0].(*object.String).Value)
	}
	if arr.Elements[1].(*object.String).Value != "Active: true, Double brace: {literal}" {
		t.Fatalf("expected literal braces escape, got %q", arr.Elements[1].(*object.String).Value)
	}
	if arr.Elements[2].(*object.String).Value != "Template: Bramand (v1)" {
		t.Fatalf("expected template literal string, got %q", arr.Elements[2].(*object.String).Value)
	}
}

func TestEnterpriseErrorStackTrace(t *testing.T) {
	input := `
	let level3 = fn() {
		throw error("database connection timeout");
	};
	let level2 = fn() {
		return level3();
	};
	let level1 = fn() {
		return level2();
	};

	let caught = null;
	try {
		level1();
	} catch err {
		caught = err;
	}

	[caught.message, caught.stack_trace, len(caught.frames)];
	`
	res := runVM(t, input)
	arr := res.(*object.Array)
	msg := arr.Elements[0].(*object.String).Value
	if msg != "database connection timeout" {
		t.Fatalf("expected error message 'database connection timeout', got %q", msg)
	}
	stack := arr.Elements[1].(*object.String).Value
	if len(stack) == 0 {
		t.Fatalf("expected non-empty stack trace, got %q", stack)
	}
	numFrames := arr.Elements[2].(*object.Integer).Value
	if numFrames < 3 {
		t.Fatalf("expected at least 3 stack frames, got %d", numFrames)
	}
}
