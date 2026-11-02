# BramandScript (bmd)

> A high-performance, general-purpose programming language built from scratch with a Stack-Based Bytecode Virtual Machine in pure Go standard library. Zero external dependencies. Completely offline buildable.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Codeleafly/bramandScript)](https://goreportcard.com/report/github.com/Codeleafly/bramandScript)
[![Documentation](https://img.shields.io/badge/Docs-Live%20Website-green.svg)](https://codeleafly.github.io/bramandScript/)

---

## ✦ Overview

**BramandScript** (`bmd`) is designed for modern developers and young learners alike. It combines the expressive simplicity of Python with the runtime predictability and efficiency of a stack-based Bytecode Virtual Machine, requiring **zero third-party dependencies** (`0 external packages`).

- **Live Documentation**: [https://codeleafly.github.io/bramandScript/](https://codeleafly.github.io/bramandScript/)
- **Host Runtime**: Pure Go Standard Library (`stdlib` only)
- **Target Environments**: Android Termux (ARM64), Linux (x86_64/ARM64), macOS, and Windows

---

## ✦ Key Features

- **Beginner-Friendly Ergonomics**: Optional semicolons, optional `if` condition parentheses, natural mathematical operators (`**` for power, `%` for modulo).
- **Stack-Based Bytecode VM**: Complete custom Pratt Parser, AST compiler, constant pools, instruction set (`code/`), and frame execution engine (`vm/`).
- **Modern Collections & Dot Access**:
  - Lists: `let list = [10, 20, 30]; list[1] = 99`
  - Hashes: `let user = {"name": "Aryan", "role": "dev"}`
  - Direct Dot Access: `print(user.name); user.role = "lead"`
  - Universal `.length` / `.len` properties across strings, arrays, and maps.
- **For-In Iterators**: Native `for item in list`, `for idx, val in list`, and string/hash iteration with `break` and `continue`.
- **First-Class Functions & Mutable Closures**: Lexical scoping, first-class `fn` objects, free-variable capture, and mutable state (`OpClosure`, `OpGetFree`, `OpSetFree`).
- **Structs & Object-Oriented Methods**:
  ```bmd
  struct Point {
      x, y
      fn dist() {
          return (x * x + y * y) ** 0.5
      }
  }
  let p = Point(3, 4)
  print(p.dist()) // 5.0
  ```
- **Pattern Matching**:
  ```bmd
  let msg = match statusCode {
      case 200 => { "OK" }
      case 400, 404 => { "Client Error" }
      default => { "Server Error" }
  }
  ```
- **Deno-Style Centralized Module Architecture**:
  - Direct import: `import "helpers.bmd"`
  - Namespaced import: `import "calc.bmd" as calc`
  - Centralized module caching at `$HOME/.bramand_libs/` avoiding `node_modules` per-project bloat.
- **Rust/Clang-Style 10-Line Diagnostic Engine**: Exact column and line pointer (`--> file:line:col`), exactly 5 lines of contextual code above and 5 below, caret token underlining (`^~~~`), intelligent suggestions (`[Hint: ...]`), and zero ASCII box borders.
- **In-Terminal Multi-Line Editor (`:editor`)**: Multi-line coding, running, and saving directly to disk from inside the interactive REPL.

---

## ✦ Quickstart & Installation

### Option 1: One-Step Automated Installation (Termux / Linux / macOS)
```bash
git clone https://github.com/Codeleafly/bramandScript.git
cd bramandScript
bash install.sh
```
This builds the binary using local Go toolchain and creates global symlinks for `bramand`, `bmd`, and the task CLI `bmd-task`.

### Option 2: Manual Go Build
```bash
go build -o bramand main.go
./bramand --help
```

---

## ✦ Usage

### 1. Interactive REPL
```bash
bmd
# or
bramand repl
```
Commands inside the REPL:
- `:editor` — Enter multi-line code editor mode
- `:editor <filename.bmd>` — Code and save directly to file
- `:clear` — Clear terminal screen
- `:reset` — Reset runtime state and variables
- `:help` — Comprehensive documentation and syntax guide
- `:exit` — Quit REPL

### 2. Execute a Source File
```bash
bmd run script.bmd
# or shortcut:
bmd script.bmd
```

### 3. Run All Automated Unit Tests
```bash
go test ./... -v
```

---

## ✦ Code Example

```bmd
// Struct declaration with closure methods
struct Counter {
    val
    fn inc() {
        val += 1
        return val
    }
}

let c = Counter(10)
c.inc()
c.inc()
print("Counter value:", c.val)

// Pattern matching
let status = 200
let response = match status {
    case 200 => { "Success" }
    case 404 => { "Not Found" }
    default => { "Unknown" }
}
print("Status:", response)

// For-in loop
let fruits = ["Apple", "Mango", "Banana"]
for i, item in fruits {
    print("Item", i, "is", item)
}
```

---

## ✦ Architecture Pipeline

```
Source Code (.bmd)
       │
       ▼
 1. Lexer (Tokens, Source Locations, String Escapes)
       │
       ▼
 2. Pratt Parser (Abstract Syntax Tree / AST Nodes)
       │
       ▼
 3. Bytecode Compiler (Symbol Table, Constants Pool, Opcodes)
       │
       ▼
 4. Bramand VM Engine (Stack Machine, Call Frames, Closures)
```

---

## ✦ Contributing

Contributions are warmly welcomed! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on code standards, our strict **Zero External Dependencies** rule, and our **No Emojis Policy**.

---

## ✦ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.  
Copyright (c) 2026 Codeleafly (BramandScript Authors and Contributors).
