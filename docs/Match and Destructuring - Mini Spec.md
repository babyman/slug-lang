# Mini-Spec: `match` on Primitives (Draft 0.1)

---

## Overview

When `match` is applied to a **primitive value** (like a number, string, or boolean),  
it checks each pattern **top to bottom** until a match is found.

- If a pattern matches, its body executes.
- If no patterns match, and no wildcard `_` is present, a runtime error is thrown.

---

## Pattern Kinds for Primitives

| Pattern            | Meaning                                                              |
|:-------------------|:---------------------------------------------------------------------|
| `literal`          | Match if value equals the literal (e.g., `5`, `"hello"`, `true`)     |
| `list of literals` | Match if value matches **any** literal in the list (e.g., `1, 2, 3`) |
| `_`                | Wildcard; matches anything                                           |

---

## Syntax Examples

### Basic Match

```slug
match x {
    1 => println("one")
    2 => println("two")
    _ => println("something else")
}
```

---

### Multiple Values in One Arm

```slug
match x {
    1, 2, 3 => println("small number")
    4, 5    => println("medium number")
    _       => println("big number")
}
```

---

## Semantics

**Matching Process:**

1. Evaluate the `match` expression target once (e.g., `x`).
2. Walk arms top-to-bottom.
3. For each arm:
    - If a **literal** matches `==` target, execute body.
    - If `_`, match unconditionally (wildcard).
4. If no match found, `nil` is returned.

---

### Matching Rule Details

- **Equality Comparison** is shallow:
    - Numbers compared by value.
    - Strings compared by content.
    - Booleans compared by value.
- No implicit type coercion (e.g., `1` doesn't match `"1"`).

---

## Example: Full Behavior

```slug
var greet = fn(x) {
    match x {
        "hello", "hi" => "greeting";
        "bye"         => "farewell";
        true          => "truthy";
        42            => "magic number";
        _             => "something else";
    }
}
```

---

# Key Advantages

- **Consistent**: works exactly like pattern matching for structured types later.
- **Lightweight**: only equality checks needed.
- **Friendly Syntax**: commas separate multiple literal matches naturally.
- **Predictable Errors**: missing match falls through to runtime error unless `_` used.

---

# Bonus Idea (implemented)

You could allow small *guard* conditions too:

```slug
match x {
    0 => println("zero")
    n if n > 0 => println("positive")
    _ => println("negative")
}
```

# Mini-Spec: Destructuring in Slug (Draft 0.1)

---

## Overview

Destructuring allows **breaking apart** structured data (lists, hashes) into individual variables in:

- `var` statements
- `match` expressions
- (optionally later) function parameters

---

## Patterns

| Pattern       | Matches                                              | Binds         |
|:--------------|:-----------------------------------------------------|:--------------|
| `[]`          | Empty list                                           | Nothing       |
| `[...]`       | Any list                                             | Nothing       |
| `[_]`         | Any with exactly one element                         | Nothing       |
| `[a, b, c]`   | List with exactly three elements                     | `a`, `b`, `c` |
| `[a, b, ...]` | List with at least two elements, discards rest       | `a`, `b`      |
| `[a, ...]`    | List with at least one element, discards rest        | `a`           |
| `{}`          | Empty map/hash                                       | Nothing       |
| `{...}`       | Any map/hash                                         | Nothing       |
| `{name}`      | Map with a `name` key                                | `name`        |
| `{name, age}` | Map with `name` and `age` keys                       | `name`, `age` |
| `{name, ...}` | Map with `name` and potentially others, ignores rest | `name`        |

---

## `var` Destructuring

Syntax:

```slug
var [h1, h2, ...rest] = mylist;
var {name, age} = user;
var {name: n, age: a} = user;
```

Semantics:

- Match the structure
- Bind variables
- Raise runtime error if structure doesn't match (or if you want to be stricter: fail gracefully)

---

## `match` Destructuring

Syntax:

```slug
match something {
    [] => println("empty list")
    [h, t, _] => println(h, t)
    {} => println("empty hash")
    {name, ...} => println(name)
}
```

Semantics:

- Try patterns **top to bottom**.
- First successful match wins.
- `_` matches any single item (wildcard).
- `...` spread anything remaining entries.
- Failure to match any branch = `nil` return.

---

## Future Extensions **IMPLEMENTED**

- `var {name, ...rest} = user` → bind rest of fields into `rest`
- `[head, ...tail]` → proper spread operator in lists
- Optional matching guards: `{name} if name.startsWith("A") => {}`

---

# AST Shape: Minimal and Flexible

(assuming something like a simple node system)

---

### Binding Patterns

```rust
enum Pattern {
    Wildcard,               // _
    Identifier(name: String),// x, name, etc
    ListPattern(elements: Vec<Pattern>),  // [a:b:_]
    MapPattern(fields: Vec<(String, Pattern)>), // {name, age}
}
```

---

### `var` Statement

```rust
struct VarStatement {
    pattern: Pattern,
    value: Expr,
}
```

---

### `match` Expression

```rust
struct MatchArm {
    pattern: Pattern,
    body: Block,
}

struct MatchExpr {
    target: Option<Expr>, // None = match without expression
    arms: Vec<MatchArm>,
}
```

---

# Why This AST Design Works

- `Pattern` can be reused for `var`, `match`, and function parameters.
- Easy to walk in interpreter:
    - `Pattern::Identifier` → bind variable
    - `Pattern::Wildcard` → ignore
    - `Pattern::ListPattern` → recursively destructure
    - `Pattern::MapPattern` → lookup fields

Keeps your runtime super simple and readable while still feeling *very expressive* for users.

