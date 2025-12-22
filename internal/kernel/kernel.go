package kernel

// Slug Microkernel v1.1 — Actors-in-Kernel with HTTP REPL Service
// ---------------------------------------------------------------
// What’s new vs v1.0:
//  - REPL is now a first-class *service* (actor) exposed over HTTP.
//  - Evaluator service (stub) executes source sent by REPL; easy drop-in for your real tree-walker.
//  - Clean capability checks preserved: REPL talks to EVAL; EVAL talks to FS/TIME via granted Caps.
//
// Build & run
//   go run main.go
//
// Try HTTP REPL
//   curl -s -X POST localhost:8080/repl/eval \
//     -H 'content-type: application/json' \
//     -d '{"source":"print \"hi\""}' | jq
//
// Other control plane endpoints still work:
//   curl -s localhost:8080/Actors | jq
//   curl -s -X POST localhost:8080/send \
//     -H 'content-type: application/json' \
//     -d '{"from":"demo","to":"fs","op":"read","payload":{"path":"/tmp/hello.txt"}}' | jq
//
// Notes
//  - Memory remains Go-managed (v1). Budgets & quotas are soft counters.
//  - REPL/EVAL boundaries make it trivial to swap in your real Slug interpreter.
//  - No external deps; HTTP only (WebSocket/SSE can be added later).

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ActorMailboxSize   = 64
	defaultSendTimeout = 5 * time.Second
	fullMailboxTimeout = 2 * time.Second
)

// ===== Kernel =====

type Kernel struct {
	Mu                 sync.RWMutex
	NextActorID        int64
	NextCapID          int64
	Actors             map[ActorID]*Actor
	NameIdx            map[string]ActorID
	OpsBySvc           map[ActorID]OpRights
	PrivilegedServices map[string]PrivilegedService
	CapIndex           map[ActorID]map[ActorID]map[int64]bool
}

func NewKernel() *Kernel {

	kernel := &Kernel{
		Actors:             make(map[ActorID]*Actor),
		NameIdx:            make(map[string]ActorID),
		OpsBySvc:           make(map[ActorID]OpRights),
		PrivilegedServices: make(map[string]PrivilegedService),
		CapIndex:           make(map[ActorID]map[ActorID]map[int64]bool),
	}

	kernel.RegisterService(KernelService, Operations, kernel.handler)

	return kernel
}

func (k *Kernel) MailboxLen(caller ActorID, target ActorID) (int, error) {
	k.Mu.RLock()
	defer k.Mu.RUnlock()

	if caller != target && !(k.hasCapWithMuLock(caller, target, RightWrite) || k.hasCapWithMuLock(caller, target, RightRead)) {
		return 0, fmt.Errorf("E_POLICY: no read permission for actor %d", target)
	}

	a, ok := k.Actors[target]
	if !ok {
		return 0, fmt.Errorf("E_NO_SUCH: actor %d", target)
	}
	return len(a.inbox), nil
}

func (k *Kernel) MailboxCapacity(caller ActorID, target ActorID) (int, error) {
	k.Mu.RLock()
	defer k.Mu.RUnlock()

	if caller != target && !(k.hasCapWithMuLock(caller, target, RightWrite) || k.hasCapWithMuLock(caller, target, RightRead)) {
		return 0, fmt.Errorf("E_POLICY: no read permission for actor %d", target)
	}

	a, ok := k.Actors[target]
	if !ok {
		return 0, fmt.Errorf("E_NO_SUCH: actor %d", target)
	}
	return cap(a.inbox), nil
}

// RegisterActor wires an actor into the kernel.
func (k *Kernel) RegisterActor(name string, handler Handler) ActorID {
	k.Mu.Lock()
	id := ActorID(k.NextActorID)
	k.NextActorID++
	actCtx, cancel := context.WithCancel(context.Background())

	slog.Info("Registering actor",
		slog.Any("id", id),
		slog.String("name", name))

	act := &Actor{
		Id:       id,
		Name:     name,
		inbox:    make(chan Message, ActorMailboxSize),
		handler:  handler,
		ctx:      actCtx,
		cancel:   cancel,
		children: make(map[ActorID]bool),
		Caps:     make(map[int64]*Capability),
	}
	k.Actors[id] = act
	if name != "" {
		k.NameIdx[name] = id
	}
	k.Mu.Unlock()

	go k.runActor(act)
	return id
}

