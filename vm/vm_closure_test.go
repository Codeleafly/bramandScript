package vm

import (
	"testing"

	"bramandscript/compiler"
)

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

func TestStructCompositionAndFieldEmbedding(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				struct Base {
					prefix,
					fn greet(name) {
						return prefix + " " + name;
					}
				}
				struct App {
					embed Base,
					title,
					fn run() {
						return title + " ready";
					}
				}
				let b = Base("Hello");
				let a = App(b, "MyApp");
				a.prefix + " - " + a.title;
			`,
			expected: "Hello - MyApp",
		},
		{
			input: `
				struct Base {
					prefix,
					fn greet(name) {
						return prefix + " " + name;
					}
				}
				struct App {
					embed Base,
					title
				}
				let b = Base("Hi");
				let a = App(b, "Test");
				a.greet("World");
			`,
			expected: "Hi World",
		},
	}
	runVmTests(t, tests)
}

func TestCallableStructInstancesAndMagicMethods(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
				struct Multiplier {
					factor,
					fn __call__(x) {
						return x * factor;
					}
				}
				let double = Multiplier(2);
				double(21);
			`,
			expected: 42,
		},
		{
			input: `
				struct Person {
					name,
					__str__,
					fn __call__(greeting) {
						return greeting + " " + name;
					}
				}
				let p = Person("Alice", "Person(Alice)");
				p("Hello");
			`,
			expected: "Hello Alice",
		},
		{
			input: `
				struct Item {
					id,
					__str__
				}
				let item = Item(42, "CustomItem#42");
				str(item);
			`,
			expected: "CustomItem#42",
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

func TestStructMethodWithLocals(t *testing.T) {
	input := `
	struct Store {
		filepath,
		index,
		fn set(key, val) {
			let skey = "" + key
			index[skey] = val
			if filepath != null {
				let line = "entry"
			}
			return val
		}
	}
	let s = Store("myfile", {})
	s.set("foo", "bar")
	s.filepath
	`
	program := parse(input)
	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		t.Fatalf("compile error: %s", err)
	}
	vm := New(comp.Bytecode())
	err = vm.Run()
	if err != nil {
		t.Fatalf("vm error: %s", err)
	}
	stackElem := vm.LastPoppedStackElem()
	testExpectedObject(t, "myfile", stackElem)
}
