---
title: Long-running loops (use `recur`)
tags: [nursery, await, recur]
---

```slug
nursery fn acceptLoop(listener, app) {
    var conn = accept(listener)
    spawn { handleConn(conn, app) }
    recur(listener, app)
}
```

Important: `recur` is a loop construct; it must **not** trigger nursery joins.
