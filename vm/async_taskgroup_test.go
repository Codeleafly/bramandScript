package vm

import (
	"testing"
)

func TestUncoloredAsyncAwait(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let get_val = async fn() {
				return 42;
			};
			await get_val();
			`,
			expected: 42,
		},
		{
			input: `
			let greet = async fn(name) {
				return "Hello, " + name;
			};
			await greet("BramandScript");
			`,
			expected: "Hello, BramandScript",
		},
		{
			// Uncolored async/await: awaiting regular non-future value works directly
			input: `
			let x = 100;
			await x;
			`,
			expected: 100,
		},
		{
			// Uncolored async/await on string literal
			input: `
			await "direct_value";
			`,
			expected: "direct_value",
		},
		{
			// Yield voluntary time-slicing
			input: `
			let a = 1;
			yield;
			let b = 2;
			yield b;
			a + b;
			`,
			expected: 3,
		},
	}

	runVmTests(t, tests)
}

func TestStructuredTaskGroupWaitAll(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let tg = task_group();
			tg.spawn(fn() { return 10; });
			tg.spawn(fn() { return 20; });
			tg.spawn(fn() { return 30; });
			let res = tg.wait_all();
			res.is_ok();
			`,
			expected: true,
		},
		{
			input: `
			let tg = task_group();
			tg.spawn(fn() { return 100; });
			tg.spawn(fn() { return 200; });
			let res = tg.wait_all();
			let list = res.unwrap();
			list[0] + list[1];
			`,
			expected: 300,
		},
		{
			input: `
			let tg = task_group();
			tg.spawn(fn() { return 1; });
			tg.spawn(fn() { return 2; });
			tg.count();
			`,
			expected: 2,
		},
	}

	runVmTests(t, tests)
}

func TestStructuredTaskGroupWaitAny(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let tg = task_group();
			tg.spawn(fn() { return "winner"; });
			let res = tg.wait_any();
			res.unwrap();
			`,
			expected: "winner",
		},
	}

	runVmTests(t, tests)
}

func TestStructuredTaskGroupCancellation(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let tg = task_group();
			tg.cancel("cancelled by user");
			tg.is_cancelled();
			`,
			expected: true,
		},
	}

	runVmTests(t, tests)
}

func TestStructuredTaskGroupCascadeCancellation(t *testing.T) {
	tests := []vmTestCase{
		{
			input: `
			let tg = task_group();
			tg.spawn(fn() {
				panic("deliberate failure");
			});
			tg.spawn(fn() {
				sleep(50);
				return 42;
			});
			let res = tg.wait_all();
			res.is_err();
			`,
			expected: true,
		},
	}

	runVmTests(t, tests)
}
