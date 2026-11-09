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
- **Pattern Matching & Destructuring**:
  ```bmd
  let [first, second, ...rest] = [10, 20, 30, 40]
  let { name, role } = { "name": "Bramand", "role": "Architect" }

  let msg = match statusCode {
      case 200 => { "OK" }
      case 400, 404 => { "Client Error" }
      default => { "Server Error" }
  }
  ```
- **Pipeline Operator (`|>`)**:
  Fluent data-first chaining lowered directly to nested call chains at compile time with zero overhead:
  ```bmd
  let result = 3 |> inc |> square // square(inc(3)) -> 16
  ```
- **Adaptive Numeric Core & Transparent BigInt**:
  Standard integer arithmetic automatically promotes to arbitrary-precision `BigInt` upon 64-bit overflow (`2 ** 70`), eliminating silent overflow wrapping while maintaining unboxed int64 speed on small integers.
- **Scientific Computing & Linear Algebra Suite**:
  - `std/math/decomp`: LU decomposition with row pivoting, QR via Householder reflections, Cholesky factorization, linear solvers (`A * x = b`), and polynomial least-squares regression.
  - `std/math/eigen`: Spectral analysis, power iteration, and shifted QR eigenvalue algorithms.
  - `std/math/sparse`: Compressed Sparse Row (CSR) matrix representation and sparse matrix-vector multiplication.
  - `std/math/calculus` & `std/math/opt` & `std/math/ode`: Numerical differentiation, Simpson/Gauss quadrature integration, Brent-Dekker root finding, scalar minimization, and RK4 ODE integrators.
  - `std/math/rational` & `std/math/decimal`: Exact fractional arithmetic with automatic GCD reduction and fixed/arbitrary-scale financial decimals.
  - `std/math/complex` & `std/math/ntheory`: Complex Cartesian/polar numbers, Euler identity verification, deterministic Miller-Rabin primality, Pollard-rho, and modular inverse.
- **Modern Cryptography & Binary Protocols**:
  - `std/crypto/advanced`: Authenticated symmetric encryption (AES-256-GCM), Ed25519 digital signatures, and JSON Web Token (JWT) signing and verification.
  - `std/binary/pack`: Python-style binary struct packing (`>hhi4s`) and MessagePack binary serialization.
  - `std/compress`: Pure Go stdlib gzip, zlib, and POSIX tar archiving without external CLI tools.
- **Terminal Visualizations (`std/tui/plot`)**:
  High-resolution 2x4 Braille dot-matrix mathematical function curves and Unicode box bar charts directly inside the terminal.
- **64-Bit Opcode Virtual Machine**: Full 64-bit (`uint64`, 8-byte) operand encoding across constants (`OpConstant`), branch jumps (`OpJump`, `OpJumpNotTruthy`, `OpJumpTruthy`), globals (`OpSetGlobal`, `OpGetGlobal`), and closures (`OpClosure`), supporting unbounded programs with millions of instructions.
- **Dynamic Stack & Configurable Call Frames**: Frames and value stack dynamically expand to accommodate deep recursions, with user-configurable bounds (`--max-frames <N>`, `--max-stack <N>`).
- **Capability-Based Permission Guards**: Host environment security protecting subprocess execution (`--allow-run`), filesystem writes (`--allow-write`), and network listening (`--allow-net`), or complete capability access (`--allow-all`).
- **Structured Exception Handling (try-catch-finally)**: First-class grammar statements for resilient error interception:
  ```bmd
  try {
      let val = 100 / 0
  } catch err {
      println("Intercepted error:", err)
  } finally {
      println("Guaranteed cleanup")
  }
  ```
