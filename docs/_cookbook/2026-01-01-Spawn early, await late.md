---
title: “Spawn early, await late”
tags: [nursery, spawn, await]
---

Start tasks as soon as you can, await when you must:

```slug
nursery fn handler(req) {
    var aT = spawn { fetchA(req) }
    var bT = spawn { fetchB(req) }

    var a = await aT
    // do some CPU work here while b runs...
    var b = await bT

    combine(a, b)
}
```
