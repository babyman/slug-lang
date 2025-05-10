## Slug Import System – Mini Spec

### 1. **Basic Import Forms**

#### 1.1 Full Module Import

```slug
import math.*;
```

- Makes `math` available as a namespace in the module scope.
- Functions and constants must be accessed with `math.name`.

---

#### 1.2 Selective Import

```slug
import math.{sqrt, sin};
```

- Injects `sqrt` and `sin` directly into the module scope.
- These can be used as top-level functions:
  ```slug
  sqrt(9)  // valid
  ```
- Enables method chaining via call-sugar (see Section 3):
  ```slug
  9.sqrt()  // desugars to sqrt(9)
  ```

---

#### 1.3 Selective Import with Aliasing

```slug
import math.{sqrt as msqrt};
```

- Imports `sqrt` under the alias `msqrt`.
- Useful to avoid name collisions or clarify origin.

---

### 2. **Scoping and Name Resolution**

#### 2.1 Import Scope Model

- All imported symbols (selectively or via full module import) reside in a **read-only parent scope** known as the *
  *import scope**.
- The module scope shadows the import scope.
- Constants in the import scope **cannot be redefined** locally.

---

#### 2.2 Lookup Order

When resolving an identifier:

1. **Module-local scope** (variables, constants, functions)
2. **Import scope** (selective imports)
3. **Namespace access** (via `a.ns::f()` desugaring; see below)

---

### 3. **Call-Sugar / Method Syntax**

Slug supports syntactic sugar for function calls using dot syntax:

```slug
a.f(b, c);
```

**Desugars to:**

```slug
f(a, b, c);
```

Applies only if:

- `f` is a function visible in the module or import scope.
- The call target `a` is passed as the first argument.

---

### 4. **Namespace Access and Fallback**

If a function isn’t available in scope (neither local nor imported), use **explicit namespace fallback**:

```slug
x.math::sqrt()
```

**Desugars to:**

```slug
math.sqrt(x)
```

#### Rules:

- `x.ns::f()` is valid if:
    - `ns` is a namespace (imported module).
    - `f` is a callable member of `ns`.
- Only works in the context of a function call.
- Supports chaining:
  ```slug
  x.math::sqrt().round()
  ```

---

### 5. **Shadowing Behavior**

- Selectively imported names **can be shadowed** by module-local definitions.
- `import math.{sqrt}` followed by:
  ```slug
  val sqrt = something_else
  ```
  is valid and masks the imported `sqrt`.

---

### 6. **Errors and Safety**

- Redefining an imported **constant** in the import scope is an error:
  ```slug
  val pi = 3.14;  // error if `pi` was imported as a constant
  ```
- Reimporting the same module multiple times is **idempotent**; re-imports with different selective sets are merged.

---

## Summary

Slug’s import system emphasizes:

- **Simplicity for common cases** (selective import + chaining).
- **Explicit clarity** when needed (`x.ns::f()`).
- **Safety** through scope separation.
- **Minimal syntax with high expressiveness**.
t shadowing errors?
