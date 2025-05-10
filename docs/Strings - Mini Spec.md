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

## Design Philosophy

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
