# Slug Language Mini-Spec: Map Syntax and Behavior (Expanded)

## 1. Overview

In Slug, **maps** are flexible, first-class dynamic key/value containers that support:

- **Mixed key types** (strings, numbers, etc.)
- **Concise, safe literals**
- **Natural access** through dot (`.`) and bracket (`[]`) syntax
- **Fluent method chaining** for building and modifying maps
- **Lightweight "method" behavior** through callable keys

Maps are the foundational structure for object-like and dynamic behaviors in Slug.

---

## 2. Map Definition

### 2.1 Static Key Map Literals

```slug
var user = { name: "Sluggy", age: 5 }
```

Equivalent to:

```slug
var user = { "name": "Sluggy", "age": 5 }
```

### 2.2 Dynamic Key Map Literals

```slug
var key = "speed"
var stats = { [key]: 88 }
```

Equivalent to:

```slug
var stats = put({}, key, 88)
```

---

## 3. Map Access

### 3.1 Dot Notation (`.`)

```slug
user.name   // get(user, "name")
numbers.42  // get(numbers, 42)
```

### 3.2 Bracket Notation (`[]`)

```slug
var field = "name"
user[field]   // get(user, field)
```

---

## 4. Function Call Syntax (Method-like Calls) **NOT IMPLEMENTED**

### 4.1 Calling a Map Key as a Function

```slug
user.greet()
// Desugars to:
get(user, "greet")(user)
```

If the retrieved key is not a function, the runtime throws an error:

```
Cannot call non-function value from map key 'greet'
```

---

## 5. Built-in Map Operations

Maps come with **core built-in functions**: `keys`, `put`, `get`, and `remove`.

Each **returns a new updated map**, allowing fluent chaining.

### 5.0 `keys(map) -> list`

- Returns a list containing all keys in the map.
- Keys in the returned list have no guaranteed order.
- If the map is empty, returns an empty list.

  Example:

```slug
var m = {name: "Sluggy"};
m.keys();  // ["name"]
```

### 5.1 `put(map, key, value) -> map`

- Inserts (or replaces) the entry for `key` with `value`.
- Returns the updated map.

Example:

```slug
var m = {}
m = put(m, "name", "Sluggy")
```

Chaining version:

```slug
var m = {}.put("name", "Sluggy").put("age", 5)
```

### 5.2 `get(map, key) -> value`

- Retrieves the value for the given key.
- If the key is not found, returns `empty` (or error based on context).

Example:

```slug
var name = get(m, "name")
```

Or using dot syntax:

```slug
var name = m.name
```

### 5.3 `remove(map, key) -> map`

- Removes the given key from the map if it exists.
- Returns the updated map.

Example:

```slug
var m2 = m.remove("age")
```

Chaining example:

```slug
var m3 = m.remove("age").put("city", "Toronto")
```

---

## 6. Fluent Chaining Examples

Chaining `put` and `remove` operations creates expressive, builder-style flows:

```slug
var config = {}
  .put("host", "localhost")
  .put("port", 8080)
  .put("debug", true)
  .remove("debug")
  .put("env", "production")

println(config.host)   // "localhost"
println(config.port)   // 8080
println(config.env)    // "production"
```

---

# Final Summary Table

| Operation             | Syntax Example         | Meaning / Desugared Form   |
|-----------------------|------------------------|----------------------------|
| Static key access     | `m.foo`                | `get(m, "foo")`            |
| Numeric key access    | `m.1`                  | `get(m, 1)`                |
| Dynamic key access    | `m[expr]`              | `get(m, expr)`             |
| Static function call  | `m.foo()`              | `get(m, "foo")(m)`         |
| Dynamic function call | `m[expr]()`            | `get(m, expr)(m)`          |
| Static map literal    | `{ key: value }`       | Put static keys into map   |
| Dynamic map literal   | `{ [expr]: value }`    | Put computed keys into map |
| Insert key-value      | `m.put(k, v)`          | `put(m, k, v)`             |
| Remove key            | `m.remove(k)`          | `remove(m, k)`             |
| Fluent map building   | `{}` + `.put()` chains | Builder-style map creation |

---

# Closing Notes

This **unified** map model gives Slug:

- Highly **expressive** map building
- A lightweight **object system** without special types
- Predictable, consistent syntax across literals, access, and methods
- A foundation to evolve more complex features like traits, prototypes, or objects later if desired

---