- **Enterprise Language Operators & Syntax**:
  - **Null-Coalescing (`??`) & Nullish Assignment (`??=`)**:
    ```bmd
    let port = config.port ?? 8080
    settings["timeout"] ??= 30
    ```
  - **Optional Chaining (`?.`)**: Short-circuiting property, index, and method access on nullable targets:
    ```bmd
    let name = user?.profile?.name
    let item = matrix?.[row]?.[col]
    let result = service?.compute?.(x, y)
    ```
  - **Ternary Conditional Expression (`?:`)**:
    ```bmd
    let status = active ? "online" : "offline"
    let grade = score >= 90 ? "A" : score >= 80 ? "B" : "C"
    ```
  - **String Interpolation (`f"..."` & Template Literals)**:
    ```bmd
    let greeting = f"Hello, {user.name}! Score is {score + 10}."
    let banner = `Server running on ${host}:${port}`
    ```
  - **Enterprise Call Stack Trace Inspection**:
    ```bmd
    try {
        processOrder(orderId)
    } catch err {
        println("Error message:", err.message)
        println("Stack trace:\n", err.stack_trace)
        for frame in err.frames {
            println("  at", frame.fn, "line", frame.line)
        }
    }
    ```
  - **Enterprise Standard Library Modules**:
    - `import "std/log" as log` — Leveled structured logger with JSON and console text outputs
    - `import "std/retry" as retry` — Exponential backoff retry engine with jitter and error filters
    - `import "std/config" as config` — Environment variable loader, `.env` dotenv parser, and INI configuration parser
    - `import "std/collections" as coll` — `partition`, `key_by`, `flat_map`, `compact`, `sum_by`, `min_by`, `max_by`, `frequencies`, `difference`, `intersection`, `union`
  - **LIFO `defer` Statement Runtime**:
    Guaranteed LIFO-ordered cleanup executed upon normal function returns, early returns, and exception unwinding:
    ```bmd
    fn processFile(path) {
        let file = open(path)
        defer file.close()

        let mu = mutex()
        mu.lock()
        defer mu.unlock()

        // processing logic...
    }
    ```
  - **Default Parameter Values in Functions**:
    Declarative default parameter expressions with preceding parameter reference capability:
    ```bmd
    fn connect(host, port = 8080, timeout = 30) {
        return f"Connecting to {host}:{port} (timeout {timeout}s)"
    }
    connect("localhost") // port defaults to 8080, timeout to 30
    ```
  - **Array & Hash Spread Expressions (`...`)**:
    In-line unpacking and deterministic dictionary merging with progressive overriding:
    ```bmd
    let items = [1, ...numbers, 4]
    let config = { ...defaults, "port": 9000, "debug": true }
    ```
  - **Enterprise Resiliency & Concurrency Standard Library**:
    - `import "std/uuid" as uuid` — RFC 4122 v4 UUID generation, parsing, and format validation
    - `import "std/semaphore" as semaphore` — Counting semaphore concurrency limiter with `with_permit()`
    - `import "std/circuitbreaker" as circuitbreaker` — Fault-tolerant circuit breaker (Closed, Open, Half-Open) with automatic recovery timeouts and fallbacks
    - `import "std/pool" as pool` — High-throughput worker pool abstraction with order-preserving parallel `map()`
- **AOT Constant Folding & Peephole Optimizer**:
  Compiler optimization pass (`bmd run -O script.bmd`, `bmd build -O app.bmd`) pre-evaluating constant arithmetic expressions and compacting bytecode opcodes.
- **Modular Standard Library (`stdlib/`)**:
  - Lean VM core with dedicated high-level standard library modules written in BramandScript:
  - `import "std/math" as math` (constants `PI`, `E`, `clamp`, `gcd`, `lcm`, `min`, `max`)
  - `import "std/strings" as str` (`capitalize`, `pad_left`, `pad_right`, `slugify`, `reverse`)
  - `import "std/collections" as coll` (`map`, `filter`, `reduce`, `find`, `unique`, `chunk`, `group_by`)
  - `import "std/iter" as iter` (resumable coroutine generators, `yield`, lazy stream evaluation)
  - `import "std/io" as io` (`read_text`, `write_text`, `append_text`, `exists`, `remove`)
  - `import "std/json" as json` (`parse`, `stringify`, `safe_parse`)
  - `import "std/testing" as t` (`assert_true`, `assert_false`, `assert_null`)
- **Centralized Package & Module Architecture (`$HOME/.bmd_libs/`)**:
  - **Zero `node_modules` Bloat**: Shared system-wide store eliminating per-project disk waste.
  - **4-Stage Import Resolution Pipeline**: Stage 1 (`embed:stdlib/`), Stage 2 (relative paths), Stage 3 (`~/.bmd_libs/registry/`), Stage 4 (immutable singleton exports with `OpModule`).
  - **Three-Tier Fallback Engine**: `bmdpkg.json` "entry" -> `main.bmd` -> `index.bmd` -> diagnostic failure.
  - **SemVer Diamond Resolution**: Unifies compatible dependency versions to highest SemVer with side-by-side major version isolation.
