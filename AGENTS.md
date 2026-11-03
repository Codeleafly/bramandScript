# AGENTS.md — BramandScript Development Guide for AI Agents

> **Specification Standard**: Tool-agnostic repository instructions for autonomous and pair-programming AI coding agents.

---

## 1. Project Overview & Philosophy
- **Project Name**: BramandScript (`bramand` / `bmd`)
- **Nature**: A high-performance, general-purpose programming language built from scratch with an English, clean, international syntax.
- **Runtime Model**: Stack-based Bytecode Virtual Machine (VM) with Lexer, Pratt Parser, AST, Bytecode Compiler, and Opcode execution engine.
- **Host Platform & Toolchain**: Pure Go Standard Library (`stdlib`). Zero third-party dependencies (`0 external packages`). Completely offline buildable.
- **Target Deployment**: Termux (Android ARM64), Linux, macOS, and Windows.

---

## 2. Directory & Task Tracking Convention (`./date/MMYY/` & `bmd-task` CLI)
Agents must strictly follow the repository's date-based progress tracking system:
1. **Folder Path**: `./date/MMYY/` where `MM` is two-digit month and `YY` is two-digit year (e.g., `date/1126/` for November 2026).
2. **Weekly Task Files**:
   - `First_week_task.md`: Tasks for Days 1 to 7.
   - `Second_week_task.md`: Tasks for Days 8 to 14.
   - Subsequent weeks follow the same convention.
3. **Mandatory CLI Task Tool (`bmd-task` / `node tools/task.js`)**:
   - AI agents MUST NOT manually edit markdown task files to avoid syntax corruption or formatting errors.
   - Always execute the built-in task tool to inspect and update task items:
     - View tasks: `bmd-task list` or `bmd-task list Day 2`
     - View task details: `bmd-task view <task_number_or_keyword>`
     - Mark task complete: `bmd-task check <task_number_or_keyword>`
     - Mark task pending: `bmd-task uncheck <task_number_or_keyword>`
     - Edit task text: `bmd-task edit <task_number_or_keyword> "New Task Spec"`
     - Add new task: `bmd-task add "Task title" --desc="Detailed description" [--includes="..."] [--day=N]` (Note: `--desc` is strictly compulsory)
     - Initialize new month: `bmd-task init [MMYY]`
4. **Live Status Tracking**: Checkboxes `- [ ]` must be updated to `- [x]` using `bmd-task check` as features are implemented and verified with passing unit tests. Never mark a feature complete without passing tests.

---

## 3. Architecture & Core Pipeline
Every BramandScript execution follows this strict Bytecode VM pipeline:
```
Source Code (.bmd)
       │
       ▼
 1. Lexer (Tokens & Locations)
       │
       ▼
 2. Pratt Parser (Abstract Syntax Tree / AST)
       │
       ▼
 3. Bytecode Compiler (Emits Opcodes, Constants Pool, Symbol Table)
       │
       ▼
 4. Bramand VM Engine (Stack-based Virtual Machine, Globals, Frames)
```

### Key Packages
- `token/`: Token specifications, keywords, and identifiers.
- `lexer/`: Unicode-compliant tokenization and lexical scanner.
- `ast/`: AST node structures (Expressions, Statements, Program).
- `parser/`: Top-down operator precedence (Pratt) parser.
- `code/`: Bytecode instructions, opcode definitions, bytecode encoding/decoding.
- `compiler/`: AST-to-Bytecode compiler, symbol scoping, and constant table management.
- `vm/`: Stack-based virtual machine execution loop.
- `object/`: Runtime value representations (Integer, Float, Boolean, String, Array, Hash, Function, Null, Error).
- `repl/`: Interactive terminal REPL.
- `main.go`: Unified CLI entrypoint (`run`, `repl`, `test`, `build`).

---

## 4. Build, Test, and Validation Commands
Every agent MUST execute and verify using the local Go toolchain without internet access:
- **Run all unit tests**:
  ```bash
  go test ./... -v
  ```
- **Build the native binary**:
  ```bash
  go build -o bramand main.go
  ```
- **Run a BramandScript source file**:
  ```bash
  ./bramand run script.bmd
  ```
- **Start the BramandScript REPL**:
  ```bash
  ./bramand repl
  ```

---

