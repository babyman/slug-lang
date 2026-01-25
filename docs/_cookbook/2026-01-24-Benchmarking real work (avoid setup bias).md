---
title: Benchmarking Real Work (avoid setup bias)
tags: [ slug.benchmark, performance ]
---

### **Problem**

You want to benchmark a function without accidentally measuring setup or allocation work.

### **Idiom: Keep Setup Outside the Benchmarked Function**

Move any setup outside the function passed to `micro`, so the benchmark measures only the work you care about.

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
````

This produces more stable and meaningful results.
