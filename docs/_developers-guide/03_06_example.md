# Module 6: Mini Project - Data Pipeline

Time to build a tiny project. You will process a list of numbers by:

- Squaring each number.
- Filtering for even numbers.
- Summing the remaining values.

```slug
var {*} = import("slug.std")

val numbers = [1, 2, 3, 4, 5, 6]

val result = numbers
    /> map(fn(x) { x * x })
    /> filter(fn(x) { x % 2 == 0 })
    /> reduce(0, fn(acc, x) { acc + x })

println("Result:", result)
```

### Challenge

Change the pipeline so it squares only odd numbers, then sums them.
