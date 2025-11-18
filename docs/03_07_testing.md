
## 7. Writing and Running Tests in Slug

Slug provides an integrated testing mechanism with the use of the tags `@test` and `@testWith`. These tags
simplify the process of writing unit tests for your Slug code and enable test-driven development by allowing you to
define and execute tests directly within your modules.

### Using `@testWith`

The `@testWith` tag is used for parameterized tests, where a single function can be tested with multiple sets of
inputs and expected outputs. This allows for concise and comprehensive test coverage.

To create a test with `@testWith`:

```slug
@testWith(
    [3, 5], 8,
    [10, -5], 5,
    [0, 0], 0
)
var parameterizedTest = fn(a, b) {
 a + b; 
}
```

- **Definition**: The `@testWith` tag takes a series of arguments. Each pair consists of input parameters and the
  expected output.
- **Execution**: The test runner executes the function for each input-output pair.
- **Pass Criteria**: For each set of inputs, if the function's return matches the expected value, the test passes.
- **Fail Criteria**: A mismatch between the actual output and expected output reports a failure.

### Using `@test`

The `@test` tag marks a function as a test case. These functions are executed independently, and the results of
assertions or errors during their execution determine if the test passes or fails.

To create a simple test using `@test`:

```slug
var {*} = import("slug.test");
@test
var simpleTest = fn() {
    val result = 1 + 1;
    result /> assertEqual(2);
}
```

- **Definition**: A function annotated with `@test` is recognized as a standard unit test.
- **Execution**: All such functions are automatically executed by the test runner.
- **Pass Criteria**: The function completes without throwing errors or exceptions.
- **Fail Criteria**: If the function throws an error, it is reported as a failure.

### Running Tests

Slug automatically detects and runs all test functions (`@test` and `@testWith`) in the given module. You can run tests
for one or more modules by specifying their paths when invoking the test runner:

```shell
slug test path_to_source.slug
````

- **Output**: The output displays the number of test cases run, along with detailed pass, fail, and error counts. Each
  test's result is also printed for quick debugging.

**Example Output:**

```
Results:

Tests run: 33, Failures: 0, Errors: 0

Total time 1ms
```

With `@test` and `@testWith`, Slug empowers you to write robust, maintainable tests that enhance code quality and
reliability.
