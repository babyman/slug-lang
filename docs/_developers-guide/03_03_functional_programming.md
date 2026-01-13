## 3. Functional Programming Constructs

Slug excels at **functional programming**, empowering you with concise tools and expressive patterns.

### Map, Filter, and Reduce

- **Map**: Transform a list using a function.
- **Filter**: Keep elements that satisfy a condition.
- **Reduce**: Aggregate a list into a single value.

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

---

### Pattern Matching

Use `match` to destructure and inspect values directly.

```slug
var {*} = import("slug.std")

val classify = fn(value) {
    match value {
        0 => "zero"
        1 => "one"
        _ => "other"  // Catch-all case
    }
}

classify(1) /> println()  // Output: one
classify(5) /> println()  // Output: other
```

`match` can also destructure complex data like lists:

```slug
var {*} = import("slug.std")

val sumList = fn(list) {
    match list {
        [h, ...t] => h + sumList(t)  // Head and Tail destructuring
        [] => 0                      // Base case
    }
}

sumList([1, 2, 3]) /> println()  // Output: 6
```

---

### Higher-Order Functions

Slug supports higher-order functions: functions that accept and return functions.

Example:

```slug
var {*} = import("slug.std")

val applyTwice = fn(f, v) { f(f(v)) }

val increment = fn(x) { x + 1 }
applyTwice(increment, 10) /> println()  // Output: 12
```

