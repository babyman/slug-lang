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


Generalized `match` to Replace `if`/`else`
===

Expanding `match` to handle conditional logic beyond just matching patterns could streamline the language even further, possibly removing the need for `if`/`else`.

Simple Conditional Match
---

``` slug
var x = 5;

match x {
  5 => print("x is 5");
  10 => print("x is 10");
  _ => print("x is something else");
}
```

How it works:
- `match` compares `x` against values in order.
- The `_` case acts as the "else".

Match with Expressions
---

``` slug
var x = 15;

match {
  x < 10 => {"x is less than 10"}
  x == 15 => {"x is exactly 15"}
  x > 20 => {"x is greater than 20"}
  _ => {"x is something else"}
}
```
Key Difference from Standard `match`:
- Instead of being tied to a single object like `x`, each branch here evaluates a condition

Replacing Switch-like Logic:
---

For cases where multiple conditions depend on a common variable, you can preserve traditional switch-like behavior:

``` slug
match x {
  1, 2, 3 => {"x is 1, 2, or 3"}
  _ => {"x is unexpected"}
}
```
4..6 => {"x is within the range 4-6"}

Errors
===

Throw
---

| Syntax                      | Expansion                                     |
|-----------------------------|-----------------------------------------------|
| `throw FileError`           | `throw { "type": "FileError" }`               |
| `throw FileError()`         | `throw { "type": "FileError" }`               |
| `throw FileError({ path })` | `throw { "type": "FileError", "path": path }` |

This approach guarantees that errors always follow the format`Hash`, which keeps the error-handling mechanism consistent, predictable, and extensible.

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
