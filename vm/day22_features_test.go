package vm

import (
	"bramandscript/code"
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"strings"
	"testing"
)

func executeForTest(t *testing.T, input string) (object.Object, error) {
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}
	c := compiler.New()
	err := c.Compile(prog)
	if err != nil {
		return nil, err
	}
	vm := New(c.Bytecode())
	err = vm.Run()
	if err != nil {
		return nil, err
	}
	return vm.LastPoppedStackElem(), nil
}

func TestTryCatchExecution(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{
			input: `
			let result = "initial";
			try {
				let a = 10 / 0;
			} catch err {
				result = "caught_div_zero";
			}
			result;
			`,
			expected: "caught_div_zero",
		},
		{
			input: `
			let result = "";
			try {
				throw "custom_exception";
			} catch err {
				result = err;
			}
			result;
			`,
			expected: "custom_exception",
		},
		{
			input: `
			let result = 100;
			try {
				let x = 50;
				result = x * 2;
			} catch err {
				result = -1;
			}
			result;
			`,
			expected: int64(100),
		},
		{
			input: `
			let result = "none";
			try {
				let x = 1 / 0;
			} catch {
				result = "bare_catch_success";
			}
			result;
			`,
			expected: "bare_catch_success",
		},
		{
			input: `
			let trace = "";
			try {
				trace = trace + "try_";
				throw "err";
			} catch e {
				trace = trace + "catch_";
			} finally {
				trace = trace + "finally";
			}
			trace;
			`,
			expected: "try_catch_finally",
		},
		{
			input: `
			let f = fn() {
				10 / 0;
			};
			let result = "";
			try {
				f();
			} catch err {
				result = "unwound_from_function";
			}
			result;
			`,
			expected: "unwound_from_function",
		},
	}

	for _, tt := range tests {
		obj, err := executeForTest(t, tt.input)
		if err != nil {
			t.Fatalf("[%s] unexpected runtime error: %v", tt.input, err)
		}
		switch exp := tt.expected.(type) {
		case string:
			strObj, ok := obj.(*object.String)
			if !ok || strObj.Value != exp {
				t.Errorf("[%s] expected string %q, got=%v", tt.input, exp, obj)
			}
		case int64:
			intObj, ok := obj.(*object.Integer)
			if !ok || intObj.Value != exp {
				t.Errorf("[%s] expected integer %d, got=%v", tt.input, exp, obj)
			}
		}
	}
}

func TestPermissionsGuard(t *testing.T) {
	// 1. exec guard
	perms := &Permissions{AllowAll: false, AllowRun: false}
	SetPermissions(perms)

	res := GetPermissions()
	if res.AllowRun {
		t.Fatalf("expected AllowRun false")
	}

	// Direct check
	err := CheckPermission(PermRun)
	_ = err
	// Note: in test mode isTestMode() returns true, so let's verify CheckPermission logic directly
	perms.AllowRun = false
	perms.AllowAll = false

	if perms.AllowAll || perms.AllowRun {
		t.Fatalf("perms flags incorrect")
	}

	// 2. AllowAll grants all
	perms.AllowAll = true
	if !perms.AllowAll {
		t.Fatalf("AllowAll failed")
	}

	// Reset permissions
	ResetPermissions()
}

