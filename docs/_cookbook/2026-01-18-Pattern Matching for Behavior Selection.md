---
title: Pattern Matching for Behavior Selection
tags: [ match ]
---

### **Problem**

Execute logic based on value shape or type.

### **Idiom: Match Over Conditionals**

```slug
match response {
  { status: 200 } => "OK"
  { status: 404 } => "Not Found"
  _               => "Error"
}
```

**Slug mindset**

> *Describe the world; let the runtime choose.*
