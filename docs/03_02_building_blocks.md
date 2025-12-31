## 2. Core Building Blocks

### Keywords in Slug

Below is a concise reference list of the keywords in Slug, along with their descriptions. These keywords form the
foundational building blocks of the language.

#### **Types**

- `nil`: Represents the absence of a value or "nothing."
- `true` / `false`: Boolean constants representing logical truth and falsehood.
- `number`: a DEC64 inspired floating decimal point value, very experimental,
  see [DEC64: Decimal Floating Point]( https://www.crockford.com/dec64.html )
- `string`
- `list`: an ordered collection of values `[1, 2, 3]`
- `map`: a collection of key-value pairs `{k:v, k:v, ...}`
- `bytes`: a byte sequence `0x"ff00"`
- `function`: a function `fn(){}`
- `task`: a task handle, returned by `spawn`

#### **Comments**

- `//` is supported since the language follows C language style conventions.
- `#` is supported to allow easy execution as a shell script with the inclusion of `#!/usr/bin/env slug`. For example,
  if `SLUG_HOME` is
  exported and `slug` is on the user path.

#### **Variable Declarations**

- `var`: Declares a mutable variable, allowing its value to be reassigned.
- `val`: Declares an immutable constant, ensuring its value cannot be changed once initialized.

#### **Control Flow**

- `if` / `else`: Define conditional logic. The `if` block executes when the condition evaluates to true, while `else`
  provides the alternative block.
- `match`: A powerful pattern-matching construct for handling various cases based on the structure of values.
- `return`: Exits a function and optionally returns a value.
- `recur` restarts the current function in tail position with a new set of arguments, providing loop-like control flow
  without growing the call stack.

#### **Functionality**

- `fn`: Declares a function or closure, a reusable logic block that can be called with arguments.

#### **Error Handling**

- `throw`: Explicitly raises an error within the program.
- `defer`: Ensures a block of code runs after its enclosing scope exits, often used for cleanup tasks.
- `defer onsuccess`: Runs the block of code only if the enclosing scope exits without error.
- `defer onerror(err)`: Runs the block of code only if the enclosing scope throws with an error.

#### **Semicolons Optional**

Slug does **not require semicolons**.

Statements are normally terminated by **newlines**, not `;`. You can write clean, line-oriented code without worrying
about hidden rules or ambiguous formatting.

```slug
var a = 1
var b = 2
(a + b) /> println
```

Semicolons are still **allowed**, but they are never required.

### When a line continues

A newline does **not** end a statement when continuation is visually obvious, such as when a line starts with an
operator:

```slug
var sql =
    "select *"
    + " from users"
    + " where active = true"
```

or when using pipelines:

```slug
value
    /> transform
    /> validate
    /> save
```

### When a line always ends

To avoid ambiguity, some constructs must stay on the same line:

```slug
f(x)     // valid
f
(x)      // invalid
```

This rule prevents accidental or confusing parses and keeps Slug predictable. The result is code that’s concise,
readable, and easy to reason about—without relying on punctuation to make it work.

#### **Dangling Commas**

Slug supports dangling commas in lists, maps, tag, and function call parameter lists. This allows you to write code that
is easy to refactor and rearrange. Dangling commas are **NOT** allowed in function definitions (i.e `fn(a,b,)` is
invalid).

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
- `@bytes`: Matches objects of type `Bytes`.
- `@fun`: Matches objects of type `Function`.
- `@task`: Matches objects of type `Task`.

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
