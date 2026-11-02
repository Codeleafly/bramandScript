# Verification Task 03: Arrays, Maps, Structs, and Pattern Matching

## 1. Test Objective
Verify BramandScript's compound data structures: dynamic Arrays, Hash Maps, Structs with methods, Pattern Matching (`match`/`case`), `for-in` iterators, and out-of-bounds/missing-key safety (evaluating to `null`).

## 2. Features Tested
- Array literal creation, zero-based indexing, mutation (`arr[i] = val`), and bounds checking (returns `null` for invalid indices)
- Array appending via `push(arr, val)`
- Map literal creation, key indexing (`map["key"]`), mutation, and missing-key lookup (returns `null`)
- Nested data structures combining arrays, maps, and scalar primitives
- `for-in` iteration over array collections
- `struct` definitions with fields and closure-based stateful methods
- `match` pattern matching with distinct literal cases and `default` branch fallback

## 3. Source Files
- `verification/task-03/student_app.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-03/student_app.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (18/18 assertions passed)

## 5. Actual Output
```
✦ TASK 03: Arrays, Maps, Structs, and Pattern Matching
  ✓ [PASS] Array length
  ✓ [PASS] Array indexing
  ✓ [PASS] Array index mutation
  ✓ [PASS] Out of bounds index returns null
  ✓ [PASS] Array push length
  ✓ [PASS] Array push element
  ✓ [PASS] Map key lookup
  ✓ [PASS] Map integer lookup
  ✓ [PASS] Map key mutation
  ✓ [PASS] Missing map key returns null
  ✓ [PASS] Nested collection access
  ✓ [PASS] Nested map lookup
  ✓ [PASS] For-in loop array summation
  ✓ [PASS] Struct property access
  ✓ [PASS] Struct method call
  ✓ [PASS] Match case 100
  ✓ [PASS] Match case 90
  ✓ [PASS] Match default
[OK] All Task 03 data structures and OOP checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
