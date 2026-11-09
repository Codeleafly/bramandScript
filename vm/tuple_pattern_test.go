package vm

import (
	"bramandscript/compiler"
	"testing"
)

func TestTupleLiteralsAndIndexing(t *testing.T) {
	tests := []vmTestCase{
		{
			input:    `let t = (1, 2, 3); t[0];`,
			expected: 1,
		},
		{
			input:    `let t = (10, 20, 30); t[1] + t[2];`,
			expected: 50,
		},
		{
			input:    `let t = (100, 200, 300); t[-1];`,
			expected: 300,
		},
		{
			input:    `let t = (); len(t);`,
			expected: 0,
		},
		{
			input:    `let t = (42,); len(t);`,
			expected: 1,
		},
		{
			input:    `let t = (42,); t[0];`,
			expected: 42,
		},
		{
			input:    `let t = (1, 2); t == (1, 2);`,
			expected: true,
		},
		{
			input:    `let t = (1, 2); t == (1, 3);`,
			expected: false,
		},
	}

	runVmTests(t, tests)
}

func TestTupleImmutability(t *testing.T) {
	input := `
	let t = (1, 2, 3);
	t[0] = 10;
	`
	program := parse(input)
	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %s", err)
	}
	vm := New(comp.Bytecode())
	err = vm.Run()
	if err == nil {
		t.Fatalf("expected error assigning to immutable tuple element, got nil")
	}
}

func TestTupleDestructuring(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let (x, y, z) = (10, 20, 30);
			x + y + z;
			`,
			expected: 60,
		},
		{
			input: `
			const (a, b) = ("hello", "world");
			a + " " + b;
			`,
			expected: "hello world",
		},
		{
			input: `
			let (first, second, third) = (100, 200, 300);
			second;
			`,
			expected: 200,
		},
	}

	runVmTests(t, tests)
}

func TestDeepPatternMatchingTuplesAndGuards(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let classify = fn(pair) {
				return match pair {
					case (0, 0) => { "origin" }
					case (0, y) => { "y-axis: " + str(y) }
					case (x, 0) => { "x-axis: " + str(x) }
					case (x, y) if x == y => { "diagonal" }
					case (x, y) => { "point(" + str(x) + ", " + str(y) + ")" }
					default => { "unknown" }
				};
			};
			classify((0, 0));
			`,
			expected: "origin",
		},
		{
			input: `
			let classify = fn(pair) {
				return match pair {
					case (0, 0) => { "origin" }
					case (0, y) => { "y-axis: " + str(y) }
					case (x, 0) => { "x-axis: " + str(x) }
					case (x, y) if x == y => { "diagonal" }
					case (x, y) => { "point(" + str(x) + ", " + str(y) + ")" }
					default => { "unknown" }
				};
			};
			classify((0, 5));
			`,
			expected: "y-axis: 5",
		},
		{
			input: `
			let classify = fn(pair) {
				return match pair {
					case (0, 0) => { "origin" }
					case (0, y) => { "y-axis: " + str(y) }
					case (x, 0) => { "x-axis: " + str(x) }
					case (x, y) if x == y => { "diagonal" }
					case (x, y) => { "point(" + str(x) + ", " + str(y) + ")" }
					default => { "unknown" }
				};
			};
			classify((7, 0));
			`,
			expected: "x-axis: 7",
		},
		{
			input: `
			let classify = fn(pair) {
				return match pair {
					case (0, 0) => { "origin" }
					case (0, y) => { "y-axis: " + str(y) }
					case (x, 0) => { "x-axis: " + str(x) }
					case (x, y) if x == y => { "diagonal" }
					case (x, y) => { "point(" + str(x) + ", " + str(y) + ")" }
					default => { "unknown" }
				};
			};
			classify((4, 4));
			`,
			expected: "diagonal",
		},
		{
			input: `
			let classify = fn(pair) {
				return match pair {
					case (0, 0) => { "origin" }
					case (0, y) => { "y-axis: " + str(y) }
					case (x, 0) => { "x-axis: " + str(x) }
					case (x, y) if x == y => { "diagonal" }
					case (x, y) => { "point(" + str(x) + ", " + str(y) + ")" }
					default => { "unknown" }
				};
			};
			classify((3, 8));
			`,
			expected: "point(3, 8)",
		},
	}

	runVmTests(t, tests)
}

func TestDeepPatternMatchingArraysAndGuards(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let cmp = fn(pair) {
				return match pair {
					case [a, b] if a > b => { "descending" }
					case [a, b] if a < b => { "ascending" }
					case [a, b] => { "equal" }
					default => { "other" }
				};
			};
			cmp([10, 5]);
			`,
			expected: "descending",
		},
		{
			input: `
			let cmp = fn(pair) {
				return match pair {
					case [a, b] if a > b => { "descending" }
					case [a, b] if a < b => { "ascending" }
					case [a, b] => { "equal" }
					default => { "other" }
				};
			};
			cmp([3, 9]);
			`,
			expected: "ascending",
		},
		{
			input: `
			let cmp = fn(pair) {
				return match pair {
					case [a, b] if a > b => { "descending" }
					case [a, b] if a < b => { "ascending" }
					case [a, b] => { "equal" }
					default => { "other" }
				};
			};
			cmp([7, 7]);
			`,
			expected: "equal",
		},
	}

	runVmTests(t, tests)
}

func TestDeepPatternMatchingHashAndGuards(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let check_user = fn(u) {
				return match u {
					case { "role": "admin" } => { "admin_access" }
					case { "name": n, "age": a } if a >= 18 => { n + "_adult" }
					case { "name": n, "age": a } => { n + "_minor" }
					default => { "unknown" }
				};
			};
			check_user({ "role": "admin" });
			`,
			expected: "admin_access",
		},
		{
			input: `
			let check_user = fn(u) {
				return match u {
					case { "role": "admin" } => { "admin_access" }
					case { "name": n, "age": a } if a >= 18 => { n + "_adult" }
					case { "name": n, "age": a } => { n + "_minor" }
					default => { "unknown" }
				};
			};
			check_user({ "name": "alice", "age": 25 });
			`,
			expected: "alice_adult",
		},
		{
			input: `
			let check_user = fn(u) {
				return match u {
					case { "role": "admin" } => { "admin_access" }
					case { "name": n, "age": a } if a >= 18 => { n + "_adult" }
					case { "name": n, "age": a } => { n + "_minor" }
					default => { "unknown" }
				};
			};
			check_user({ "name": "bob", "age": 15 });
			`,
			expected: "bob_minor",
		},
	}

	runVmTests(t, tests)
}
