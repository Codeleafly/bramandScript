# BramandScript — 20-Task Deep Capability Verification Suite

This directory contains the independent, practical verification suite designed to rigorously validate BramandScript (`bramand` / `bmd`) against 20 real-world functional areas.

---

## ✦ Overview of Verification Groups

### Group A — Claimed Capabilities (Tasks 01 to 10)
- **Task 01**: Core Language & Expression Evaluation
- **Task 02**: Functions, Closures, and Control Flow
- **Task 03**: Arrays, Maps, Structs, and Pattern Matching
- **Task 04**: Modules and Package Management
- **Task 05**: File System and Process Execution
- **Task 06**: Concurrency, Channels, and Mutexes (1,000 Deterministic Jobs)
- **Task 07**: BigInt, BigFloat, Matrices, and Statistics
- **Task 08**: JSON, CSV, Base64, and Cryptography
- **Task 09**: Networking and Web Server (TCP, UDP, HTTP, WebSockets)
- **Task 10**: AI Integration and Developer Toolchain

### Group B — Essential Language Basics (Tasks 11 to 15)
- **Task 11**: Terminal Printing, ANSI Colors, and Formatting
- **Task 12**: Terminal Input and Interactive Programs
- **Task 13**: Strings, Type Conversion, and Error Handling
- **Task 14**: Collection Utilities and Gradebook Application
- **Task 15**: Exceptions, Exit Codes, and Debugging

### Group C — Additional Capabilities (Tasks 16 to 20)
- **Task 16**: Command-Line Arguments and CLI Applications (`bmd-calc`)
- **Task 17**: Timers, Date/Time, and Scheduling
- **Task 18**: HTTP Client, Headers, and Network Error Handling
- **Task 19**: Environment, Paths, and Portable Applications
- **Task 20**: Library Quality, Documentation, and Developer Experience

---

## ✦ Running the Verification Suite

Run all verification programs from the repository root:

```bash
# Group A
./bramand run verification/task-01/test_core.bmd
./bramand run verification/task-02/test_functions_control.bmd
./bramand run verification/task-03/student_app.bmd
./bramand run verification/task-04/main.bmd
./bramand run verification/task-05/test_fs_process.bmd
./bramand run verification/task-06/test_concurrency.bmd
./bramand run verification/task-07/test_math_stats.bmd
./bramand run verification/task-08/test_crypto_encoding.bmd
./bramand run verification/task-09/test_networking.bmd
./bramand run verification/task-10/test_ai_toolchain.bmd

# Group B
./bramand run verification/task-11/test_terminal_printing.bmd
printf "12\n+\n8\n" | ./bramand run verification/task-12/calculator.bmd
./bramand run verification/task-13/test_strings_types.bmd
./bramand run verification/task-14/gradebook.bmd
./bramand run verification/task-15/result_pattern.bmd

# Group C
./bramand run verification/task-16/bmd_calc.bmd add 10 20
./bramand run verification/task-17/timer_app.bmd
./bramand run verification/task-18/test_http_client.bmd
./bramand run verification/task-19/config_inspector.bmd
./bramand test verification/task-20/test_library.bmd
```

Consult [`REPORT.md`](./REPORT.md) for detailed analysis, assertion counts, defects identified, and fixes applied.
