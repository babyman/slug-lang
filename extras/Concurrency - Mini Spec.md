# **Slug Concurrency: Mini-Spec**

## **Core Concepts**

### 1. **Process**

A **process** in Slug is:

- An isolated unit of execution with:
    - Its own **stack** and **environment**
    - A **mailbox** (message queue)
- Lightweight and scheduled by the Slug runtime (not OS threads)

Processes **do not share memory** and interact only via **message passing**.

---

### 2. **Process ID (PID)**

- An opaque handle representing a process.
- Can be sent in messages, stored in data structures, and compared.

---

### 3. **Spawn**

```slug
var pid = spawn fn (arg1, arg2) {
    ...
}(a, b);
```

- Starts a new process running the given function.
- The function must **not** implicitly capture outer scope—pass all needed state explicitly via arguments.
- Returns a `pid`.

---

### 4. **Self**

```slug
var me = self()
```

- Returns the current process's `pid`.
- Bound automatically in each process created by `spawn`.

---

### 5. **Send**

```slug
send(pid, message)
```

- Appends `message` to the inbox of the process identified by `pid`.
- Messages are:
    - First-in, first-out
    - Immutable values
    - Must be pattern-matchable

---

### 6. **Receive**

```slug
receive msg {
    {tag: "foo", data} => ...
    {tag: "bar"} => ...
}
```

- Blocks until a matching message is available in the current process’s mailbox.
- Matches are evaluated **in order**.
- Pattern-matching follows Slug's existing semantics.
- A fallback (e.g. `_ =>`) is recommended for exhaustiveness.

#### Optional timeout extension (not in min-spec, but natural):

```slug
receive msg timeout 1000 {
    {tag: "foo"} => ...
    _ => log("timed out")
}
```

---

## **Runtime Behavior**

- Slug maintains a **scheduler** that maps many lightweight processes to a small number of OS threads (or a single
  thread for simplicity).
- `spawn`ed processes are added to a **run queue**.
- `receive` suspends a process until a matching message is available.
- Messages are delivered **asynchronously**, in the order sent per sender.

---

## **Design Principles**

- **Explicit state**: no implicit environment capture in spawned functions.
- **No shared memory**: all state is passed via messages or arguments.
- **Composable concurrency**: processes are simple and isolated.
- **Pure semantics**: all concurrency constructs are functions or expressions.
- **No implicit reply**: use `self()` and explicit `from` values in message formats.

---

## **Example: Ping-Pong**

```slug
var responder = fn() {
    receive msg {
        {tag: "ping", from} => {
            from.send({tag: "pong"});
            responder();
        }
    }
}

var res = spawn responder();

res.send({tag: "ping", from: self()});

receive msg {
    {tag: "pong"} => log("Got pong");
}
```

---

## **Future Extensions (Outside the Min-Spec)**

- **`after` or `timeout` in `receive`**
- **Process groups or registries**
- **Supervision trees**
- **Message priorities or filters**
- **Remote messaging (distributed)**
