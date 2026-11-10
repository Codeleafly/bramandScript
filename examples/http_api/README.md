# Pure BramandScript REST API Service (`examples/http_api/`)

A production-grade, 100% pure BramandScript HTTP/1.1 REST API service built using `std/http` and the Trie-based Radix Router (`std/net/router`).

## Architecture & Features

- **Pure BMD HTTP Engine**: 100% self-hosted wire parser, HTTP server, and HTTP client operating directly over raw duplex TCP sockets.
- **Radix-Trie Routing**:
  - Parameterized route matching (`:id`) extracting URL path params into `req.params`.
  - HTTP method dispatch (`GET`, `POST`, `PUT`, `DELETE`).
- **RESTful Endpoints**:
  - `GET /`: Health check & API documentation metadata.
  - `GET /api/v1/items`: List all items with count.
  - `POST /api/v1/items`: Create resource from JSON payload with validation.
  - `GET /api/v1/items/:id`: Read resource by ID or return 404.
  - `PUT /api/v1/items/:id`: Update resource by ID or return 404.
  - `DELETE /api/v1/items/:id`: Remove resource by ID or return 404.
- **Thread-Safe Data Store**: Protects in-memory storage dictionary using BramandScript's native `mutex()`.
- **Automated Verification**: End-to-end self-test testing the full CRUD lifecycle against the running server.

## Running the Application

```bash
# Run automated API service demo
./bramand run --allow-net examples/http_api/main.bmd
```