func (k *Kernel) Register(name string, pid ActorID) {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	if _, ok := k.Actors[pid]; ok {
		k.NameIdx[name] = pid
	}
}

func (k *Kernel) Unregister(name string) ActorID {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	id := k.NameIdx[name]
	delete(k.NameIdx, name)
	return id
}

func (k *Kernel) Registered() []string {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	var result []string
	for k, _ := range k.NameIdx {
		result = append(result, k)
	}
	return result
}

func (k *Kernel) Lookup(name string) ActorID {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	return k.NameIdx[name]
}

func (k *Kernel) SpawnChild(parent ActorID, name string, ops OpRights, handler Handler) (ActorID, error) {

	k.Mu.Lock()

	id := ActorID(k.NextActorID)
	k.NextActorID++
	actCtx, cancel := context.WithCancel(context.Background())

	child := &Actor{
		Id:       id,
		Name:     name,
		Parent:   parent,
		inbox:    make(chan Message, ActorMailboxSize),
		handler:  handler,
		ctx:      actCtx,
		cancel:   cancel,
		children: make(map[ActorID]bool),
		Caps:     make(map[int64]*Capability),
	}

	k.Actors[child.Id] = child
	k.OpsBySvc[id] = ops

	// register as child of parent
	if parent, ok := k.Actors[parent]; ok {
		parent.children[child.Id] = true

		// Copy capabilities from parent to child
		for _, c := range parent.Caps {
			k.createCapWithMuLock(id, c.Target, c.Rights, c.Scope)
		}

		// Create capabilities for parent to child
		k.createCapWithMuLock(parent.Id, child.Id, RightRead|RightWrite|RightExec, nil)

		// Create capabilities for child to parent
		k.createCapWithMuLock(child.Id, parent.Id, RightRead|RightWrite|RightExec, nil)

		// Create capabilities for child to itself
		k.createCapWithMuLock(child.Id, child.Id, RightRead|RightWrite|RightExec, nil)
	}
	k.Mu.Unlock()

	go k.runActor(child)

	slog.Info("Spawned child actor",
		slog.Any("parent-id", parent),
		slog.Any("child-id", id),
		slog.Any("name", name))
	return id, nil
}

func (k *Kernel) SpawnPassiveChild(parent ActorID, name string) (ActorID, error) {
	k.Mu.Lock()

	id := ActorID(k.NextActorID)
	k.NextActorID++
	actCtx, cancel := context.WithCancel(context.Background())

	child := &Actor{
		Id:       id,
		Name:     name,
		Parent:   parent,
		Passive:  true,
		inbox:    make(chan Message, ActorMailboxSize),
		handler:  nil,
		ctx:      actCtx,
		cancel:   cancel,
		children: make(map[ActorID]bool),
		Caps:     make(map[int64]*Capability),
	}

	k.Actors[child.Id] = child

	// register as child of parent
	if parentAct, ok := k.Actors[parent]; ok {
		parentAct.children[child.Id] = true

		// Copy capabilities from parent to child
		for _, c := range parentAct.Caps {
			k.createCapWithMuLock(id, c.Target, c.Rights, c.Scope)
		}

		// Create capabilities for parent to child
		k.createCapWithMuLock(parentAct.Id, child.Id, RightRead|RightWrite|RightExec, nil)

		// Create capabilities for child to parent
		k.createCapWithMuLock(child.Id, parentAct.Id, RightRead|RightWrite|RightExec, nil)

		// Create capabilities for child to itself
		k.createCapWithMuLock(child.Id, child.Id, RightRead|RightWrite|RightExec, nil)
	}

	k.Mu.Unlock()

	slog.Info("Spawned passive child actor",
		slog.Any("parent-id", parent),
		slog.Any("child-id", id),
		slog.Any("name", name))
	return id, nil
}

