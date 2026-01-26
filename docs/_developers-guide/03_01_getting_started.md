# Module 1: Getting Started

In this module, you will run your first Slug program and learn how the runtime finds your files.

## Lesson 1.1: Your first program

Create a file called `hello.slug`:

```slug
println("Hello, Slug!")
```

Run it:

```shell
slug hello.slug
```

Expected output:

```
Hello, Slug!
```

### Try it

Change the string, run again, and make sure the new text shows up. You just completed your first Slug program.

## Lesson 1.2: How Slug resolves imports

Slug resolves the entry module and all `import(...)` paths relative to the command-line target first, then falls back to
the global library directory (`$SLUG_HOME/lib`).

### 1) When the CLI target is a file path

Example:

```sh
slug ./tests/bytes.slug
```

Imports are searched relative to the directory of the entry file. For example:

```slug
import("slug.std")
```

is resolved as `./slug/std.slug` and searched in this order:

1. `./tests/slug/std.slug`
2. `$SLUG_HOME/lib/slug/std.slug`

### 2) When the CLI target is not a file path

If the command-line target is not found locally, Slug treats it as a library or module name.

Example:

```sh
slug hello world
```

Slug attempts to load the entry module `hello` in this order:

1. `./hello`
2. `./hello.slug`
3. `$SLUG_HOME/lib/hello.slug`

## Lesson 1.3: Program arguments

In all cases, remaining command-line tokens after the entry target are available via `argv()` and `argm()`.

- `argv() == ["world"]` for `slug hello world`

### Try it

Run a program with two extra args and print `argv()` to confirm the list order.
