
## 6. Working Example: Functional Data Pipeline

We’ll build a pipeline that processes a list of numbers by:

- Squaring each number.
- Filtering for even numbers.
- Finding the sum of the remaining elements.

```slug
var {*} = import("slug.std");

val numbers = [1, 2, 3, 4, 5, 6];

val result = numbers
    /> map(fn(x) { x * x })          // [1, 4, 9, 16, 25, 36]
    /> filter(fn(x) { x % 2 == 0 })  // [4, 16, 36]
    /> reduce(0, fn(acc, x) { acc + x });  // 56

println("Result:", result);  // Output: Result: 56
```
