# Introduction to the Slug Programming Language: A Developer's Guide

Welcome to the **Slug Programming Language**! Slug is a versatile, functional-first language that blends simplicity,
expressiveness, and power to enable developers to write readable and maintainable code. This tutorial will introduce you
to the core concepts of Slug, its syntax, and its idiomatic use cases.

By the end of this guide, you will have a foundational understanding of Slug’s features, allowing you to build
structured, functional, and elegant programs.

---

## 1. Getting Started with Slug

### Writing Your First Slug Program

Create a file called `hello_world.slug` and add the following:

```slug
var {*} = import("slug.std");

println("Hello, Slug!");
```

Run it with:

```shell
slug hello_world.slug
```

You should see:

```
Hello, Slug!
```

---

## 2. Core Building Blocks

### Keywords in Slug

Below is a concise reference list of the keywords in Slug, along with their descriptions. These keywords form the
foundational building blocks of the language.

#### **Types**

- `nil`: Represents the absence of a value or "nothing."
- `true` / `false`: Boolean constants representing logical truth and falsehood.
- `number`: DEC64 inspired, very experimental, see [DEC64: Decimal Floating Point]( https://www.crockford.com/dec64.html )
- `string`
- `list`: []
- `map`: {}
- `function`: fn(){}

#### **Comments**

- `//` is supported since the language follows C language style conventions.
- `#` is supported to allow easy execution as a shell script with the inclusion of `#!/usr/bin/env slug`. For example, if `SLUG_HOME` is
  exported and `slug` is on the user path.

#### **Variable Declarations**

- `var`: Declares a mutable variable, allowing its value to be reassigned.
- `val`: Declares an immutable constant, ensuring its value cannot be changed once initialized.

#### **Control Flow**

- `if` / `else`: Define conditional logic. The `if` block executes when the condition evaluates to true, while `else`
  provides the alternative block.
- `match`: A powerful pattern-matching construct for handling various cases based on the structure of values.
- `return`: Exits a function and optionally returns a value.

#### **Functionality**

- `fn`: Declares a function or closure, a reusable logic block that can be called with arguments.

#### **Error Handling**

- `try` / `catch`: Used for exception handling. Code within `try` is monitored for errors, and `catch` provides the
  response logic.
- `throw`: Explicitly raises an error within the program.
- `defer`: Ensures a block of code runs after its enclosing scope exits, often used for cleanup tasks.

#### **Dangling Commas**

Slug supports dangling commas in lists and maps. This allows you to write code that is easy to refactor and rearrange.

```slug
var {*} = import(
	"slug.std",
);

val map = {
	k: 50,
};

var list = [
	1,
	[1,2,],
	11,
];

println(map, list,);
```

---

### Built-in Functions

Slug provides a small set of built-in functions designed to serve core operations and promote simplicity in program
design. These built-ins are globally available and do not require explicit imports. Here's an overview of the built-ins:

#### **`import`**

- **Purpose**: Dynamically loads external modules and provides access to their exported variables and functions.
- **Usage**: Accepts one or more module paths as string arguments. Returns a map containing the bindings from the
  imported modules.
- **Example**:

```slug
val {*} = import("slug.std");
```

#### **`len`**

- **Purpose**: Returns the length of a supported object.
- **Usage**: Accepts a single argument, which can be a list, map, or string. Returns the length as an integer.
- **Example**:

```slug
val size = len([1, 2, 3]);       // 3
val textLength = len("hello");   // 5
```

#### **`print`** and **`println`**

- **Purpose**: Output formatted text to the console.
    - **`print`**: Outputs without a trailing newline.
    - **`println`**: Outputs followed by a newline.
- **Usage**: Both accept one or more arguments of any type, outputting their string representations separated by spaces.
- **Examples**:

```slug
  print("Hello", "Slug!");       // Outputs: Hello Slug!
  println("Welcome to Slug!");   // Outputs: Welcome to Slug!\n
```

---

### Variables in Slug

Slug supports two types of variable declarations:

1. **Mutable Variables** (`var`): Can change over time.
2. **Immutable Variables** (`val`): Fixed once initialized.

Examples:

```slug
var {*} = import("slug.std");

var counter = 0;  // Mutable variable
val greeting = "Hello"; // Immutable constant

counter = counter + 1;      // Reassigning is allowed with var
counter /> println();       // Prints: 1
```

---

### Functions and Closures

Functions are first-class citizens in Slug. They can be passed as arguments, returned from other functions, or stored in
variables.

Example of defining and calling functions:

```slug
var {*} = import("slug.std");

val add = fn(a, b) { a + b }  // A function that adds two numbers
add(3, 4) /> println();       // Output: 7
```

Functions can close over their surrounding environment, making them closures:

```slug
var {*} = import("slug.std");

val multiplier = fn(factor) {
    fn(num) { num * factor }
};

val double = multiplier(2);
double(5) /> println();  // Output: 10
```

### Function Chaining, The Trail Operator (`/>`)

Slug values simplicity and flow - and the **trail operator** (`/>`) captures both.

Each value *slides forward* through the trail, passed into the next function or expression. It’s clean, readable, and
feels like watching data follow its own path.

```slug
var double = fn(n) { n * 2 }
var map = { double: double }
var lst = [nil, double]

10 /> map.double /> lst[1] /> println("is 40")

// is the equivalent of:
println(lst[1](map.double(10)), "is 40")
```

…but the trail reads the way you think: left to right, step by step. A small bit of syntax that makes code feel alive -
like a slug gliding gracefully across a path of transformations.

#### Function Dispatch

- In Slug, you can define multiple variations (overloads) of a function using the same name but with different
  signatures.
- During a function call:
    1. Each candidate function within the group is inspected.
    2. The number of arguments and their types are matched against the function's signature.
    3. The **best match** is determined based on:
        - Exact type matches.
        - Compatibility with **type tags** (described below).
        - Preference towards non-variadic functions if possible.

If no suitable match is found, an error is returned mentioning that no valid function exists for the given arguments.

#### Using Type Hints

Type hints, represented by **tags** in Slug, help guide the dispatch process by associating function parameters or
objects with specific types.

**Supported Type Tags:**

- `@num`: Matches objects of type `Number`.
- `@str`: Matches objects of type `String`.
- `@bool`: Matches objects of type `Boolean`.
- `@list`: Matches objects of type `List`.
- `@map`: Matches objects of type `Map`.
- `@fun`: Matches objects of type `Function`.

**Example: Function with Type Hints**

Suppose we define two functions where each variation operates on different types:

```slug
fn add(@num a, @num b) { a + b }
fn add(@str a, @str b) { a + b }
```

- Calling `add(3, 5)` would match the first function (`@num`), returning `8`.
- Calling `add("Hello, ", "world!")` would match the second function (`@str`), returning `"Hello, world!"`.

#### Tips for Writing Functions with Hints

1. Use **type tags** (`@num`, `@str`, etc.) to clarify expected parameter types.
2. Define fallback or general-purpose functions to handle unexpected cases.

This function concatenates any number of strings passed as arguments.

---

## 3. Functional Programming Constructs

Slug excels at **functional programming**, empowering you with concise tools and expressive patterns.

### Map, Filter, and Reduce

- **Map**: Transform a list using a function.
- **Filter**: Keep elements that satisfy a condition.
- **Reduce**: Aggregate a list into a single value.

```slug
var {*} = import("slug.std");

val list = [1, 2, 3, 4, 5];

val squares = list /> map(fn(v) { v * v });           // [1, 4, 9, 16, 25]
val evens = list /> filter(fn(v) { v % 2 == 0 });     // [2, 4]
val sum = list /> reduce(0, fn(acc, v) { acc + v });  // 15

squares /> println();
evens /> println();
sum /> println();
```

---

### Pattern Matching

Use `match` to destructure and inspect values directly.

```slug
var {*} = import("slug.std");

val classify = fn(value) {
    match value {
        0 => "zero";
        1 => "one";
        _ => "other";  // Catch-all case
    }
};

classify(1) /> println();  // Output: one
classify(5) /> println();  // Output: other
```

`match` can also destructure complex data like lists:

```slug
var {*} = import("slug.std");

val sumList = fn(list) {
    match list {
        [h, ...t] => h + sumList(t);  // Head and Tail destructuring
        [] => 0;                      // Base case
    }
};

sumList([1, 2, 3]) /> println();  // Output: 6
```

---

### Higher-Order Functions

Slug supports higher-order functions: functions that accept and return functions.

Example:

```slug
var {*} = import("slug.std");

val applyTwice = fn(f, v) { f(f(v)) };

val increment = fn(x) { x + 1 };
applyTwice(increment, 10) /> println();  // Output: 12
```

---

## 4. Data Structures

### Lists

A list is a collection of elements. It supports operations like indexing, appending, and slicing.

```slug
var {*} = import("slug.std");

val list = [10, 20, 30];
list[1] /> println();    // Output: 20
list[-1] /> println();   // Output: 30
list[:1] /> println();   // Output: [10]
list[1:] /> println();   // Output: [20, 30]
```

### Maps

Maps are key-value stores in Slug.

```slug
var {*} = import("slug.std");

var myMap = {};
myMap = put(myMap, "name", "Slug");
get(myMap, "name") /> println();  // Output: Slug
```

---

## 5. Flow Control

### Conditionals: `if`/`else`

```slug
var {*} = import("slug.std");

val max = fn(a, b) {
    if (a > b) {
        a
    } else {
        b
    }
};

max(3, 5) /> println();  // Output: 5
```

### Error Handling with `try`/`catch` and `throw`

```slug
var {*} = import("slug.std");

val process = fn(value) {
    try {
        if (value < 0) {
            throw NegativeValueError({msg:"Negative value not allowed"});
        }
        value * 2
    } catch (err) {
        {...} => println("Caught error:", err.msg);
        _ => nil;
    }
};

process(-1) /> println();  // Output: Caught error: Negative value not allowed
```

---

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

---

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

## 8. Reference

### Operator Precedence and Associativity

| Prec | Operator  | Description                      | Associates |
|------|-----------|----------------------------------|------------|
| 1    | () [] .   | Grouping, Subscript, Method call | Left       |
| 2    | - ! ~     | Negate, Not, Complement          | Right      |
| 3    | * / %     | Multiply, Divide, Modulo         | Left       |
| 4    | + -       | Add, Subtract                    | Left       |
| 6    | << >>     | Left shift, Right shift          | Left       |
| 7    | &         | Bitwise and                      | Left       |
| 8    | ^         | Bitwise xor                      | Left       |
| 9    | \|        | Bitwise or                       | Left       |
| 10   | < <= > >= | Comparison                       | Left       |
| 12   | == !=     | Equals, Not equal                | Left       |
| 13   | &&        | Logical and                      | Left       |
| 14   | \|\|      | Logical or                       | Left       |
| 15   | ?:        | Conditional*                     | Right      |
| 16   | =         | Assignment                       | Right      |
