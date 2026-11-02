# Verification Task 19: Environment, Paths, and Portable Applications

## 1. Test Objective
Verify BramandScript's capabilities for building portable environment-aware utilities: reading and mutating environment variables (`env`, `set_env`), safe handling of missing variables, dynamic JSON configuration loading, file cleanup, and building a secret-masking configuration inspection tool.

## 2. Features Tested
- Reading standard system environment variables (`env("HOME")`)
- Missing environment variable handling (returns `null` safely without runtime panic)
- Setting process-level environment variables (`set_env`)
- Writing, reading, and parsing localized configuration files (`write_file`, `read_file`, `json_parse`)
- Production secret-masking utility that sanitizes sensitive keys (`secret`, `password`, `token`) with `********`
- Safe configuration file deletion (`delete_file`)

## 3. Source Files
- `verification/task-19/config_inspector.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-19/config_inspector.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (8/8 assertions passed)

## 5. Actual Output
```
✦ TASK 19: Environment, Paths, and Portable Applications
  ✓ [PASS] HOME environment variable is non-null
  ✓ [PASS] Missing environment variable returns null
  ✓ [PASS] Set and retrieve environment variable
  ✓ [PASS] Config file created
  ✓ [PASS] Config parsed successfully
  ✓ [PASS] Public config key preserved
  ✓ [PASS] Secret config key masked
  ✓ [PASS] Config file cleaned up
[OK] All Task 19 environment, path, and config inspection checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found).
