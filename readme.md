Slug
===

An interpreted programming language.

Comments
===

`//` is supported since the language follows `C` language style conventions.

`#` is supported to allow easy execution as a shell script with the inclusion of `#!`.  For example 
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
