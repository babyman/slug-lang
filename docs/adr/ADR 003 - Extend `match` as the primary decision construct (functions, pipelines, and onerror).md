# ADR 003 - Extend `match` as the primary decision construct (functions, pipelines, and `onerror`)

## Status

Accepted

## Context

Slug has been converging on a small set of orthogonal primitives that are:

- explicit in control flow
- composable
- refactor-friendly
- predictable at runtime

Recent work unlocked several “unexpected wins” by leaning further into `match`:

- `match` as a function body (especially powerful with recursion + `recur`)
- nested patterns (e.g. `[[h, ...t], acc]`)
- multi-parameter matching with list semantics and `...` to ignore irrelevant parameters (e.g. `[[], ...]`)

At the same time, certain constructs (notably `defer onerror`) risk becoming “one-off” control forms that duplicate
branching semantics and increase the mental surface area of the language.

We want one primary way to express “decision based on the shape/outcome of a value” across:

- normal functions
- pipelines
- error handling paths (including deferred error handlers)

## Decision

We will standardize on `match` as Slug’s primary decision construct across the language, including extended usage in
function bodies, pipelines, and `defer onerror`.

### 1) `match` as a function body

Slug supports `match` as the entire body of a function, enabling multi-arm definitions without introducing a new
construct.

- **Single-parameter functions**: `match { ... }` matches the single parameter.
- **Multi-parameter functions**: `match { ... }` matches a *positional list* of parameters in declaration order.

Example:

```slug
var sum = fn(numbers, acc = 0) match {
    [[], acc] => acc;
    [[h, ...t], acc] => recur(t, acc + h);
}
````

### 2) `...` in positional parameter patterns

When multi-parameter matching is used, `...` may be used to acknowledge additional parameters without repeating `_`
placeholders.

Example:

```slug
@export
var toCsv = fn(rows, sep = ",", quote = "\"", eol = "\r\n", acc = "") match {
    [[], ...] => acc;
    [[h, ...t], ...] => recur(t, sep, quote, eol, acc + rowToCsv(h, sep, quote, eol));
}
```

### 3) `match` in pipelines (decision boundaries)

Pipelines are for value transformation; `match` marks the boundary where meaning diverges.

Slug will support using `match` as a pipeline stage. When `match` appears without an explicit scrutinee in a pipeline
position, it consumes the prior pipeline value as its scrutinee.

Example:

```slug
input
/> parse()
/> validate()
/> match {
    {ok: v} => save(v);
    {error: e} => report(e);
}
```

### 4) `defer onerror` will use `match` semantics

`defer onerror` will be defined (or refactored) to branch via `match` over an explicit error/outcome value shape, rather
than relying on a bespoke “error-only” branching construct.

This preserves the “single decision mechanism” principle:

* outcomes are values
* branching is `match`
* error handling is explicit and structurally visible

(Exact outcome shapes are defined by the error/result conventions in use, but the branching mechanism is always
`match`.)

## Consequences

### Positive

* **Fewer special cases**: one primary decision construct reduces language surface area.
* **Improved readability**: decisions are visibly marked by `match`, especially at pipeline boundaries.
* **Refactor-friendly**: function-body matching + `recur` avoids name-dependent recursion and supports anonymous
  recursion patterns cleanly.
* **Composable semantics**: the same pattern matching rules apply in function bodies, pipelines, message handlers, and
  error handling paths.
* **Better teaching story**: “transform with pipelines, decide with match” becomes a core idiom.

### Negative

* **Parser complexity increases**: patterns are recursive and must be parsed as a dedicated grammar distinct from
  expressions.
* **More reliance on conventions for outcome shapes**: error/outcome values must be matchable in a consistent way or
  code becomes inconsistent.
* **Potential overuse**: developers may use `match` for trivial boolean branching where `if` might be simpler (style
  guidance mitigates this).

### Neutral

* **No new runtime model required**: these are primarily parsing/AST and idiom-level consolidations; evaluation remains
  structurally similar.
* **Future types remain compatible**: this ADR does not require `Option`/`Box`/`Result`, but will work well if such
  types are introduced later.
