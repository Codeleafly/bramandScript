# Verification Task 15: Exceptions, Exit Codes, and Debugging

## 1. Test Objective
Verify BramandScript's diagnostic reporting, error handling mechanism, line/column coordinate precision, and process exit code propagation. Determine whether `try/catch` or an alternate error pattern is supported.

## 2. Findings on Error Handling Model
- **No `try/catch` Grammar**: BramandScript does not feature `try` or `catch` keywords in its Pratt parser or lexer.
- **Idiomatic Error Pattern**: BramandScript follows the modern Go / Rust style Result dictionary pattern (`{"ok": true, "value": ...}` vs `{"ok": false, "error": ...}`) or multi-value returns.
- **Process Exit Codes**: The `exit(code)` builtin terminates the process and delivers the integer status code directly to the host OS shell.
- **10-Line Diagnostic Engine**:
  - Exact file path, line, and column coordinates: `--> file:line:col`.
  - Exactly 5 lines of contextual code above and 5 below the failure point.
  - Caret underline pointer (`^`) highlighting the offending token.
  - Contextual diagnostic hints (e.g. `[Hint: ...]`).

## 3. Source Files
- `verification/task-15/syntax_error.bmd`
- `verification/task-15/runtime_error.bmd`
- `verification/task-15/exit_code.bmd`
- `verification/task-15/result_pattern.bmd`

## 4. Execution Commands & Exit Codes

### Command 1: Syntax Error
```bash
./bramand run verification/task-15/syntax_error.bmd
```
- **Exit Code**: `1`
- **Output**: `error[SyntaxError]: unexpected token ';'` at line 2, col 9.

### Command 2: Runtime Error
```bash
./bramand run verification/task-15/runtime_error.bmd
```
- **Exit Code**: `1`
- **Output**: `error[RuntimeError]: unsupported types for binary operation: INTEGER 4 NULL`

### Command 3: Custom Exit Code
```bash
./bramand run verification/task-15/exit_code.bmd
```
- **Exit Code**: `42`
- **Output**: `Exiting with status code 42`

### Command 4: Idiomatic Result Pattern
```bash
./bramand run verification/task-15/result_pattern.bmd
```
- **Exit Code**: `0`
- **Output**: All result assertions passed cleanly.

## 5. Verdict
**Status**: `PASS` (Real execution, error diagnostic engine and exit codes verified, absence of `try/catch` documented accurately).
