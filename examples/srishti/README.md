# Srishti (सृष्टि) Programming Language

> **Phase 3 Language Architecture Milestone**: A high-performance, general-purpose programming language built 100% in pure BramandScript (`.bmd`) with an English natural syntax and zero external dependencies.

---

## 1. Overview & Philosophy

**Srishti** (Sanskrit: *सृष्टि* — creation, universe) is an expressive, general-purpose, bytecode-compiled programming language designed and implemented entirely within pure BramandScript. It serves as the ultimate real-world dogfooding milestone for BramandScript, establishing language maturity with over **12,000+ lines of native code** and zero host dependency.

### Core Architectural Principles
- **100% Native BramandScript**: Lexer, Pratt Parser, Bytecode Compiler, Optimizer, Stack VM, Runtime Object System, Standard Library, REPL, and CLI are written in `.bmd`.
- **English Natural Syntax**: Natural English operator phrases (`is greater than or equal to`, `combined with`, `divided by`, `repeat while`, `iterate through`) eliminate cryptic punctuation without sacrificing grammar rigor.
- **Stack-based Bytecode Virtual Machine**: Features 35+ specialized bytecode instructions, CallFrames, lexical scoping, closures with free variable capture, and struct instantiation with property mutation.
- **Offline-First & Zero Dependencies**: Fully buildable and testable offline with zero third-party packages or runtime dependencies.
- **Strict Typographic Standard**: 0-emoji policy, utilizing clean typography (`✦`, `❯`, `·`, `::`) and bracketed tokens (`[OK]`, `[ERROR]`, `[INFO]`).

---

## 2. Architecture & Execution Pipeline

```
Srishti Source Code (.sri)
            │
            ▼
 1. Lexer (Scanner & Coordinates)
            │  Tokens, Spans, Keywords, Comments (-- and //)
            ▼
 2. Pratt Parser (Top-Down Precedence)
            │  Abstract Syntax Tree (AST) Program
            ▼
 3. Bytecode Compiler & Optimizer
            │  Opcodes, Constant Pool, Lexical Symbol Table
            ▼
 4. Virtual Machine Engine
            │  CallFrames, Stack, Globals, Closures, Structs
            ▼
 Execution Result / Side Effects
```

### Module Layout
```
examples/srishti/
├── token/               # Token types, coordinates, keyword lookup
│   ├── position.bmd     # File coordinates and source spans
│   ├── token.bmd        # Token definitions and keyword tables
│   └── index.bmd        # Token module exports
├── lexer/               # Lexical scanner and diagnostics
│   ├── unicode.bmd      # Unicode/ASCII character classifiers
│   ├── errors.bmd       # Source line formatting and caret diagnostics
│   ├── lexer.bmd        # Multi-word lookahead scanner
│   └── index.bmd        # Lexer module exports
├── ast/                 # Abstract Syntax Tree hierarchy
│   ├── ast.bmd          # Base AST interfaces and Program node
│   ├── expressions.bmd  # Infix, prefix, call, struct, index nodes
│   ├── statements.bmd   # Let, set, if, loop, match, return nodes
│   ├── printer.bmd      # S-expression and AST tree dumper
│   └── index.bmd        # AST module exports
├── parser/              # Top-down Pratt parser
│   ├── precedence.bmd   # Operator precedence levels
│   ├── errors.bmd       # Syntax error diagnostics
│   ├── parser.bmd       # Indentation-aware Pratt parsing engine
│   └── index.bmd        # Parser module exports
├── compiler/            # Bytecode compiler & symbol resolution
│   ├── opcode.bmd       # VM instruction definitions and disassembly
│   ├── symbol_table.bmd # Lexical scopes (Global, Local, Free, Builtin)
│   ├── bytecode.bmd     # Bytecode and CompiledFunction artifacts
│   ├── optimizer.bmd    # Constant folding and peephole optimizer
│   ├── compiler.bmd     # AST to Bytecode compiler engine
│   └── index.bmd        # Compiler module exports
├── object/              # Runtime object representation
│   ├── object.bmd       # Int, Float, String, Bool, Array, Struct, Closure
│   ├── builtins.bmd     # len, print, println, str, int, float, slice, push
│   └── index.bmd        # Runtime object module exports
├── vm/                  # Virtual machine runtime execution
│   ├── frame.bmd        # CallFrame with closure and instruction pointer
│   ├── vm.bmd           # Stack machine loop with 35+ opcodes
│   └── index.bmd        # VM module exports
├── stdlib/              # Standard library in pure BramandScript
│   ├── math.bmd         # Math constants and scientific functions
│   ├── strings.bmd      # String search, split, casing, formatting
│   ├── collections.bmd  # Functional higher-order utilities (map, filter, sort)
│   ├── io.bmd           # Safe file operations and terminal tables
│   ├── json.bmd         # Pure JSON serializer and parser
│   └── index.bmd        # Standard library unified entrypoint
├── repl/                # Interactive terminal REPL
│   ├── repl.bmd         # State-persisting REPL loop with colon commands
│   └── index.bmd        # REPL module exports
├── sample_programs/     # Comprehensive real-world Srishti applications
│   ├── 01_banking_system.sri
│   ├── 02_math_and_statistics.sri
│   ├── 03_text_analyzer.sri
│   ├── 04_task_scheduler.sri
│   └── 05_game_simulation.sri
├── tests/               # Automated test suites
│   ├── 01_lexer_test.bmd
│   ├── 02_parser_test.bmd
│   ├── 03_compiler_test.bmd
│   ├── 04_vm_execution_test.bmd
│   └── 05_e2e_programs_test.bmd
├── cli.bmd              # Unified command-line interface
├── index.bmd            # Master language package entrypoint
└── README.md            # Comprehensive documentation manual
```

