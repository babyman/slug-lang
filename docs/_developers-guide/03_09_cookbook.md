## 9. Cookbook

A collection of Slug idioms and useful coding patterns.

### Concurrency Idioms in Slug

Slug concurrency is **structured, explicit, and boring** by design.

#### The rules

* `async` marks code that may suspend (may use `await`).
* `spawn { ... }` starts work concurrently and returns a **task handle**.
* `await handle` waits for a task handle to complete.
* `await handle within T` waits up to `T`, then throws a timeout.
* **Nursery boundaries** are created explicitly by `async limit N ...` (and root).
* **Join points** are only:

    1. **nursery exit**, and
    2. **`await` on a task handle**.

There is no implicit waiting on variable reads or function calls.

---

#### Parallel fan-out / fan-in

```slug
async fn handler(req) {
    var userT  = spawn { fetchUser(req.id) }
    var postsT = spawn { fetchPosts(req.id) }

    var user  = await userT
    var posts = await postsT

    render(user, posts)
}
```

Parallel work is explicit and easy to see.

---

#### Timeout any work (the canonical pattern)

Timeouts apply to `await`, so to timeout a call (sync or async), wrap it in a spawn:

```slug
async fn handler(req) {
    var resT = spawn { doWork(req) }
    var res  = await resT within 2000
    res
}
```

This works for:

* normal `fn`
* `async fn`
* blocking foreign calls (FFI)

---

#### Timeout a blocking FFI call safely

If `read/accept/write` are blocking FFI, do this:

```slug
async fn readWithTimeout(conn, n, ms) {
    var t = spawn { read(conn, n) }
    await t within ms
}
```

This is the simplest portable approach.

---

#### Bounded concurrency (backpressure) with a nursery

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

---

#### “Spawn early, await late”

Start tasks as soon as you can, await when you must:

```slug
async fn handler(req) {
    var aT = spawn { fetchA(req) }
    var bT = spawn { fetchB(req) }

    var a = await aT
    // do some CPU work here while b runs...
    var b = await bT

    combine(a, b)
}
```

---

#### Gather a list of tasks, then join

```slug
async fn fetchAll(ids) {
    var tasks = ids.map(fn(id) { spawn { fetchUser(id) } })
    tasks.map(async fn(t) { await t })
}
```

This is the cleanest “fan-out/fan-in” shape for N items.

---

#### Fail-fast behavior (structured cancellation)

By default, if a task fails and you `await` it:

* the error propagates normally
* sibling tasks in the same nursery should be cancelled (fail-fast)

Typical pattern:

```slug
async fn handler(req) {
    var aT = spawn { taskA() }
    var bT = spawn { taskB() }

    var a = await aT;    // if this throws, nursery cancels siblings
    var b = await bT

    {a:a, b:b}
}
```

(Your runtime can also enforce fail-fast on nursery exit if any child failed.)

---

#### Per-request nursery (server pattern)

Put limits at the right scope: connection, request, or whole server.

```slug
var handleRequest = async limit 20 fn(req) {
    // at most 20 concurrent spawns inside this request
    var userT  = spawn { fetchUser(req.id) }
    var postsT = spawn { fetchPosts(req.id) }

    var user  = await userT within 200ms
    var posts = await postsT within 400ms

    render(user, posts)
}
```

This gives request-local backpressure without affecting the whole server.

---

#### 10) Long-running loops (use `recur`)

```slug
async fn acceptLoop(listener, app) {
    var conn = accept(listener)
    spawn { handleConn(conn, app) }
    recur(listener, app)
}
```

Important: `recur` is a loop construct; it must **not** trigger nursery joins.

---

#### 11) Always close resources with `defer`

```slug
async fn handleConn(conn, app) {
    defer { close(conn) }

    var rawT = spawn { read(conn, 64_000) }
    var raw  = await rawT within 30s

    var resT = spawn { app(parseRequest(raw)) }
    var res  = await resT within 2s

    write(conn, formatResponse(res))
}
```

`defer` keeps cleanup deterministic even with timeouts/errors.

---

#### 12) Pipeline-friendly timeout helper (optional)

If you like keeping the spawn+await pattern tidy:

```slug
var withTimeout = async fn(ms, f) {
    await (spawn { f() }) within ms
}
```

Usage:

```slug
var res = await withTimeout(2000, fn() { app(req) })
```

No language magic—just an idiom.

---

#### Anti-patterns

##### Don’t share mutable bindings across tasks

Even with immutable values, rebinding a `var` from multiple tasks can lose updates.

Prefer:

* returning values from tasks and awaiting them
* a synchronized primitive (later: `cell`, `collector`, channels)

##### Don’t expect function calls to “join”

A function call is not a join point. Only:

* nursery exit
* `await` on a handle

##### Don’t hide concurrency

If it runs concurrently, you should see `spawn`.

---

#### Rule of thumb

> **Spawn early. Await late.
> Limit at the nursery.
> Timeout at await.**

That’s Slug concurrency.
