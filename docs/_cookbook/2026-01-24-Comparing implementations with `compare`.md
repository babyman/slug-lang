---
title: Comparing Implementations with `compare`
tags: [ slug.benchmark, performance ]
---

### **Problem**

You have multiple implementations of the same idea and want to know which one is faster.

### **Idiom: Comparing Benchmarks by Median Time**

The `compare` function runs `micro` for each benchmark, sorts the results by median time (p50),
and computes a relative ratio against the fastest implementation.

```slug
var {*} = import(
	"slug.time",
	"slug.benchmark",
)

var slow = fn() { sleep(10) }
var fast = fn() { sleep(5) }

var report =
	compare([
		{name: "slow version", fun: slow},
		{name: "fast version", fun: fast},
	],
	)

report /> printCompareReport
````

The fastest benchmark is shown as `x1`, with slower implementations reported relative to it.
