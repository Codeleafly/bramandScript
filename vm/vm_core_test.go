package vm

import (
	"fmt"
	"strings"
	"testing"

	"bramandscript/ast"
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
)

type vmTestCase struct {
	input    string
	expected interface{}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)
	return p.ParseProgram()
}

func testExpectedObject(t *testing.T, expected interface{}, actual object.Object) {
	t.Helper()

	switch expected := expected.(type) {
	case int:
		err := testIntegerObject(int64(expected), actual)
		if err != nil {
			t.Errorf("testIntegerObject failed: %s", err)
		}
	case float64:
		err := testFloatObject(expected, actual)
		if err != nil {
			t.Errorf("testFloatObject failed: %s", err)
		}
	case bool:
		err := testBooleanObject(expected, actual)
		if err != nil {
			t.Errorf("testBooleanObject failed: %s", err)
		}
	case string:
		err := testStringObject(expected, actual)
		if err != nil {
			t.Errorf("testStringObject failed: %s", err)
		}
	case []int:
		array, ok := actual.(*object.Array)
		if !ok {
			t.Errorf("object not Array: %T (%+v)", actual, actual)
			return
		}
		if len(array.Elements) != len(expected) {
			t.Errorf("wrong num elements. want=%d, got=%d", len(expected), len(array.Elements))
			return
		}
		for i, expectedElem := range expected {
			err := testIntegerObject(int64(expectedElem), array.Elements[i])
			if err != nil {
				t.Errorf("element %d failed: %s", i, err)
			}
		}
	case *object.Null:
		if actual != Null {
			t.Errorf("object is not Null: %T (%+v)", actual, actual)
		}
	}
}

func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)
	if !ok {
		return fmt.Errorf("object is not Integer. got=%T (%+v)", actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
	}
	return nil
}

func testFloatObject(expected float64, actual object.Object) error {
	result, ok := actual.(*object.Float)
	if !ok {
		return fmt.Errorf("object is not Float. got=%T (%+v)", actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%g, want=%g", result.Value, expected)
	}
	return nil
}

func testBooleanObject(expected bool, actual object.Object) error {
	result, ok := actual.(*object.Boolean)
	if !ok {
		return fmt.Errorf("object is not Boolean. got=%T (%+v)", actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%t, want=%t", result.Value, expected)
	}
	return nil
}

func testStringObject(expected string, actual object.Object) error {
	result, ok := actual.(*object.String)
	if !ok {
		return fmt.Errorf("object is not String. got=%T (%+v)", actual, actual)
	}
	if result.Value != expected {
		return fmt.Errorf("object has wrong value. got=%q, want=%q", result.Value, expected)
	}
	return nil
}

func runVmTests(t *testing.T, tests []vmTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("compiler error: %s", err)
		}

		vm := New(comp.Bytecode())
		err = vm.Run()
		if err != nil {
			t.Fatalf("vm error: %s", err)
		}

		stackElem := vm.LastPoppedStackElem()
		testExpectedObject(t, tt.expected, stackElem)
	}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"1", 1},
		{"2", 2},
		{"1 + 2", 3},
		{"1 - 2", -1},
		{"2 * 3", 6},
		{"6 / 2", 3},
		{"10 % 3", 1},
		{"2 ** 4", 16},
		{"50 / 2 * 2 + 10 - 5", 55},
		{"-5", -5},
		{"-10 + 20 - 5", 5},
	}
	runVmTests(t, tests)
}

func TestFloatArithmetic(t *testing.T) {
	tests := []vmTestCase{
		{"3.5 + 2.5", 6.0},
		{"10.0 - 4.5", 5.5},
		{"2.0 * 3.5", 7.0},
		{"7.0 / 2.0", 3.5},
		{"1.5e3", 1500.0},
		{"1e-3", 0.001},
		{"2.5e+2", 250.0},
	}
	runVmTests(t, tests)
}

func TestBooleanExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 >= 1", true},
		{"1 == 1", true},
		{"1 != 1", false},
		{"true == true", true},
		{"false == false", true},
		{"true != false", true},
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"true && true", true},
		{"true && false", false},
		{"false || true", true},
		{`null || "fallback"`, "fallback"},
		{`"hello" || "fallback"`, "hello"},
		{`42 && "yes"`, "yes"},
		{`null && "never"`, Null},
		{`"a" < "b"`, true},
		{`"apple" > "banana"`, false},
		{`"hello" <= "hello"`, true},
		{`"world" >= "hello"`, true},
	}
	runVmTests(t, tests)
}

func TestConditionals(t *testing.T) {
	tests := []vmTestCase{
		{"if (true) { 10 }", 10},
		{"if (true) { 10 } else { 20 }", 10},
		{"if (false) { 10 } else { 20 }", 20},
		{"if (1 < 2) { 10 }", 10},
		{"if (1 > 2) { 10 } else { 20 }", 20},
		{"if true { 99 } else { 0 }", 99},
		{"if 5 > 2 { 42 } else { 0 }", 42},
	}
	runVmTests(t, tests)
}

