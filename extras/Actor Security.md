# The Kernel Actor Security Model

The current actor security model combines **capability-based access control** for service operations with a *
*parent/child authority model** for actor supervision, and it now includes a first-class, secure mechanism for **reply
channels** via passive actors.

At the core is the idea that sending a message is not “always allowed”: before a message is enqueued, the kernel checks
whether the sender is permitted to invoke that specific operation on the target. For *service actors*, this is driven by
`OpRights`: each service declares which message payload types it understands and what `Rights` (read/write/exec) are
required for each. When an actor sends a message whose payload type matches an entry in the target’s `OpRights`, the
sender must hold a non-revoked capability granting at least those rights to that target. This keeps service APIs
explicit and prevents arbitrary actors from invoking privileged operations unless a capability was granted.

Separately, the kernel enforces a **supervision boundary** using parent/child relationships. When an actor spawns a
child, the kernel automatically creates baseline capabilities and establishes that the parent has broad authority over
that child (and vice versa, depending on how you use it). In practice this means: if there is no defined `OpRights`
entry for a given payload type on a target actor, the kernel still allows communication *when the sender is the parent
of the target*. This “parent can fully manage its children” rule is what makes actor composition workable without
requiring every internal control message to be modeled as an explicit service operation.

Passive actors introduce a new kind of endpoint: they have an ID and a mailbox, but **execute no handler code**. They
are intended as reply mailboxes and protocol handles, so their security rules are stricter. First,
`receiveFrom(pid, timeout)` is a privileged read: it is only valid when `pid` refers to a **passive** actor, and only
the **parent** of that passive actor is allowed to read from it. Attempting to receive from a non-passive actor is a
hard error. Second, sending to a passive actor is not treated like normal service messaging: **only the passive actor’s
parent** may send to it *unless* the sender holds an explicit `RightWrite` capability targeting that passive mailbox.
This prevents passive mailboxes from becoming globally-writable “public inboxes” if an ID leaks.

To keep reply channels simple in multi-hop call chains (client → service → worker), the kernel now supports an *
*implicit reply-to delegation** mechanism. Each message may carry an explicit `ReplyTo` field. When an actor sends a
message with `ReplyTo` set to a passive mailbox that it owns (i.e., the passive mailbox’s parent is the sender), the
kernel automatically grants the recipient a `RightWrite` capability to that `ReplyTo` mailbox. Because forwarding
preserves the original `From` and `ReplyTo`, this delegation naturally propagates through intermediaries without
developers manually managing grants. The delegation is narrow (write-only, mailbox-specific) and the kernel avoids
capability accumulation by not issuing duplicate `RightWrite` caps when the recipient already has one.

Finally, lifecycle interacts with security in predictable ways: passive actors persist until explicitly terminated or
until their parent exits; termination removes the actor from the kernel, so subsequent sends fail with “no such actor”,
and any capabilities pointing at it are effectively useless (and may be revoked during cleanup). The net result is a
model where **service APIs are capability-checked**, **actor supervision stays ergonomic**, and **reply channels are
both secure and easy to use** via passive mailboxes and explicit `ReplyTo` delegation.
