# Module 0.1: Running Slug

This is your quick-start cheat sheet for running programs.

## Lesson 0.1: Shell scripts

You can run Slug files as scripts if `slug` is on your `PATH`:

```shell
#!/usr/bin/env slug
println("Hello Slug!")
```

## Lesson 0.2: CLI usage

```shell
slug --root [path to module root] script[.slug] [args...]
```

- `--root` sets the module root for imports.
- `script` can be a file path or a module name.
- extra tokens become program arguments.

## Lesson 0.3: REPL

Stay tuned for this feature.
