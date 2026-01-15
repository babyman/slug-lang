---
title: Gather a list of tasks, then join
tags: [async, spawn, await]
---

```slug
async fn fetchAll(ids) {
    var tasks = ids /> map(fn(id) { spawn { fetchUser(id) } })
    tasks /> map(async fn(t) { await t })
}
```

This is the cleanest “fan-out/fan-in” shape for N items.
