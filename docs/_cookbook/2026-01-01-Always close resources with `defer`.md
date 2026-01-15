---
title: Always close resources with `defer`
tags: [async, spawn, await, defer]
---

```slug
async fn handleConn(conn, app) {
    defer { close(conn) }

    var rawT = spawn { read(conn, 64_000) }
    var raw  = await rawT within 30000

    var resT = spawn { app(parseRequest(raw)) }
    var res  = await resT within 2000

    write(conn, formatResponse(res))
}
```

`defer` keeps cleanup deterministic even with timeouts/errors.
