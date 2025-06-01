Slug
===

A small, opinionated programming language.

Slug Command
===

Setup
---

```shell
# slug home
export SLUG_HOME=[[path to slug home directory]]
export PATH="$SLUG_HOME/bin:$PATH"
```

Shell scripts
---
The following shell script works.

```shell
#!/usr/bin/env slug
println("Hello Slug!")
```

CLI
---

```shell
slug --root [path to module root] script[.slug] [args...]
```

Repl
---

Slug has a simple repl if launched without a script.

# Introduction to the Slug Programming Language: A Developer's Guide

Welcome to the **Slug Programming Language**! Slug is a versatile, functional-first language that blends simplicity,
expressiveness, and power to enable developers to write readable and maintainable code. This tutorial will introduce you
to the core concepts of Slug, its syntax, and its idiomatic use cases.

By the end of this guide, you will have a foundational understanding of Slug’s features, allowing you to build
structured, functional, and elegant programs.

---

## Table of Contents

1. **Getting Started with Slug**
    - Writing Your First Slug Program
2. **Core Building Blocks**
    - Keywords
    - Built-in Functions
    - Variables: `var` and `val`
    - Functions and Closures
3. **Functional Programming Constructs**
    - Map, Filter, and Reduce
    - Pattern Matching
    - Higher-Order Functions
4. **Data Structures**
    - Lists
    - Maps
5. **Flow Control**
    - Conditionals (`if`/`else`)
    - Error Handling with `try`/`catch` and `throw`
6. **Working Example**
    - Building a Functional Pipeline

---

## 1. Getting Started with Slug

### Writing Your First Slug Program

Create a file called `hello_world.slug` and add the following:

```
var {*} import("slug.std");

println("Hello, Slug!");
```

Run it with:

```shell script
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

#### **Constants**

- `nil`: Represents the absence of a value or "nothing."
- `true` / `false`: Boolean constants representing logical truth and falsehood.

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

```
val size = len([1, 2, 3]);       // 3
val textLength = len("hello");   // 5
```

#### **`print`** and **`println`**

- **Purpose**: Output formatted text to the console.
    - **`print`**: Outputs without a trailing newline.
    - **`println`**: Outputs followed by a newline.
- **Usage**: Both accept one or more arguments of any type, outputting their string representations separated by spaces.
- **Examples**:

``` slug
  print("Hello", "Slug!");       // Outputs: Hello Slug!
  println("Welcome to Slug!");   // Outputs: Welcome to Slug!\n
```

### Variables in Slug

Slug supports two types of variable declarations:

1. **Mutable Variables** (`var`): Can change over time.
2. **Immutable Variables** (`val`): Fixed once initialized.

Examples:

```
var {*} import("slug.std");

var counter = 0;  // Mutable variable
val greeting = "Hello"; // Immutable constant

counter = counter + 1;   // Reassigning is allowed with var
counter.println();       // Prints: 1
```

---

### Functions and Closures

Functions are first-class citizens in Slug. They can be passed as arguments, returned from other functions, or stored in
variables.

Example of defining and calling functions:

```
var {*} import("slug.std");

val add = fn(a, b) { a + b }  // A function that adds two numbers
add(3, 4).println();          // Output: 7
```

Functions can close over their surrounding environment, making them closures:

```
var {*} import("slug.std");

val multiplier = fn(factor) {
    fn(num) { num * factor }
};

val double = multiplier(2);
double(5).println();  // Output: 10
```

### Function Chaining in Slug

Slug supports function chaining, which allows for a cleaner and more expressive syntax. When a variable is placed before
a function call in the format `var.call()`, it is automatically passed as the first parameter to the function. The
result is equivalent to invoking the function as `call(var)`.

Example:

```
slug var {*} import("slug.std");

1.println(); // Using function chaining 1.println(); Outputs: 1
println(1);  // Equivalent traditional function call println(1);
```

By supporting function chaining, Slug simplifies code readability and enables a more fluid programming style, especially
when writing pipelines or working with multiple transformations.

---

## 3. Functional Programming Constructs

Slug excels at **functional programming**, empowering you with concise tools and expressive patterns.

### Map, Filter, and Reduce

- **Map**: Transform a list using a function.
- **Filter**: Keep elements that satisfy a condition.
- **Reduce**: Aggregate a list into a single value.

```
var {*} import("slug.std");

