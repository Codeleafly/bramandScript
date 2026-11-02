# Verification Task 12: Terminal Input and Interactive Programs

## 1. Test Objective
Verify BramandScript's ability to receive terminal input from standard input (`input()`, `read_line()`), handle interactive prompts, parse numeric input (`int()`, `float()`), and operate accurately when driven by stdin pipes or redirections.

## 2. Features Tested
- Interactive terminal prompt display (`input("Enter first number: ")`)
- Sequential line reading across multiple invocations without losing buffered bytes
- String to numeric parsing via `float(...)`
- Arithmetic evaluation based on parsed operator tokens
- Piped standard input processing (`printf "12\n+\n8\n" | ./bramand run ...`)

## 3. Source Files
- `verification/task-12/calculator.bmd`

## 4. Execution Command & Exit Code
```bash
printf "12\n+\n8\n" | ./bramand run verification/task-12/calculator.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (`Result: 20`)

```bash
printf "15\n*\n4\n" | ./bramand run verification/task-12/calculator.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (`Result: 60`)

## 5. Actual Output
```
✦ Interactive BramandScript Calculator
Enter first number: Enter operator (+, -, *, /): Enter second number: Result: 20
```

## 6. Verdict
**Status**: `PASS` (Discovered critical stdin buffer desynchronization bug across multiple `input()` calls; implemented unified `stdinReader` with thread-safe locking).
