Slug
===

A small, opinionated programming language.

Slug Command
===

Setup
---

```shell
# slug home
export SLUG_HOME=[[path to slug home directory]]
export PATH="$SLUG_HOME/bin:$PATH"
```

Shell scripts
---
The following shell script works.

```shell
#!/usr/bin/env slug
println("Hello Slug!")
```

CLI
---

```shell
slug --root [path to module root] script[.slug] [args...]
```

Repl
---

Slug has a simple repl if launched without a script.

Comments
===

`//` is supported since the language follows `C` language style conventions.

`#` is supported to allow easy execution as a shell script with the inclusion of `#!`. For example, if `SLUG_HOME` is
exported and `slug` is on the user path.


Types
===

- `Nil`
- `Boolean`: true or false
- `Integer`
- `String`
- `List`: []
- `Map`: {}
- `Function`: fn(){}

Operator Precedence and Associativity
===

| Prec | Operator  | Description                      | Associates |
|------|-----------|----------------------------------|------------|
| 1    | () [] .   | Grouping, Subscript, Method call | Left       |
| 2    | - ! ~     | Negate, Not, Complement          | Right      |
| 3    | * / %     | Multiply, Divide, Modulo         | Left       |
| 4    | + -       | Add, Subtract                    | Left       |
| 6    | << >>     | Left shift, Right shift          | Left       |
| 7    | &         | Bitwise and                      | Left       |
| 8    | ^         | Bitwise xor                      | Left       |
| 9    | \|        | Bitwise or                       | Left       |
| 10   | < <= > >= | Comparison                       | Left       |
| 12   | == !=     | Equals, Not equal                | Left       |
| 13   | &&        | Logical and                      | Left       |
| 14   | \|\|      | Logical or                       | Left       |
| 15   | ?:        | Conditional*                     | Right      |
| 16   | =         | Assignment                       | Right      |

Imports
===

```slug
// import all exports from slug.system
import slug.system.*;

// import only sqr and sum 
import functions.{sqr, sum};

// import `sqr` as square and `sum` as foo
import functions.{sqr as square, sum as foo};
```

Imports are loaded during on demand, circular imports are supported. The search for an import will check for files by
substituting the `.` for file path separators, for example `slug.system` will become `/slug/system.slug`

- project root (default current directory)
- the $SLUG_HOME/lib directory

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
    - If a **list** contains target, execute body.
    - If `_`, match unconditionally (wildcard).
4. If no match found, runtime error unless `_` was present.

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
        "hello", "hi" => println("greeting")
        "bye"         => println("farewell")
        true          => println("truthy")
        42            => println("magic number")
        _             => println("something else")
    }
}
```

---

## Parser/AST Notes

Each `MatchArm` in the AST could look like:

```rust
struct MatchArm {
    patterns: Vec<PrimitivePattern>, // one or more literals or wildcard
    body: Block,
}

