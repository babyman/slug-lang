Errors
===

# **Final Slug Error Model Summary**

### **Throw**

```
throw X     // X is any value
```

### **Defer Blocks**

```
defer { ... }              // always
defer onsuccess { ... }    // only if no throw escapes
defer onerror(err) { ... } // only if a throw escapes; err = thrown value
```

### **Stacktrace Builtin**

```
stacktrace(err) → List<StackFrame>
```

### **Propagation Rules**

* Throws capture a stacktrace snapshot at the throw site
* Stacktrace persists until error is *handled*
* `defer onerror` *handles* the error if it returns normally
* If `onerror` throws again, new error replaces the old one
* `onsuccess` blocks never see the return value (simpler, clearer)
* Defers execute LIFO within their categories
* Defer blocks do not appear in the stacktrace

### **Return Semantics**

* Returning normally → onsuccess + always defers run
* Throwing → onerror + always defers run
* Onerror returning a value completes the function with that value
* Onerror rethrowing creates a new error context
* Tail call optimization works cleanly because errors are early returns

### **Stacktrace Contents**

```
StackFrame {
    func: string
    module: string
    line: int
    column: int
    sourceLine: string (optional)
}
```

Printed nicely:

```
THROW: "DIV by Zero!"
    at div (math:3)
    at f (main:12)
    at <root> (main:20)
```
