---
title: Inline Unit Testing with `@testWith`
tags: [ test, @testWith ]
---

### **Problem**

You want to verify the correctness of a function immediately after defining it, without setting up a separate test file
or complex test suite.

### **Idiom: Table-Driven Testing with `@testWith`**

The `@testWith` decorator allows you to define a list of inputs and expected outputs directly above your function. Use
the slug test runner to execute the test cases `slug test slug.std` (or `slug -root . test hello`).

```slug
@testWith(
	[1, 2], false,
	[1, 1], true,
	["1", "2"], false,
	["1", "1"], true,
	[true, true], true,
	[false, true], false,
)
var equals = fn(v1, v2) {
	v1 == v2
}
```
