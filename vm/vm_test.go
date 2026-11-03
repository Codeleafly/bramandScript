package vm

import (
	"bramandscript/ast"
	"bramandscript/compiler"
	"bramandscript/lexer"
	"bramandscript/object"
	"bramandscript/parser"
	"fmt"
	"os"
	"strings"
	"testing"
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

func TestArrayLiterals(t *testing.T) {
	tests := []vmTestCase{
		{"[]", []int{}},
		{"[1, 2, 3]", []int{1, 2, 3}},
		{"[1 + 2, 3 * 4, 5 + 6]", []int{3, 12, 11}},
	}
	runVmTests(t, tests)
}

func TestIndexExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"[1, 2, 3][0]", 1},
		{"[1, 2, 3][1]", 2},
		{"[1, 2, 3][2]", 3},
		{"let i = 0; [1][i]", 1},
		{"[1, 2, 3][1 + 1]", 3},
		{"let arr = [1, 2, 3]; arr[0] + arr[1] + arr[2]", 6},
		{`{"name": "Brahmand"}["name"]`, "Brahmand"},
		{`{1: 10, 2: 20}[2]`, 20},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithoutArguments(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let fivePlusTen = fn() { 5 + 10; };
			fivePlusTen();
			`,
			expected: 15,
		},
		{
			input: `
			let one = fn() { 1; };
			let two = fn() { 2; };
			one() + two();
			`,
			expected: 3,
		},
		{
			input: `
			let earlyExit = fn() { return 99; 100; };
			earlyExit();
			`,
			expected: 99,
		},
	}
	runVmTests(t, tests)
}

func TestCallingFunctionsWithArgumentsAndBindings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let identity = fn(a) { a; };
			identity(4);
			`,
			expected: 4,
		},
		{
			input: `
			let sum = fn(a, b) { a + b; };
			sum(1, 2);
			`,
			expected: 3,
		},
		{
			input: `
			let sum = fn(a, b) {
				let c = a + b;
				c;
			};
			sum(10, 20);
			`,
			expected: 30,
		},
	}
	runVmTests(t, tests)
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []vmTestCase{
		{`len("")`, 0},
		{`len("four")`, 4},
		{`len([1, 2, 3])`, 3},
		{`type("universe")`, "STRING"},
		{`type(42)`, "INTEGER"},
		{`len(push([1, 2], 3))`, 3},
		{`let a = [1, 2]; push(a, 3); len(a)`, 3},
		{`let a = [1, 2]; push(a, 3); a[2]`, 3},
		{`int("42")`, 42},
		{`float("3.14")`, 3.14},
		{`str(100)`, "100"},
		{`bool(1)`, true},
		{`bool(null)`, false},
		{`len([1, 2] + [3, 4, 5])`, 5},
		{`([10, 20] + [30, 40])[2]`, 30},
		{`len(slice([1, 2, 3, 4, 5], 1, 4))`, 3},
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

func TestClosures(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let newClosure = fn(a) {
				return fn() { return a; };
			};
			let closure = newClosure(99);
			closure();
			`,
			expected: 99,
		},
		{
			input: `
			let newAdder = fn(a, b) {
				return fn(c) { return a + b + c; };
			};
			let adder = newAdder(1, 2);
			adder(8);
			`,
			expected: 11,
		},
	}
	runVmTests(t, tests)
}

func TestIndexAssignment(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let arr = [10, 20, 30]
			arr[0] = 99
			arr[0]
			`,
			expected: 99,
		},
		{
			input: `
			let arr = [1, 2, 3]
			arr[1] = 50
			arr[0] + arr[1] + arr[2]
			`,
			expected: 54,
		},
		{
			input: `
			let user = {"name": "Aryan", "score": 100}
			user["score"] = 250
			user["score"]
			`,
			expected: 250,
		},
	}
	runVmTests(t, tests)
}

