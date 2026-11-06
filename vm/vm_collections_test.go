package vm

import (
	"testing"

	"bramandscript/compiler"
)

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

func TestSliceExpressions(t *testing.T) {
	tests := []vmTestCase{
		{"[10, 20, 30, 40, 50][1:3]", []int{20, 30}},
		{"[10, 20, 30, 40, 50][:2]", []int{10, 20}},
		{"[10, 20, 30, 40, 50][3:]", []int{40, 50}},
		{"[10, 20, 30][:]", []int{10, 20, 30}},
		{`"hello world"[0:5]`, "hello"},
		{`"hello world"[6:]`, "world"},
		{`"hello"[:3]`, "hel"},
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

func TestSliceWithStepAndReverse(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"let a = [10, 20, 30, 40, 50]; a[::]", []int{10, 20, 30, 40, 50}},
		{"let a = [10, 20, 30, 40, 50]; a[::-1]", []int{50, 40, 30, 20, 10}},
		{"let a = [10, 20, 30, 40, 50]; a[::2]", []int{10, 30, 50}},
		{"let a = [10, 20, 30, 40, 50]; a[1:4:2]", []int{20, 40}},
		{`let s = "hello"; s[::-1]`, "olleh"},
		{`let s = "abcdef"; s[::2]`, "ace"},
	}

	for _, tt := range tests {
		program := parse(tt.input)
		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			t.Fatalf("[%s] compile error: %s", tt.input, err)
		}
		vm := New(comp.Bytecode())
		err = vm.Run()
		if err != nil {
			t.Fatalf("[%s] vm error: %s", tt.input, err)
		}
		stackElem := vm.LastPoppedStackElem()
		testExpectedObject(t, tt.expected, stackElem)
	}
}

func TestEnumDeclarations(t *testing.T) {
	tests := []vmTestCase{
		{`enum Status { PENDING, ACTIVE, DONE }; Status.PENDING`, 0},
		{`enum Status { PENDING, ACTIVE, DONE }; Status.ACTIVE`, 1},
		{`enum Status { PENDING, ACTIVE, DONE }; Status.DONE`, 2},
		{`enum Status { PENDING, ACTIVE, DONE = 100, FAILED }; Status.DONE`, 100},
		{`enum Status { PENDING, ACTIVE, DONE = 100, FAILED }; Status.FAILED`, 101},
		{`enum Mode { FAST = "fast", SLOW = "slow" }; Mode.FAST`, "fast"},
		{`enum Status { A, B }; Status.__enum__`, true},
		{`enum Status { A, B }; Status.__name__`, "Status"},
	}
	runVmTests(t, tests)
}

func TestInterfaceChecking(t *testing.T) {
	tests := []vmTestCase{
		{`let obj = {"name": "test", "run": fn() { return true; }}; implements(obj, "run")`, true},
		{`let obj = {"name": "test", "run": fn() { return true; }}; implements(obj, ["run", "name"])`, true},
		{`let obj = {"name": "test"}; implements(obj, "missing")`, false},
		{`let obj = {"name": "test"}; implements(obj, ["name", "missing"])`, false},
	}
	runVmTests(t, tests)
}
