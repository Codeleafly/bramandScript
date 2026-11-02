# Verification Task 01: Language Core & Expression Evaluation

## 1. Test Objective
Verify BramandScript's core evaluation engine: literal types, arithmetic operations, operator precedence, parentheses nesting, variable declarations (`let`, `const`), reassignments, logical expressions, and runtime division by zero diagnostics.

## 2. Features Tested
- Integer, Float, String (with escapes `\n\t\"`), Boolean, and Null literals
- Infix arithmetic: `+`, `-`, `*`, `/`, `%`, `**`
- Relational comparisons: `==`, `!=`, `<`, `>`, `<=`, `>=`
- Logical operations: `&&`, `||`
- Precedence ordering and grouping parentheses
- Variables & mutability: `let`, reassignments, `const` immutability
- Runtime zero-division detection and diagnostic hints

## 3. Source Files
- `verification/task-01/test_core.bmd`
- `verification/task-01/test_div_zero.bmd`

## 4. Execution Commands & Exit Codes

### Command 1: Core Expressions
```bash
./bramand run verification/task-01/test_core.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (26/26 assertions passed)

### Command 2: Division by Zero Edge Case
```bash
./bramand run verification/task-01/test_div_zero.bmd
```
- **Exit Code**: `1`
- **Result**: PASS (Emitted `error[RuntimeError]: division by zero` with contextual hint `[Hint: Denominator cannot be 0]`)

## 5. Actual Output
```
✦ TASK 01: Core Language & Expression Evaluation
  ✓ [PASS] Integer literal
  ✓ [PASS] Float literal
  ✓ [PASS] String literal
  ✓ [PASS] Boolean literals
  ✓ [PASS] Null literal
  ✓ [PASS] Addition
  ✓ [PASS] Subtraction
  ✓ [PASS] Multiplication
  ✓ [PASS] Integer division
  ✓ [PASS] Modulo operator
  ✓ [PASS] Exponentiation
  ✓ [PASS] Float arithmetic
  ✓ [PASS] Operator precedence * over +
  ✓ [PASS] Parentheses override precedence
  ✓ [PASS] Complex precedence
  ✓ [PASS] Equal ==
  ✓ [PASS] Not equal !=
  ✓ [PASS] Less than <
  ✓ [PASS] Greater than >
  ✓ [PASS] Less than equal <=
  ✓ [PASS] Greater than equal >=
  ✓ [PASS] Logical AND
  ✓ [PASS] Logical OR
  ✓ [PASS] Short-circuit OR
  ✓ [PASS] Let reassignment
  ✓ [PASS] Const declaration
[OK] All Task 01 core expression checks passed successfully.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