func TestElseIfConditionals(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let x = 2
			if x == 1 {
				10
			} else if x == 2 {
				20
			} else {
				30
			}
			`,
			expected: 20,
		},
		{
			input: `
			let x = 99
			if x == 1 {
				10
			} else if x == 2 {
				20
			} else {
				30
			}
			`,
			expected: 30,
		},
	}
	runVmTests(t, tests)
}

func TestCleanBeginnerFriendlySyntax(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let a = 10
			let b = 20
			a + b
			`,
			expected: 30,
		},
		{
			input: `
			let age = 14
			if age >= 14 {
				100
			} else {
				50
			}
			`,
			expected: 100,
		},
		{
			input: `
			fn add(x, y) {
				return x + y
			}
			add(7, 8)
			`,
			expected: 15,
		},
	}
	runVmTests(t, tests)
}

func TestGlobalLetStatements(t *testing.T) {
	tests := []vmTestCase{
		{"let one = 1; one", 1},
		{"let one = 1; let two = 2; one + two", 3},
		{"let one = 1; let two = one + one; two", 2},
	}
	runVmTests(t, tests)
}

func TestStringExpressions(t *testing.T) {
	tests := []vmTestCase{
		{`"brahmand"`, "brahmand"},
		{`"space" + " " + "travel"`, "space travel"},
	}
	runVmTests(t, tests)
}

func TestBitwiseOperators(t *testing.T) {
	tests := []vmTestCase{
		{"5 & 3", 1},
		{"5 | 3", 7},
		{"5 ^ 3", 6},
		{"1 << 4", 16},
		{"16 >> 2", 4},
		{"~0", -1},
		{"~1", -2},
		{"(255 & 15)", 15},
		{"1 << 3 | 2", 10},
		{"10 ^ 10", 0},
	}
	runVmTests(t, tests)
}

func TestBytesAndBinaryIO(t *testing.T) {
	tests := []vmTestCase{
		{`let b = bytes(4); b[0] = 65; b[1] = 66; b[0] + b[1]`, 131},
		{`let b = bytes("hello"); len(b)`, 5},
		{`let b = bytes([10, 20, 30]); b[1]`, 20},
		{`let b = bytes(4); b.write_u8(0, 255); b.read_u8(0)`, 255},
		{`let b = bytes(4); b.write_u16(0, 4660, "be"); b.read_u16(0, "be")`, 4660},
		{`let b = bytes(4); b.write_u16(0, 4660, "le"); b.read_u16(0, "le")`, 4660},
		{`let b = bytes(8); b.write_u32(0, 100000); b.read_u32(0)`, 100000},
		{`let b = bytes("BramandScript"); let s = slice(b, 0, 7); str(s)`, "Bramand"},
		{`let b = bytes("ABC"); b.to_string()`, "ABC"},
		{`let b = bytes([222, 173, 190, 239]); b.to_hex()`, "deadbeef"},
	}
	runVmTests(t, tests)
}

func TestUnicodeCodePointAndRunePrimitives(t *testing.T) {
	tests := []vmTestCase{
		{`ord("A")`, 65},
		{`ord("a")`, 97},
		{`ord("0")`, 48},
		{`ord("✦")`, 10022},
		{`chr(65)`, "A"},
		{`chr(97)`, "a"},
		{`chr(10022)`, "✦"},
		{`chr(ord("Z"))`, "Z"},
	}
	runVmTests(t, tests)
}

func TestWhileLoops(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let i = 0
			let sum = 0
			while i < 5 {
				i = i + 1
				sum = sum + i
			}
			sum
			`,
			expected: 15,
		},
		{
			input: `
			let i = 0
			while i < 10 {
				i = i + 1
				if i == 5 {
					break
				}
			}
			i
			`,
			expected: 5,
		},
	}
	runVmTests(t, tests)
}

func TestLargeLocalVariables(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("let testFn = fn() {\n")
	for i := 0; i < 300; i++ {
		sb.WriteString(fmt.Sprintf("    let v%d = %d;\n", i, i))
	}
	sb.WriteString("    return v0 + v299;\n")
	sb.WriteString("};\n")
	sb.WriteString("testFn();\n")

	tests := []vmTestCase{
		{input: sb.String(), expected: 299},
	}
	runVmTests(t, tests)
}

func TestBlockScopingExecution(t *testing.T) {
	tests := []vmTestCase{
		{
			// Shadowing in an if block: outer x remains 10
			input: `
			let x = 10;
			let innerVal = 0;
			if (true) {
				let x = 99;
				innerVal = x;
			}
			x + innerVal;
			`,
			expected: 109,
		},
		{
			// Function-level block scoping
			input: `
			let f = fn() {
				let a = 1;
				if (true) {
					let b = 2;
					let a = 10;
					return a + b;
				}
				return a;
			};
			f();
			`,
			expected: 12,
		},
		{
			// For-in loop block scoping
			input: `
			let sum = 0;
			for x in [1, 2, 3] {
				sum = sum + x;
			}
			sum;
			`,
			expected: 6,
		},
	}
	runVmTests(t, tests)
}
