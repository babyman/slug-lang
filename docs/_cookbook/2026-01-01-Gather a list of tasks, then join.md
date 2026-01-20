---
title: Gather a list of tasks, then join
tags: [nursery, spawn, await]
---

```slug
nursery fn fetchAll(ids) {
    var tasks = ids /> map(fn(id) { spawn { fetchUser(id) } })
    tasks /> map(fn(t) { await t })
}
```

This is the cleanest “fan-out/fan-in” shape for N items.