## 5. Coding & Guardrail Rules
1. **Zero External Dependencies**: NEVER add third-party modules or run `go get`. All capabilities (networking, crypto, json, math, os, concurrency) MUST use Go's standard library.
2. **Bytecode First**: Do NOT fall back to tree-walk interpretation. All features must be compiled into bytecodes and executed on the Bramand VM.
3. **Documentation Integrity**: Maintain English documentation and comments in all Go code and markdown specifications.
4. **No Fake Implementations**: Every implemented feature must have concrete execution logic and automated unit test coverage.
5. **No Emojis Policy (Compulsory Unicode / ASCII Symbols)**: Never use emojis anywhere in the language codebase, CLI, REPL banners, runtime diagnostics, error handlers, example scripts, or documentation. Emojis compromise international programming language standards and break terminal compatibility. Instead, use clean Unicode symbols (e.g., `✦`, `❯`, `→`, `·`, `●`, `■`, `::`) or standard bracketed ASCII tokens (e.g., `[INFO]`, `[OK]`, `[ERROR]`, `[WARN]`).
6. **Production-Ready Mandate (Direct-to-Release Standard)**: Every feature must be built to commercial, production-ready standards. No half-baked stubs, no toy compromises, and no unhandled edge cases. All code must pass zero-defect scrutiny so that it can be directly published to production without hesitation. REPL banners and CLI outputs must adhere to clean, industry-standard conventions (e.g., Python / Node / Go style minimal branding).
7. **Feature Branching Protocol**: Always create and switch to a dedicated feature branch (e.g., `feature/<feature-name>` or `feat/<day-N>`) before implementing or updating any feature.
8. **Main Branch Protection**: Never develop or commit new feature implementations directly onto the `main`/`master` branch. Keep `main` in a pristine, releasable state at all times.
9. **User Confirmation Protocol**: Upon completing feature implementations, test verifications, or milestones on a feature branch, always summarize progress and prompt the user for review and confirmation before proceeding to the next phase or merging.
10. **Strict 1GB Daily Data Limit & Offline-First Mandate**: The developer has a strict mobile data allotment of **1 GB per day**. Agents must treat internet bandwidth as extremely scarce and precious:
    - **100% Offline-First Development**: BramandScript uses zero third-party dependencies. All compilation, unit tests (`go test ./...`), end-to-end testing (`bramand test`), formatting (`bramand fmt`), and benchmarks MUST be executed strictly locally without internet access.
    - **No Network Bandwidth Waste**: NEVER stream or poll GitHub Actions or network logs continuously (e.g. avoid `gh run watch` or polling loops that consume data rapidly).
    - **Lean Git Transfers**: Minimize git push/fetch payload sizes. Avoid repeatedly committing or pushing large compiled binary blobs across remote branches.
    - **Zero Unnecessary Network Requests**: Never download external assets, packages, or tools unless explicitly requested by the user.
11. **Real-World Verification & Application-Driven Dogfooding Protocol**: When requested by the user to test the language with real-world tasks and modular applications, agents must build genuine, executable test programs and modular applications under `verification/` and `examples/<app_name>/`. If any task reveals missing features, unsupported syntax, or runtime defects, agents MUST implement and fix them directly in the BramandScript compiler, VM, and standard library to commercial production standards with zero external dependencies, automated unit tests, and thorough documentation.
12. **Modular Standard Library vs Lean VM Principle**: Keep the VM core execution loop and opcode set minimal, compact, and fast. High-level domain logic, mathematical algorithms, and collection utilities must NOT be added as hardcoded Go VM builtins. Instead, organize them into clean, modular `.bmd` packages under `stdlib/` (e.g., `stdlib/math/`, `stdlib/strings/`, `stdlib/collections/`, `stdlib/io/`, `stdlib/json/`, `stdlib/testing/`). The compiler must natively resolve `import "std/<module>"` from standard library search locations.
13. **Modular Documentation Architecture Mandate**: Never dump language documentation into a single giant monolithic HTML file. Web documentation must be structured into dedicated, topic-focused pages (`index.html`, `fundamentals.html`, `data-structures.html`, `concurrency.html`, `stdlib.html`, `networking.html`, `tooling.html`, `architecture.html`) with shared responsive navigation, live cross-page search, and dark/light mode persistence.
14. **OOP Struct State Synchronization Protocol**: Structs in BramandScript instantiate as Hash maps containing method closures. All struct methods must maintain strict bidirectional state synchronization with their instance Hash:
    - *On Method Call*: Instance Hash properties must be synchronized into the closure's `Free` variables before execution begins.
    - *On Method Return*: Modified `Free` variables must be written back to the instance Hash upon `OpReturnValue` and `OpReturn`.
    - This eliminates state desynchronization between method mutations and direct property mutations (`c.val = 50; c.inc()`).
15. **Thread-Safety for Concurrency & Shared Memory**: Runtime objects shared across concurrent tasks (`spawn`) or multi-threaded server requests—specifically `object.Hash` and `object.Channel`—MUST be protected with synchronization primitives (`sync.RWMutex` / `sync.Mutex`). Never expose raw Go map reads/writes to concurrent VM executions to prevent fatal Go runtime panics (`concurrent map writes`).
16. **Recursion Depth & Cyclic Reference Protection Guardrail**: All recursive runtime operations—including `ObjectsEqual()` and `Inspect()` on nested reference types (`Array`, `Hash`)—MUST enforce depth limits (e.g., `depth > 64` or `depth >= 16`) and pointer identity shortcuts (`l == r`). Circular references (`a[0] = a; a == a`) must never exhaust the call stack or trigger 1GB Go stack overflow crashes. VM frame allocation (`pushFrame`) must guard against `vm.framesIndex >= MaxFrames` and return a clean BramandScript stack overflow error.
17. **Short-Circuit Evaluation & Compound Assignment Single-Evaluation**: Logical operators (`&&`, `||`) must compile to conditional jump opcodes (`OpJumpNotTruthy`, `OpJump`) with true short-circuit behavior; never evaluate the right operand eagerly. Compound index assignments (`arr[0] += 5`, `dict["key"] += 10`) must evaluate container and key expressions exactly once using temporary scoped variables to eliminate duplicate side effects.
18. **Production Security & Cryptographic Defaults (Strict TLS & SNI)**: Network clients and server connections (e.g., WebSocket `wss://`, HTTPS) must NEVER use insecure bypasses like `InsecureSkipVerify: true` by default. Always enforce valid TLS certificate verification and set `ServerName` for Server Name Indication (SNI).
19. **Module Export Completeness (`const`, `let`, `fn`, `struct`)**: When importing modules with a namespace alias (`import "module" as mod`), the compiler must export all top-level statements including `LetStatement`, `ConstStatement`, named `FunctionLiteral`, and `StructStatement`. Never omit `ConstStatement` or other declarations when assembling the exported module hash.
