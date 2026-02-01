---
title: Always close resources with `defer`
tags: [nursery, spawn, await, defer]
---

```slug
var {*} = import("slug.channel")

nursery fn handleConn(conn, app) {
    defer { close(conn) }

    var rawT = spawn { read(conn, 64_000) }
    var raw  = await(rawT, 30000)

    var resT = spawn { app(parseRequest(raw)) }
    var res  = await(resT, 2000)

    write(conn, formatResponse(res))
}
```

`defer` keeps cleanup deterministic even with timeouts/errors.
