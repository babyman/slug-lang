# ADR 010 - Environment Shallow Capture for Spawned Tasks

## Status

Accepted

## Context

In Slug, the `spawn` expression creates a concurrent task that captures its lexical environment. Simultaneously, Slug
uses Tail Call Optimization (TCO) via the `recur` keyword to allow infinite recursion in constant stack space.

A conflict arises in long-running loops (e.g., a web server's `acceptLoop`). When `recur` is called, the current block's
environment is "reset" (`ResetForTCO`) to prevent memory leaks and clear local bindings for the next iteration. However,
if a task was `spawned` in that block, its reference to the environment becomes invalid (the variables it needs are
wiped), leading to "identifier not found" errors.

Simply removing `ResetForTCO` leads to unbounded memory growth, while chaining environments leads to logical stack
growth and memory leaks.

## Decision

To preserve lexical integrity without leaking memory, Slug will implement a **Shallow Capture** mechanism for all
`spawn` expressions.

### Shallow Copy of Local Bindings

When a `spawn` expression is evaluated, the runtime will create a shallow copy of the *current* environment's local
bindings. This copy:

* Contains pointers to all local variables defined in the immediate block.
* Shares the same `Outer` environment link (preserving live access to module-level and global variables).
* Is private to the spawned task.

### Non-Destructive TCO

The `ResetForTCO` operation will continue to wipe the `Bindings` map of the active environment. Because the spawned task
holds its own copied map, it remains unaffected by the reset.

## Consequences

### Positive

* **Safety:** Spawned tasks can safely access local variables even if the parent loop has moved on to a new `recur`
  iteration.
* **Memory Efficiency:** The main loop remains $O(1)$ in memory. The copied bindings for a task are garbage collected as
  soon as that specific task completes.
* **Predictability:** Matches the mental model that a `spawn` captures a "snapshot" of the local world at the moment it
  was called.

### Negative

* **Performance Overhead:** Every `spawn` now incurs the cost of copying the local bindings map (usually small, but
  non-zero).
* **Mutable Locals:** If a local variable is reassigned *after* a `spawn` but *before* a `recur`, the spawned task will
  not see the update (since it has a copy of the binding pointer). However, this is generally considered a "pro" in
  concurrent programming as it prevents race conditions on local stack-like variables.

### Neutral

* **Outer Scopes:** Changes to variables in the `Outer` chain (module-level) are still "live" and shared, as the `Outer`
  reference is not copied but shared.
