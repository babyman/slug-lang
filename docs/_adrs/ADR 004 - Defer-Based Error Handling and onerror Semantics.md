# ADR 004 – Defer-Based Error Handling and `onerror` Semantics

## Status

**Accepted**

## Context

Slug originally supported a `try / catch` style error-handling mechanism similar to Java and other imperative languages.
While functional, this approach conflicted with several core Slug design principles:

* Explicit control flow over implicit mechanisms
* Single, orthogonal constructs rather than overlapping abstractions
* Simplicity and predictability, especially in the presence of recursion and tail-call optimization (TCO)

As Slug evolved, the language adopted `defer` as the primary mechanism for cleanup, control-flow interception, and error
handling. This mirrors successful patterns seen in languages such as Go and C3, while remaining fully consistent with
Slug’s own semantics.

During development, several questions arose:

* How should errors be intercepted without reintroducing `try / catch`?
* How should `throw` interact with `defer`?
* How should error handlers behave under tail-call optimization?
* Should returning a value from an `onerror` handler rethrow or resolve the error?

Early implementations treated returning the original error value from an `onerror` handler as “unhandled”, implicitly
continuing the throw. While workable, this introduced a special-case semantic that violated Slug’s otherwise uniform
rules around return values.

This ADR captures the final, refined model.

## Decision

Slug adopts a **defer-centric error handling model** with **explicit throw semantics** and **no implicit rethrow
behavior**.

### 1. `throw` Semantics

* `throw value` is defined as an *early return with error intent*.
* Throwing triggers deferred execution in the same way as a normal return.
* The thrown value may be **any Slug value** (string, map, number, etc.).

There is no separate exception type hierarchy enforced by the language.

### 2. `defer` Variants

Slug supports three defer forms:

#### `defer { ... }`

* Always executes when the enclosing scope exits.
* Executes on both normal return and `throw`.

#### `defer onsuccess { ... }`

* Executes only if the scope exits normally (no `throw` occurred).

#### `defer onerror(err) { ... }`

* Executes only if the scope exits due to a `throw`.
* The thrown value is bound to `err`.

Deferred handlers execute in **LIFO order** within their lexical scope.

### 3. Return Semantics of `onerror`

**Returning a value from an `onerror` handler is always treated as a normal return.**

There are **no special cases** based on the returned value.

#### Example

```slug
var f = fn() {
    defer onerror(err) { err }
    throw "bad"
}

f()   // returns "bad"
```

This behavior is intentional and consistent with all other Slug functions.

### 4. Explicit Rethrow

To rethrow an error from an `onerror` handler, the handler **must explicitly use `throw`**.

#### Example

```slug
var f = fn() {
    defer onerror(err) {
        throw err
    }
    throw "bad"
}
```

This removes all implicit or value-based rethrow behavior.

### 5. Error Transformation and Recovery

Because `onerror` handlers return normally unless they explicitly throw, they naturally support:

* Error transformation
* Error recovery
* Error-to-value conversion
* Chained error enrichment

#### Example (error recovery)

```slug
defer onerror(err) { 0 }
```

#### Example (error transformation)

```slug
defer onerror(err) { "wrapped: " + err }
```

### 6. Interaction with Tail-Call Optimization (TCO)

* `defer` handlers are associated with lexical scopes, not stack frames.
* In tail-recursive functions, `defer` is intended for **local, non-surviving scopes**.
* Error handling logic naturally belongs in non-recursive wrapper functions.
* The presence of `defer` that must survive a tail call conservatively disables TCO.

This preserves correctness without introducing hidden allocations or runtime surprises.

### 7. Stack Traces

* Stack traces are captured at the point of `throw`.
* A built-in `stacktrace(value)` function exposes the captured trace.
* Error chaining is supported by capturing causal context when new errors are thrown during handling.

This integrates naturally with `defer onerror` and tooling such as `runSafe`.

## Consequences

### Positive

* Fully consistent semantics: return always means return, throw always means throw
* No implicit or magical control-flow rules
* Eliminates the need for `try / catch`
* Composes naturally with recursion and TCO
* Enables expressive, user-defined testing and error-handling utilities
* Aligns with Slug’s design philosophy: explicit, minimal, and easy to reason about

### Negative

* Requires users to explicitly write `throw err` to rethrow
* Slightly more verbose in rare rethrow scenarios
* Migration required from earlier `onerror` implementations that relied on implicit rethrow behavior

### Neutral

* Errors remain untyped at the language level (by design)
* Tooling and conventions (e.g. `{type, msg}` maps) remain library concerns
* The model does not preclude future typed-error systems, but does not depend on them

## Summary

This ADR formalizes Slug’s final error-handling model:

> **Errors are values.
> Defer is the control point.
> Return is always normal.
> Throw is always explicit.**

The resulting system is simple, orthogonal, TCO-safe, and deeply Slug-like.
