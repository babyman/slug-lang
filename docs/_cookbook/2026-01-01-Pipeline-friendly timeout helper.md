---
title: Pipeline-friendly timeout helper
tags: [spawn, await]
---

If you like keeping the spawn+await pattern tidy:

```slug
var withTimeout = fn(ms, f) {
    await (spawn { f() }) within ms
}
```

Usage:

```slug
var res = await withTimeout(2000, fn() { app(req) })
```

No language magic—just an idiom.