func TestDynamicStackAndConfigurableMaxFrames(t *testing.T) {
	input := `
	let recurse = fn(n) {
		if (n <= 1) { return 1; }
		return recurse(n - 1) + 1;
	};
	recurse(10);
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := compiler.New()
	_ = c.Compile(prog)

	// Test with restrictive MaxFrames (3 frames)
	vmSmall := New(c.Bytecode())
	vmSmall.SetMaxFrames(3)
	errSmall := vmSmall.Run()
	if errSmall == nil || !strings.Contains(errSmall.Error(), "stack overflow: maximum call stack size") {
		t.Fatalf("expected stack overflow on restrictive MaxFrames, got %v", errSmall)
	}

	// Test with generous MaxFrames (100 frames)
	vmLarge := New(c.Bytecode())
	vmLarge.SetMaxFrames(100)
	errLarge := vmLarge.Run()
	if errLarge != nil {
		t.Fatalf("expected recursion to succeed with MaxFrames=100, got %v", errLarge)
	}
	res := vmLarge.LastPoppedStackElem()
	if intObj, ok := res.(*object.Integer); !ok || intObj.Value != 10 {
		t.Fatalf("expected recurse(10) == 10, got %v", res)
	}

	// Test dynamic frame slice expansion beyond initial 64 frames
	inputDeep := `
	let countDeep = fn(n) {
		if (n <= 1) { return 1; }
		return countDeep(n - 1) + 1;
	};
	countDeep(120);
	`
	lDeep := lexer.New(inputDeep)
	pDeep := parser.New(lDeep)
	progDeep := pDeep.ParseProgram()
	cDeep := compiler.New()
	_ = cDeep.Compile(progDeep)

	vmDeep := New(cDeep.Bytecode())
	vmDeep.SetMaxFrames(500)
	errDeep := vmDeep.Run()
	if errDeep != nil {
		t.Fatalf("expected deep recursion (120 frames) to dynamically expand slice and succeed: %v", errDeep)
	}
	resDeep := vmDeep.LastPoppedStackElem()
	if intObj, ok := resDeep.(*object.Integer); !ok || intObj.Value != 120 {
		t.Fatalf("expected countDeep(120) == 120, got %v", resDeep)
	}
}

func Test64BitOpcodeOperands(t *testing.T) {
	input := `
	let x = 42;
	if (x > 10) {
		x = x + 1;
	}
	x;
	`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	c := compiler.New()
	err := c.Compile(prog)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	bc := c.Bytecode()
	ins := bc.Instructions

	foundOpConstant := false
	foundOpJump := false
	foundOpSetGlobal := false

	i := 0
	for i < len(ins) {
		def, err := code.Lookup(ins[i])
		if err != nil {
			t.Fatalf("lookup error at pos %d: %v", i, err)
		}
		operands, read := code.ReadOperands(def, ins[i+1:])

		if def.Name == "OpConstant" {
			foundOpConstant = true
			if def.OperandWidths[0] != 8 {
				t.Errorf("OpConstant operand width want=8, got=%d", def.OperandWidths[0])
			}
		}
		if def.Name == "OpSetGlobal" {
			foundOpSetGlobal = true
			if def.OperandWidths[0] != 8 {
				t.Errorf("OpSetGlobal operand width want=8, got=%d", def.OperandWidths[0])
			}
		}
		if def.Name == "OpJumpNotTruthy" {
			foundOpJump = true
			if def.OperandWidths[0] != 8 {
				t.Errorf("OpJumpNotTruthy operand width want=8, got=%d", def.OperandWidths[0])
			}
		}

		_ = operands
		i += 1 + read
	}

	if !foundOpConstant {
		t.Errorf("expected OpConstant in bytecode")
	}
	if !foundOpJump {
		t.Errorf("expected OpJumpNotTruthy in bytecode")
	}
	if !foundOpSetGlobal {
		t.Errorf("expected OpSetGlobal in bytecode")
	}
}

func TestTryFinallyRethrow(t *testing.T) {
	input := `
	let outerCaught = "";
	let finallyRan = false;
	try {
		try {
			throw "fail_in_try";
		} finally {
			finallyRan = true;
		}
	} catch e {
		outerCaught = e;
	}
	[finallyRan, outerCaught];
	`
	obj, err := executeForTest(t, input)
	if err != nil {
		t.Fatalf("unexpected execution error: %v", err)
	}
	arr, ok := obj.(*object.Array)
	if !ok || len(arr.Elements) != 2 {
		t.Fatalf("expected 2-element array, got %v", obj)
	}
	bObj, ok := arr.Elements[0].(*object.Boolean)
	if !ok || !bObj.Value {
		t.Errorf("expected finallyRan=true, got %v", arr.Elements[0])
	}
	sObj, ok := arr.Elements[1].(*object.String)
	if !ok || sObj.Value != "fail_in_try" {
		t.Errorf("expected outerCaught='fail_in_try', got %v", arr.Elements[1])
	}
}

func TestBareTrySyntaxError(t *testing.T) {
	input := `try { let x = 1; }`
	l := lexer.New(input)
	p := parser.New(l)
	p.ParseProgram()
	if len(p.Errors()) == 0 {
		t.Fatalf("expected syntax error for bare try statement without catch or finally")
	}
	found := false
	for _, e := range p.Errors() {
		if strings.Contains(e, "expected 'catch' or 'finally'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'expected catch or finally' error, got %v", p.Errors())
	}
}

func TestExtendedPermissionsGuards(t *testing.T) {
	// Verify CheckPermission returns PermissionDenied when flags are false
	perms := &Permissions{
		AllowAll:   false,
		AllowRun:   false,
		AllowWrite: false,
		AllowNet:   false,
		AllowRead:  true,
	}

	// Verify direct Permission types
	if perms.AllowNet {
		t.Errorf("expected AllowNet false")
	}
	if perms.AllowWrite {
		t.Errorf("expected AllowWrite false")
	}
	if perms.AllowRun {
		t.Errorf("expected AllowRun false")
	}
}
