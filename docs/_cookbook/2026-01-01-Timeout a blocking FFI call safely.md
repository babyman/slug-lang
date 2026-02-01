---
title: Timeout a blocking FFI call safely
tags: [spawn, await]
---

If `read/accept/write` are blocking FFI, do this:

```slug
var {*} = import("slug.channel")
fn readWithTimeout(conn, n, ms) {
    var t = spawn { read(conn, n) }
    await(t, ms)
}
```

This is the simplest portable approach.
