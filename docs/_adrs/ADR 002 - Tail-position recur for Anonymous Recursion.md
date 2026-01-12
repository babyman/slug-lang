# ADR 002 - Tail-position `recur` for Anonymous Recursion

## Status

Accepted

## Context

Slug emphasizes recursion over loops and already implements tail-call optimization (TCO) in the evaluator. However,
writing recursive functions currently requires naming the function (or otherwise capturing a self-reference), which:

* Makes anonymous recursion awkward (especially for function values/lambdas).
* Creates refactor hazards (renaming functions can break recursive call sites).
* Encourages patterns that accidentally defeat TCO (non-tail recursion) without a clear, ergonomic “this must be tail”
  signal.

Many real-world recursive idioms in Slug are effectively “looping with updated parameters,” which should be explicit,
safe, and TCO-friendly.

## Decision

Slug will introduce a new keyword: `recur`.

### Syntax

`recur` is a keyword expression that takes zero or more arguments:

```slug
recur(a, b, c)
```

The arity must match the containing function’s parameter list (including defaulted parameters once resolved by call
rules).

### Semantics

`recur(args...)` performs a tail-recursive call to the *innermost containing function* (the function literal in which
the `recur` appears), re-invoking it with the provided argument values.

This is a **self-call** construct:

* It does not look up a function by name.
* It is robust under renames and refactors.
* It is intended to model iteration via parameter rebinding.

### Tail-position restriction

`recur` is valid **only in tail position**.

* If `recur` is not in tail position, it is a compile/validation error (AST validation phase).
* Tail position is defined consistently with Slug’s existing TCO rules (e.g., final expression of a block, both branches
  of an `if`/`match` arm when returned as the arm result, etc.).

### Binding rule in nested functions

If functions are nested, `recur` binds to the *nearest* enclosing function literal:

```slug
fn outer(n) {
  fn inner(x) {
    recur(x - 1) // recurs inner, not outer
  }
}
```

Using `recur` outside any function is an error.

### Runtime implementation

`recur` compiles/lowers to the same mechanism as tail-call optimization:

* In evaluation, `recur` returns a `TailCall` (or equivalent) that targets the current function object and carries
  evaluated arguments.
* The existing tail-call unwinding loop executes the call without growing the stack.

### Errors and diagnostics

Slug will produce clear validation errors:

* `recur is only allowed in tail position`
* `recur must appear within a function`
* `recur arity mismatch: expected N arguments, got M`

### Interaction with multi-arm function definitions

Within any arm body, `recur(...)` targets the overall containing function, not a specific arm. Dispatch/matching occurs
as normal on the new arguments when the function is re-entered.

### Tooling expectation (non-blocking)

The AST debug output should represent `recur` distinctly (e.g., `RecurExpr(args=...)`) to help users understand
tail-position legality during development.

## Consequences

### Positive

* Enables **anonymous recursion** cleanly (especially for function values).
* Eliminates recursion-by-name refactor hazards (renames do not break recursion).
* Makes TCO-friendly iteration idioms **ergonomic and explicit**.
* Prevents accidental non-tail recursion by construction (tail-position enforcement).
* Very small runtime complexity increase; reuses the existing TCO machinery.

### Negative

* Introduces a construct that is only valid in a specific AST shape (tail position), which may confuse new users.
* Requires precise tail-position analysis in the validator, including across blocks/conditionals/match arms.
* Needs crisp documentation to avoid ambiguity about which function `recur` targets in nested scopes.

### Neutral

* `recur` is syntactic/semantic sugar over existing tail-call mechanics; it does not change Slug’s fundamental execution
  model.
* Existing code remains valid; `recur` is additive.
* Encourages a “loop via recursion” style that is already aligned with Slug’s design philosophy.

