// benchmarks/baseline_comparison.js
// Node.js Baseline Benchmarks for BramandScript Comparison

const { performance } = require('perf_hooks');

function timeIt(name, fn, iters) {
    fn(); // Warmup
    const start = performance.now();
    for (let i = 0; i < iters; i++) {
        fn();
    }
    const elapsed = performance.now() - start;
    const avgOp = elapsed / iters;
    const opsSec = (iters / (elapsed / 1000.0));
    console.log(`✦ Node.js    ${name.padEnd(28)} ${String(iters).padStart(8)} iters   ${elapsed.toFixed(2).padStart(9)}ms total   ${avgOp.toFixed(3).padStart(8)}ms/op   (${Math.round(opsSec).toString().padStart(8)} ops/sec)`);
}

function fib(n) {
    if (n <= 1) return n;
    return fib(n - 1) + fib(n - 2);
}

function tightLoop() {
    let s = 0;
    for (let i = 0; i < 10000; i++) {
        s += i;
    }
    return s;
}

function matmul4x4() {
    const A = [[1.0, 2.0, 3.0, 4.0], [5.0, 6.0, 7.0, 8.0], [9.0, 1.0, 2.0, 3.0], [4.0, 5.0, 6.0, 7.0]];
    const B = [[2.0, 0.0, 1.0, 3.0], [1.0, 2.0, 0.0, 1.0], [3.0, 1.0, 2.0, 0.0], [0.0, 3.0, 1.0, 2.0]];
    const C = [[0,0,0,0], [0,0,0,0], [0,0,0,0], [0,0,0,0]];
    for (let i = 0; i < 4; i++) {
        for (let k = 0; k < 4; k++) {
            for (let j = 0; j < 4; j++) {
                C[i][j] += A[i][k] * B[k][j];
            }
        }
    }
    return C;
}

const doc = {
    user_id: 42091,
    username: "bramand_dev",
    roles: ["developer", "maintainer", "admin"],
    metrics: {
        requests: 14920,
        latency_p99: 1.25,
        uptime: 99.98
    },
    tags: ["compiler", "vm", "termux", "offline"]
};

function jsonRoundtrip() {
    const s = JSON.stringify(doc);
    const o = JSON.parse(s);
    return o;
}

console.log("✦ Node.js Performance Baseline Benchmarks");
console.log("────────────────────────────────────────────────────────────────────────────");
timeIt("fibonacci_recursive_15", () => fib(15), 100);
timeIt("tight_loop_10k_ops", tightLoop, 20);
timeIt("tensor_matmul_4x4", matmul4x4, 100);
timeIt("json_roundtrip", jsonRoundtrip, 200);
console.log("────────────────────────────────────────────────────────────────────────────");