---

## 3. Grammar & Syntax Specification

### 3.1 Variables & Constants
```srishti
-- Mutable variable declaration
let balance be 50000.0

-- Variable mutation
set balance to balance plus 15000.0

-- Constant declaration
constant MAX_RETRIES be 5
```

### 3.2 Data Types
- **Integer**: `42`, `-100`, `0`
- **Float**: `3.14159`, `0.001`, `-45.5`
- **String**: `"Srishti Language"`, `"Hello\nWorld"`
- **Boolean**: `true`, `false`
- **Null**: `null`
- **Array**: `[10, 20, 30, 40]`, indexing via `arr[0]`, mutation via `set arr[0] to 99`
- **Struct**: User-defined structured types with named fields

### 3.3 Arithmetic Operators
| Operation | Operator Phrase | Example |
| :--- | :--- | :--- |
| Addition | `plus` | `a plus b` |
| Subtraction | `minus` | `total minus discount` |
| Multiplication | `times` | `rate times hours` |
| Division | `divided by` | `sum divided by count` |
| Modulo | `modulo` | `n modulo 2` |
| Power | `raised to` | `factor raised to years` |
| Negation | `-` | `-x` |

### 3.4 Relational & Logical Operators
| Comparison | Operator Phrase | Example |
| :--- | :--- | :--- |
| Equal | `is equal to` | `status is equal to 200` |
| Not Equal | `is not equal to` | `user is not equal to null` |
| Greater Than | `is greater than` | `score is greater than 90` |
| Less Than | `is less than` | `age is less than 18` |
| Greater or Equal | `is greater than or equal to` | `balance is greater than or equal to 1000` |
| Less or Equal | `is less than or equal to` | `i is less than or equal to n` |
| Logical Conjunction | `and` | `has_key and is_valid` |
| Logical Disjunction | `or` | `is_admin or is_moderator` |
| Logical Negation | `not` | `not is_empty` |
| String Concatenation | `combined with` | `"Score: " combined with str(score)` |

### 3.5 Control Flow & Conditionals
```srishti
if balance is greater than or equal to 100000.0 then:
    display "VIP Account"
otherwise:
    display "Standard Account"
```

### 3.6 Looping Constructs
```srishti
-- Repeat-While Loop
let i be 0
repeat while i is less than 5:
    display i
    set i to i plus 1

-- Iterate-Through Collection Loop
let items be ["Alpha", "Beta", "Gamma"]
iterate elem through items:
    display elem
```

### 3.7 Pattern Matching
```srishti
match tx_code:
    when 1:
        deposit(account, amount)
    when 2:
        withdraw(account, amount)
    when 3:
        display account.balance
    otherwise:
        display "Unknown transaction code"
```

### 3.8 Functions & Closures
```srishti
-- Function with parameters
define function calculate_interest with principal, rate, years:
    let factor be 1.0 plus (rate divided by 100.0)
    return principal times (factor raised to years)

-- Function without parameters
define function get_version:
    return "1.0.0"

-- Stateful Closure capturing lexical environment
define function make_counter:
    let count be 0
    define function increment:
        set count to count plus 1
        return count
    return increment
```