func (k *Kernel) ReceiveFromPassive(parent ActorID, passive ActorID, timeout time.Duration) (any, bool, error) {
	k.Mu.RLock()
	a := k.Actors[passive]
	k.Mu.RUnlock()

	if a == nil {
		return nil, false, fmt.Errorf("E_NO_SUCH: actor %d", passive)
	}
	if !a.Passive {
		return nil, false, fmt.Errorf("E_NOT_PASSIVE: actor %d", passive)
	}
	if a.Parent != parent {
		return nil, false, fmt.Errorf("E_POLICY: actor %d is not parent of %d", parent, passive)
	}

	// Timeout semantics:
	//  - timeout < 0 => block forever
	//  - timeout == 0 => poll (non-blocking)
	//  - timeout > 0 => wait up to timeout
	if timeout < 0 {
		msg, ok := <-a.inbox
		if !ok {
			return nil, false, fmt.Errorf("E_NO_SUCH: actor %d", passive)
		}
		return msg.Payload, true, nil
	}

	if timeout == 0 {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil, false, fmt.Errorf("E_NO_SUCH: actor %d", passive)
			}
			return msg.Payload, true, nil
		default:
			return nil, false, nil
		}
	}

	select {
	case msg, ok := <-a.inbox:
		if !ok {
			return nil, false, fmt.Errorf("E_NO_SUCH: actor %d", passive)
		}
		return msg.Payload, true, nil
	case <-time.After(timeout):
		return nil, false, nil
	}
}

func (k *Kernel) runActor(a *Actor) {
	ctx := &ActCtx{K: k, Self: a.Id, Context: a.ctx}
	for {
		select {
		case <-a.ctx.Done():
			// The actor was cancelled externally (interrupted)
			return
		case msg, ok := <-a.inbox:
			if !ok {
				return
			}

			if exit, ok := msg.Payload.(Exit); ok {
				k.cleanupActor(a, exit.Reason)
				slog.Info("actor exiting",
					slog.Any("actor-id", a.Id),
					slog.String("actor-name", a.Name),
					slog.String("reason", exit.Reason))
				return
			}

			start := time.Now()
			sig := a.handler(ctx, msg)
			atomic.AddUint64(&a.CpuOps, uint64(time.Since(start).Microseconds()))

			switch signal := sig.(type) {
			case nil, Continue:
				continue
			case Terminate:
				k.cleanupActor(a, signal.Reason)
				slog.Info("actor terminating",
					slog.Any("actor-id", a.Id),
					slog.String("actor-name", a.Name),
					slog.String("reason", signal.Reason))
				return
			case Restart:
				k.cleanupActor(a, signal.Reason)
				k.restartActor(a, signal)
				return
			case Error:
				k.handleActorError(a, signal)
				// could escalate, log, terminate, or ignore depending on policy
			}
		}
	}
}

// cleanupActor removes the actor from kernel tracking and closes resources
func (k *Kernel) cleanupActor(a *Actor, reason string) {
	// 1. If it's an active actor, try to tell it to shut down nicely first
	if !a.Passive {
		select {
		case a.inbox <- Message{Payload: Shutdown{Reason: reason}}:
			// We successfully queued an exit message.
			// In a more robust system, you might wait a few ms here
			// to see if the goroutine exits on its own.
		default:
			// Inbox is full/blocked; we must force it.
			if a.cancel != nil {
				a.cancel()
			}
		}
	} else {
		// Passive actors have no goroutine, just cancel the context
		if a.cancel != nil {
			a.cancel()
		}
	}

	k.Mu.Lock()
	defer k.Mu.Unlock()

	slog.Info("Cleaning up actor",
		slog.Any("actor-id", a.Id),
		slog.Any("actor-name", a.Name),
		slog.Any("cpu-time", a.CpuOps),
		slog.Any("ipc-in", a.IpcIn),
		slog.Any("ipc-out", a.IpcOut),
		slog.Any("caps", len(a.Caps)),
		slog.Any("reason", reason))

	// Kill children first
	for childID := range a.children {
		if child, ok := k.Actors[childID]; ok {
			k.cleanupActor(child, "parent terminated")
		}
	}

	// Remove self from parent's children
	if a.Parent != 0 {
		if parent, ok := k.Actors[a.Parent]; ok {
			delete(parent.children, a.Id)
		}
	}

	// Remove from actors map
	delete(k.Actors, a.Id)

	// Remove from name index if named
	if a.Name != "" {
		delete(k.NameIdx, a.Name)
	}

	// Remove from operations map if it's a service
	delete(k.OpsBySvc, a.Id)

	// 1. Revoke and remove capabilities held BY this actor
	for capID, c := range a.Caps {
		c.Revoked.Store(true)
		// Remove from reverse index
		if holders, ok := k.CapIndex[c.Target]; ok {
			if caps, ok := holders[a.Id]; ok {
				delete(caps, capID)
				if len(caps) == 0 {
					delete(holders, a.Id)
				}
			}
			if len(holders) == 0 {
				delete(k.CapIndex, c.Target)
			}
		}
		delete(a.Caps, capID)
	}

	// 2. Revoke and remove capabilities held by OTHERS targeting this actor
	if holders, ok := k.CapIndex[a.Id]; ok {
		for holderID, caps := range holders {
			if holder, ok := k.Actors[holderID]; ok {
				for capID := range caps {
					if c, ok := holder.Caps[capID]; ok {
						c.Revoked.Store(true)
						delete(holder.Caps, capID)
					}
				}
			}
		}
		delete(k.CapIndex, a.Id)
	}

	// Close the inbox channel to prevent further messages
	close(a.inbox)
}

