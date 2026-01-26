# Module 8: Concurrency

Slug uses structured concurrency. That means every task has a clear owner and a clear lifetime. It is explicit and
predictable on purpose.

## Lesson 8.1: Core principles

1. Concurrency is explicit: parallel work uses `spawn`, suspension uses `await`.
2. Lifetime is lexical: a scope owns the work it spawns.
3. Values are values: `await` produces concrete values, not futures.
4. Errors and cancellation are structural: failures bubble up, cancellations flow down.

## Lesson 8.2: `nursery`

`nursery` marks a function or block as suspending.

```slug
var fetchUser = nursery fn(id) {
    ...
}
```

Key rules:

- Each task has a `currentNursery` pointer.
- Entering a `nursery` scope pushes a new nursery.
- Leaving a `nursery` scope joins or cancels its children.
- Ordinary function calls do not create a nursery.
- `spawn { ... }` registers with the nearest enclosing nursery, not the immediate call frame.

Escaping task handles:

- A task handle can be returned or stored in data.
- If it escapes its nursery scope, it is guaranteed to be settled.
- `await` on an escaped handle is always idempotent and returns immediately (or re-throws the stored error).

## Lesson 8.3: `spawn`

`spawn` creates a child task and returns a task handle.

```slug
var t = spawn {
    work()
}
```

Semantics:

- Child tasks are owned by the current nursery scope.
- Spawned tasks run on a managed worker pool.
- A nursery cannot exit until its children settle.
- `spawn` registers with the nearest enclosing nursery.

## Lesson 8.4: `await`

`await` suspends the current task until a handle completes.

```slug
var value = await taskHandle
```

With a timeout:

```slug
var value = await taskHandle within 500
```

Notes:

- Suspension happens only at `await`.
- On timeout, a `Timeout` error is raised.
- Errors propagate like normal runtime errors.

## Lesson 8.5: Task type tags

Task handles are first-class values. Use `@task` with tagged dispatch:

```slug
var awaitAll = fn(@list hs) {
    hs /> map(fn(@task h) { await h })
}
```

`await` is idempotent: awaiting an already-settled handle returns immediately (or re-throws its error).

## Lesson 8.6: Concurrency limits and timeouts

### Limits

```slug
var handler = nursery limit 10 fn() {
    ...
}
```

- Default limit is 2 * CPU cores or 4.
- Limits apply only to direct `spawn` calls in the current scope.
- Excess spawns wait for capacity.

### Timeouts

```slug
var v = await task within 1
```

- Timeouts are in milliseconds.
- A timeout raises `Timeout` and cancels the awaited task.
- Handle errors via `defer onerror` or other constructs.

## Lesson 8.7: Failure and cancellation

- If a child task fails, siblings are cancelled and the error propagates.
- If a parent scope exits early, all children are cancelled.
- Cancellation is observed at `await`.
- The runtime attempts to detect circular awaits and raises `Deadlock`.

## Lesson 8.8: Fan-out and fan-in example

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

## Lesson 8.9: Pipelines and `await`

Because `await` is syntax, use a helper for pipelines:

```slug
var awaitWithin = fn(v, dur) {
    await v within dur
}

var user = userT /> awaitWithin(500)
```

## Lesson 8.10: What Slug does not provide

Slug intentionally avoids:

- Actors or mailboxes.
- Implicit futures.
- Automatic parallelization.
- Implicit blocking on variable reads.
- Global cancellation tokens.
- Detached background tasks without explicit APIs.
