---
title: Timeout any work (the canonical pattern)
tags: [async, spawn, await]
---

Timeouts apply to `await`, so to timeout a call (sync or async), wrap it in a spawn:

```slug
async fn handler(req) {
    var resT = spawn { doWork(req) }
    var res  = await resT within 2000
    res
}
```

This works for:

* normal `fn`
* `async fn`
* blocking foreign calls (FFI)
