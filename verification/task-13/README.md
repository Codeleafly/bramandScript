# Verification Task 13: Strings, Type Conversion, and Error Handling

## 1. Test Objective
Verify BramandScript's string manipulation engine: concatenation, indexing, substring slicing, Unicode rune fidelity (proper rune counting vs byte counting), string mutation utilities (`split`, `join`, `trim`, `to_lower`, `to_upper`, `replace`), bidirectional type conversions (`int()`, `float()`, `str()`), runtime type inspection (`type()`), and build a practical input validation utility.

## 2. Features Tested
- String concatenation with `+`
- String length measurement via both `len(str)` and `.length` property
- Rune-aware character indexing (`str[i]`) returning single-character strings or `null` on out-of-bounds
- Substring slicing via `slice(str, start, end)` and `substring(str, start, end)`
- Multi-byte UTF-8 Unicode rune handling (`✦ नमस्ते Bramand ॐ`)
- Delimited splitting via `split(str, sep)` returning array of tokens
- Array joining via `join(arr, sep)` returning combined string
- Whitespace trimming via `trim(str)`
- Case conversion (`to_upper`, `to_lower`) and global replacement (`replace(str, old, new)`)
- Type conversions: integer, float, string, and boolean
- Dynamic runtime type introspection with `type(obj)`
- Production input validation utility verifying email structure

## 3. Source Files
- `verification/task-13/test_strings_types.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-13/test_strings_types.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (32/32 assertions passed)

## 5. Actual Output
```
✦ TASK 13: Strings, Type Conversion, and Validation
  ✓ [PASS] String concatenation
  ✓ [PASS] String length via len()
  ✓ [PASS] String length via .length property
  ✓ [PASS] Character indexing [0]
  ✓ [PASS] Character indexing [7]
  ✓ [PASS] Character index out of bounds returns null
  ✓ [PASS] Substring slice(0, 5)
  ✓ [PASS] Substring alias substring(7, 12)
  ✓ [PASS] Unicode string length in runes
  ✓ [PASS] Unicode indexing first rune
  ✓ [PASS] split array count
  ✓ [PASS] split element
  ✓ [PASS] join result
  ✓ [PASS] trim whitespace
  ✓ [PASS] to_upper
  ✓ [PASS] to_lower
  ✓ [PASS] replace
  ✓ [PASS] int to str
  ✓ [PASS] str to int
  ✓ [PASS] float to str
  ✓ [PASS] str to float
  ✓ [PASS] bool to str
  ✓ [PASS] type inspection int
  ✓ [PASS] type inspection float
  ✓ [PASS] type inspection str
  ✓ [PASS] type inspection bool
  ✓ [PASS] type inspection null
  ✓ [PASS] type inspection array
  ✓ [PASS] type inspection hash
  ✓ [PASS] Validate valid email
  ✓ [PASS] Validate invalid email (no @)
  ✓ [PASS] Validate invalid email (no dot)
[OK] All Task 13 string, type conversion, and validation checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, defect in byte-based `len(str)` resolved to Unicode rune counting).