func TestDotAccessAndAssign(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let user = {"name": "Aryan", "role": "dev"}
			user.role
			`,
			expected: "dev",
		},
		{
			input: `
			let user = {"name": "Aryan", "role": "dev"}
			user.role = "lead"
			user.role
			`,
			expected: "lead",
		},
		{
			input: `
			let arr = [1, 2, 3, 4, 5]
			arr.length
			`,
			expected: 5,
		},
		{
			input: `
			let s = "hello"
			s.length
			`,
			expected: 5,
		},
	}
	runVmTests(t, tests)
}

func TestForInLoops(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let sum = 0
			for x in [1, 2, 3, 4] {
				sum += x
			}
			sum
			`,
			expected: 10,
		},
		{
			input: `
			let sumIdx = 0
			for i, val in [10, 20, 30] {
				sumIdx += i
			}
			sumIdx
			`,
			expected: 3, // 0 + 1 + 2
		},
		{
			input: `
			let sum = 0
			for x in [1, 2, 3, 4, 5] {
				if x == 4 { break }
				sum += x
			}
			sum
			`,
			expected: 6, // 1 + 2 + 3
		},
		{
			input: `
			let sum = 0
			for x in [1, 2, 3, 4, 5] {
				if x == 3 { continue }
				sum += x
			}
			sum
			`,
			expected: 12, // 1 + 2 + 4 + 5
		},
	}
	runVmTests(t, tests)
}

func TestPatternMatching(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let status = 200
			let res = match status {
				case 200 => { "OK" }
				case 404 => { "Not Found" }
				default => { "Error" }
			}
			res
			`,
			expected: "OK",
		},
		{
			input: `
			let code = 404
			let res = match code {
				case 200 => { "OK" }
				case 400, 404 => { "Client Error" }
				default => { "Server Error" }
			}
			res
			`,
			expected: "Client Error",
		},
		{
			input: `
			let code = 500
			let res = match code {
				case 200 => { "OK" }
				case 404 => { "Not Found" }
				default => { "Other" }
			}
			res
			`,
			expected: "Other",
		},
	}
	runVmTests(t, tests)
}

func TestStructsAndMethods(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			struct Point {
				x, y
			}
			let pt = Point(15, 25)
			pt.x + pt.y
			`,
			expected: 40,
		},
		{
			input: `
			struct Counter {
				count
				fn inc() {
					count += 1
					return count
				}
			}
			let c = Counter(10)
			c.inc()
			c.inc()
			`,
			expected: 12,
		},
	}
	runVmTests(t, tests)
}

func TestModuleImport(t *testing.T) {
	// Create temporary module file
	modCode := `
	fn add(a, b) {
		return a + b
	}
	let multiplier = 5
	`
	os.WriteFile("test_calc.bmd", []byte(modCode), 0644)
	defer os.Remove("test_calc.bmd")

	tests := []vmTestCase{
		{
			input: `
			import "test_calc.bmd"
			add(20, 22)
			`,
			expected: 42,
		},
		{
			input: `
			import "test_calc.bmd" as calc
			calc.add(10, calc.multiplier)
			`,
			expected: 15,
		},
	}
	runVmTests(t, tests)
}

func TestSpawnAndChannels(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let ch = chan(1);
			ch <- 100;
			<-ch;
			`,
			expected: 100,
		},
		{
			input: `
			let ch = chan();
			spawn fn() {
				ch <- 42;
			}();
			<-ch;
			`,
			expected: 42,
		},
		{
			input: `
			let ch = chan(2);
			ch <- 10;
			ch <- 20;
			let a = <-ch;
			let b = <-ch;
			a + b;
			`,
			expected: 30,
		},
		{
			input: `
			let ch = chan(5);
			ch.capacity;
			`,
			expected: 5,
		},
		{
			input: `
			let ch = chan(1);
			ch <- "ping";
			close(ch);
			<-ch;
			`,
			expected: "ping",
		},
	}
	runVmTests(t, tests)
}

func TestMutexAndSynchronization(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let m = mutex();
			m.lock();
			m.unlock();
			let ok = m.try_lock();
			m.unlock();
			ok;
			`,
			expected: true,
		},
		{
			input: `
			let wg = wait_group();
			let ch = chan(1);
			wg.add(1);
			spawn fn() {
				ch <- 77;
				wg.done();
			}();
			wg.wait();
			<-ch;
			`,
			expected: 77,
		},
	}
	runVmTests(t, tests)
}

