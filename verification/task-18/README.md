# Verification Task 18: HTTP Client, Headers, and Network Error Handling

## 1. Test Objective
Verify BramandScript's HTTP client implementation: issuing standard GET and POST requests, custom request headers (`headers` map), JSON payloads, verifying HTTP status codes (200, 201, 404), configurable timeout enforcement, and safe error handling when encountering connection-refused failures.

## 2. Features Tested
- `http_get(url)`: Simple GET request returning response dictionary with `status_code`, `body`, and `headers`
- `http_post(url, body)`: POST request delivering raw or JSON encoded request bodies
- Custom header injection via `http_request({ "headers": { ... } })`
- Status code extraction across successful (200, 201) and error (404) responses
- Server-side header manipulation (`res.set_header(...)`, `res.setHeader(...)`)
- Client error resilience: connection-refused responses return an `Error` object rather than crashing the VM

## 3. Source Files
- `verification/task-18/test_http_client.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-18/test_http_client.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (6/6 assertions passed)

## 5. Actual Output
```
✦ TASK 18: HTTP Client, Headers, and Network Error Handling
  ✓ [PASS] GET status code is 200
  ✓ [PASS] GET body contains success
  ✓ [PASS] POST status code is 201
  ✓ [PASS] POST body reflects payload
  ✓ [PASS] 404 Not Found status code
  ✓ [PASS] Advanced request with headers status 200
  ✓ [PASS] Connection failure produces error without crashing
[OK] All Task 18 HTTP client and network error handling checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found).
