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
     - Mark task complete: `bmd-task check <task_number_or_keyword>`
     - Mark task pending: `bmd-task uncheck <task_number_or_keyword>`
     - Add new task: `bmd-task add "Task name" --day=N`
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
