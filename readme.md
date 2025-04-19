Slug
===

An interpreted programming language.

Types
===

- Integer
- Boolean
- String
- Map
- List
- Null

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

Comments
===

`//` is supported since the language follows `C` language style conventions.

`#` is supported to allow easy execution as a shell script with the inclusion of `#!`. For example
if `SLUG_HOME` is exported and `slug` is on the users path.

```shell
# slug home
export SLUG_HOME=[[path to slug home directory]]
export PATH="$SLUG_HOME/bin:$PATH"
```

The following shell script works.

```shell
#!/usr/bin/env slug
puts("Hello from a Slug script!")
```