val list = [1, 2, 3, 4, 5];

val squares = list.map(fn(v) { v * v });           // [1, 4, 9, 16, 25]
val evens = list.filter(fn(v) { v % 2 == 0 });     // [2, 4]
val sum = list.reduce(0, fn(acc, v) { acc + v });  // 15

squares.println();
evens.println();
sum.println();
```

---

### Pattern Matching

Use `match` to destructure and inspect values directly.

```
var {*} import("slug.std");

val classify = fn(value) {
    match value {
        0 => "zero";
        1 => "one";
        _ => "other";  // Catch-all case
    }
};

classify(1).println();  // Output: one
classify(5).println();  // Output: other
```

`match` can also destructure complex data like lists:

```
var {*} import("slug.std");

val sumList = fn(list) {
    match list {
        [h, ...t] => h + sumList(t);  // Head and Tail destructuring
        [] => 0;                      // Base case
    }
};

sumList([1, 2, 3]).println();  // Output: 6
```

---

### Higher-Order Functions

Slug supports higher-order functions: functions that accept and return functions.

Example:

```
var {*} import("slug.std");

val applyTwice = fn(f, v) { f(f(v)) };

val increment = fn(x) { x + 1 };
applyTwice(increment, 10).println();  // Output: 12
```

---

## 4. Data Structures

### Lists

A list is a collection of elements. It supports operations like indexing, appending, and slicing.

```
var {*} import("slug.std");

val list = [10, 20, 30];
list[1].println();    // Output: 20
list[-1].println();   // Output: 30
list[:1].println();   // Output: [10]
list[1:].println();   // Output: [20, 30]
```

### Maps

Maps are key-value stores in Slug.

```
var {*} import("slug.std");

var myMap = {};
myMap = put(myMap, "name", "Slug");
get(myMap, "name").println();  // Output: Slug
```

---

## 5. Flow Control

### Conditionals: `if`/`else`

```
var {*} import("slug.std");

val max = fn(a, b) {
    if (a > b) {
        a
    } else {
        b
    }
};

max(3, 5).println();  // Output: 5
```

### Error Handling with `try`/`catch` and `throw`

```
var {*} import("slug.std");

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

process(-1).println();  // Output: Caught error: Negative value not allowed
```

---

## 6. Working Example: Functional Data Pipeline

We’ll build a pipeline that processes a list of numbers by:

- Squaring each number.
- Filtering for even numbers.
- Finding the sum of the remaining elements.

```
var {*} import("slug.std");

val numbers = [1, 2, 3, 4, 5, 6];

val result = numbers
    .map(fn(x) { x * x })          // [1, 4, 9, 16, 25, 36]
    .filter(fn(x) { x % 2 == 0 })  // [4, 16, 36]
    .reduce(0, fn(acc, x) { acc + x });  // 56

println("Result:", result);  // Output: Result: 56
```

Comments
===

`//` is supported since the language follows `C` language style conventions.

`#` is supported to allow easy execution as a shell script with the inclusion of `#!`. For example, if `SLUG_HOME` is
exported and `slug` is on the user path.


Types
===

- `Nil`
- `Boolean`: true or false
- `Integer`
- `String`
- `List`: []
- `Map`: {}
- `Function`: fn(){}

Operator Precedence and Associativity
===

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

Imports
===

```slug
// import all exports from slug.system
var {*} = import("slug.system");

// import only sqr and sum 
var {sqr, sum} = import("functions");

// import `sqr` as square and `sum` as foo
var {sqr: square, sum: foo} = import("functions");
```

Imports are loaded on demand, circular imports are supported. The search for an import will check for files by
substituting the `.` for file path separators, for example `slug.system` will become `/slug/system.slug`

- project root (default current directory)
- the $SLUG_HOME/lib directory
