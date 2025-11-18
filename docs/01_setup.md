# Installing Slug

## Download a Precompiled Binary from GitHub

You can download the latest release from the Slug [releases page](https://github.com/babyman/slug-lang/releases). Grab
the latest release for your platform and architecture and extract the binary.

Once you the Slug release you will need to configure your `PATH` to include the `slug` binary,
see [local setup](#local-setup) for more details.

> **IMPORTANT**: If you are running OSX you will also need to `fix` permissions on the binary:
>
> ```shell
> xattr -d com.apple.quarantine ./bin/slug
> codesign -s - --deep --force ./bin/slug
> ```
>
> What do these commands do?
>
> - `xattr -d com.apple.quarantine ./bin/slug` Removes the macOS quarantine attribute added to files downloaded from the
    internet, allowing the binary to run without “cannot be opened” security prompts.
>
> - `codesign -s - --deep --force ./bin/slug` Applies an ad-hoc code signature to the binary (and any nested code with
    --deep), which satisfies macOS Gatekeeper requirements and prevents “not signed” execution errors.

## Build from source

If you have Go installed, you can build from source:

```shell
git clone https://github.com/babyman/slug-lang.git
cd slug-lang
make build
```

## Local setup

Once you have a `slug` binary, you will need to export `$SLUG_HOME` and add the binary to your `$PATH`. `$SLUG_HOME`
is the directory where slug will find its libraries.

```shell
# slug home
export SLUG_HOME=[[path to slug home directory]]
export PATH="$SLUG_HOME/bin:$PATH"
```

## Sublime Syntax Highlighting

If you are using [Sublime Text](https://www.sublimetext.com/3), you can install
the [Slug Syntax Highlighting](https://github.com/babyman/slug-lang/tree/master/extras/Slug.sublime-package) by
downloading the package and placing it in your Sublime Text `Packages/User` directory.
