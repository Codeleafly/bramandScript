---
name: Bug Report
about: Create a detailed report to help us reproduce and fix a defect
title: "[BUG] "
labels: ["bug", "triage"]
assignees: []
---

## ✦ Bug Summary
A concise description of what the unexpected behavior or error is.

## ✦ Minimal Reproduction Script
Create the smallest possible `.bmd` script that triggers the issue:

```bmd
// repro.bmd
let x = ...
```

## ✦ Execution Command & Output
Run command:
```bash
./bramand run repro.bmd
```

Actual output / diagnostic trace:
```
[Paste output here]
```

Expected behavior:
```
[Describe what should have happened]
```

## ✦ Environment & System Details
- **BramandScript Version**: `0.1.0` (run `bramand version`)
- **Operating System**: Linux / Termux (Android ARM64) / macOS / Windows
- **Architecture**: `amd64` / `arm64` / `armv7`
- **Go Version** (if built from source): `go version`

## ✦ Additional Context
Add any other context about the defect, including affected packages (`lexer`, `parser`, `compiler`, `vm`, `builtins`).
