---
title: Pipeline-friendly timeout helper
tags: [spawn, await]
---

If you like keeping the spawn+await pattern tidy:

```slug
var {*} = import("slug.channel")
var withTimeout = fn(ms, f) {
    await(spawn { f() }, ms)
}
```

Usage:

```slug
var res = withTimeout(2000, fn() { app(req) })
```

No language magic—just an idiom.