func TestFileIO(t *testing.T) {
	testFile := "test_day4_io.txt"
	defer os.Remove(testFile)

	tests := []vmTestCase{
		{
			input: `
			write_file("test_day4_io.txt", "BramandScript Day 4");
			file_exists("test_day4_io.txt");
			`,
			expected: true,
		},
		{
			input: `
			read_file("test_day4_io.txt");
			`,
			expected: "BramandScript Day 4",
		},
		{
			input: `
			append_file("test_day4_io.txt", " Concurrency");
			read_file("test_day4_io.txt");
			`,
			expected: "BramandScript Day 4 Concurrency",
		},
		{
			input: `
			delete_file("test_day4_io.txt");
			file_exists("test_day4_io.txt");
			`,
			expected: false,
		},
		{
			input: `
			is_dir(".");
			`,
			expected: true,
		},
		{
			input: `
			let files = list_dir(".");
			len(files) > 0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestProcessExecutionAndEnv(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			set_env("BMD_DAY4_TEST", "verified");
			env("BMD_DAY4_TEST");
			`,
			expected: "verified",
		},
		{
			input: `
			let res = exec("echo", "bramand");
			res.code;
			`,
			expected: 0,
		},
		{
			input: `
			let res = exec("echo", "test_output");
			res.stdout;
			`,
			expected: "test_output\n",
		},
	}
	runVmTests(t, tests)
}

