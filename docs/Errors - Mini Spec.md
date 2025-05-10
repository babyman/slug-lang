Errors
===

## Throw

| Syntax                      | Expansion                                     |
|-----------------------------|-----------------------------------------------|
| `throw FileError`           | `throw { "type": "FileError" }`               |
| `throw FileError()`         | `throw { "type": "FileError" }`               |
| `throw FileError({ path })` | `throw { "type": "FileError", "path": path }` |

This ensures all errors are structured as `Hash` values with a required `"type"` key, enabling consistent, extensible,
and pattern-matchable error handling.

### Runtime Behavior

- Executing `throw` immediately halts the current function.
- The provided error `Hash` is wrapped internally by the runtime in a `RuntimeError` object:
  ```go
  type RuntimeError struct {
      Payload    Hash
      StackTrace []StackFrame
  }
  ```
- Each `StackFrame` includes the current function name, source file, line, and column number.
- As the error propagates up the call stack, the runtime appends frames to the `StackTrace`.
- The `Payload` (just the `Hash`) is what reaches `catch` blocks.
- If an error is uncaught, the full `RuntimeError` is printed, including the stack trace.
- Thrown values **must be Hashes**. Throwing a non-Hash value causes a runtime error.

---

## Try/Catch

```slug
try {
    connectToDatabase();
} catch err {
    { "type": "ConnectionError", "message": msg } => print("Connection failed: " + msg);
    { "type": "TimeoutError" } => print("Operation timed out!");
    _ => throw err // Re-throw unexpected errors;
}
```

### Runtime Behavior

- The `try` block executes normally unless a `throw` occurs.
- If a `throw` is triggered (directly or indirectly), control transfers to the nearest enclosing `catch`.
- The runtime extracts the `Payload` `Hash` from the internal `RuntimeError` and binds it to the identifier following
  `catch`.
- The `catch` block is a pattern match block:
    - Arms are evaluated in order.
    - On the first match, the corresponding block executes.
    - If no arm matches and `_ =>` is present, it is used.
    - If no match is found, the original `RuntimeError` is re-thrown with its stack trace preserved.

---

## Stack Trace Introspection

Users can access stack trace information from a caught error using the built-in `trace(err)` function:

```slug
try {
    runJob();
} catch err {
    _ =>
      println("Something went wrong!");
      println("Details: " + err);
      println("Stack trace:");
      for line in trace(err) {
          println("  at " + line.function + " (" + line.file + ":" + line.line + ")");
      }
}
```

The `trace(err)` function returns a list of hashes, each representing a stack frame:

```slug
[
  { "function": "readFile", "file": "fs.sl", "line": 12, "col": 5 },
  { "function": "loadConfig", "file": "app.sl", "line": 3, "col": 5 },
  ...
]
```

> Note: If `err` is not a tracked runtime error, `trace(err)` returns an empty list.

---

## Example: Propagation Across Multiple Stack Frames

```slug
var readFile = fn(path) {
    var content = readFromDisk(path); // might throw
    return content;
}

var loadConfig = fn() {
    var configText = readFile("config.json");
    return parseConfig(configText);
}

try {
    loadConfig();
} catch err {
    { "type": "FileError", "path": p } => {
        print("Failed to read file: " + p);
        for frame in trace(err) {
            print("  at " + frame.function + " (" + frame.file + ":" + frame.line + ")");
        }
    }
    _ => throw err;
}
```

---

## Example: Uncaught Error Output

If an error is not caught:

```slug
var main = fn() {
    openSocket(); // throws, uncaught
}
```

Runtime output:

```text
Uncaught error: { "type": "SocketUnavailable", "port": 8080 }
Stack trace:
  at openSocket (net.sl:42)
  at main (app.sl:2)
```
