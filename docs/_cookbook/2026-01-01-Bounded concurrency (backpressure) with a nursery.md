---
title: Bounded concurrency (backpressure) with a nursery
tags: [nursery, limit, spawn, await]
---

Create a nursery to limit concurrent spawns:

```slug
var {*} = import("slug.channel")
var loadUsers = nursery limit 10 fn(ids) {
    ids
        /> map(fn(id) { spawn { fetchUser(id) } })
        /> map(fn(t) { await(t) })  // join handles
}
```

Notes:

* `limit 10` caps in-flight tasks in this nursery.
* Join is explicit (await each handle), or implicit at nursery exit if you don’t need results.