### 3.9 Struct Declarations & Instantiations
```srishti
-- Define Struct
create struct Account with id, owner, balance, is_active:

-- Instantiate Struct
let my_acc be Account(101, "Aryabhata", 50000.0, true)

-- Read Property
display my_acc.owner

-- Mutate Property
set my_acc.balance to my_acc.balance plus 5000.0
```

### 3.10 Concurrency
```srishti
-- Spawn non-blocking background task executed on a separate CallFrame
spawn task audit_accounts(accounts_list)
```

---

## 4. Standard Library Reference

### 4.1 Math Module (`std/math`)
- **Constants**: `PI` (`3.141592653589793`), `E` (`2.718281828459045`)
- **Functions**:
  * `abs(x)`: Absolute value.
  * `min(a, b)`: Smaller of two numbers.
  * `max(a, b)`: Larger of two numbers.
  * `clamp(val, min, max)`: Restricts value within bounds.
  * `floor(x)`: Greatest integer less than or equal to `x`.
  * `ceil(x)`: Smallest integer greater than or equal to `x`.
  * `round(x)`: Rounds to nearest integer.
  * `sqrt(x)`: High-precision Newton-Raphson square root.
  * `pow(base, exp)`: Power exponentiation.
  * `hypot(a, b)`: Euclidean hypotenuse `sqrt(a*a + b*b)`.
  * `deg_to_rad(deg)`: Converts degrees to radians.
  * `rad_to_deg(rad)`: Converts radians to degrees.

### 4.2 Strings Module (`std/strings`)
- `repeat(s, count)`: Generates string repeated `count` times.
- `pad_left(s, len, char)`: Pads left side of string to target length.
- `pad_right(s, len, char)`: Pads right side of string to target length.
- `starts_with(s, prefix)`: Returns `true` if string begins with `prefix`.
- `ends_with(s, suffix)`: Returns `true` if string terminates with `suffix`.
- `contains(s, substr)`: Returns `true` if `substr` is found in `s`.
- `to_upper(s)`: Converts string to uppercase.
- `to_lower(s)`: Converts string to lowercase.
- `trim_left(s)`: Removes leading whitespace.
- `trim_right(s)`: Removes trailing whitespace.
- `trim(s)`: Removes leading and trailing whitespace.
- `split(s, sep)`: Splits string by separator into an array of substrings.
- `join(arr, sep)`: Joins array of strings with separator into a string.
- `replace(s, old, new)`: Substitutes occurrences of `old` with `new`.

### 4.3 Collections Module (`std/collections`)
- `map(coll, fn)`: Transforms each element via callback.
- `filter(coll, fn)`: Selects elements matching predicate.
- `reduce(coll, fn, initial)`: Combines elements into single accumulator.
- `find(coll, fn)`: Returns first matching element or `null`.
- `find_index(coll, fn)`: Returns index of first match or `-1`.
- `any(coll, fn)`: Returns `true` if any element matches.
- `all(coll, fn)`: Returns `true` if all elements match.
- `reverse(coll)`: Returns new array with reversed order.
- `sort(coll, cmp)`: Stable mergesort array ordering.
- `flatten(coll)`: Recursively flattens nested arrays.
- `chunk(coll, size)`: Partitions array into chunks of specified length.
- `zip(arr1, arr2)`: Combines two arrays into pairs `[a, b]`.
- `range(start, end, step)`: Generates numeric sequence.

### 4.4 I/O Module (`std/io`)
- `read_file(path)`: Reads entire file content into string.
- `write_file(path, content)`: Writes content to file.
- `append_file(path, content)`: Appends content to file.
- `file_exists(path)`: Checks if file exists on disk.
- `format_table(headers, rows)`: Formats data into an aligned Unicode table.
- `print_banner(title, subtitle)`: Prints a clean, professional banner.

### 4.5 JSON Module (`std/json`)
- `serialize(val, indent)`: Serializes primitives, arrays, hashes, and structs to JSON string.
- `deserialize(json_str)`: Parses JSON string into native values.

---

## 5. Command-Line Interface (`cli.bmd`)

Execute the unified CLI runner via BramandScript:

