---
title: Connecting to a MySQL database
tags: [ slug.io.db, defer ]
---

To interact with MySQL, import `slug.io.db` and use a standard DSN (Data Source Name) string.

```slug
var db = import("slug.io.db")

// Format: "user:password@tcp(host:port)/dbname"
var dsn = "root:secret@tcp(127.0.0.1:3306)/my_app?parseTime=true"

var conn = dsn /> db.connect(db.MYSQL_DRIVER)
defer { db.close(conn) }

// Execute a query
var users = db.query(conn, "SELECT id, email FROM users WHERE active = ?", true)

println(users)
```
