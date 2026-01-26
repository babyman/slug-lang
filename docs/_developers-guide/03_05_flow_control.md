# Module 5: Flow Control

Now you can build logic: conditions, loops via tail recursion, and error handling.

## Lesson 5.1: Conditionals

```slug
val max = fn(a, b) {
    if (a > b) {
        a
    } else {
        b
    }
}

max(3, 5) /> println()
```

## Lesson 5.2: Tail-recursive looping with `recur`

`recur` restarts the current function in tail position without growing the call stack.

```slug
// Sum 1..n using tail recursion in an anonymous function
fn(n, acc) {
    if (n == 0) {
        acc
    } else {
        recur(n - 1, acc + n)
    }
}(5, 0) /> println()
```

## Lesson 5.3: Error handling with `throw` and `defer onerror`

```slug
val process = fn(value) {
    defer onerror(err) { println("Caught error:", err.msg) }

    if (value < 0) {
        throw {msg: "Negative value not allowed"}
    }

    value * 2
}

process(-1) /> println()
```

## Lesson 5.4: `defer`, `defer onsuccess`, and `defer onerror`

Use `defer` to run cleanup or logging when a scope exits.

```slug
val writeFile = fn(path, text) {
    defer { println("closing file") }
    defer onsuccess { println("write ok") }
    defer onerror(err) { println("write failed:", err) }

    // ... write logic here ...
}
```

### Try it

Write a function that divides two numbers and throws an error when the divisor is zero.
