---
title: Timeout a blocking FFI call safely
tags: [async, spawn, await]
---

If `read/accept/write` are blocking FFI, do this:

```slug
async fn readWithTimeout(conn, n, ms) {
    var t = spawn { read(conn, n) }
    await t within ms
}
```

This is the simplest portable approach.
