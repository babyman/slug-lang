---
title: Timeout any work (the canonical pattern)
tags: [nursery, spawn, await]
---

Timeouts apply to `await`, so to timeout a call (sync or nursery), wrap it in a spawn:

```slug
var {*} = import("slug.channel")
nursery fn handler(req) {
    var resT = spawn { doWork(req) }
    var res  = await(resT, 2000)
    res
}
```

This works for:

* normal `fn`
* `nursery fn`
* blocking foreign calls (FFI)
