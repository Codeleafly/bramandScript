# BramandScript — Month 11/26: First Week Task (Day 1 to Day 7)

**Target**: Complete, General-Purpose Programming Language with Bytecode VM  
**Host Architecture**: Pure Go Standard Library (Zero External Dependencies)  
**Tracking Period**: 01-11-2026 to 07-11-2026  

---

## Day 1: The Core Foundation & Bytecode VM Pipeline
- [x] Complete Token definitions & Unicode Lexer
- [x] Abstract Syntax Tree (AST) node structures
- [x] Pratt Parser (Top-Down Operator Precedence)
- [x] Literals: Integers, Floats, Strings, Booleans, Null
- [x] Prefix & Infix Operators (`+`, `-`, `*`, `/`, `%`, `**`, `!`, `==`, `!=`, `<`, `>`, `<=`, `>=`)
- [x] Logical Operators (`&&`, `||`)
- [x] Variables declarations: `let`, `const`, assignment `=`
- [x] Bytecode Instruction Set (`code/`) with Opcode encoding/decoding
- [x] AST-to-Bytecode Compiler (`compiler/`) with Constants pool & Global Symbol Table
- [x] Stack-based Bytecode Virtual Machine (`vm/`)
- [x] Object runtime system (`object/`)
- [x] Unified CLI with interactive REPL & file execution (`main.go`)
- [x] Comprehensive Unit Tests for Lexer, Parser, Compiler, and VM (100% Passing)

---

## Day 2: Functions, Closures, Scopes & Control Flow
- [x] First-class functions (`fn`)
- [x] Local scope management & stack frames
- [x] Function arguments & return values
- [x] Closures & Free variables (`OpClosure`, `OpGetFree`)
- [x] Conditionals: `if`, `else if`, `else` with bytecode jumps
- [x] Loops: `while` loops, `break`, `continue`
- [x] Stack trace & runtime error reporting
- [x] Day 2 test suite verification

---

## Day 3: Data Structures, Pattern Matching & Structs
- [ ] Arrays and array operations
- [ ] Maps / Dictionaries (Hash tables)
- [ ] Index expressions (`arr[0]`, `map["key"]`)
- [ ] For-in iterators (`for item in list`)
- [ ] Structs & Object-oriented methods
- [ ] Pattern matching (`match` / `switch`)
- [ ] Module system (`import "path"`)
- [ ] Day 3 test suite verification

---

## Day 4: Concurrency, Async/Await & OS/Filesystem
- [ ] Concurrency primitives: `spawn` (lightweight tasks)
- [ ] Channels & message passing (`chan`)
- [ ] Mutexes & lock synchronization
- [ ] File I/O: read, write, append, delete, permissions
- [ ] Process execution & environment variables
- [ ] Standard I/O (stdin, stdout, stderr) & ANSI terminal formatting
- [ ] Day 4 test suite verification

---

## Day 5: Mathematics, Scientific Computing & Serialization
- [ ] Arbitrary-precision BigInt & BigFloat
- [ ] Vectors & Matrices (Linear algebra operations)
- [ ] Trigonometric, Logarithmic & Statistical standard library
- [ ] JSON parser & serializer
- [ ] CSV & Base64 encoding/decoding
- [ ] Cryptography: SHA-256, HMAC, secure random
- [ ] Day 5 test suite verification

---

## Day 6: Networking, Web Framework & AI Integrations
- [ ] TCP & UDP client and server socket APIs
- [ ] HTTP/HTTPS client
- [ ] Built-in HTTP Web Server with routing & middleware
- [ ] WebSocket client and server
- [ ] AI Integration module: HTTP streaming LLM calls, structured JSON outputs
- [ ] Vector similarity / Cosine distance operations for AI embeddings
- [ ] Day 6 test suite verification

---

## Day 7: Developer Tooling, Verification, Docs & Release Candidate
- [ ] `bramand test` CLI test runner
- [ ] `bramand fmt` code formatter
- [ ] Full end-to-end regression & benchmark tests
- [ ] Complete Language Specification & Standard Library Documentation
- [ ] Example programs suite
- [ ] Termux/Linux ARM64 standalone release binary build
