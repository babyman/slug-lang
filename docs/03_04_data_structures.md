## 4. Data Structures

### Lists

A list is a collection of elements. It supports operations like indexing, appending, and slicing.

```slug
var {*} = import("slug.std");

val list = [10, 20, 30];
list[1] /> println();    // Output: 20
list[-1] /> println();   // Output: 30
list[:1] /> println();   // Output: [10]
list[1:] /> println();   // Output: [20, 30]
```

### Maps

Maps are key-value stores in Slug.

```slug
var {*} = import("slug.std");

var myMap = {};
myMap = put(myMap, "name", "Slug");
get(myMap, "name") /> println();  // Output: Slug
```
