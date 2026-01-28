# Module 4: Data Structures

In this module, you will get comfortable with the two workhorse collections: lists and maps.

## Lesson 4.1: Lists

```slug
val list = [10, 20, 30]

list[1] /> println()    // 20
list[-1] /> println()   // 30
list[0:1] /> println()  // [10]
list[1:] /> println()   // [20, 30]
```

Use lists for ordered data, pipelines, and batches of work.

## Lesson 4.2: Maps

Maps store key-value pairs:

```slug
var myMap = {}
myMap = put(myMap, :name, "Slug")
get(myMap, :name) /> println()
```

## Lesson 4.3: Symbols

Symbols are interned labels used as map keys, struct fields, and type tags. They are written with a `:` prefix:

```slug
:ok
:"Content-Type"
```

Use `sym()` to create symbols from strings and `label()` to get the raw text:

```slug
sym("foo bar")      // :"foo bar"
label(:ok)          // "ok"
```

Maps with bare keys use symbols by default:

```slug
val headers = {contentType: "text/plain"}
headers[:contentType] /> println()
```

When a map uses symbol keys, you can use dot access as shorthand:

```slug
headers.contentType /> println()
```

## Lesson 4.4: Structs

Structs are schema-backed, immutable records. You define a schema with `struct`, then construct values from it.

```slug
val User = struct {
    name,
    @num age,
    active = true,
}

val u1 = User { name: "Slug", age: 2 }
u1.name /> println()
```

Update with `copy`:

```slug
val u2 = u1 copy { age: 3 }
```

Structs support introspection through `type()` and `keys()`:

```slug
type(u1) == User /> println()
keys(u1) /> println()    // [:name, :age, :active]
```

Match on structs:

```slug
match u2 {
    User { name, age } => println(name, age)
    _ => println("unknown")
}
```

### Try it

Create a map that stores a user id and name, then print a sentence using both values.
