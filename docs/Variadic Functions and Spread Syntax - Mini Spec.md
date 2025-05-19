## Slug Mini-Spec: Variadic Functions and Spread Syntax

### 1. Function Definition with Variadic Parameters

A function may declare a variadic parameter using the `...` prefix:

```slug
fn(...args) {
    // args is a list containing all passed values
}
```

* The variadic parameter must be the **last** parameter in the function definition.
* The parameter will receive a **list** of all remaining arguments passed to the function.
* Only one variadic parameter is allowed per function.

#### Example

```slug
var sum = fn(...nums) {
    // nums is a list: [1, 2, 3]
    return reduce(nums, 0, fn(x, y) { x + y });
};
sum(1, 2, 3); // returns 6
```

---

### 2. Spread Operator in Function Calls

A function call may **spread** the contents of a list into individual arguments using the `...` spread syntax:

```slug
fnName(...expr)
```

* `expr` must evaluate to a list. Each element in the list will be treated as a separate argument.
* Spread syntax can be used **anywhere** in the argument list.
* Multiple spreads are allowed.
* Non-list values passed with `...` result in a runtime error.

#### Example

```slug
var args = [1, 2, 3];
print("Numbers:", ...args, 4); // equivalent to print("Numbers:", 1, 2, 3, 4)
```

---

### 3. Combining Normal and Spread Arguments

Both standard arguments and spread arguments can be mixed in a function call.

```slug
fnName(arg1, ...list1, arg2, ...list2)
```

The resulting arguments will be flattened in left-to-right order.

---

### 4. Semantics

#### Function Definition

* When a function is declared with a variadic parameter, all extra arguments beyond the required ones are **collected
  into a list** and bound to that parameter name.

#### Function Call

* When evaluating a function call:

    1. Evaluate all arguments in order.
    2. For arguments with spread (`...expr`), evaluate `expr` to a list.
    3. Inline each item in the list into the argument list at that position.
    4. If a spread expression does not yield a list, raise a runtime error.

---

### 5. Errors

- **Invalid in Definition**:

```slug
fn(a, ...args, b) { }  // Error: variadic parameter must be last
```

- **Invalid in Call**:
```slug
print(...42);  // Error: cannot spread non-list
````

---

### 6. Example

```slug
foreign print = fn(...args);

var println = fn(...args) {
    print(...(args :+ "\n"));
}

var items = ["apple", "banana"];
println("Fruits:", ...items);  // prints: Fruits: apple banana \n
```
