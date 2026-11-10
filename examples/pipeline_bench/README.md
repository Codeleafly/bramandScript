# Buffer Stream Pipeline Benchmark (`examples/pipeline_bench/`)

A production-grade, 100% pure BramandScript benchmark suite exercising binary byte buffer manipulation, numeric codecs, memory copying, and stream piping using `std/buffer` and `std/io`.

## Architecture & Features

- **Binary Numeric Codecs**: Tests serialization and deserialization of Big-Endian and Little-Endian 32-bit unsigned integers and 64-bit IEEE 754 floating point numbers.
- **Fast Slice Copying (`buf.copy`)**: Measures throughput of bulk byte transfers across separate memory buffers with zero data corruption.
- **Stream Piping Combinator (`io.pipe`)**: Pipes a structured 256 KB binary stream through `io.BufferReader` into `io.BufferWriter` using chunked streaming buffers (4096 bytes per chunk).
- **Data Integrity Verification**: Uses `Buffer.compare` and offset boundary checks to verify that every single byte matches exactly between input and output.

## Running the Application

```bash
# Run buffer pipeline benchmark
./bramand run examples/pipeline_bench/main.bmd
```
