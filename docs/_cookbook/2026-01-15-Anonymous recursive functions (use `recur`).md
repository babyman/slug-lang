---
title: Anonymous recursive functions (use `recur`)
tags: [ recur ]
---

In Slug, you don't need to name a function to make it recursive. The `recur` keyword allows an anonymous function to
call itself.

```slug
var factorialResult = fn(n, acc = 1) {
    if (n <= 1) {
        acc
    } else {
        recur(n - 1, n * acc)
    }
}(5)

println("factorial(5)", factorialResult) // 120
```
