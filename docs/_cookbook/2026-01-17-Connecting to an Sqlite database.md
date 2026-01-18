---
title: Connecting to an Sqlite database
tags: [ slug.io.db, defer ]
---

To interact with Sqlite, import `slug.io.db` and use a standard DSN (Data Source Name) string.

```slug
var db = import("slug.io.db")

// Format: "file:path/to/database.db"
var dsn = "file:my_app.db"

var conn = dsn /> db.connect(db.SQLITE_DRIVER)
defer { db.close(conn) }

// Execute a query
var users = db.query(conn, "SELECT id, email FROM users WHERE active = ?", true)

println(users)
```

Sqlite also supports an in memory database using `var dsn = ":memory:"`.
