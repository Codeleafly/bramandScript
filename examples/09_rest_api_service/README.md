# Modular REST API Microservice in BramandScript

This example demonstrates how to build a production-grade, modular HTTP REST API service using BramandScript's standard library (`std/net` and `std/db`).

## Features Demonstrated

1. **Radix Trie Router (`std/net/router.bmd`)**:
   - Exact path routes (`/health`).
   - Parameterized route matching with named captures (`/api/v1/users/:id`).
   - Query string parameter parsing (`?role=admin`).

2. **HTTP Context Wrapper (`std/net/context.bmd`)**:
   - Status code mutations (`ctx.status(404)`).
   - Dynamic JSON serialization and header management (`ctx.send_json(...)`).
   - Body parsing from HTTP payloads (`ctx.json_body()`).

3. **Schema Model Validation (`std/db/model.bmd`)**:
   - Type validation (`STRING`, `INTEGER`).
   - Length constraints (`min_len`, `max_len`).
   - Value constraints (`min`, `max`, `contains`, `in`).
   - Clean 422 Unprocessable Entity error payloads.

## Running the Example

```bash
bramand run examples/09_rest_api_service/main.bmd
```
