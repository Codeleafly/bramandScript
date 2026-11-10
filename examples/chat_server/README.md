# Multi-Client TCP Chat Server (`examples/chat_server/`)

A production-grade, 100% pure BramandScript real-time chat room server built on raw duplex TCP sockets (`std/net`).

## Architecture & Features

- **Raw TCP Sockets**: Uses `net.create_server` and `net.connect` for low-level connection streaming.
- **Thread-Safe State Synchronization**: Employs `mutex()` locks to protect the global client registry across concurrent connection worker coroutines (`spawn`).
- **Command Dispatcher**:
  - `/nick <name>`: Rename active session with room-wide notifications.
  - `/who`: List currently active users in the room.
  - `/msg <user> <message>`: Point-to-point private messaging between users.
  - `/quit`: Gracefully exit the room.
  - `/help`: Print command manual.
- **Room-Wide Broadcast**: Distributes public messages to all connected clients except the sender.
- **Automated Verification**: Self-running automated simulation of concurrent clients (Alice & Bob) verifying session creation, nick changes, broadcast, direct messaging, and clean shutdown.

## Running the Application

```bash
# Run automated simulation
./bramand run --allow-net examples/chat_server/main.bmd
```
