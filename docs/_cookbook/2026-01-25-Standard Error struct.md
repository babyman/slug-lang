---
title: Standard Error struct
tags: [ slug.std.Error, throw, defer, slug.std ]
---

### **Problem**

You want consistent, idiomatic error payloads that are easy to pattern-match and inspect.

### **Idiom: Throw `Error { ... }` literals**

Slug's standard library defines a conventional `Error` struct in `slug.std`. Use struct literals at the throw site so
stacktraces point at the exact source line.

```slug
throw Error { type: "ValidationError", msg: "name is required" }
```

Add context via `data` or chain failures via `cause`:

```slug
var openConfig = fn(path) {
    defer onerror(err) {
        // Wrap and rethrow with context
        throw Error { type: "ConfigError", msg: "failed to open config", data: { path: path }, cause: err }
    }

    readFile(path) // throws its own Error on failure
}
```
