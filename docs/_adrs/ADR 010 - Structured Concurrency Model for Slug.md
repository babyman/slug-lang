---
code: ADR010
title: Structured Concurrency Model for Slug
---
# ADR 010 – Structured Concurrency Model for Slug

## Status

Accepted

## Context

Slug requires a concurrency model that is:

* **Simple and explicit**, aligned with Slug’s philosophy
* **Portable** across implementations (Go now, bytecode VM later, potentially Zig/C)
* **Predictable**, avoiding hidden scheduling decisions or implicit parallelism
* **Safe by default**, especially for long-running processes like servers
* **Compatible with recursion and TCO (tail call optimization)**, which are core language features

The previous actor-based approach introduced significant complexity:

* Passive mailboxes
* Verbose message protocols
* Ambiguous ownership and lifecycle
* Difficulty integrating with `recur`, `defer`, and tail-call optimization

This ADR captures the final design of Slug’s structured concurrency model after extensive exploration, prototyping, and
real-world testing (including HTTP/HTMX server scenarios).

---

## Decision

Slug adopts a **structured concurrency model based on Tasks and Nurseries**, with explicit scoping and ownership rules.

### 1. Runtime and Task Separation

Slug separates execution concerns into two core concepts:

#### Runtime

A shared, global object responsible for:

* Module loading and caching
* Built-in functions
* Global configuration
* Time and scheduling primitives

The Runtime is **shared by all executing code**.

#### Task

A Task represents a single unit of execution. All running code executes inside a Task, including the program entry
point (root task).

Each Task owns:

* Its lexical environment stack
* Its call stack
* Its current nursery context
* Cancellation state
* Completion state and result

Tasks are **lightweight**, cooperative, and fully managed by the language runtime.

---

### 2. Tasks Are Awaitable (Merged Task + Handle)

Slug merges the concepts of “task” and “task handle”:

* `spawn { ... }` returns a **Task**
* `await task` waits for that Task
* Cancellation, completion, and results are properties of the Task itself

There is no separate “handle” abstraction.

---

### 3. Nurseries Define Ownership and Lifetime

A **nursery** is a runtime construct that owns a set of child Tasks.

Nurseries:

* Are created **only by `async limit N` blocks**
* Track child tasks
* Enforce concurrency limits
* Implement structured cancellation and failure propagation
* Join all child tasks before exiting

Nurseries are **not lexical environments** and are **not derived from variable scope**.

---

### 4. Dynamic Nursery Context (Not Lexical)

Each Task maintains a **dynamic nursery stack**.

Rules:

* Entering an `async limit` block pushes a new nursery
* Exiting the block pops the nursery (after joining children)
* `spawn` always registers the new Task with the **current nursery**
* Spawned tasks inherit the parent Task’s current nursery

This ensures correct ownership even when spawning occurs inside:

* Functions defined at module scope
* Deep call stacks
* Higher-order functions (e.g. `map`, `reduce`)

---

### 5. Await Consumes a Task from Its Nursery

Awaiting a Task **removes it from its owning nursery**.

Implications:

* Awaited tasks are no longer considered unhandled background work
* Errors from awaited tasks do not re-propagate at nursery exit
* Prevents duplicate error handling
* Prevents nursery child buildup
* Eliminates the need for TCO child-cleanup scans

Rule:

> Awaiting a task consumes it from the nursery.

Un-awaited tasks that fail will still poison the nursery (fail-fast).

---

### 6. Error Propagation and Handling

* Tasks may fail with runtime errors (including timeouts and cancellations)
* A nursery records the **first unhandled child failure**
* On nursery exit, that failure is injected unless already handled

#### Defer semantics:

* `defer onerror` may **handle** an error by returning normally
* If handled, the error does not propagate further
* If re-propagation is desired, the handler must `throw err`
* `defer onsuccess` runs only if the final scope result is successful

Handled errors do **not** reappear at nursery exit due to removal-on-await.

---

### 7. Tail Call Optimization (`recur`) Semantics

* `recur` is a **function-level control flow operation**
* It does **not** exit any scope
* It does **not** run defers
* It does **not** close or reset nurseries

Nurseries persist across `recur` iterations and only exit on:

* Return
* Unhandled error
* Cancellation

Iteration-local state is reset, but nursery ownership and deferred cleanup are preserved.

---

### 8. Defer Semantics in Recursive Loops

* `defer` runs on **scope exit**, not per iteration
* In a `recur` loop, defers execute **once**, when the loop finally exits
* Per-iteration cleanup must be placed in a nested block or helper function

This avoids resource leaks while preserving predictable semantics.

---

## Consequences

### Positive

* Clear, teachable mental model
* No hidden concurrency decisions
* Safe server-style programming by default
* Natural integration with recursion and TCO
* Errors are neither silently dropped nor duplicated
* Portable to future VM and non-Go implementations
* Eliminates actor-model complexity

### Negative

* Requires understanding of nurseries and structured concurrency
* Some familiar patterns (e.g. “fire and forget”) require explicit design
* `defer` inside loops can be a footgun without documentation or linting

### Neutral

* No implicit parallelism; concurrency is always explicit
* Async without `limit` provides asynchrony but not structured ownership
* Long-running services require intentional scope design (by choice)
