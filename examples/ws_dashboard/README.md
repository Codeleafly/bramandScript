# Live Telemetry & WebSocket Dashboard (`examples/ws_dashboard/`)

A production-grade, 100% pure BramandScript real-time WebSocket dashboard and telemetry server built with `std/ws` and `std/os`.

## Architecture & Features

- **Pure BMD RFC 6455 Engine**: Complete handshake negotiation, frame encoding/decoding, and ping/pong keepalive management running directly over raw TCP sockets.
- **Operating System Introspection**: Fetches real-time runtime metrics (platform, architecture, CPU count, memory allocation) using `std/os`.
- **Periodic Live Telemetry Broadcaster**: Background worker coroutine (`spawn`) streaming JSON metric pulses to all connected dashboard clients.
- **Interactive Control Channel**: Bidirectional command parsing (`ping` / `pong`) and connection lifecycle management (`connect`, `message`, `close`).
- **Automated Verification**: Self-running client connecting over loopback WebSocket, verifying handshake, bidirectional ping/pong, and multi-pulse telemetry consumption before graceful termination.

## Running the Application

```bash
# Run automated WebSocket dashboard demo
./bramand run --allow-all examples/ws_dashboard/main.bmd
```
