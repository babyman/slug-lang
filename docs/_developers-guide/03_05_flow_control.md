

## 5. Flow Control

### Conditionals: `if`/`else`

```slug
var {*} = import("slug.std")

val max = fn(a, b) {
    if (a > b) {
        a
    } else {
        b
    }
}

max(3, 5) /> println()  // Output: 5
```

### Tail-recursive looping: `recur`

`recur` is a special form that restarts the current function in tail position with a new set of arguments, giving you
loop-like behavior without growing the call stack; it works in both named and anonymous functions, and the compiler
will report an error if you use `recur` in a position that is not actually tail-recursive.

```slug
// Sum 1..n using tail recursion in an anonymous function
fn(n, acc) {
	if (n == 0) { 
		acc
	} else {
		recur(n - 1, acc + n) 
	} 
}(5, 0) /> println() // Output: 15
```

### Error Handling with `try`/`catch` and `throw`

```slug
var {*} = import("slug.std")

val process = fn(value) {
    try {
        if (value < 0) {
            throw NegativeValueError({msg:"Negative value not allowed"})
        }
        value * 2
    } catch (err) {
        {...} => println("Caught error:", err.msg)
        _ => nil
    }
}

process(-1) /> println()  // Output: Caught error: Negative value not allowed
```
