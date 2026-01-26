# Module 3: Functional Programming

Slug shines when you lean into functional patterns. Let's practice the big three: map, filter, reduce, plus pattern
matching.

## Lesson 3.1: Map, filter, reduce

```slug
var {*} = import("slug.std")

val list = [1, 2, 3, 4, 5]

val squares = list /> map(fn(v) { v * v })           // [1, 4, 9, 16, 25]
val evens = list /> filter(fn(v) { v % 2 == 0 })     // [2, 4]
val sum = list /> reduce(0, fn(acc, v) { acc + v })  // 15

squares /> println()
evens /> println()
sum /> println()
```

## Lesson 3.2: Pattern matching

`match` lets you destructure values directly.

```slug
val classify = fn(value) {
    match value {
        0 => "zero"
        1 => "one"
        _ => "other"
    }
}

classify(1) /> println()
classify(5) /> println()
```

You can match lists too:

```slug
val sumList = fn(list) {
    match list {
        [h, ...t] => h + sumList(t)
        [] => 0
    }
}

sumList([1, 2, 3]) /> println()
```

### Pattern matching extras

Pin an existing value with `^name`:

```slug
val expected = 42

match value {
    ^expected => println("matched 42")
    _ => println("nope")
}
```

Use `...` to capture the rest of a list:

```slug
match list {
    [head, ...tail] => println(head, tail)
    [] => println("empty")
}
```

## Lesson 3.3: Higher-order functions

```slug
val applyTwice = fn(f, v) { f(f(v)) }

val increment = fn(x) { x + 1 }
applyTwice(increment, 10) /> println()
```

### Try it

Write a function `times` that takes `n` and a function `f`, then applies `f` to an input value `n` times.
