---
title: Slug
subtitle: No Shell. All Strength.
header_type: hero
show_breadcrumb: false
---

> **Slug is a language for writing code you can understand tomorrow.**
> Code is written once and read many times, so Slug optimizes for reading.
> It favors explicit structure over cleverness, simplicity over feature count, and predictability over magic.

## Why developers pick Slug

- Readability-first syntax that keeps intent clear and refactors safe.
- Functional-first pipelines and pattern matching for clean data flow.
- Structured concurrency with explicit `spawn` and `await`.
- Small standard library and predictable modules over hidden magic.

## Build with confidence

Slug is opinionated on purpose: explicit control flow, data-first composition, and a compact core you can keep in your
head. Spend less time decoding code and more time shipping.

## See it in action

```slug
var {*} = import("slug.std")

val result = [1, 2, 3, 4, 5, 6]
    /> map(fn(x) { x * x })
    /> filter(fn(x) { x % 2 == 0 })
    /> reduce(0, fn(acc, x) { acc + x })

println("Result:", result)
```

## Get started

- Installation and setup: http://www.sluglang.org
- Developers guide: `docs/developers-guide.md`
- Setup guide: `docs/setup.md`
- Cookbook: `docs/cookbook.md`