func TestStandardIOAndFormatting(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let colored = color("green", "OK");
			colored == "\033[32mOK\033[0m";
			`,
			expected: true,
		},
		{
			input: `
			let styledText = styled("HELLO", "bold");
			styledText == "\033[1mHELLO\033[0m";
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestBigIntAndBigFloat(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let a = big_int("1000000000000000000000000000000");
			let b = big_int("2000000000000000000000000000000");
			let c = a + b;
			c.str();
			`,
			expected: "3000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("1000000000000000000000000000000");
			let b = big_int("3000000000000000000000000000000");
			let c = b - a;
			c.str();
			`,
			expected: "2000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("1000000000000000");
			let b = big_int("2000000000000000");
			let c = a * b;
			c.str();
			`,
			expected: "2000000000000000000000000000000",
		},
		{
			input: `
			let a = big_int("100000000000000000000");
			let b = big_int("10000000000");
			let c = a / b;
			c.str();
			`,
			expected: "10000000000",
		},
		{
			input: `
			let a = big_int("105");
			let b = big_int("10");
			let c = a % b;
			c.str();
			`,
			expected: "5",
		},
		{
			input: `
			let a = big_int("2");
			let b = big_int("100");
			let c = a ** b;
			c.str();
			`,
			expected: "1267650600228229401496703205376",
		},
		{
			input: `
			let a = big_int("100");
			let b = big_int("200");
			a < b && b > a && a != b && (a == big_int("100"));
			`,
			expected: true,
		},
		{
			input: `
			let a = big_int("50");
			let b = -a;
			b.str();
			`,
			expected: "-50",
		},
		{
			input: `
			let a = big_int("-99");
			a.abs().str();
			`,
			expected: "99",
		},
		{
			input: `
			let a = big_float("1.25");
			let b = big_float("2.5");
			let c = a + b;
			c.str(3);
			`,
			expected: "3.75",
		},
		{
			input: `
			let a = big_float("10.5");
			let b = big_float("2.5");
			let c = a - b;
			c.str(3);
			`,
			expected: "8",
		},
		{
			input: `
			let a = big_float("2.5");
			let b = big_float("4.0");
			let c = a * b;
			c.str(3);
			`,
			expected: "10",
		},
		{
			input: `
			let a = big_float("15.0");
			let b = big_float("2.0");
			let c = a / b;
			c.str(3);
			`,
			expected: "7.5",
		},
		{
			input: `
			let a = big_float("16.0");
			let b = a.sqrt();
			b.str(3);
			`,
			expected: "4",
		},
		{
			input: `
			let a = big_float("1.5");
			let b = big_float("2.5");
			a < b && b > a && a != b && (a == big_float("1.5"));
			`,
			expected: true,
		},
		{
			input: `
			let x = 99999999999999999999999999999 ** 2;
			x.str();
			`,
			expected: "9999999999999999999999999999800000000000000000000000000001",
		},
	}
	runVmTests(t, tests)
}

func TestBigIntPowerOverflowSafety(t *testing.T) {
	input := `let x = 99999999999999999999999999999 ** 99999999999999999999999;`
	program := parse(input)
	compiler := compiler.New()
	err := compiler.Compile(program)
	if err != nil {
		t.Fatalf("unexpected compiler error: %s", err)
	}
	vm := New(compiler.Bytecode())
	err = vm.Run()
	if err == nil {
		t.Fatalf("expected exponent overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "exponent too large") {
		t.Fatalf("expected 'exponent too large' error, got %s", err)
	}
}

func TestVectorsAndMatrices(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let v = vec([1.0, 2.0, 3.0]);
			v.dim;
			`,
			expected: 3,
		},
		{
			input: `
			let v = vec([10.0, 20.0, 30.0]);
			v[1];
			`,
			expected: 20.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0, 3.0]);
			let v2 = vec([4.0, 5.0, 6.0]);
			v1.dot(v2);
			`,
			expected: 32.0,
		},
		{
			input: `
			let v1 = vec([1.0, 0.0, 0.0]);
			let v2 = vec([0.0, 1.0, 0.0]);
			let v3 = v1.cross(v2);
			v3[0] == 0.0 && v3[1] == 0.0 && v3[2] == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			let v = vec([3.0, 4.0]);
			v.norm();
			`,
			expected: 5.0,
		},
		{
			input: `
			let v = vec([0.0, 5.0]);
			let u = v.normalize();
			u[0] == 0.0 && u[1] == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0]);
			let v2 = vec([3.0, 4.0]);
			let v3 = v1.add(v2);
			v3[0] == 4.0 && v3[1] == 6.0;
			`,
			expected: true,
		},
		{
			input: `
			let v = vec([2.0, 3.0]);
			let scaled = v.scale(3.0);
			scaled[0] == 6.0 && scaled[1] == 9.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.rows == 2 && m.cols == 2;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.get(1, 0);
			`,
			expected: 3.0,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[5.0, 6.0], [7.0, 8.0]]);
			let m3 = m1.add(m2);
			m3.get(0, 0) == 6.0 && m3.get(1, 1) == 12.0;
			`,
			expected: true,
		},
		{
			input: `
			let m1 = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let m2 = matrix([[2.0, 0.0], [1.0, 2.0]]);
			let m3 = m1.mul(m2);
			// [1*2 + 2*1, 1*0 + 2*2] = [4, 4]
			// [3*2 + 4*1, 3*0 + 4*2] = [10, 8]
			m3.get(0, 0) == 4.0 && m3.get(0, 1) == 4.0 && m3.get(1, 0) == 10.0 && m3.get(1, 1) == 8.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let v = vec([2.0, 1.0]);
			let res = m.mul_vec(v);
			res[0] == 4.0 && res[1] == 10.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			let t = m.transpose();
			t.get(0, 1) == 3.0 && t.get(1, 0) == 2.0;
			`,
			expected: true,
		},
		{
			input: `
			let m = matrix([[1.0, 2.0], [3.0, 4.0]]);
			m.det();
			`,
			expected: -2.0,
		},
		{
			input: `
			let m = matrix([[4.0, 7.0], [2.0, 6.0]]);
			let inv = m.inv();
			let ident = m.mul(inv);
			// Check diagonal is ~1.0
			floor(ident.get(0, 0) + 0.5) == 1.0 && floor(ident.get(1, 1) + 0.5) == 1.0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestScientificMathAndStats(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			floor(pi() * 100);
			`,
			expected: 314,
		},
		{
			input: `
			floor(e() * 100);
			`,
			expected: 271,
		},
		{
			input: `
			sin(0.0) == 0.0 && cos(0.0) == 1.0;
			`,
			expected: true,
		},
		{
			input: `
			sqrt(144.0);
			`,
			expected: 12.0,
		},
		{
			input: `
			cbrt(27.0);
			`,
			expected: 3.0,
		},
		{
			input: `
			hypot(3.0, 4.0);
			`,
			expected: 5.0,
		},
		{
			input: `
			log10(1000.0);
			`,
			expected: 3.0,
		},
		{
			input: `
			log2(32.0);
			`,
			expected: 5.0,
		},
		{
			input: `
			pow(2.0, 8.0);
			`,
			expected: 256.0,
		},
		{
			input: `
			abs(-42.5);
			`,
			expected: 42.5,
		},
		{
			input: `
			floor(3.9) == 3.0 && ceil(3.1) == 4.0 && round(3.5) == 4.0 && trunc(3.7) == 3.0;
			`,
			expected: true,
		},
		{
			input: `
			deg_to_rad(180.0) == pi();
			`,
			expected: true,
		},
		{
			input: `
			rad_to_deg(pi());
			`,
			expected: 180.0,
		},
		{
			input: `
			min([5, 2, 9, 1, 7]);
			`,
			expected: 1,
		},
		{
			input: `
			max([5, 2, 9, 1, 7]);
			`,
			expected: 9,
		},
		{
			input: `
			sum([10, 20, 30, 40]);
			`,
			expected: 100,
		},
		{
			input: `
			mean([10.0, 20.0, 30.0]);
			`,
			expected: 20.0,
		},
		{
			input: `
			median([1.0, 3.0, 2.0, 5.0, 4.0]);
			`,
			expected: 3.0,
		},
		{
			input: `
			median([1.0, 2.0, 3.0, 4.0]);
			`,
			expected: 2.5,
		},
		{
			input: `
			mode([1, 2, 2, 3, 3, 3, 4]);
			`,
			expected: 3,
		},
		{
			input: `
			variance([2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0]);
			`,
			expected: 4.0,
		},
		{
			input: `
			std_dev([2.0, 4.0, 4.0, 4.0, 5.0, 5.0, 7.0, 9.0]);
			`,
			expected: 2.0,
		},
	}
	runVmTests(t, tests)
}

