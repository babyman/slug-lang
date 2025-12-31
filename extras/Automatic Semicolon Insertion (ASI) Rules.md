# Slug Automatic Semicolon Insertion (ASI) Rules

Slug treats semicolons as **optional**. Statement boundaries are determined by `NEWLINE`, block structure, and a small set of explicit continuation rules. The goal is **predictability over guesswork**.

---

## 1. Statement terminators

A statement may be terminated by any of the following:

* `NEWLINE`
* `;`
* `}`
* end-of-file

Multiple terminators in a row are allowed (blank lines are fine).

---

## 2. NEWLINE usually terminates a statement

By default, a `NEWLINE` **ends the current statement**.

This means:

```slug
var a = 1
var b = 2
```

is always two statements.

---

## 3. NEWLINE does *not* terminate when the expression must continue

A `NEWLINE` is treated as whitespace (not a terminator) when **continuation is unambiguous**.

### 3.1 Incomplete expressions

If the parser is in a state where a right-hand side is required, the newline is ignored:

```slug
var x = a +
        b +
        c
```

This applies after tokens such as:

* binary operators
* assignment operators
* pipeline operators
* member access (`.`)

---

### 3.2 Line-start continuation operators

A `NEWLINE` does **not** terminate a statement if the next non-whitespace token is an explicit continuation operator.

**Continuation tokens include:**

* arithmetic operators: `+ - * / %`
* comparison operators: `== != < <= > >=`
* boolean operators: `&& ||`
* pipeline operator: `/>`
* member access: `.`

This enables styles like:

```slug
"select *"
+ " from users"
+ " where active = true"
```

and:

```slug
value
/> transform
/> validate
/> save
```

---

## 4. Calls and indexing may NOT start on a new line

A `NEWLINE` **always terminates** a statement if the next token is:

* `(`
* `[`

This rule forbids newline-call and newline-index syntax:

```slug
f
(x)      // invalid

xs
[i]      // invalid
```

Calls and indexing must be written with the callee on the same line:

```slug
f(x)
xs[i]
```

This rule prevents accidental parses like:

```slug
spawn { ... }
(await t)
```

being treated as a function call.

---

## 5. Map literals vs blocks

`{ ... }` is disambiguated by **parse context**:

* In **statement position**, `{ ... }` is a block
* In **expression position**, `{ ... }` is a map literal

Example:

```slug
{
    // block
}

var m = {
    a: 1,
    b: 2,
}
```

Blocks are **not expressions**.

---

## 6. NEWLINE handling inside map literals

Inside map literals:

* NEWLINE is treated as whitespace
* Entries are separated by `,`
* Trailing commas are allowed
* Closing `}` may appear on its own line

Valid examples:

```slug
var m = {
    a: 1,
    b: 2,
}

var m = {
    a:
        1,
}
```

---

## 7. Match expressions

### 7.1 Case separators

Match cases may be separated by:

* `NEWLINE`
* `;`
* any combination of the above

Blank lines are allowed.

```slug
match x {
    1 => "one"

    2 => "two";
    _ => "other"
}
```

---

### 7.2 Case bodies

The body of a match case is a **single statement**, not just an expression.

This allows:

```slug
_ => throw { type: "error", msg: "bad value" }
```

as well as expression bodies.

Case bodies may start on the next line:

```slug
1 =>
    a + b
```

---

### 7.3 Pinned patterns (`^`)

The `^` token is used to **pin constants/variables in match patterns**.

To avoid ambiguity with ASI:

* `^` is **not** a line-start continuation token
* `^` must appear at the start of the pattern line

Valid:

```slug
^NIL_TYPE => nil
```

Invalid as continuation:

```slug
expr
^ other   // does not continue
```

This keeps pattern matching reliable and explicit.

---

## 8. Semicolons remain valid but optional

Semicolons may be used anywhere a `NEWLINE` would terminate a statement.
They are never required.

---

## Design summary

* Slug’s ASI is **explicit, not heuristic**
* NEWLINE is structural, not cosmetic
* Continuation only happens when it is visually obvious
* Dangerous ambiguities (newline-call/index) are forbidden
* Formatting choices are flexible but predictable
