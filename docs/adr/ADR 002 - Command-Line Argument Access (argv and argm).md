# ADR 002: Command-Line Argument Access (`argv()` and `argm()`)

## Status

Accepted

## Context

Slug programs are commonly invoked from the command line, but the previous approach of implicitly binding command-line
arguments into the base script scope introduced several issues:

* Implicit global bindings conflicted with Slug’s design philosophy of explicitness.
* A single binding did not serve both low-level (raw) and high-level (parsed) use cases well.
* Common names such as `args` are highly collision-prone in user code.
* Existing language ecosystems either expose only raw argument vectors or require heavy, opinionated CLI frameworks for
  parsing.

The goal was to provide a **small, explicit, and composable mechanism** for accessing command-line arguments without
imposing schemas, validation rules, or side effects.

## Decision

Slug will provide **two built-in functions** for command-line argument access:

### 1. `argv()`

* Returns the raw command-line arguments as a list of strings.
* Excludes the `slug` executable itself.
* Preserves ordering and performs no parsing.

```slug
argv() -> list<string>
```

This function provides the lowest-level, POSIX-style view of invocation arguments and allows programs to implement
custom or non-standard parsing when required.


### 2. `argm()`

* Returns a simple, opinion-free structural parse of the command line.
* Produces a map with two keys:

    * `options`: a map of parsed options and flags
    * `positional`: a list of positional arguments

```slug
argm() -> { options: map, positional: list }
```

Parsing behavior:

* Short flags are expanded (`-abc` → `{a:true, b:true, c:true}`)
* Long options capture values (`--user john` → `{user:"john"}`)
* All non-option arguments are treated as positional
* No validation, defaults, aliases, or required fields are enforced

`argm()` is intentionally minimal and returns **data only**.

### Naming rationale

* `argv()` follows long-standing POSIX conventions and clearly signals a raw argument vector.
* `argm()` mirrors `argv()` in length and intent, with `m` indicating a map-based, structured representation.
* This avoids common identifier collisions such as `args`, while keeping the API compact and primitive in nature.


## Consequences

### Positive

* Eliminates implicit global bindings in module scope.
* Provides both raw and structured views of command-line arguments.
* Avoids naming collisions with common user variables.
* Keeps CLI handling explicit, local, and easy to reason about.
* Aligns with Slug’s philosophy of small primitives and composability.
* Allows higher-level argument validation or specification layers to be added later in the standard library without
  breaking changes.


### Negative

* `argm()` requires minimal documentation to explain its abbreviated name.
* No built-in validation or help generation is provided out of the box.
* Users unfamiliar with POSIX conventions may need examples to understand option parsing behavior.


### Neutral

* Argument schemas, defaults, aliases, and validation are intentionally deferred to future standard library utilities.
* Programs are expected to interpret argument data explicitly using `match` and other core language constructs.
