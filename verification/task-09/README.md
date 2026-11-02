# Verification Task 09: Networking and Web Server

## 1. Test Objective
Verify BramandScript's networking stack: TCP stream sockets, UDP datagram sockets, HTTP web server with routing, middleware, and JSON responses, HTTP client (`http_get`, `http_post`), and full RFC 6455 WebSocket client/server communications over localhost.

## 2. Features Tested
- TCP server listening (`tcp_listen`), connection accept (`srv.accept()`), bi-directional messaging (`conn.send`, `conn.recv`), and graceful closure
- UDP socket binding (`udp_listen`), connection (`udp_connect`), and packet transfer (`send`, `recv`)
- Built-in HTTP web server (`http_server()`) with REST route handlers (`app.get`, `app.post`)
- HTTP response builders (`res.json()`, `res.status(code)`)
- HTTP client methods (`http_get`, `http_post`) verifying status codes (200, 201) and payload roundtrips
- WebSocket server (`ws_server`), connection handshake (`ws_connect`), and bidirectional frame exchange (`send`, `recv`)

## 3. Source Files
- `verification/task-09/test_networking.bmd`

## 4. Execution Command & Exit Code
```bash
./bramand run verification/task-09/test_networking.bmd
```
- **Exit Code**: `0`
- **Result**: PASS (7/7 assertions passed)

## 5. Actual Output
```
✦ TASK 09: Networking, Web Server, Sockets, and WebSockets
  ✓ [PASS] TCP client/server roundtrip
  ✓ [PASS] UDP client/server datagram
  ✓ [PASS] HTTP GET status 200
  ✓ [PASS] HTTP GET body content
  ✓ [PASS] HTTP POST status 201
  ✓ [PASS] HTTP POST body content
  ✓ [PASS] WebSocket client/server frame roundtrip
[OK] All Task 09 networking and web server checks passed.
```

## 6. Verdict
**Status**: `PASS` (Real execution, 0 defects found)