- **Enterprise Concurrency, Actors & Monadic Safety (Week 5)**:
  - **Date, Time & Calendar Suite (`std/datetime`)**: High-performance Date, Time, Duration, Timestamp, Unix milliseconds, ISO 8601 / RFC 3339 parsing and formatting, calendar arithmetic (`add_days`, `add_hours`, `diff`), timezone offsets, and leap year checks.
  - **Enterprise File System & Path (`std/fs` & `std/path`)**: Recursive directory traversal (`fs_walk`), glob matching, temp file creation, and path manipulation (`join`, `clean`, `base`, `dir`, `ext`, `abs`, `split`).
  - **Universal Data Encoding Suite (`std/encoding`)**: CSV parsing/formatting with headers and custom delimiters, Hex encoding/decoding, and URL query encoding/decoding.
  - **Lightweight Actor Model & Supervision Trees (`actor`, `std/actor`)**: `actor` keyword, FIFO mailboxes, async `act.send(msg)`, synchronous `act.ask(msg, timeout_ms)`, global registry (`actor_register`, `actor_whereis`), and supervision trees (`one_for_one`, `one_for_all`, `rest_for_one`).
  - **Monadic Safety (`Result` & `Option`)**: `Ok(v)`, `Err(e)`, `Some(v)`, `None()` runtime object types with methods (`.unwrap()`, `.unwrap_or()`, `.is_ok()`, `.is_err()`, `.map()`, `.flat_map()`).
  - **Try-Propagation Operator (`?`)**: Postfix `?` unwraps `Ok`/`Some` or early-returns `Err`/`None` from enclosing functions.
  - **Fixed Tuples & Destructuring**: Immutable `(a, b, c)` tuple runtime object with literal syntax, `OpTuple` opcode, and multi-variable destructuring (`let (x, y, z) = point`).
  - **Deep Pattern Matching with Guard Expressions**: Extended `match-case` with nested tuple, array, and hash/struct destructuring and `if <condition>` guard clauses.
  - **Uncolored Async/Await Coroutines**: `async fn` and `await` compiling to suspendable VM call frames allowing any caller to await without function coloring.
  - **Cooperative Scheduler & Structured Task Groups (`task_group`, `std/concurrency`)**: `yield` voluntary time-slicing and `task_group()` coordinator supporting parallel branch execution, `wait_all()`, `wait_any()`, and cascade cancellation on failure.
  - **Supply-Chain Permission Contracts**: Sandboxing declared host permissions (`network`, `filesystem_read`, `filesystem_write`, `subprocess`) with install-time security alerts.
  - **Single-Binary Embedding (`bmd build`)**: Direct embedding of global dependencies into self-contained standalone binaries.
  - **Offline-First Determinism**: Complete offline operation via `--offline` flag and `BMD_OFFLINE=1`.
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

### 2. Execute a Source File & Permission Options
```bash
# Standard execution
bmd run script.bmd

# Execute with explicit security permission flags
bmd run --allow-all script.bmd          # Allow all permissions
bmd run --allow-run script.bmd          # Allow exec()
bmd run --allow-write script.bmd        # Allow write_file(), delete_file()
bmd run --allow-net server.bmd          # Allow tcp_listen()

# Custom call stack depth limit
bmd run --max-frames 2048 deep.bmd

# Direct shortcut execution
bmd script.bmd
```

### 3. Centralized Package Manager (`bmd pkg`, `bmd install`, `bmd get`, `bmd cache`)

BramandScript provides a unified, offline-first package manager with centralized storage in `$HOME/.bmd_libs/`:

```bash
# Install package from GitHub (version tag or latest)
bmd install github.com/alice/json-tools
bmd install github.com/alice/json-tools 1.2.0

# Install from remote CDN archive URL
bmd install https://cdn.bmd.io/packages/matrix-math.tar.gz

# Link local development directory
bmd install ./libs/my_utils
bmd install local/my_utils@dev

# Synchronize project dependencies declared in bmdpkg.json
bmd get

# Offline dependency synchronization (strictly local/cache)
bmd get --offline

# Inspect centralized storage metrics and counts
bmd cache info

# Prune orphaned tarball archives while preserving registry trees
bmd cache prune

# Cryptographically verify all installed packages against integrity.db
bmd cache verify

# Initialize a new package with bmdpkg.json manifest and main.bmd
bmd pkg init my-library

# List project dependencies and globally available registry packages
bmd pkg list

# Compile standalone executable embedding global dependencies
bmd build app.bmd -o my_app
```

#### Package Manifest (`bmdpkg.json`)
```json
{
  "name": "my-service",
  "version": "1.0.0",
  "description": "High-throughput microservice in BramandScript",
  "entry": "src/main.bmd",
  "license": "MIT",
  "authors": ["Codeleafly <ziprecovery2024@gmail.com>"],
  "permissions": {
    "network": true,
    "filesystem_read": true,
    "filesystem_write": false,
    "subprocess": false
  },
  "dependencies": {
    "github.com/alice/json-tools": "^1.2.0"
  },
  "compatibility": {
    "bmd_min": "0.1.0"
  }
}
```

