# Real-Time Chat Room Hub & SSE Telemetry in BramandScript

This example demonstrates how to build real-time communication architectures using BramandScript's WebSocket room hub (`std/net/ws_room`) and Server-Sent Events engine (`std/net/sse`).

## Features Demonstrated

1. **Multi-Room WebSocket Broadcaster (`std/net/ws_room.bmd`)**:
   - Dynamic room creation and subscriptions (`hub.join(room, client_id, conn, metadata)`).
   - Targeted broadcasting with sender exclusion (`hub.broadcast(room, message, exclude_id)`).
   - Global multi-room dispatch (`hub.broadcast_all(...)`).
   - Clean disconnection and unsubscription handling (`hub.leave(...)`, `hub.leave_all(...)`).

2. **W3C Server-Sent Events (SSE) Engine (`std/net/sse.bmd`)**:
   - Formats standards-compliant event streams (`event: ...\nid: ...\ndata: ...\n\n`).
   - Heartbeat/keepalive ping comments (`: keepalive <timestamp>`).
   - Inbound event stream chunk parser (`sse.parse_sse_event(...)`).

## Running the Example

```bash
bramand run examples/14_realtime_chat_hub/main.bmd
```
