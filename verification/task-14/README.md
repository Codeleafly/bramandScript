# Verification Task 14: Collection Utilities and Common Data Operations

## 1. Test Objective
Determine and verify BramandScript's standard collection capabilities: array element addition (`push`), element removal (`pop`), membership tests (`contains`), sorting (`sort`), reversing (`reverse`), slicing (`slice`), map key/value inspection (`keys`, `values`), key deletion (`delete`), and build a complete grade-book application computing counts, totals, high/low bounds, and averages.

## 2. Features Tested
- Dynamic array mutation and extraction (`push`, `pop`, `len`)
- Collection membership lookup (`contains`)
- Numeric ascending array sorting (`sort`)
- Array reversing (`reverse`)
- Sub-array slicing (`slice`)
- Map key and value enumeration (`keys`, `values`)
- Map key deletion (`delete`)
- Iterative aggregation and statistics computation over dictionary keys
- Gradebook application calculating min, max, sum, and mean

## 3. Source Files
- `verification/task-14/gradebook.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-14/gradebook.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (21/21 assertions passed)

## 5. Actual Output
```
✦ TASK 14: Collection Utilities and Gradebook Application
  ✓ [PASS] Initial scores length
  ✓ [PASS] Scores length after push
  ✓ [PASS] Popped score is 91
  ✓ [PASS] Scores length after pop
  ✓ [PASS] Contains score 95
  ✓ [PASS] Contains missing score 100
  ✓ [PASS] Sorted first element (min)
  ✓ [PASS] Sorted last element (max)
  ✓ [PASS] Reversed first element (max)
  ✓ [PASS] Reversed last element (min)
  ✓ [PASS] Top three length
  ✓ [PASS] Top three elements
  ✓ [PASS] Keys enumeration count
  ✓ [PASS] Values enumeration count
  ✓ [PASS] Deleted key no longer in map
  ✓ [PASS] Keys count after deletion
  ✓ [PASS] Gradebook count is 3
  ✓ [PASS] Gradebook total is 274.0
  ✓ [PASS] Gradebook highest is 95
  ✓ [PASS] Gradebook lowest is 88
  ✓ [PASS] Gradebook average > 91.0
[OK] All Task 14 collection utilities and gradebook checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, collection builtins `sort`, `reverse`, `slice` verified).