#### CLI Command Cheat Sheet
| Command | Arguments | Purpose & Functionality |
| :--- | :--- | :--- |
| `bmd install` | `<target> [version] [--offline]` | Installs package from GitHub, CDN, or local path into `~/.bmd_libs/registry/` |
| `bmd get` | `[dir] [--offline]` | Resolves and downloads missing project dependencies from `bmdpkg.json` |
| `bmd cache info` | *(none)* | Displays disk usage, tarball archive count, and registry package count |
| `bmd cache prune` | *(none)* | Reclaims space by removing orphaned tarballs while preserving registry trees |
| `bmd cache verify` | *(none)* | Validates SHA-256 cryptographic checksums against `locks/integrity.db` |
| `bmd pkg init` | `[name]` | Scaffolds a new package directory with `bmdpkg.json` and `main.bmd` |
| `bmd pkg list` | *(none)* | Lists dependencies in current project and all globally installed packages |
| `bmd build` | `<file.bmd> -o <exe>` | Compiles self-contained executable embedding dependencies from `~/.bmd_libs/` |

### 4. Run All Automated Unit Tests
```bash
# Pure Go test suite
go test ./... -v

# BramandScript test suite
bmd test tests/
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

// Pattern matching & Destructuring
let [head, ...tail] = [10, 20, 30, 40]
let { name, role } = { "name": "Bramand", "role": "Architect" }

let status = 200
let response = match status {
    case 200 => { "Success" }
    case 404 => { "Not Found" }
    default => { "Unknown" }
}
print("Status:", response)

// Pipeline Operator (|>)
let inc = fn(x) { return x + 1; }
let sq = fn(x) { return x * x; }
let res = 3 |> inc |> sq
print("Pipeline (3 + 1)^2:", res) // 16

// For-in loop
let fruits = ["Apple", "Mango", "Banana"]
for i, item in fruits {
    print("Item", i, "is", item)
}
```

---

## ✦ Modular Documentation Portal

The documentation is organized into dedicated, fast-loading modular guides:
- [Getting Started & Overview](https://codeleafly.github.io/bramandScript/) — Installation, CLI commands, permission flags, and REPL `:editor` mode.
- [Language Fundamentals](https://codeleafly.github.io/bramandScript/fundamentals.html) — Syntax, variables, operators (null-coalescing `??`, optional chaining `?.`, ternary `?:`), default parameters, LIFO `defer`, and `try-catch-finally` exceptions.
- [Data Structures & OOP](https://codeleafly.github.io/bramandScript/data-structures.html) — Arrays, hash maps, dot access, structs, spread operators (`...`), enums, and pattern matching.
- [Concurrency & System I/O](https://codeleafly.github.io/bramandScript/concurrency.html) — Lightweight tasks (`spawn`), channels, mutexes, counting semaphores (`std/semaphore`), worker pools (`std/pool`), and file I/O.
- [Modular Standard Library](https://codeleafly.github.io/bramandScript/stdlib.html) — 21 embedded modules: `std/math`, `std/strings`, `std/collections`, `std/json`, `std/io`, `std/log`, `std/retry`, `std/config`, `std/uuid`, `std/semaphore`, `std/circuitbreaker`, `std/pool`, BigInt/BigFloat, and 180+ builtins reference.
- [Networking, Web & AI](https://codeleafly.github.io/bramandScript/networking.html) — HTTP server/client, WebSockets, TCP/UDP sockets, resilient network requests, AI streaming, and vector search.
- [Developer Tooling](https://codeleafly.github.io/bramandScript/tooling.html) — Centralized package manager (`bmd install`, `bmd get`, `bmd cache`), test runner (`bmd test`), formatter (`bmd fmt`), compiler (`bmd build`), disassembler, profiler, and permission flags.
- [Architecture & Internals](https://codeleafly.github.io/bramandScript/architecture.html) — 4-stage runtime import resolution pipeline, inter-process atomic locking, SemVer diamond dependency resolution, 64-bit opcode VM, dynamic stack frames, thread safety, embedded filesystem (`embed.FS`), diagnostics engine, and contributing guide.
- [Runtime & Diagnostics Reference](https://codeleafly.github.io/bramandScript/errors.html) — Complete error catalog (`SyntaxError`, `CompileError`, `RuntimeError`, `PermissionDenied`, `ConcurrencyError`, `PackageManagerError`), visual diagnostic anatomy, and defensive recovery strategies.

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
