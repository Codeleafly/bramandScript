# Tabular Data Analytics with DataFrames in BramandScript

This example demonstrates how to process, query, and aggregate tabular datasets using `std/dataframe`.

## Features Demonstrated

1. **Tabular DataFrame Architecture (`std/dataframe/index.bmd`)**:
   - Automatic type-inferred CSV parser (`from_csv`, `read_csv`).
   - Dimension introspection (`frame.shape()`).
   - Column projection (`frame.select(...)`) and slicing (`frame.head(N)`, `frame.tail(N)`).

2. **Predicate Filtering**:
   - Functional row predicates (`frame.filter(fn(row) { ... })`).

3. **Grouped Aggregations (`group_by`)**:
   - Hash-based partitioning by key column (`frame.group_by("department")`).
   - Multi-metric reduction specs: `mean`, `sum`, `count`, `min`, `max`.

4. **Descriptive Statistics**:
   - Automated identification of numeric columns and summary metrics via `frame.describe()`.

5. **CSV Exporting**:
   - Canonical CSV formatting and disk persistence (`frame.save_csv(path)`).

## Running the Example

```bash
bramand run --allow-all examples/13_tabular_data_analytics/main.bmd
```
> Note: `--allow-all` (or `--allow-write`) is required for CSV reading and disk persistence (`save_csv`).
