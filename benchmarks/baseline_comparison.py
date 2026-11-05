# benchmarks/baseline_comparison.py
# Python 3 Baseline Benchmarks for BramandScript Comparison

import time
import json

def time_it(name, fn, iters):
    fn() # Warmup
    start = time.perf_counter()
    for _ in range(iters):
        fn()
    elapsed = time.perf_counter() - start
    avg_op = elapsed / iters
    ops_sec = iters / elapsed
    print(f"✦ Python 3   {name:<28} {iters:8d} iters   {elapsed*1000:9.2f}ms total   {avg_op*1000:8.3f}ms/op   ({ops_sec:8.0f} ops/sec)")

def fib(n):
    if n <= 1:
        return n
    return fib(n - 1) + fib(n - 2)

def tight_loop():
    s = 0
    for i in range(10000):
        s += i
    return s

def matmul_4x4():
    A = [[1.0, 2.0, 3.0, 4.0], [5.0, 6.0, 7.0, 8.0], [9.0, 1.0, 2.0, 3.0], [4.0, 5.0, 6.0, 7.0]]
    B = [[2.0, 0.0, 1.0, 3.0], [1.0, 2.0, 0.0, 1.0], [3.0, 1.0, 2.0, 0.0], [0.0, 3.0, 1.0, 2.0]]
    C = [[0.0]*4 for _ in range(4)]
    for i in range(4):
        for k in range(4):
            for j in range(4):
                C[i][j] += A[i][k] * B[k][j]
    return C

doc = {
    "user_id": 42091,
    "username": "bramand_dev",
    "roles": ["developer", "maintainer", "admin"],
    "metrics": {
        "requests": 14920,
        "latency_p99": 1.25,
        "uptime": 99.98
    },
    "tags": ["compiler", "vm", "termux", "offline"]
}

def json_roundtrip():
    s = json.dumps(doc)
    o = json.loads(s)
    return o

if __name__ == "__main__":
    print("✦ Python 3 Performance Baseline Benchmarks")
    print("────────────────────────────────────────────────────────────────────────────")
    time_it("fibonacci_recursive_15", lambda: fib(15), 100)
    time_it("tight_loop_10k_ops", tight_loop, 20)
    time_it("tensor_matmul_4x4", matmul_4x4, 100)
    time_it("json_roundtrip", json_roundtrip, 200)
    print("────────────────────────────────────────────────────────────────────────────")