func TestJSONSerialization(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let raw = "{\"name\": \"bramand\", \"version\": 1, \"active\": true}";
			let obj = json_parse(raw);
			obj["name"] == "bramand" && obj["version"] == 1 && obj["active"] == true;
			`,
			expected: true,
		},
		{
			input: `
			let raw = "[10, 20, 30]";
			let arr = json_parse(raw);
			arr[0] == 10 && arr[1] == 20 && arr[2] == 30;
			`,
			expected: true,
		},
		{
			input: `
			let data = {"title": "Language", "id": 42};
			let str = json_stringify(data);
			let back = json_parse(str);
			back["title"] == "Language" && back["id"] == 42;
			`,
			expected: true,
		},
		{
			input: `
			let data = [1, 2, 3];
			let str = json_stringify(data, 2);
			let back = json_parse(str);
			back[0] == 1 && back[1] == 2 && back[2] == 3;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestCSVAndBase64(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let csvStr = "name,age,city\nAlice,30,Tokyo\nBob,25,Paris";
			let rows = csv_parse(csvStr);
			rows[0][0] == "name" && rows[1][1] == "30" && rows[2][2] == "Paris";
			`,
			expected: true,
		},
		{
			input: `
			let csvStr = "a;b;c\n1;2;3";
			let rows = csv_parse(csvStr, ";");
			rows[0][1] == "b" && rows[1][2] == "3";
			`,
			expected: true,
		},
		{
			input: `
			let rows = [["id", "name"], ["1", "alpha"], ["2", "beta"]];
			let str = csv_stringify(rows);
			let parsed = csv_parse(str);
			parsed[1][1] == "alpha" && parsed[2][1] == "beta";
			`,
			expected: true,
		},
		{
			input: `
			let enc = base64_encode("BramandScript ✦ High Performance");
			let dec = base64_decode(enc);
			dec == "BramandScript ✦ High Performance";
			`,
			expected: true,
		},
		{
			input: `
			let enc = base64_url_encode("hello+world/test?foo=bar");
			let dec = base64_url_decode(enc);
			dec == "hello+world/test?foo=bar";
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestCryptoAndRandom(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			// SHA-256 of "hello"
			sha256("hello");
			`,
			expected: "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
		},
		{
			input: `
			// MD5 of "hello"
			md5("hello");
			`,
			expected: "5d41402abc4b2a76b9719d911017c592",
		},
		{
			input: `
			// SHA-512 of "hello" starts with 9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043
			sha512("hello");
			`,
			expected: "9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca72323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043",
		},
		{
			input: `
			// HMAC-SHA256 of "message" with key "secret"
			hmac_sha256("secret", "message");
			`,
			expected: "8b5f48702995c1598c573db1e21866a9b825d4a794d169d7060a03605796360b",
		},
		{
			input: `
			let rnd = random_bytes(16);
			len(rnd);
			`,
			expected: 32,
		},
		{
			input: `
			let n = random_int(10, 20);
			n >= 10 && n <= 20;
			`,
			expected: true,
		},
		{
			input: `
			let u = uuid();
			len(u) == 36;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestTcpAndUdpSockets(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = tcp_listen("127.0.0.1:19081");
			spawn fn() {
				let sConn = srv.accept();
				let line = sConn.read_line();
				sConn.write(line + "_reply\n");
				sConn.close();
			}();

			let client = tcp_connect("127.0.0.1:19081");
			client.write("hello_tcp\n");
			let resp = client.read_line();
			client.close();
			srv.close();
			resp;
			`,
			expected: "hello_tcp_reply",
		},
		{
			input: `
			let srv = udp_listen("127.0.0.1:19082");
			let client = udp_connect("127.0.0.1:19082");
			client.write("ping_udp");

			let packet = srv.recv_from();
			srv.close();
			client.close();
			packet["data"];
			`,
			expected: "ping_udp",
		},
	}
	runVmTests(t, tests)
}

func TestHttpClientAndServer(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = http_server();
			srv.get("/api/greet", fn(req, res) {
				res.send("hello from server");
			});
			srv.post("/api/echo", fn(req, res) {
				res.json(req.json());
			});
			srv.get("/api/custom_status", fn(req, res) {
				res.status(201).send("created");
			});
			srv.listen("127.0.0.1:19083");

			// Test GET
			let res1 = http_get("http://127.0.0.1:19083/api/greet");
			let b1 = res1.body;

			// Test POST
			let res2 = http_post("http://127.0.0.1:19083/api/echo", "{\"title\": \"Bramand\"}");
			let parsed = res2.json();
			let title = parsed["title"];

			// Test Status
			let res3 = http_get("http://127.0.0.1:19083/api/custom_status");
			let code = res3.status;

			srv.close();
			b1 == "hello from server" && title == "Bramand" && code == 201;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestWebSocketClientAndServer(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let srv = ws_server("127.0.0.1:19084", "/chat");
			spawn fn() {
				let sConn = srv.accept();
				let msg = sConn.recv();
				sConn.send("ack:" + msg);
				sConn.close();
			}();

			let client = ws_connect("ws://127.0.0.1:19084/chat");
			client.send("hello_bramand_ws");
			let reply = client.recv();
			client.close();
			srv.close();
			reply;
			`,
			expected: "ack:hello_bramand_ws",
		},
	}
	runVmTests(t, tests)
}

func TestVectorSimilarityAndEmbeddings(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let v1 = vec([1.0, 0.0]);
			let v2 = vec([0.0, 1.0]);
			cosine_similarity(v1, v2);
			`,
			expected: 0.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0, 3.0]);
			let v2 = vec([1.0, 2.0, 3.0]);
			floor(cosine_similarity(v1, v2) * 100);
			`,
			expected: 100,
		},
		{
			input: `
			let v1 = vec([1.0, 1.0]);
			let v2 = vec([1.0, 1.0]);
			cosine_distance(v1, v2);
			`,
			expected: 0.0,
		},
		{
			input: `
			let v1 = vec([0.0, 0.0]);
			let v2 = vec([3.0, 4.0]);
			euclidean_distance(v1, v2);
			`,
			expected: 5.0,
		},
		{
			input: `
			let v1 = vec([1.0, 2.0]);
			let v2 = vec([4.0, 6.0]);
			manhattan_distance(v1, v2);
			`,
			expected: 7.0,
		},
		{
			input: `
			let query = vec([1.0, 0.0]);
			let doc1 = {"id": "doc1", "embedding": vec([0.0, 1.0])};
			let doc2 = {"id": "doc2", "embedding": vec([0.9, 0.1])};
			let doc3 = {"id": "doc3", "embedding": vec([-1.0, 0.0])};

			let results = vector_search(query, [doc1, doc2, doc3], 1, "cosine");
			let topMatch = results[0]["item"];
			topMatch["id"];
			`,
			expected: "doc2",
		},
	}
	runVmTests(t, tests)
}

