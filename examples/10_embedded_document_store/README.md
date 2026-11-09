# Embedded Document Store & Query Engine in BramandScript

This example demonstrates how to use the embedded JSON document database and query generation capabilities in `std/db`.

## Features Demonstrated

1. **Document Collections (`std/db/doc.bmd`)**:
   - Auto-incrementing, collision-free document keys (`_id`).
   - Query filters with predicate evaluation (`$gt`, `$in`, `$eq`, `$contains`).
   - In-place mutation with operators like `$set`.
   - Automatic disk persistence and JSON serialization.

2. **Query Builder (`std/db/query.bmd`)**:
   - Fluent API for building SQL-like queries.
   - Dual execution:
     - In-memory execution against existing dataset arrays (`qb.execute(dataset)`).
     - ANSI SQL dialect compilation (`qb.to_sql()`).

## Running the Example

```bash
bramand run --allow-all examples/10_embedded_document_store/main.bmd
```
> Note: `--allow-all` (or `--allow-write`) is required because the database writes collection records to disk.