// restartActor recreates an actor with the same configuration
func (k *Kernel) restartActor(a *Actor, restart Restart) {
	slog.Info("Restarting actor",
		slog.Any("id", a.Id),
		slog.String("name", a.Name))

	// todo this will require new capabilities or nothing can call it!

	// Store original configuration
	name := a.Name
	handler := a.handler

	// Register a new actor with the same name and handler
	newID := k.RegisterActor(name, handler)

	slog.Info("Actor restarted",
		slog.String("name", name),
		slog.Any("new-id", newID),
		slog.Any("old-id", a.Id))
}

// handleActorError processes actor errors based on policy
func (k *Kernel) handleActorError(a *Actor, err Error) {
	slog.Error("Actor reported error",
		slog.Any("id", a.Id),
		slog.String("name", a.Name),
		slog.Any("error", err.Err))

	// Error handling policy - can be configured based on requirements
	// For now, we'll log the error and continue
	// In a production system, this might:
	// - Escalate to supervisor
	// - Restart the actor
	// - Terminate the actor
	// - Increment error counters and take action on threshold
}

// RegisterPrivilegedService registers a service that needs kernel access
func (k *Kernel) RegisterPrivilegedService(name string, svc PrivilegedService) {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	k.PrivilegedServices[name] = svc
}

// Declare a service (actor) with op→Operations mapping, enabling cap checks.
func (k *Kernel) RegisterService(name string, ops OpRights, handler Handler) ActorID {
	id := k.RegisterActor(name, handler)
	k.Mu.Lock()
	defer k.Mu.Unlock()
	k.OpsBySvc[id] = ops
	return id
}

func (k *Kernel) GrantChildAccess(granter ActorID, grantee ActorID, target ActorID, rights Rights, scope map[reflect.Type]any) (*Capability, error) {
	k.Mu.Lock()
	defer k.Mu.Unlock()

	// todo...
	// granter can grant permission to its children
	// and access to it's children
	// but nothing that exceeds it's own permissions or those of the accessor

	//// Verify that granter is the parent of target
	//targetActor, ok := k.Actors[target]
	//if !ok {
	//	return nil, fmt.Errorf("target actor %d does not exist", target)
	//}
	//
	//if targetActor.Parent != granter {
	//	return nil, fmt.Errorf("granter actor %d is not the parent of target actor %d", granter, target)
	//}

	// Grant the capability
	return k.createCapWithMuLock(grantee, target, rights, scope), nil
}

// GrantCap issues a capability from kernel to a specific actor.
func (k *Kernel) GrantCap(from ActorID, target ActorID, rights Rights, scope map[reflect.Type]any) *Capability {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	return k.createCapWithMuLock(from, target, rights, scope)
}

func (k *Kernel) createCapWithMuLock(from ActorID, target ActorID, rights Rights, scope map[reflect.Type]any) *Capability {
	capID := k.NextCapID
	k.NextCapID++
	capability := &Capability{ID: capID, Target: target, Rights: rights, Scope: scope}
	if a, ok := k.Actors[from]; ok {
		a.Caps[capID] = capability

		// Update reverse index
		if k.CapIndex[target] == nil {
			k.CapIndex[target] = make(map[ActorID]map[int64]bool)
		}
		if k.CapIndex[target][from] == nil {
			k.CapIndex[target][from] = make(map[int64]bool)
		}
		k.CapIndex[target][from][capID] = true

		return capability
	}
	return nil
}