```bash
# Execute Srishti source file
./bramand run examples/srishti/cli.bmd run examples/srishti/sample_programs/01_banking_system.sri

# Launch Interactive REPL
./bramand run examples/srishti/cli.bmd repl

# Syntax & Semantic Static Check
./bramand run examples/srishti/cli.bmd check examples/srishti/sample_programs/02_math_and_statistics.sri

# Dump Abstract Syntax Tree (AST)
./bramand run examples/srishti/cli.bmd ast examples/srishti/sample_programs/03_text_analyzer.sri

# Disassemble VM Bytecodes and Constant Pool
./bramand run examples/srishti/cli.bmd disasm examples/srishti/sample_programs/04_task_scheduler.sri

# Run All Srishti Test Suites
./bramand run --allow-run examples/srishti/cli.bmd test

# Display Architecture & Version Banner
./bramand run examples/srishti/cli.bmd version
```

---

## 6. Interactive REPL

Start the REPL session:
```bash
./bramand run examples/srishti/cli.bmd repl
```

```text
╔══════════════════════════════════════════════════════════════════╗
║  ✦ Srishti Interactive Language REPL (सृष्टि) v1.0.0             ║
║    Pure BramandScript Stack Machine Architecture                 ║
║    Type :help for reference, :exit or Ctrl+D to exit             ║
╚══════════════════════════════════════════════════════════════════╝

srishti> let x be 100
srishti> let y be 25
srishti> x divided by y
❯ 4

srishti> define function square with n:
...          return n times n
srishti> square(9)
❯ 81

srishti> :vars
── Active Global Variables ───────────────────────────────────────
  x (idx 0): 100 [INTEGER]
  y (idx 1): 25 [INTEGER]
  square (idx 2): CompiledFunction(square, 9 bytes, 2 locals) [CLOSURE]
──────────────────────────────────────────────────────────────────

srishti> :exit
[INFO] Exiting Srishti REPL. Namaste!
```

---

## 7. Real-World Sample Programs

The `sample_programs/` directory contains 5 comprehensive real-world applications:

1. **`01_banking_system.sri`**:
   - Models bank accounts, deposits, withdrawals, and inter-account transfers.
   - Computes compound interest yields with floating-point exponentiation.
   - Dispatches transactions using pattern matching (`match-when`).
   - Concurrently audits VIP accounts using `spawn task`.

2. **`02_math_and_statistics.sri`**:
   - Scientific computing suite for numerical datasets.
   - Implements mean, median, variance, and standard deviation algorithms.
   - Computes prime numbers via the Sieve of Eratosthenes.
   - Implements memoized Fibonacci and matrix-vector multiplication.

3. **`03_text_analyzer.sri`**:
   - Natural Language Processing (NLP) text analyzer.
   - Tokenizes text, computes word and character counts, line metrics, and unique vocabulary.
   - Computes document lexical diversity.
   - Validates word palindromes and capitalizes sentence tokens.

4. **`04_task_scheduler.sri`**:
   - Distributed priority job scheduler engine.
   - Manages asynchronous jobs with priority ratings.
   - Filters urgent jobs using higher-order functions.
   - Sorts priority queues descending and executes jobs to completion.

5. **`05_game_simulation.sri`**:
   - Turn-based fantasy RPG dungeon crawl and combat simulator.
   - Hero and Monster structured data modeling with attack, defense, and HP pools.
   - Combat turn loops with damage reduction calculations.
   - Inventory item consumption (`match-when`) and hero level-up mechanics.

---

## 8. Verification & Test Metrics

Run all test suites locally:
```bash
# Run Srishti E2E test suite
./bramand run examples/srishti/tests/05_e2e_programs_test.bmd

# Run complete repo test suite (544+ tests)
./bramand test
```

### Milestone Summary
- **Total Srishti Codebase**: 12,328 lines across 48 files
- **Srishti Unit & E2E Tests**: 149 / 149 passed (100%)
  * `01_lexer_test.bmd`: 20 / 20 passed
  * `02_parser_test.bmd`: 29 / 29 passed
  * `03_compiler_test.bmd`: 29 / 29 passed
  * `04_vm_execution_test.bmd`: 33 / 33 passed
  * `05_e2e_programs_test.bmd`: 38 / 38 passed
- **Complete Repository Tests**: 582 / 582 passed (433 core + 149 Srishti)
- **Zero Third-Party Dependencies**: 100% pure standard library and pure BramandScript
