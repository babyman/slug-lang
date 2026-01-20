## 8. Concurrency in Slug (Structured, Explicit, Boring)

This document defines the **official concurrency model** for the Slug programming language.

Slug favors **explicit concurrency**, **lexical ownership**, and **predictable lifetime rules** over implicit
parallelism or message-passing abstractions.

---

### Core Principles

1. **Concurrency is explicit**

    * Parallel work is created only with `spawn`
    * Suspension happens only at `await`

2. **Lifetime is lexical**

    * A scope owns the work it spawns
    * A scope cannot exit until its child tasks have settled

3. **Values are values**

    * `await` produces concrete values
    * No futures/promises are visible in user code

4. **Errors and cancellation are structural**

    * Failure propagates upward
    * Cancellation propagates downward
    * Sibling tasks are cancelled on first failure (fail-fast)

---

### Language Constructs

#### `nursery`

Marks a function (or block) as **suspending**.

```slug
var fetchUser = nursery fn(id) {
    ...
}
```

**Nursery ownership:** Each executing task has a `currentNursery` pointer. Entering a `nursery` *nursery scope* (e.g.,
`nursery limit N fn` or `nursery {}`) pushes a new nursery; leaving it joins/cancels its children. Ordinary function
calls do not create a nursery.

**Spawn registration:** `spawn { ... }` registers the child task with `currentNursery` (the nearest enclosing nursery),
not with the immediate call frame or block scope.

**Escaping task handles:** A task handle may be stored in values (lists/maps) and returned from functions. If a task
handle escapes its nursery scope (for example, it is returned from a `nursery` scope), it is guaranteed to be
**settled** (completed, failed, or cancelled), because the nursery cannot exit until all its children settle. As a
result, `await` on an escaped handle is always **idempotent** and will return immediately (or re-throw the stored
error).

---

#### `spawn`

Creates a **child task** that runs concurrently.

```slug
var t = spawn {
    work()
}
```

Semantics:

* Returns a **task handle**
* The child task is owned by the current scope
* **Execution**: Spawned tasks are executed on a managed worker pool.
* A **nursery scope** cannot exit until all its spawned children settle
* `spawn` registers its child task with the nearest enclosing nursery scope (or root), not the immediate
  function-call environment.

`spawn` may execute:

* `nursery fn` bodies (cooperative suspension)
* plain `fn` bodies (blocking or CPU work)

---

#### `await`

Suspends the current task until a task handle completes.

```slug
var value = await taskHandle
```

Optional timeout:

```slug
var value = await taskHandle within 500
```

Semantics:

* Suspension happens **only** at `await`
* On timeout, a `Timeout` error is raised
* Errors propagate like normal runtime errors

---

### Type Tags (Task Handles)

Task handles are first-class values and can be passed through lists/maps and into functions. When writing polymorphic
functions that use type-tagged dispatch, Slug provides the `@task` type tag for task handles.

```slug
var awaitAll = fn(@list hs) {
    hs /> map(fn(@task h) { await h })
}
```

Notes:

* `@task` matches task handles returned by `spawn`.
* `await` is idempotent: awaiting an already-settled handle returns immediately (or re-throws its stored error).

---

### Scoped Concurrency Policies

#### Concurrency Limits

`nursery limit N` limits the number of concurrently executing child tasks **within the nursery scope**.

```slug
var handler = nursery limit 10 fn() {
    ...
}
```

Rules:

* Default limit is 2 * CPU cores or 4.
* Applies strictly to **direct** `spawn` calls in the current scope (limits are not inherited by child tasks).
* Excess spawns wait until capacity is available.
* Limits are lexical and deterministic.

---

#### Timeouts

Timeouts are expressed at `await` points:

```slug
var v = await task within 1
```

Timeout behavior:

* If no timeout is specified the default is infinite.
* Timeouts are in millisecond units.
* Raises a `Timeout` error.
* **Error Handling**: Must be handled via `defer onerror` or similar error-trapping constructs.
* Cancels the awaited task.
* Triggers normal error propagation and `defer onerror`.

---

### Failure & Cancellation Semantics

* **Deadlocks**: The runtime will attempt to detect circular dependencies (e.g., two tasks awaiting each other). If
  detected, a `Deadlock` error is raised. Otherwise, tasks will remain suspended until a timeout occurs.
* If a child task fails:

    * Sibling tasks are cancelled
    * The error propagates to the parent
* If a parent scope exits early:

    * All child tasks are cancelled
* Cancellation is observed at `await`

This ensures **fail-fast, structured execution**.

---

### Example: Parallel Fan-Out / Fan-In

```slug
var fetchUser  = nursery fn(id) { ... }
var fetchPosts = fn(id) { ... }
var renderProfile = fn(user, posts) { ... }

var showProfile = nursery limit 10 fn(id) {
    var userT  = spawn { id /> fetchUser }
    var postsT = spawn { id /> fetchPosts }

    var user  = await userT  within 500
    var posts = await postsT within 1000

    renderProfile(user, posts)
}
```

Properties:

* `fetchUser` and `fetchPosts` run in parallel
* `await` points are explicit
* Scope cannot exit until both complete
* Timeout and concurrency policies are scoped

---

### Idiomatic Helper for Pipelines

Because `await` is syntax (not a function), pipelines should use small helpers:

```slug
var awaitWithin = fn(v, dur) {
    await v within dur
}

var user = userT /> awaitWithin(500);
```

This keeps pipelines readable while preserving explicit suspension.

---

### What Slug Does *Not* Provide

Slug intentionally does **not** include:

* Actors or mailboxes
* Implicit futures
* Automatic parallelization
* Implicit blocking on variable reads
* Global cancellation tokens
* Detached background tasks (without explicit APIs)

These are considered sources of hidden complexity and unpredictable lifetime.

---

### Mental Model (Authoritative)

> * `nursery` — *I will not leave until my children are done*
> * `spawn` — *do this in parallel*
> * `await` — *pause here*

If you remember only this, you understand Slug concurrency.

---

### Summary

Slug’s concurrency model is:

* **Explicit**: no hidden parallelism
* **Structured**: scope owns lifetime
* **Portable**: implementable in Go, Zig, etc.
* **Boring**: by design

This model favors clarity, debuggability, and long-term maintainability over cleverness.

