## 1. Getting Started with Slug

### Writing Your First Slug Program

Create a file called `hello.slug` and add the following:

```slug
println("Hello, Slug!")
```

Run it with:

```shell
slug hello.slug
```

You should see:

```
Hello, Slug!
```

### Initial code load and library search order

Slug resolves the entry module and all `import(...)` paths relative to the command-line target first, and falls back to
the global library directory (`$SLUG_HOME/lib`) if not found.

#### 1) When the CLI target is a file path

Example:

```sh
slug ./tests/bytes.slug
````

Imports are searched **relative to the directory of the entry file**. For example:

```slug
import("slug.std")
```

is resolved as `./slug/std.slug` and searched in this order:

1. `./tests/slug/std.slug`
2. `$SLUG_HOME/lib/slug/std.slug`

#### 2) When the CLI target is not a file path

If the command-line target is not found locally, Slug treats it as a library/module name.

Example:

```sh
slug hello world
```

Slug attempts to load the entry module `hello` in this order:

1. `./hello`
2. `./hello.slug`
3. `$SLUG_HOME/lib/hello.slug`

#### Program arguments

In all cases, remaining command-line tokens after the entry target are passed through to the program:

* `args[] == ["world"]` for `slug hello world`
