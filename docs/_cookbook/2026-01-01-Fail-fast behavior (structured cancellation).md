---
title: Fail-fast behavior (structured cancellation)
tags: [nursery, spawn, await]
---

By default, if a task fails and you `await` it:

* the error propagates normally
* sibling tasks in the same nursery should be cancelled (fail-fast)

Typical pattern:

```slug
var {*} = import("slug.channel")
nursery fn handler(req) {
    var aT = spawn { taskA() }
    var bT = spawn { taskB() }

    var a = await(aT);    // if this throws, nursery cancels siblings
    var b = await(bT)

    {a:a, b:b}
}
```
