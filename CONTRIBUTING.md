# Contributing to BramandScript

> **Standard**: Production-Grade, Zero External Dependencies, Offline-First Language Architecture.

Thank you for your interest in contributing to **BramandScript** (`bmd`)! BramandScript is designed from scratch to be a high-performance, general-purpose programming language powered by a stack-based Bytecode Virtual Machine, implemented strictly using the standard library of Go (`stdlib`).

Please read this document carefully before proposing architectural changes, filing issues, or submitting pull requests.

---

## 1. Guiding Philosophy & Architectural Rules

1. **Zero Third-Party Dependencies (0 external packages)**:
   - The entire compiler toolchain, bytecode virtual machine, lexer, Pratt parser, runtime, standard library, and CLI must build using standard Go library tools only.
   - Do NOT run `go get` or add dependencies to `go.mod`. Any external dependency is grounds for immediate PR rejection.
2. **Bytecode-First Execution Pipeline**:
   - Every language feature must compile to bytecode instructions (`code/`) executed on the stack-based VM (`vm/`).
   - Tree-walk interpreters or AST-walking evaluation are strictly prohibited.
3. **No Emojis Policy (Compulsory Unicode / ASCII Standard)**:
   - Never use emojis anywhere in the codebase, CLI prompts, REPL banners, runtime diagnostics, error handlers, examples, or documentation.
   - Use clean Unicode typographic symbols (such as `✦`, `❯`, `→`, `·`, `●`, `■`, `::`) or bracketed status tokens (e.g., `[INFO]`, `[OK]`, `[ERROR]`, `[WARN]`).
4. **Direct-to-Release Quality Mandate**:
   - Every commit and pull request must represent complete, zero-defect code.
   - Stubs, placeholder mocks, or half-implemented features are not accepted.
   - Every feature must be backed by unit tests with 100% pass status.
5. **Cross-Platform Compatibility**:
   - BramandScript targets Termux (Android ARM64), Linux (x86_64 / aarch64), macOS (Apple Silicon / Intel), and Windows. Do not use OS-specific assumptions without standard library abstractions.

---

## 2. Environment Setup

### Prerequisites
- **Go 1.20+** installed on your system or Termux environment.
- Git.
- Optional: Node.js 16+ (for `tools/task.js` / `bmd-task` development roadmap tracker).

### Clone the Repository
```bash
git clone https://github.com/Codeleafly/bramandScript.git
cd bramandScript
```

### Build Locally
To compile the unified native CLI binary:
```bash
go build -o bramand main.go
```

Verify the build:
```bash
./bramand version
```
Expected output:
```
BramandScript 0.1.1 (arm64-linux)
```

---

## 3. Running the Test Suite

Before making any changes, ensure all existing unit tests pass:
```bash
go test ./... -v
```

To run tests for a specific subsystem:
```bash
# Test Lexer
go test ./lexer -v

# Test Parser
go test ./parser -v

# Test Compiler & Symbol Table
go test ./compiler -v

# Test Virtual Machine & Bytecode Execution
go test ./vm -v

# Test Diagnostics Engine
go test ./diagnostics -v
```

### Coverage Check
To verify test coverage:
```bash
go test -cover ./...
```

---

## 4. Repository Structure & Subsystems

```
bramandScript/
├── token/         ✦ Token definitions, types, and keyword tables
├── lexer/         ✦ Unicode lexical scanner, character streaming
├── ast/           ✦ Abstract Syntax Tree node interfaces, expressions, statements
├── parser/        ✦ Top-down operator precedence (Pratt) parser
├── code/          ✦ Bytecode definitions, Opcode constants, BigEndian encoding
├── compiler/      ✦ AST-to-Bytecode compiler, symbol scoping, constant table
├── vm/            ✦ Stack-based virtual machine, execution loop, call frames
├── object/        ✦ Runtime data types (Integer, Float, String, Closure, Hash, Array)
├── diagnostics/   ✦ Rust/Clang-style visual error engine with context window
├── repl/          ✦ Interactive terminal REPL and multi-line :editor mode
├── examples/      ✦ Standard reference programs (.bmd)
├── tools/         ✦ Internal developer CLI utilities (task.js)
├── date/          ✦ Roadmap and task tracking logs (MMYY format)
├── docs/          ✦ Official web documentation portal
├── main.go        ✦ CLI entrypoint (run, repl, version, help)
└── install.sh     ✦ Automated installer script
```

---

## 5. Development Roadmap & Task CLI (`bmd-task`)

All active developments are logged in `./date/MMYY/First_week_task.md` following project tracking standards.

- Do **not** manually edit markdown checkboxes in `date/MMYY/*.md`.
- Always use the provided tool:
  ```bash
  node tools/task.js list           # View current tasks and progress
  node tools/task.js check <number> # Mark task as complete [x] after tests pass
  node tools/task.js add "..."      # Add a new task item
  ```

---

## 6. Coding Standards & Conventions

### Go Code Style
- Run `gofmt -s -w .` before committing code.
- Follow idiomatic Go naming conventions: camelCase for private variables, PascalCase for exported identifiers.
- Errors must be informative and returned cleanly; do not panic in library packages (`lexer`, `parser`, `compiler`, `vm`).
- Provide line and column numbers for syntax, compilation, and runtime errors to integrate with the `diagnostics` package.

### Commit Messages
Use the Conventional Commits specification:
- `feat: add support for bitwise shift operators`
- `fix: resolve closure variable capture in nested functions`
- `test: add unit test for hash table index assignment`
- `docs: update standard library builtins reference`
- `refactor: optimize stack frame allocation in VM`

---

## 7. Step-by-Step Pull Request Workflow

1. **Fork and Branch**:
   Create a dedicated branch for your feature or bug fix:
   ```bash
   git checkout -b feat/your-feature-name
   ```

2. **Implement Feature with Tests**:
   - Write unit tests covering both happy paths and edge cases.
   - Implement the minimal, robust logic following the Bytecode VM pipeline.

3. **Verify All Tests Pass**:
   ```bash
   go test ./... -v
   go vet ./...
   ```

4. **Verify Clean Build**:
   ```bash
   go build -o bramand main.go
   ./bramand run examples/day3_features.bmd
   ```

5. **Commit and Push**:
   ```bash
   git add .
   git commit -m "feat: implement feature with comprehensive tests"
   git push origin feat/your-feature-name
   ```

6. **Submit PR**:
   - Fill out the PR template with clear description, rationale, and sample `.bmd` code snippet.
   - Ensure continuous integration checks pass without warnings.

---

## 8. License

By contributing to BramandScript, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
