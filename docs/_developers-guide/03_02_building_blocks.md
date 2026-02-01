# Module 2: Core Building Blocks

In this module, you will learn the essential pieces of the language: types, variables, functions, and the builtins you
will use all the time.

## Lesson 2.1: Types at a glance

Slug has a small, focused set of core types:

- `nil`: absence of a value.
- `true` / `false`: booleans.
- `number`: DEC64-inspired floating decimal values.
- `string`.
- `list`: ordered collection, e.g. `[1, 2, 3]`.
- `map`: key-value collection, e.g. `{k: v}`.
- `bytes`: byte sequence, e.g. `0x"ff00"`.
- `function`: a `fn(){}` value.
- `task`: a task handle, returned by `spawn`.

## Lesson 2.2: Comments

Slug supports two comment styles:

- `//` C-style comments.
- `#` for script-friendly shebang usage, like `#!/usr/bin/env slug`.

## Lesson 2.3: Strings

Slug supports regular strings, raw strings, and interpolation.

```slug
val name = "Slug"
val greeting = "Hello {{name}}!"
val path = 'C:\temp\file.txt' // raw string, no escapes
```

## Lesson 2.4: Numeric literals

Numbers can include underscores for readability:

```slug
val maxUsers = 1_000_000
```

## Lesson 2.5: Variables

Use `var` for mutable values and `val` for constants:

```slug
var counter = 0
val greeting = "Hello"

counter = counter + 1
counter /> println()
```

## Lesson 2.6: Semicolons are optional

Statements end at newlines, not semicolons:

```slug
var a = 1
var b = 2
(a + b) /> println
```

A line continues when it clearly should:

```slug
var sql =
    "select *"
    + " from users"
    + " where active = true"
```

A line ends when the next token would be confusing:

```slug
f(x)     // valid
f
(x)      // invalid
```

## Lesson 2.7: Dangling commas

Slug supports trailing commas in lists, maps, tags, and call arguments. It does not allow them in function definitions.

```slug
var {*} = import(
    "slug.std",
)

val map = {
    k: 50,
}

var list = [
    1,
    [1, 2,],
    11,
]

println(map, list,)
```

## Lesson 2.8: Built-in functions you will use first

### `import`

```slug
val {*} = import("slug.std")
```

### `len`

```slug
val size = len([1, 2, 3])
val textLength = len("hello")
```

### `print` and `println`

```slug
print("Hello", "Slug!")
println("Welcome to Slug!")
```

## Lesson 2.9: Modules and exports

Use `@export` to expose values from a module. `import(...)` returns a map of exports.

```slug
// math.slug
@export
val add = fn(a, b) { a + b }

// app.slug
val math = import("math")
math.add(2, 3) /> println()
```

Imports are live bindings. In cyclic imports, accessing a value before it is initialized raises a clear runtime error.

## Lesson 2.10: Command-line arguments

Slug provides two tiny, explicit builtins for arguments:

- `argv()` returns raw args as a list.
- `argm()` returns a parsed map and positionals.

```slug
// slug playground.slug -abc --user john foo.txt

argv()
// => ["-abc", "--user", "john", "foo.txt"]

argm()
// => { options: { a: true, b: true, c: true, user: "john" }, positional: ["foo.txt"] }
```

Typical usage:

```slug
var cli = argm()

match cli.options {
  {help: true} => showHelp()
  {user: u} => run(u, cli.positional[0])
  _ => fail("missing --user")
}
```

## Lesson 2.11: Unified configuration with `cfg()`

```slug
val port = cfg("port", 8080)
val dbUrl = cfg("db.url", "postgres://localhost:5432")
```

Precedence is:

1. CLI args: `--key=value` or `-k=value`.
2. Environment: `SLUG__db__port=5432`.
3. Local `slug.toml` in the current directory.
4. Global `slug.toml` in `$SLUG_HOME/lib/`.
5. In-code default.

Keys without dots are automatically namespaced to the current module.

## Lesson 2.12: Functions and closures

```slug
val add = fn(a, b) { a + b }
add(3, 4) /> println()
```

Closures capture their environment:

```slug
val multiplier = fn(factor) {
    fn(num) { num * factor }
}

val double = multiplier(2)
double(5) /> println()
```

## Lesson 2.13: Default parameters

Defaults are evaluated at call time in the function's defining module.

```slug
val dbHost = cfg("db.host", "localhost")

val connect = fn(host = dbHost, port = 5432) {
    println(host, port)
}

connect()
```

## Lesson 2.14: Named parameters

Named parameters are supported in function calls:

```slug
val greet = fn(name, title) { "Hello {{title}} {{name}}" }
greet(title: "Mr", name: "Slug") /> println()
```

## Lesson 2.15: Pipelines with the trail operator

The trail operator (`/>`) passes the value to the next function, left to right:

```slug
var double = fn(n) { n * 2 }
var map = { double: double }
var lst = [nil, double]

10 /> map.double /> lst[1] /> println("is 40")

// Equivalent to:
println(lst[1](map.double(10)), "is 40")
```

## Lesson 2.16: Function dispatch and type tags

Slug can dispatch by argument count and type tags:

```slug
fn add(@num a, @num b) { a + b }
fn add(@str a, @str b) { a + b }
```

Supported tags: `@num`, `@str`, `@bool`, `@list`, `@map`, `@bytes`, `@fn`, `@task`.

### Try it

Write an `add` overload for lists that concatenates two lists, then call it with `[1]` and `[2]`.
