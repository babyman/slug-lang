---
title: Per-request nursery (server pattern)
tags: [async, limit, spawn, await]
---

Put limits at the right scope: connection, request, or whole server.

```slug
var handleRequest = async limit 20 fn(req) {
    // at most 20 concurrent spawns inside this request
    var userT  = spawn { fetchUser(req.id) }
    var postsT = spawn { fetchPosts(req.id) }

    var user  = await userT within 200
    var posts = await postsT within 400

    render(user, posts)
}
```

This gives request-local backpressure without affecting the whole server.
