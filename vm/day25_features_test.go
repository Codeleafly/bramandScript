package vm

import (
	"bramandscript/object"
	"testing"
)

func TestDeferStatementExecution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name: "LIFO defer ordering",
			input: `
			let order = [];
			fn run() {
				defer fn() { order = append(order, 1); }();
				defer fn() { order = append(order, 2); }();
				defer fn() { order = append(order, 3); }();
				return "done";
			}
			run();
			order;
			`,
			expected: "[3, 2, 1]",
		},
		{
			name: "Defer on early return",
			input: `
			let cleaned = false;
			fn early(val) {
				defer fn() { cleaned = true; }();
				if val > 10 {
					return 999;
				}
				return 0;
			}
			let res = early(20);
			[res, cleaned];
			`,
			expected: "[999, true]",
		},
		{
			name: "Defer on exception unwinding",
			input: `
			let ran = false;
			fn will_fail() {
				defer fn() { ran = true; }();
				throw "kaboom";
			}
			try {
				will_fail();
			} catch err {
				// handled
			}
			ran;
			`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := executeForTest(t, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expStr, ok := tt.expected.(string); ok {
				if res.Inspect() != expStr {
					t.Errorf("expected %s, got %s", expStr, res.Inspect())
				}
			} else if expBool, ok := tt.expected.(bool); ok {
				if boolObj, ok := res.(*object.Boolean); !ok || boolObj.Value != expBool {
					t.Errorf("expected %v, got %s", expBool, res.Inspect())
				}
			}
		})
	}
}

func TestDefaultParameterExecution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Single default parameter omitted",
			input: `
			fn greet(name, prefix = "Hello") {
				return f"{prefix}, {name}";
			}
			greet("World");
			`,
			expected: "Hello, World",
		},
		{
			name: "Single default parameter provided",
			input: `
			fn greet(name, prefix = "Hello") {
				return f"{prefix}, {name}";
			}
			greet("World", "Hi");
			`,
			expected: "Hi, World",
		},
		{
			name: "Multiple default parameters partially overridden",
			input: `
			fn config(host, port = 80, ssl = false) {
				return [host, port, ssl];
			}
			config("localhost", 443);
			`,
			expected: "[localhost, 443, false]",
		},
		{
			name: "Default parameter referring to prior parameter",
			input: `
			fn rect(w, h = w) {
				return w * h;
			};
			let res = [rect(4), rect(4, 6)];
			res;
			`,
			expected: "[16, 24]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := executeForTest(t, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.Inspect())
			}
		})
	}
}

func TestArrayAndHashSpreadExecution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Array spread",
			input: `
			let mid = [2, 3];
			let arr = [1, ...mid, 4];
			arr;
			`,
			expected: "[1, 2, 3, 4]",
		},
		{
			name: "Multiple array spreads",
			input: `
			let a = [1, 2];
			let b = [3, 4];
			[...a, ...b];
			`,
			expected: "[1, 2, 3, 4]",
		},
		{
			name: "String spread",
			input: `
			[..."hey"];
			`,
			expected: "[h, e, y]",
		},
		{
			name: "Hash spread with overrides",
			input: `
			let base = { "a": 1, "b": 2 };
			let final = { ...base, "b": 20, "c": 30 };
			[final["a"], final["b"], final["c"]];
			`,
			expected: "[1, 20, 30]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := executeForTest(t, tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Inspect() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, res.Inspect())
			}
		})
	}
}