enum PrimitivePattern {
    Literal(LiteralValue),
    Wildcard,
}
```

Where `LiteralValue` could be:

```rust
enum LiteralValue {
    Number(f64),
    String(String),
    Boolean(bool),
}
```

---

# Key Advantages

- **Consistent**: works exactly like pattern matching for structured types later.
- **Lightweight**: only equality checks needed.
- **Friendly Syntax**: commas separate multiple literal matches naturally.
- **Predictable Errors**: missing match falls through to runtime error unless `_` used.

---

# ⚡ Bonus Idea (optional later)

You could allow small *guard* conditions too:

```slug
match x {
    0 => println("zero")
    n if n > 0 => println("positive")
    _ => println("negative")
}
```

But for now — **perfect to leave that out** and focus on pure matching!

# Mini-Spec: Destructuring in Slug (Draft 0.1)

---

## Overview

Destructuring allows **breaking apart** structured data (lists, hashes) into individual variables in:

- `let` statements
- `match` expressions
- (optionally later) function parameters

---

## Patterns

| Pattern       | Matches                                              | Binds         |
|:--------------|:-----------------------------------------------------|:--------------|
| `[]`          | Empty list                                           | Nothing       |
| `[a:b:c]`     | List with exactly three elements                     | `a`, `b`, `c` |
| `[a:b:_]`     | List with at least two elements, discards rest       | `a`, `b`      |
| `[a:_]`       | List with at least one element, discards rest        | `a`           |
| `{}`          | Empty map/hash                                       | Nothing       |
| `{name}`      | Map with a `name` key                                | `name`        |
| `{name, age}` | Map with `name` and `age` keys                       | `name`, `age` |
| `{name, _}`   | Map with `name` and potentially others, ignores rest | `name`        |
| `{_}`         | Any map/hash                                         | Nothing       |

- `_` **discards** unmatched parts.
- No spread/rest binding (like `rest`) inside hashes yet — keep it simple first.

---

## `let` Destructuring

Syntax:

```slug
let [h1:h2:rest] = mylist
let {name, age} = user
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
    [h:t:_] => println(h, t)
    {} => println("empty hash")
    {name, _} => println(name)
}
```

Semantics:

- Try patterns **top to bottom**.
- First successful match wins.
- `_` matches anything (wildcard).
- Failure to match any branch = runtime error (or default error handling).

---

## Future Extensions

(Easy to add later if needed:)

- `let {name, ...rest} = user` → bind rest of fields into `rest`
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

### `let` Statement

```rust
struct LetStatement {
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

# ✨ Why This AST Design Works

- `Pattern` can be reused for `let`, `match`, and function parameters.
- Easy to walk in interpreter:
    - `Pattern::Identifier` → bind variable
    - `Pattern::Wildcard` → ignore
    - `Pattern::ListPattern` → recursively destructure
    - `Pattern::MapPattern` → lookup fields

Keeps your runtime super simple and readable while still feeling *very expressive* for users.

# Slug Language Mini-Spec: Strings

## 1. String Literals

Slug supports two types of string literals:

- **Raw Strings**: Enclosed in single quotes `'...'`.
- **Interpolated Strings**: Enclosed in double quotes `"..."`.

---

## 2. Raw Strings (`'...'`)

- Raw strings preserve the exact characters between the single quotes.
- No escape sequences are interpreted.
- No interpolation is performed.

**Examples**:

```slug
var path = 'C:\Program Files\Slug'
var text = 'Line1\nLine2' // \n is two characters, not a newline
var template = 'Hello, {{user}}!' // {{user}} stays literal
```

---

## 3. Interpolated Strings (`"..."`)

- Interpolated strings support **escape sequences** and **expression interpolation**.
- Escape sequences inside interpolated strings:
    - `\n` → newline
    - `\t` → tab
    - `\\` → backslash
    - `\"` → double quote
- Expressions enclosed in `{{ ... }}` are evaluated dynamically using the current scope.

**Examples**:

```slug
var user = "Sluggo"
var greeting = "Hello, {{user}}!" // Hello, Sluggo!

var count = 5
var message = "You have {{count}} new messages."

var pi = 3.14
var display = "Pi is approximately {{pi}}."
```

**Function calls inside interpolation** are allowed:

```slug
var today = getDate()
var report = "Today's date is {{formatDate(today)}}."
```

---

## 4. Multi-line Strings (`"""..."""`)

- Multi-line strings are created using triple double-quotes `"""`.
- They may span multiple lines and support the same escape and interpolation features as regular interpolated strings.
- Leading and trailing whitespace inside the triple quotes is preserved exactly.

**Examples**:

```slug
var email = """
Hello {{user}},

Your subscription will expire on {{formatDate(expirationDate)}}.

Regards,
The Slug Team
"""
```

---

## 5. Syntax Rules

- In interpolated strings, each `{{` must be matched by a corresponding `}}`.
- Unmatched or malformed interpolation blocks cause a compile-time error.
- Escape sequences must be recognized only inside interpolated strings; invalid escape sequences cause a compile-time
  error.
- Nested `{{...}}` inside interpolation is not supported initially.

---

## 6. Summary Table

| Feature                 | Raw Strings `'...'` | Interpolated Strings `"..."` / `"""..."""` |
|-------------------------|---------------------|--------------------------------------------|
| Escape Sequences        | No                  | Yes                                        |
| Interpolation `{{...}}` | No                  | Yes                                        |
| Multiline Support       | No                  | Yes (`"""..."""`)                          |

---

## Design Philosophy

- **Raw by default** with `'...'` for exact text.
- **Smart and powerful** with `"..."` when escape and interpolation are needed.
- **Simple, explicit, developer-friendly behavior.**
- **No hidden magic** outside of clearly delimited `{{ ... }}` blocks.

# Slug Language Mini-Spec: Future String Enhancements (Handlebars Logic)

---

## 1. Overview

Slug plans to extend interpolated strings (`"..."` and `"""..."""`) with embedded **Handlebars-style logic blocks** for
dynamic content generation.

These blocks allow conditionals, loops, and scoped evaluations directly inside strings while maintaining a simple,
readable syntax.

---

## 2. Planned Features

### 2.1 Conditional Blocks (`#if`, `#else`, `#unless`)

- `#if` begins a conditional block.
- `#else` provides an alternate path.
- `#unless` is a shorthand for negated conditions.
- `/if` closes the conditional block.

**Example**:

```slug
var loggedIn = true

var message = """
{{#if loggedIn}}
Welcome back, {{user}}!
{{else}}
Please log in to continue.
{{/if}}
"""
```

**Rules**:

- Every opening block must have a matching closing block.
- Conditions are simple expressions evaluated at runtime.
- Nesting of blocks may be supported later.

---

### 2.2 Iteration Blocks (`#each`)

- `#each` loops over a list or iterable.
- Inside the block, `this` refers to the current item.

**Example**:

```slug
var items = ["apple", "banana", "cherry"]

var list = """
Items:
{{#each items}}
- {{this}}
{{/each}}
"""
```

**Optional Future Enhancements**:

- Expose `@index` (zero-based index) within the loop body.

---

### 2.3 Scoped Context (`#with`)

- `#with` temporarily changes the scope to a specific object or map.
- Allows concise access to nested properties.

**Example**:

```slug
var user = { name = "Sluggo", age = 7 }

var intro = """
{{#with user}}
Name: {{name}}
Age: {{age}}
{{/with}}
"""
```

---

## 3. Syntax and Validation Rules

- Each block (`#if`, `#each`, `#with`) must have a corresponding closing block (`/if`, `/each`, `/with`).
- Mismatched or incomplete blocks are **compile-time errors**.
- Expressions inside `{{ ... }}` are full Slug expressions, including function calls.
- No nested interpolation inside `{{ ... }}` expressions (flat structure only, initially).

---

## 4. Summary Table

| Feature            | Description                                       |
|--------------------|---------------------------------------------------|
| `#if / else / /if` | Conditional content based on a boolean expression |
| `#unless`          | Conditional content based on negated expression   |
| `#each / /each`    | Loop over lists or arrays                         |
| `#with / /with`    | Change evaluation scope to a sub-object           |

---

##  Design Philosophy

- **Keep templates simple, readable, and predictable.**
- **Fail early**: Compilation should catch unmatched or invalid blocks.
- **Scope is clear**: No hidden magic or implicit behaviors.
- **Power comes from full Slug expressions inside `{{...}}`**, not special helpers.

---

## Recommended Rollout Plan

| Phase   | Feature Set                                  |
|---------|----------------------------------------------|
| Phase 1 | Inline `{{expression}}` interpolation only   |
| Phase 2 | Add `#if / else / unless` conditional blocks |
| Phase 3 | Add `#each`, `#with` scoped evaluation       |

Errors
===

Throw
---

| Syntax                      | Expansion                                     |
|-----------------------------|-----------------------------------------------|
| `throw FileError`           | `throw { "type": "FileError" }`               |
| `throw FileError()`         | `throw { "type": "FileError" }`               |
| `throw FileError({ path })` | `throw { "type": "FileError", "path": path }` |

This approach guarantees that errors always follow the format`Hash`, which keeps the error-handling mechanism
consistent, predictable, and extensible.

Try/Catch
---

``` slug
try {
    connectToDatabase();
} catch err {
    { "type": "ConnectionError", "message": msg } => print("Connection failed: " + msg);
    { "type": "TimeoutError" } => print("Operation timed out!");
    _ => throw err // Re-throw unexpected errors;
}
```

Build In Functions
===

assert(test boolean, message string*)
---

Assert something as true, will error if false. If an optional message string is provided that will be included in the
error.
