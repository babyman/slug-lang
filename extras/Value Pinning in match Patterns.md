## Value Pinning in `match` Patterns

Slug patterns bind identifiers by default.
This enhancement introduces **value pinning**, which allows a pattern to match against an **existing value from the current scope** rather than binding a new variable.

Value pinning is **explicit**, **lexically scoped**, and works uniformly across all pattern forms.

---

## Binding vs Matching

Within a pattern:

| Syntax                       | Behaviour                         |
| ---------------------------- | --------------------------------- |
| `name`                       | Binds a new variable              |
| `"literal"` / `123` / `true` | Matches by value                  |
| `^name`                      | Matches against an existing value |
| `_`                          | Ignores the value                 |
| `...`                        | Matches remaining list elements   |

By default, identifiers in patterns **always bind**.
Pinning opts out of binding.

---

## Pinned Identifiers

A **pinned identifier** uses the `^` prefix:

```slug
^name
```

### Semantics

* `name` must already be defined in the current lexical scope
* The matcher compares the matched value with `name` using equality
* The identifier is **not bound**
* If the comparison fails, the pattern fails

Pinned identifiers behave like literals whose value is supplied from scope.

---

## Map Patterns

### Binding

```slug
match map {
    {k1:key} => ...
}
```

### Literal matching

```slug
match map {
    {k1:"v1"} => ...
}
```

### Pinned value matching

```slug
val expected = "v1";

match map {
    {k1:^expected, k2} => ...
    _ => ...
}
```

The pattern matches only if `map.k1 == expected`.

---

## List Patterns

Pinning works identically in list patterns.

### Binding

```slug
match list {
    [h1, ...] => ...
}
```

### Literal matching

```slug
match list {
    ["v1", h2, ...] => ...
}
```

### Pinned value matching

```slug
val expected = "v1";

match list {
    [^expected, h2, ...] => ...
    _ => ...
}
```

The first list element must equal `expected`.

---

## Guards and Pinning

Pinned values reduce the need for guards but do not replace them.

### Guard-based matching

```slug
match map {
    {k1, k2} if k1 == expected => ...
}
```

### Pinned pattern matching

```slug
match map {
    {k1:^expected, k2} => ...
}
```

Patterns describe **structure and fixed values**.
Guards describe **logical conditions**.

---

## Scope and Resolution Rules

* Pinned identifiers are resolved **before** pattern bindings occur
* Pinning always refers to the identifier in the **enclosing lexical scope**
* Pinned identifiers cannot be shadowed by pattern bindings
* If a pinned identifier is undefined, the match is invalid

---

## Restrictions

To keep patterns declarative and predictable:

* Pinned identifiers are **atomic only**

  * No expressions: `^a + b` ❌
  * No destructuring: `^x:y` ❌
* `...` must remain the final element in list patterns
* Pinning does not evaluate code or perform conversions

---

## Failure Behaviour

If a pinned value comparison fails:

* The pattern is considered non-matching
* Evaluation continues to the next match arm
* Guards are not evaluated for that arm

---

## Design Rationale

* Avoids implicit constant matching
* Eliminates ambiguity between binding and comparison
* Preserves simple, structural pattern matching
* Aligns with Erlang / Elixir’s proven approach
* Scales naturally to future pattern types

---

## Summary

Value pinning introduces a **clear and explicit way to match against existing values** in Slug patterns, without adding magic or complexity.

It works uniformly across maps and lists, integrates cleanly with guards, and preserves Slug’s core principles of explicitness and lexical clarity.

---

If you want, next we can:

* Turn this into a formal grammar diff
* Sketch the matcher algorithm step-by-step
* Define diagnostics and error messages
* Look ahead to how this extends to `Option` / `Box` matching

Just tell me where to go next.