// resolveRights returns required Operations for an op against a target service.
func (k *Kernel) resolveRights(target ActorID, op reflect.Type) (Rights, bool) {
	k.Mu.RLock()
	ops, ok := k.OpsBySvc[target]
	k.Mu.RUnlock()
	if !ok {
		return 0, false
	}
	r, ok := ops[op]
	return r, ok
}

// hasCap checks if sender owns a non-revoked cap to target with required Operations.
func (k *Kernel) hasCap(sender ActorID, target ActorID, want Rights) bool {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	return k.hasCapWithMuLock(sender, target, want)
}

func (k *Kernel) hasCapWithMuLock(sender ActorID, target ActorID, want Rights) bool {
	a := k.Actors[sender]
	if a == nil {
		return false
	}
	for _, c := range a.Caps {
		if c.Target == target && !c.Revoked.Load() && (c.Rights&want) == want {
			return true
		}
	}
	return false
}

func (k *Kernel) isPermitted(from ActorID, to ActorID, payload any) error {
	if payload != nil {

		// Passive mailbox policy: only parent or explicit RightWrite cap may send to passive actors.
		k.Mu.RLock()
		targetActor := k.Actors[to]
		k.Mu.RUnlock()
		if targetActor != nil && targetActor.Passive {
			if targetActor.Parent == from || k.hasCap(from, to, RightWrite) {
				return nil
			}
			slog.Error("E_POLICY: no permission to send to passive mailbox",
				slog.Any("from", from),
				slog.Any("to", to))
			return fmt.Errorf("E_POLICY: no permission to send to passive mailbox %d from %d", to, from)
		}

		msgType := reflect.TypeOf(payload)
		if rights, ok := k.resolveRights(to, msgType); ok {
			if !k.hasCap(from, to, rights) {
				slog.Error("E_POLICY: cap not granted for Operation",
					slog.Any("rights", rights),
					slog.Any("from", from),
					slog.Any("to", to),
					slog.Any("payload-type", reflect.TypeOf(payload).String()))
				return fmt.Errorf("E_POLICY: cap not granted for Operations=%v for op %T to target %d", rights, payload, to)
			}
		} else {
			k.Mu.RLock()
			defer k.Mu.RUnlock()
			if child, ok := k.Actors[to]; ok {
				if child.Parent == from {
					// parent has full rwx on child
					return nil
				}
				slog.Error("E_POLICY: no defined Operation",
					slog.Any("from", from),
					slog.Any("to", to),
					slog.Any("payload-type", reflect.TypeOf(payload).String()))
				return fmt.Errorf("E_POLICY: no defined Operations for op %T from %d to target %d", payload, from, to)
			} else {
				slog.Error("E_POLICY: no actor for target",
					slog.Any("from", from),
					slog.Any("to", to),
					slog.Any("payload-type", reflect.TypeOf(payload).String()))
				return fmt.Errorf("E_POLICY: no actor found for op %T from %d to target %d", payload, from, to)
			}
		}
	} else {
		slog.Warn("E_POLICY: nil payload",
			slog.Any("from", from),
			slog.Any("to", to),
			slog.Any("payload-type", reflect.TypeOf(payload).String()))
		return fmt.Errorf("E_POLICY: nil payload to target %d", to)
	}
	return nil
}

