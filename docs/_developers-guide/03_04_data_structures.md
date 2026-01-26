# Module 4: Data Structures

In this module, you will get comfortable with the two workhorse collections: lists and maps.

## Lesson 4.1: Lists

```slug
val list = [10, 20, 30]

list[1] /> println()    // 20
list[-1] /> println()   // 30
list[:1] /> println()   // [10]
list[1:] /> println()   // [20, 30]
```

Use lists for ordered data, pipelines, and batches of work.

## Lesson 4.2: Maps

Maps store key-value pairs:

```slug
var myMap = {}
myMap = put(myMap, "name", "Slug")
get(myMap, "name") /> println()
```

## Lesson 4.3: Structs

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

Match on structs:

```slug
match u2 {
    User { name, age } => println(name, age)
    _ => println("unknown")
}
```

### Try it

Create a map that stores a user id and name, then print a sentence using both values.
