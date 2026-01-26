# Module 7: Testing

Slug has built-in testing tags so you can keep tests next to the code they verify.

## Lesson 7.1: Parameterized tests with `@testWith`

```slug
@testWith(
    [3, 5], 8,
    [10, -5], 5,
    [0, 0], 0
)
var parameterizedTest = fn(a, b) {
    a + b
}
```

- Each pair is inputs plus the expected output.
- The test runner executes the function for each pair.

## Lesson 7.2: Standard tests with `@test`

```slug
var {*} = import("slug.test")

@test
var simpleTest = fn() {
    val result = 1 + 1
    result /> assertEqual(2)
}
```

## Lesson 7.3: Running tests

Run tests for a module with:

```shell
slug test path_to_source.slug
```

Example output:

```
Results:

Tests run: 33, Failures: 0, Errors: 0

Total time 1ms
```

### Try it

Add a new `@testWith` case that checks subtraction, then run the tests.
