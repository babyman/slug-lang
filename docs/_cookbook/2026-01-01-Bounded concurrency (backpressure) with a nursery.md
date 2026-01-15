---
title: Bounded concurrency (backpressure) with a nursery
tags: [async, limit, spawn, await]
---

Create a nursery to limit concurrent spawns:

```slug
var loadUsers = async limit 10 fn(ids) {
    ids
        /> map(fn(id) { spawn { fetchUser(id) } })
        /> map(async fn(t) { await t })  // join handles
}
```

Notes:

* `limit 10` caps in-flight tasks in this nursery.
* Join is explicit (await each handle), or implicit at nursery exit if you don’t need results.