// sendInternal enqueues a message, enforcing capability checks for service ops.
func (k *Kernel) SendInternal(from ActorID, to ActorID, replyTo ActorID, payload any, resp chan Message) error {
	// Implicit reply-to delegation:
	// If replyTo is fromActor passive child mailbox of `from`, allow `to` to write to it.
	// Avoid capability accumulation: only grant if `to` doesn't already have RightWrite on replyTo.
	if replyTo != 0 {
		k.Mu.Lock()
		r := k.Actors[replyTo]
		if r != nil && r.Passive && r.Parent == from {
			if toActor := k.Actors[to]; toActor != nil {
				already := false
				for _, c := range toActor.Caps {
					if c.Target == replyTo && !c.Revoked.Load() && (c.Rights&RightWrite) == RightWrite {
						already = true
						break
					}
				}
				if !already {
					k.createCapWithMuLock(to, replyTo, RightWrite, nil)
				}
			}
		}
		k.Mu.Unlock()
	}

	err := k.isPermitted(from, to, payload)
	if err != nil {
		return err
	}

	k.Mu.RLock()
	toActor := k.Actors[to]
	k.Mu.RUnlock()
	if toActor == nil {
		slog.Warn("E_NO_SUCH: toActor actor",
			slog.Any("from", from),
			slog.Any("to", to))
		return errors.New("E_NO_SUCH: toActor actor")
	}

	// Passive actors do not run fromActor handler loop, so they cannot process Exit.
	// We interpret Exit as an immediate kernel-level cleanup.
	if toActor.Passive {
		if exit, ok := payload.(Exit); ok {
			k.cleanupActor(toActor, exit.Reason)
			return nil
		}
	}

	msg := Message{From: from, To: to, ReplyTo: replyTo, Payload: payload, Resp: resp}
	select {
	case toActor.inbox <- msg:
		if fromActor := k.getActor(from); fromActor != nil {
			slog.Info("Sent message",
				slog.Any("from", from),
				slog.Any("to", to),
				slog.Any("payload-type", reflect.TypeOf(payload).String()))
			atomic.AddUint64(&fromActor.IpcOut, 1)
			atomic.AddUint64(&toActor.IpcIn, 1)
		} else {
			slog.Warn("Failed to send message, to actor not found",
				slog.Any("from", from),
				slog.Any("to", to),
			)
		}
		return nil
	case <-time.After(fullMailboxTimeout):
		slog.Warn("E_BUSY: toActor inbox full",
			slog.Any("from", from),
			slog.Any("to", to))
		return errors.New("E_BUSY: toActor inbox full")
	}
}

func (k *Kernel) getActor(id ActorID) *Actor {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	return k.Actors[id]
}

// Name→ActorID lookup helpers
func (k *Kernel) ActorByName(name string) (ActorID, bool) {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	id, ok := k.NameIdx[name]
	return id, ok
}

// Name→ActorID lookup helpers
func (k *Kernel) Start() {

	k.Mu.RLock()
	for _, svc := range k.PrivilegedServices {
		svc.Initialize(k)
	}
	k.Mu.RUnlock()

	// if the CLI service is running send it a boot message
	kernelID, _ := k.ActorByName(KernelService)

	// Broadcast Boot message to all registered actors
	bootMessage := Boot{}
	k.broadcastMessage(kernelID, bootMessage)

	// Keep main alive; also show a periodic status line
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for {
			time.Sleep(time.Duration(1500+r.Intn(1500)) * time.Millisecond)
		}
	}()
	select {
	case <-ctx.Done():
		os.Exit(0)
	}
}

func (k *Kernel) broadcastMessage(kernelID ActorID, message any) {
	k.Mu.RLock()
	// snapshot the recipients you want to send to
	var recipients []ActorID
	for id := range k.Actors {
		if id != kernelID && k.isPermitted(kernelID, id, message) == nil {
			recipients = append(recipients, id)
		}
	}
	k.Mu.RUnlock()

	for actorID := range recipients {
		go func(id ActorID) {
			if err := k.SendInternal(kernelID, id, 0, message, nil); err != nil {
				slog.Warn("Failed to send payload",
					slog.Any("payload", message),
					slog.Any("to", id),
					slog.Any("error", err))
			}
		}(ActorID(actorID))
	}
}

func (k *Kernel) handler(ctx *ActCtx, msg Message) HandlerSignal {
	switch payload := msg.Payload.(type) {
	case Broadcast:
		k.broadcastMessage(msg.From, payload.Payload)
		k.reply(ctx, msg, nil)
	case RequestShutdown:
		slog.Info("RequestShutdown",
			slog.Int("exitCode", payload.ExitCode))
		k.reply(ctx, msg, nil)
		os.Exit(payload.ExitCode)
	default:
		slog.Warn("Unhandled message",
			slog.Any("message", msg))
		k.reply(ctx, msg, nil)
	}
	return Continue{}
}

func (k *Kernel) reply(ctx *ActCtx, msg Message, payload any) {
	if msg.Resp != nil {
		msg.Resp <- Message{From: ctx.Self, To: msg.From, Payload: payload}
	}
}
