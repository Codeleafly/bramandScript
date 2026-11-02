<p align="center">
  <img src="assets/logo.jpg" alt="BramandScript Official Logo" width="160" style="border-radius: 28px; box-shadow: 0 10px 30px rgba(0,0,0,0.35);">
</p>

<h1 align="center">BramandScript (bmd)</h1>

<p align="center">
  <strong>A modern, high-performance general-purpose programming language built from scratch with a Stack-Based Bytecode Virtual Machine in pure Go standard library. Zero external dependencies. Completely offline buildable.</strong>
</p>

<p align="center">
  <a href="https://github.com/Codeleafly/bramandScript/releases/latest"><img src="https://img.shields.io/github/v/release/Codeleafly/bramandScript?color=blue&label=Release" alt="Release"></a>
  <a href="https://github.com/Codeleafly/bramandScript/actions/workflows/ci.yml"><img src="https://github.com/Codeleafly/bramandScript/actions/workflows/ci.yml/badge.svg" alt="CI Testing Matrix"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
  <a href="https://codeleafly.github.io/bramandScript/"><img src="https://img.shields.io/badge/Docs-Live%20Portal-emerald.svg" alt="Documentation"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-orange.svg" alt="Dependencies"></a>
  <a href="https://github.com/Codeleafly/bramandScript/releases"><img src="https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20Termux-purple.svg" alt="Platforms"></a>
  <a href="https://github.com/Codeleafly/bramandScript"><img src="https://img.shields.io/github/stars/Codeleafly/bramandScript?style=social" alt="Stars"></a>
  <a href="https://github.com/Codeleafly/bramandScript"><img src="https://img.shields.io/github/forks/Codeleafly/bramandScript?style=social" alt="Forks"></a>
</p>

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

### Option 1: Smart Universal One-Liner (Linux, macOS, Android Termux)
Auto-detects your platform (Linux, macOS Darwin, Android Termux) and CPU architecture (`amd64`, `arm64`, `armv7`), pulls the latest release, installs the standalone binary, and links `bramand` and `bmd`:
```bash
curl -fsSL https://raw.githubusercontent.com/Codeleafly/bramandScript/master/install.sh | bash
```

### Option 2: Windows PowerShell One-Liner
Open PowerShell and run:
```powershell
irm https://raw.githubusercontent.com/Codeleafly/bramandScript/master/install.ps1 | iex
```

### Option 3: Compile from Source (Offline / Local Toolchain)
```bash
git clone https://github.com/Codeleafly/bramandScript.git
cd bramandScript
go build -ldflags="-s -w" -o bramand main.go

# Install to system PATH
cp bramand "${PREFIX:-/usr/local}/bin/bramand"
ln -sf "${PREFIX:-/usr/local}/bin/bramand" "${PREFIX:-/usr/local}/bin/bmd"
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

## ✦ Community & Stargazers

If you find BramandScript exciting or useful, please consider giving the repository a **Star** on GitHub. It directly supports development and helps other developers discover the language.

[![Star History Chart](https://api.star-history.com/svg?repos=Codeleafly/bramandScript&type=Date)](https://star-history.com/#Codeleafly/bramandScript&Date)

---

## ✦ Discoverability & Tags

`#bramandscript` `#bmd` `#programming-language` `#compiler` `#virtual-machine` `#bytecode` `#stack-machine` `#golang` `#termux` `#zero-dependencies` `#pratt-parser` `#lexer` `#ast` `#developer-tools` `#cli` `#repl` `#opensource`

---

## ✦ Contributing

Contributions are warmly welcomed! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on code standards, our strict **Zero External Dependencies** rule, and our **No Emojis Policy**.

---

## ✦ License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.  
Copyright (c) 2026 Codeleafly (BramandScript Authors and Contributors).
