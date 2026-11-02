# Verification Task 08: JSON, CSV, Base64, and Cryptography

## 1. Test Objective
Verify BramandScript's serialization formats and cryptographic standard library: JSON parsing/stringification, CSV handling with quoted fields and embedded commas, Base64 encoding/decoding, SHA-256 against official RFC known-good test vectors, HMAC-SHA256 authenticated hashes, and cryptographically secure pseudorandom number generation (`random_bytes`, `random_int`, `uuid`).

## 2. Features Tested
- JSON parsing (`json_parse`) into native maps and arrays
- JSON serialization (`json_stringify`) with formatting fidelity
- CSV parsing (`csv_parse`) with RFC 4180 compliant comma escaping within quotes (`"Doe, Jane"`)
- CSV serialisation (`csv_stringify`)
- Base64 encoding and decoding (`base64_encode`, `base64_decode`)
- SHA-256 digest calculation (`sha256("hello")`) verified against the official NIST/RFC vector: `2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824`
- HMAC-SHA256 signature generation (`hmac_sha256`)
- Hardware CSPRNG random bytes via `crypto/rand` (`random_bytes` returning hex strings)
- Cryptographic integer generation within specified bounds (`random_int`)
- RFC 4122 compliant UUID v4 generation (`uuid()`)

## 3. Source Files
- `verification/task-08/test_crypto_encoding.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-08/test_crypto_encoding.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (15/15 assertions passed)

## 5. Actual Output
```
✦ TASK 08: JSON, CSV, Base64, and Cryptography
  ✓ [PASS] JSON parse string property
  ✓ [PASS] JSON parse boolean property
  ✓ [PASS] JSON parse array property
  ✓ [PASS] JSON stringify roundtrip
  ✓ [PASS] CSV row count
  ✓ [PASS] CSV quoted comma handling
  ✓ [PASS] CSV simple field
  ✓ [PASS] CSV stringify output
  ✓ [PASS] Base64 encode not empty
  ✓ [PASS] Base64 decode matches original
  ✓ [PASS] SHA-256 RFC known-good vector
  ✓ [PASS] HMAC-SHA256 generation (64 hex chars)
  ✓ [PASS] CSPRNG random_bytes hex string length (16 bytes = 32 hex chars)
  ✓ [PASS] CSPRNG random_int in range [10, 50]
  ✓ [PASS] UUID v4 string length is 36
[OK] All Task 08 JSON, CSV, Base64, and Cryptography checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
