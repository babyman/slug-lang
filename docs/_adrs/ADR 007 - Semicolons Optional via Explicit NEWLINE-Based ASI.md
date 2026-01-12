---
code: ADR007
title: Semicolons Optional via Explicit NEWLINE-Based ASI
---
# ADR 007 – Semicolons Optional via Explicit NEWLINE-Based ASI

## Status

Accepted


## Context

Slug historically allowed semicolons to be optional, but the rules around when they were required were implicit and
inconsistent. This led to:

* unpredictable parse errors
* silent misparses
* formatting styles that appeared valid but failed at runtime
* difficulty reasoning about statement boundaries

As Slug codebases grew, this ambiguity became a major pain point. The language needed a principled, explicit, and
testable approach to semicolon elision that aligned with Slug’s core philosophy:

> explicit over implicit, predictable over permissive


## Decision

Slug adopts a **NEWLINE-based Automatic Semicolon Insertion (ASI) model** with explicit continuation rules.

Semicolons are optional everywhere, but **NEWLINE is treated as a real structural token**. The parser determines
statement boundaries using deterministic rules rather than heuristics.

### 1. Statement termination

A statement may be terminated by:

* `NEWLINE`
* `;`
* `}`
* end-of-file

Multiple terminators in a row are allowed.


### 2. Default behavior

A `NEWLINE` **terminates the current statement** by default.

This ensures line-oriented code behaves predictably and prevents accidental statement merging.


### 3. Explicit continuation rules

A `NEWLINE` is treated as whitespace (not a terminator) only when continuation is **unambiguous**.

Continuation occurs when:

#### 3.1 The expression is incomplete

The parser is expecting a right-hand side (e.g. after an operator).

```slug
var x = a +
        b +
        c
```

#### 3.2 The next line begins with a continuation operator

Supported continuation tokens include:

* arithmetic: `+ - * / %`
* comparison: `== != < <= > >=`
* boolean: `&& ||`
* pipeline: `/>`
* member access: `.`

```slug
value
/> transform
/> validate
```


### 4. Newline-call and newline-index are forbidden

A `NEWLINE` **always terminates** a statement if the next token is:

* `(`
* `[`

This forbids ambiguous constructs such as:

```slug
f
(x)    // invalid
```

and prevents accidental parses like calling the result of a block.

Calls and indexing must be written on the same line as their callee.


### 5. Map literals vs blocks

Brace usage is disambiguated by parse context:

* `{ ... }` in **statement position** → block
* `{ ... }` in **expression position** → map literal

Blocks are not expressions.


### 6. NEWLINE handling inside map literals

Inside map literals:

* NEWLINE is treated as whitespace
* entries are separated by commas
* trailing commas are allowed
* closing `}` may appear on its own line


### 7. Match expressions

* Match cases may be separated by `NEWLINE`, `;`, or both
* Case bodies are parsed as **single statements**, not only expressions
* Case bodies may start on the following line
* Pinned patterns using `^` are supported

The `^` token is **not** treated as a line-start continuation operator to avoid ambiguity with ASI.


### 8. Semicolons

Semicolons remain valid but are never required. They act as explicit statement terminators and may be used for stylistic
or transitional reasons.


## Consequences

### Positive

* Completely predictable statement boundaries
* Removal of thousands of unnecessary semicolons without breaking behavior
* Elimination of silent misparses caused by heuristic ASI
* Clear, documented formatting rules
* Strong alignment with Slug’s design philosophy

### Negative

* Some formatting styles that appear valid in other languages (e.g. newline-call) are intentionally rejected
* The parser is stricter than before and may surface errors earlier

### Neutral

* Semicolons remain supported for compatibility and clarity
* Future tooling (formatter, parser recovery) can rely on stable ASI rules

