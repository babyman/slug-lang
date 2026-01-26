# Module 0: Setup

Lets get Slug installed and ready to run. Pick one path below.

## Option A: Download a precompiled binary

1. Download the latest release from the Slug [releases page](https://github.com/babyman/slug-lang/releases).
2. Extract the archive and locate the `slug` binary.
3. Add it to your `PATH` and set `SLUG_HOME` (see Local setup below).

### macOS note

If you are on macOS, remove quarantine and apply an ad-hoc signature:

```shell
xattr -d com.apple.quarantine ./bin/slug
codesign -s - --deep --force ./bin/slug
```

What these do:

- `xattr -d com.apple.quarantine ./bin/slug` removes the download quarantine so the binary can run.
- `codesign -s - --deep --force ./bin/slug` adds an ad-hoc signature so Gatekeeper is satisfied.

## Option B: Build from source

If you have Go installed:

```shell
git clone https://github.com/babyman/slug-lang.git
cd slug-lang
make build
```

## Local setup

Once you have a `slug` binary, export `SLUG_HOME` and add the binary to your `PATH`:

```shell
export SLUG_HOME=[[path to slug home directory]]
export PATH="$SLUG_HOME/bin:$PATH"
```

`SLUG_HOME` is where Slug looks for libraries.

### Try it

Open a new terminal, run `slug --version`, and confirm it prints a version.

## Troubleshooting

If something is not working, check these first:

- `slug: command not found`: make sure `PATH` includes `$SLUG_HOME/bin`.
- `SLUG_HOME` not set: export it and open a new terminal.
- macOS "cannot be opened" or "not signed": re-run the `xattr` and `codesign` commands above.
- Imports failing: confirm `SLUG_HOME/lib` exists and your `--root` is correct.

## Sublime Text syntax highlighting

If you use [Sublime Text](https://www.sublimetext.com/3), install
[Slug Syntax Highlighting](https://github.com/babyman/slug-lang/tree/master/extras/Slug.sublime-package) by
placing the package in your `Packages/User` directory.