func TestAIIntegrationModule(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			// Set up a mock AI streaming HTTP server
			let srv = http_server();
			srv.post("/api/mock_stream", fn(req, res) {
				res.set_header("Content-Type", "text/event-stream");
				res.send("data: {\"response\": \"Hello \"}\n\ndata: {\"response\": \"from \"}\n\ndata: {\"response\": \"AI!\"}\n\ndata: [DONE]\n\n");
			});
			srv.listen("127.0.0.1:19085");

			let collected = [];
			let full = ai_stream({"url": "http://127.0.0.1:19085/api/mock_stream", "body": "{}"}, fn(token) {
				collected = push(collected, token);
			});

			srv.close();
			full == "Hello from AI!" && len(collected) == 3 && collected[0] == "Hello ";
			`,
			expected: true,
		},
		{
			input: `
			// Set up a mock AI structured response HTTP server
			let srv = http_server();
			srv.post("/api/mock_structured", fn(req, res) {
				res.json({"summary": "High Performance Language", "stars": 5});
			});
			srv.listen("127.0.0.1:19086");

			let schema = {"summary": "string", "stars": 0};
			let structured = ai_structured({"model": "test-model", "prompt": "analyze", "endpoint": "http://127.0.0.1:19086/api/mock_structured"}, schema);

			srv.close();
			structured["summary"] == "High Performance Language" && structured["stars"] == 5;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestAssertionsAndTestingFramework(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			assert(true);
			assert_eq(10 + 20, 30);
			assert_eq([1, 2, 3], [1, 2, 3]);
			assert_eq({"a": 1, "b": 2}, {"a": 1, "b": 2});
			assert_ne(10, 20);
			assert_ne([1, 2], [1, 3]);
			true;
			`,
			expected: true,
		},
		{
			input: `
			let passed = 0;
			describe("Math Suite", fn() {
				test("addition works", fn() {
					assert_eq(5 + 5, 10);
					passed = passed + 1;
				});
				test("subtraction works", fn() {
					assert_eq(10 - 4, 6);
					passed = passed + 1;
				});
			});
			passed == 2;
			`,
			expected: true,
		},
		{
			input: `
			let failedHandled = false;
			test("deliberate failure caught", fn() {
				assert_eq(1, 2, "1 should equal 2");
			});
			// test() returns false on failure, without crashing VM
			true;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestTimeAndBenchmarking(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let t1 = time_ms();
			let now = time_now();
			t1 > 0 && now > 0;
			`,
			expected: true,
		},
		{
			input: `
			let b = benchmark("simple_add", fn() {
				let x = 1 + 1;
			}, 100);
			b["name"] == "simple_add" && b["iterations"] == 100 && b["total_ms"] >= 0;
			`,
			expected: true,
		},
	}
	runVmTests(t, tests)
}

func TestLargeLocalVariables(t *testing.T) {
	// Generate a function with 300 local variables (exceeding previous 8-bit limit of 255)
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
