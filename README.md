# BramandScript (bmd) ✦

> A modern, high-performance general-purpose programming language built from scratch with a Stack-Based Bytecode Virtual Machine in pure Go standard library. Zero external dependencies. Completely offline buildable.

[![Release](https://img.shields.io/github/v/release/Codeleafly/bramandScript?color=blue&label=Release)](https://github.com/Codeleafly/bramandScript/releases/latest)
[![CI Testing Matrix](https://github.com/Codeleafly/bramandScript/actions/workflows/ci.yml/badge.svg)](https://github.com/Codeleafly/bramandScript/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://img.shields.io/badge/Docs-Live%20Portal-emerald.svg)](https://codeleafly.github.io/bramandScript/)
[![Dependencies](https://img.shields.io/badge/Dependencies-0%20(Pure%20Stdlib)-orange.svg)](https://golang.org)
[![Platforms](https://img.shields.io/badge/Platforms-Linux%20%7C%20macOS%20%7C%20Windows%20%7C%20Termux-purple.svg)](https://github.com/Codeleafly/bramandScript/releases)
[![Stars](https://img.shields.io/github/stars/Codeleafly/bramandScript?style=social)](https://github.com/Codeleafly/bramandScript)
[![Forks](https://img.shields.io/github/forks/Codeleafly/bramandScript?style=social)](https://github.com/Codeleafly/bramandScript)

---

## ✦ Overview

**BramandScript** (`bmd`) is designed for modern developers and young learners alike. It combines the expressive simplicity of Python with the runtime predictability and efficiency of a stack-based Bytecode Virtual Machine, requiring **zero third-party dependencies** (`0 external packages`).

- **Live Documentation**: [https://codeleafly.github.io/bramandScript/](https://codeleafly.github.io/bramandScript/)
- **Host Runtime**: Pure Go Standard Library (`stdlib` only)
- **Target Environments**: Android Termux (ARM64), Linux (x86_64/ARM64), macOS, and Windows

---

## ✦ Key Features

- **Beginner-Friendly Ergonomics**: Optional semicolons, optional `if` condition parentheses, natural mathematical operators (`**` for power, `%` for modulo).
- **Full UTF-8 Unicode Lexer**: Complete support for international identifiers across scripts (Devanagari, CJK, Greek, accented characters) and rune-accurate string operations and diagnostic pointers.
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
- **Modular Standard Library (`stdlib/`)**:
  - Lean VM core with dedicated high-level standard library modules written in BramandScript:
  - `import "std/math" as math` (constants `PI`, `E`, `clamp`, `gcd`, `lcm`, `min`, `max`)
  - `import "std/strings" as str` (`capitalize`, `pad_left`, `pad_right`, `slugify`, `reverse`)
  - `import "std/collections" as coll` (`map`, `filter`, `reduce`, `find`, `unique`, `chunk`, `group_by`)
  - `import "std/io" as io` (`read_text`, `write_text`, `append_text`, `exists`, `remove`)
  - `import "std/json" as json` (`parse`, `stringify`, `safe_parse`)
  - `import "std/testing" as t` (`assert_true`, `assert_false`, `assert_null`)
- **Deno-Style Centralized Module Architecture**:
  - Standard library import: `import "std/math" as math`
  - Direct file import: `import "helpers.bmd"`
  - Namespaced local import: `import "calc.bmd" as calc`
  - Centralized module caching at `$HOME/.bramand_libs/` avoiding `node_modules` per-project bloat.
- **Rust/Clang-Style 10-Line Diagnostic Engine**: Exact column and line pointer (`--> file:line:col`), exactly 5 lines of contextual code above and 5 below, caret token underlining (`^~~~`), intelligent suggestions (`[Hint: ...]`), and zero ASCII box borders.
- **In-Terminal Multi-Line Editor (`:editor`)**: Multi-line coding, running, and saving directly to disk from inside the interactive REPL.

---

## ✦ Quickstart & Installation

### Option 1: Smart Universal One-Liner (Linux, macOS, Android Termux)
Auto-detects your platform (Linux, macOS Darwin, Android Termux) and CPU architecture (`amd64`, `arm64`, `armv7`), pulls the latest release, installs the standalone binary, and links `bramand` and `bmd`:
```bash
curl -fsSL https://raw.githubusercontent.com/Codeleafly/bramandScript/main/install.sh | bash
```

### Option 2: Windows PowerShell One-Liner
Open PowerShell and run:
```powershell
irm https://raw.githubusercontent.com/Codeleafly/bramandScript/main/install.ps1 | iex
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

## ✦ Modular Documentation Portal

The documentation is organized into dedicated, fast-loading modular guides:
- [Getting Started & Overview](https://codeleafly.github.io/bramandScript/) — Installation, CLI commands, and REPL `:editor` mode.
- [Language Fundamentals](https://codeleafly.github.io/bramandScript/fundamentals.html) — Syntax, variables, data types, operators, and functions.
- [Data Structures & OOP](https://codeleafly.github.io/bramandScript/data-structures.html) — Arrays, hash maps, dot access, structs, and pattern matching.
- [Concurrency & System I/O](https://codeleafly.github.io/bramandScript/concurrency.html) — Lightweight tasks (`spawn`), channels, mutexes, and file I/O.
- [Modular Standard Library](https://codeleafly.github.io/bramandScript/stdlib.html) — `std/math`, `std/strings`, `std/collections`, `std/io`, `std/json`, `std/testing`, BigInt/BigFloat, and builtins reference.
- [Networking, Web & AI](https://codeleafly.github.io/bramandScript/networking.html) — HTTP server/client, WebSockets, TCP/UDP sockets, AI streaming, and vector search.
- [Developer Tooling](https://codeleafly.github.io/bramandScript/tooling.html) — Test runner (`bmd test`), formatter (`bmd fmt`), compiler (`bmd build`), and package manager (`bmd pkg`).
- [Architecture & Internals](https://codeleafly.github.io/bramandScript/architecture.html) — VM pipeline, diagnostics engine, and contributing guide.

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
Copyright (c) 2026 Codeleafly <ziprecovery2024@gmail.com> (https://github.com/Codeleafly). All rights reserved.
