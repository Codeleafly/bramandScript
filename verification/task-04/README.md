# Verification Task 04: Modules and Package Management

## 1. Test Objective
Verify BramandScript's modular architecture: local module imports, nested directory imports, missing-module compiler diagnostics, package management via `bramand pkg` CLI, centralized caching in `$HOME/.bramand_libs/`, and importing packages directly by identifier.

## 2. Features Tested
- Local module file import (`import "math_lib.bmd"`)
- Nested module path import (`import "nested/helper.bmd"`)
- Global variables and functions exported and called across module boundaries
- Compile-time error when an imported file does not exist (`cannot import module: not found locally or in ~/.bramand_libs`)
- Package management CLI:
  - `bramand pkg install <source_dir> <name@version>`
  - `bramand pkg list`
  - `bramand pkg remove <name@version>`
- Importing packages from `$HOME/.bramand_libs/` by package name (`import "logger@1.0.0"`)

## 3. Source Files
- `verification/task-04/math_lib.bmd`
- `verification/task-04/nested/helper.bmd`
- `verification/task-04/main.bmd`
- `verification/task-04/missing_module.bmd`
- `verification/task-04/test_pkg_import.bmd`
- `verification/task-04/pkg_source/logger@1.0.0/index.bmd`

## 4. Execution Commands & Exit Codes

### Command 1: Local & Nested Module Imports
```bash
./bramand run verification/task-04/main.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (4/4 assertions passed)

### Command 2: Missing Module Import
```bash
./bramand run verification/task-04/missing_module.bmd
```
- **Exit Code**: `1`
- **Result**: PASS (Emitted compile error `cannot import module 'does_not_exist_xyz.bmd'`)

### Command 3: Package Installation & Centralized Cache Import
```bash
./bramand pkg install verification/task-04/pkg_source/logger@1.0.0 logger@1.0.0
./bramand run verification/task-04/test_pkg_import.bmd
./bramand pkg remove logger@1.0.0
```
- **Exit Code**: `0`
- **Result**: PASS (Installed to `~/.bramand_libs/logger@1.0.0`, executed cleanly, then removed)

## 5. Actual Output
```
✦ TASK 04: Modules & Package Management
  ✓ [PASS] Imported function calc_sum
  ✓ [PASS] Imported function calc_product
  ✓ [PASS] Imported global variable
  ✓ [PASS] Nested helper import
[OK] All Task 04 module checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
