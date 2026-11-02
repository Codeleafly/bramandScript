# Verification Task 16: Command-Line Arguments and CLI Applications

## 1. Test Objective
Determine and verify BramandScript's capability to build production-grade Command-Line Interface (CLI) applications using the `args()` / `argv()` builtins: reading `argv`, parsing positional and option flags (`--help`, `--version`), validating argument counts, converting string arguments to numbers, handling errors, and returning contextual OS exit codes.

## 2. Features Tested
- Accessing command-line arguments via `args()`
- Positional argument extraction (`cli_args[0]`, `cli_args[1]`, `cli_args[2]`)
- Command routing (`add`, `subtract`, `multiply`, `divide`)
- CLI help flag (`--help`, `-h`)
- CLI version flag (`--version`, `-v`)
- Strict error exit codes:
  - Exit code `0`: Successful evaluation
  - Exit code `1`: Missing arguments or invalid syntax
  - Exit code `2`: Semantic/mathematical runtime error (division by zero)

## 3. Source Files
- `verification/task-16/bmd_calc.bmd`

## 4. Execution Commands & Exit Codes

```bash
# 1. Help message
./bramand run verification/task-16/bmd_calc.bmd --help
# Exit Code: 0

# 2. Version banner
./bramand run verification/task-16/bmd_calc.bmd --version
# Exit Code: 0 (Output: bmd-calc v0.1.0)

# 3. Addition
./bramand run verification/task-16/bmd_calc.bmd add 10 20
# Exit Code: 0 (Output: 30)

# 4. Subtraction
./bramand run verification/task-16/bmd_calc.bmd subtract 30 5
# Exit Code: 0 (Output: 25)

# 5. Multiplication
./bramand run verification/task-16/bmd_calc.bmd multiply 4 6
# Exit Code: 0 (Output: 24)

# 6. Division
./bramand run verification/task-16/bmd_calc.bmd divide 20 4
# Exit Code: 0 (Output: 5)

# 7. Division by zero error
./bramand run verification/task-16/bmd_calc.bmd divide 20 0
# Exit Code: 2 (Output: Error: Cannot divide by zero.)

# 8. Missing arguments error
./bramand run verification/task-16/bmd_calc.bmd add 10
# Exit Code: 1 (Output: Error: Command 'add' requires 2 numeric arguments.)
```

## 5. Verdict
**Status**: `PASS` (Real execution, CLI arguments builtin `args()` implemented and fully verified).
