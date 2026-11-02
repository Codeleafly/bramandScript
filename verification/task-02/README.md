# Verification Task 02: Functions, Closures, and Control Flow

## 1. Test Objective
Verify first-class functions (`fn`), arguments, return values, nested scopes, lexical closures capturing state across invocations, recursive execution (Fibonacci & Factorial), conditional branching (`if`, `else if`, `else`), `while` loops with `break` and `continue`, and runtime failure reporting.

## 2. Features Tested
- Function definitions and invocations
- Nested functions with lexical scope isolation
- State-capturing closures via free variable management (`OpClosure`, `OpGetFree`)
- Recursive call chains
- `if`/`else if`/`else` branching jumps
- `while` loop control with `break` and `continue`
- Contextual runtime error diagnostics for invalid calls

## 3. Source Files
- `verification/task-02/test_functions_control.bmd`
- `verification/task-02/test_stack_trace.bmd`

## 4. Execution Commands & Exit Codes

### Command 1: Functions, Closures, and Loops
```bash
./bramand run verification/task-02/test_functions_control.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (11/11 assertions passed)

### Command 2: Stack Trace on Invalid Call
```bash
./bramand run verification/task-02/test_stack_trace.bmd
```
- **Exit Code**: `1`
- **Result**: PASS (Clean error diagnostic emitted: `error[RuntimeError]: calling non-function and non-builtin: NULL`)

## 5. Actual Output
```
✦ TASK 02: Functions, Closures, and Control Flow
  ✓ [PASS] Basic function call
  ✓ [PASS] Nested function call
  ✓ [PASS] Closure first call
  ✓ [PASS] Closure state retention
  ✓ [PASS] Recursive Fibonacci
  ✓ [PASS] Recursive Factorial
  ✓ [PASS] If branch
  ✓ [PASS] Else-if branch
  ✓ [PASS] Else branch
  ✓ [PASS] While loop with continue
  ✓ [PASS] While loop with break
[OK] All Task 02 functions and control flow checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
