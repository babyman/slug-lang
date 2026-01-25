---
title: Benchmarking a Function with `micro`
tags: [ slug.benchmark, performance ]
---

### **Problem**

You want to measure how long a small piece of code takes to run, without setting up an external profiler or harness.

### **Idiom: Benchmarking a Zero-Argument Function**

The `micro` function benchmarks a single zero-argument function and reports timing statistics such as median (p50),
p90, and p99.

```slug
var {*} = import(
	"slug.time",
	"slug.benchmark",
)

var f = fn() {
	sleep(100)
}

micro("sleep 100ms", f)
	/> printResult
```

Focus on **p50** for typical performance, and **p90/p99** to understand jitter or tail latency.
