---
title: Concurrency Anti-patterns
tags: [spawn, await]
---

## Don’t share mutable bindings across tasks

Even with immutable values, rebinding a `var` from multiple tasks can lose updates.

Prefer:

* returning values from tasks and awaiting them

## Don’t expect function calls to “join”

A function call is not a join point. Only:

* nursery exit
* `await` on a handle

## Don’t hide concurrency

If it runs concurrently, you should see `spawn`.
