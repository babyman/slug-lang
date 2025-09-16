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
	"math/rand"
	"os"
	"reflect"
	"slug/internal/logger"
	"slug/internal/util/future"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var log = logger.NewLogger("kernel", SystemLogLevel())

// ===== Core Types =====

func (c *ActCtx) RegisterCleanup(msg Message) {
	c.K.RegisterCleanup(c.Self, msg)
}

// SendAsync fire-and-forgets.
func (c *ActCtx) SendAsync(to ActorID, payload any) error {
	return c.K.SendInternal(c.Self, to, payload, nil)
}

func (c *ActCtx) SendFuture(to ActorID, payload any) (*future.Future[Message], error) {
	f := future.New[Message](func() (Message, error) {
		respCh := make(chan Message, 1)
		err := c.K.SendInternal(c.Self, to, payload, respCh)
		if err != nil {
			log.Warnf("Error sending message to %d from %d: %v", to, c.Self, err)
			return Message{}, err
		}
		select {
		case resp := <-respCh:
			return resp, nil
		}
	})
	return f, nil
}

// SendSync sends and waits for a single reply.
func (c *ActCtx) SendSync(to ActorID, payload any) (Message, error) {
	return c.SendSyncWithTimeout(to, payload, 5*time.Second)
}

// SendSync sends and waits for a single reply.
func (c *ActCtx) SendSyncWithTimeout(to ActorID, payload any, timeout time.Duration) (Message, error) {
	f, err := c.SendFuture(to, payload)
	if err != nil {
		return Message{}, err
	}

	resp, err, ok := f.AwaitTimeout(timeout)
	if !ok {
		log.Warnf("E_DEADLINE: reply timeout %v, from %d to %d, %T", timeout, c.Self, to, payload)
		return Message{}, fmt.Errorf("E_DEADLINE: reply timeout %v", timeout)
	}
	return resp, err
}

// ===== Kernel =====

type Kernel struct {
	Mu                 sync.RWMutex
	NextActorID        int64
	NextCapID          int64
	Actors             map[ActorID]*Actor
	NameIdx            map[string]ActorID // convenient lookup by Name
	OpsBySvc           map[ActorID]OpRights
	PrivilegedServices map[string]PrivilegedService
}

func NewKernel() *Kernel {

	kernel := &Kernel{
		Actors:             make(map[ActorID]*Actor),
		NameIdx:            make(map[string]ActorID),
		OpsBySvc:           make(map[ActorID]OpRights),
		PrivilegedServices: make(map[string]PrivilegedService),
	}

	kernel.RegisterService(KernelService, Operations, kernel.handler)

	return kernel
}

func (k *Kernel) RegisterCleanup(id ActorID, msg Message) {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	a, ok := k.Actors[id]
	if ok {
		a.Cleanup = append(a.Cleanup, msg)
	}
}

// RegisterActor wires an actor into the kernel.
func (k *Kernel) RegisterActor(name string, handler Handler) ActorID {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	id := ActorID(k.NextActorID)
	k.NextActorID++
	act := &Actor{
		Id:      id,
		Name:    name,
		inbox:   make(chan Message, 64),
		handler: handler,
		Caps:    make(map[int64]*Capability),
		Cleanup: []Message{},
	}
	k.Actors[id] = act
	if name != "" {
		k.NameIdx[name] = id
	}
	go k.runActor(act)
	return id
}

func (k *Kernel) runActor(a *Actor) {
	ctx := &ActCtx{K: k, Self: a.Id}
	for msg := range a.inbox {
		if exit, ok := msg.Payload.(Exit); ok {
			k.cleanupActor(a, exit.Reason)
			println("Exiting actor", a.Name, "with reason", exit.Reason)
			return
		}

		atomic.AddUint64(&a.IpcIn, 1)
		start := time.Now()
		sig := a.handler(ctx, msg)
		atomic.AddUint64(&a.CpuOps, uint64(time.Since(start).Microseconds()))

		switch signal := sig.(type) {
		case nil, Continue:
			continue
		case Terminate:
			k.cleanupActor(a, signal.Reason)
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

// cleanupActor removes the actor from kernel tracking and closes resources
func (k *Kernel) cleanupActor(a *Actor, reason string) {
	k.Mu.Lock()
	defer k.Mu.Unlock()

	log.Infof("Cleaning up actor %d (%s): %s", a.Id, a.Name, reason)

	// Send cleanup messages in reverse order (LIFO)
	for i := len(a.Cleanup) - 1; i >= 0; i-- {
		cleanupMsg := a.Cleanup[i]
		log.Infof("Sending cleanup message from %d to %d", cleanupMsg.From, cleanupMsg.To)
		// Send cleanup message without capability checks since this is part of shutdown
		if target := k.Actors[cleanupMsg.To]; target != nil {
			select {
			case target.inbox <- cleanupMsg:
				// Message sent successfully
			default:
				// Target inbox full or closed, log and continue
				log.Warnf("Failed to send cleanup message to actor %d: inbox full or closed", cleanupMsg.To)
			}
		} else {
			log.Warnf("Cleanup target actor %d not found", cleanupMsg.To)
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

	// Revoke all capabilities granted to this actor
	for _, cap := range a.Caps {
		cap.Revoked.Store(true)
	}

	// Close the inbox channel to prevent further messages
	close(a.inbox)
}

// restartActor recreates an actor with the same configuration
func (k *Kernel) restartActor(a *Actor, restart Restart) {
	log.Infof("Restarting actor %d (%s)", a.Id, a.Name)

	// todo this will require new capabilities or nothing can call it!

	// Store original configuration
	name := a.Name
	handler := a.handler

	// Register a new actor with the same name and handler
	newID := k.RegisterActor(name, handler)

	log.Infof("Actor %s restarted with new ID %d (was %d)", name, newID, a.Id)
}

// handleActorError processes actor errors based on policy
func (k *Kernel) handleActorError(a *Actor, err Error) {
	log.Errorf("Actor %d (%s) reported error: %v", a.Id, a.Name, err.Err)

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
	k.PrivilegedServices[name] = svc
	svc.Initialize(k)
}

// Declare a service (actor) with op→Operations mapping, enabling cap checks.
func (k *Kernel) RegisterService(name string, ops OpRights, handler Handler) ActorID {
	id := k.RegisterActor(name, handler)
	k.Mu.Lock()
	k.OpsBySvc[id] = ops
	k.Mu.Unlock()
	return id
}

// GrantCap issues a capability from kernel to a specific actor.
func (k *Kernel) GrantCap(to ActorID, target ActorID, rights Rights, scope map[reflect.Type]any) *Capability {
	k.Mu.Lock()
	defer k.Mu.Unlock()
	capID := k.NextCapID
	k.NextCapID++
	capability := &Capability{ID: capID, Target: target, Rights: rights, Scope: scope}
	if a, ok := k.Actors[to]; ok {
		a.Caps[capID] = capability
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
	a := k.Actors[sender]
	k.Mu.RUnlock()
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
		msgType := reflect.TypeOf(payload)
		if rights, ok := k.resolveRights(to, msgType); ok {
			if !k.hasCap(from, to, rights) {
				log.Warnf("E_POLICY: cap not granted for Operations=%v for op %T from %d to target %d", rights, payload, from, to)
				return fmt.Errorf("E_POLICY: cap not granted for Operations=%v for op %T to target %d", rights, payload, to)
			}
		} else {
			log.Warnf("E_POLICY: no defined Operations for op %T from %d to target %d", payload, from, to)
			return fmt.Errorf("E_POLICY: no defined Operations for op %T to target %d", payload, to)
		}
	} else {
		log.Warnf("E_POLICY: nil payload from %d to target %d", from, to)
		return fmt.Errorf("E_POLICY: nil payload to target %d", to)
	}
	return nil
}

// sendInternal enqueues a message, enforcing capability checks for service ops.
func (k *Kernel) SendInternal(from ActorID, to ActorID, payload any, resp chan Message) error {
	err := k.isPermitted(from, to, payload)
	if err != nil {
		return err
	}
	k.Mu.RLock()
	target := k.Actors[to]
	k.Mu.RUnlock()
	if target == nil {
		log.Warnf("E_NO_SUCH: target actor, from %d to %d", from, to)
		return errors.New("E_NO_SUCH: target actor")
	}
	msg := Message{From: from, To: to, Payload: payload, Resp: resp}
	select {
	case target.inbox <- msg:
		if a := k.getActor(from); a != nil {
			atomic.AddUint64(&a.IpcOut, 1)
		}
		return nil
	case <-time.After(2 * time.Second):
		log.Warnf("E_BUSY: target inbox full, from %d to %d", from, to)
		return errors.New("E_BUSY: target inbox full")
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
			printStatus(k)
		}
	}()
	select {
	case <-ctx.Done():
		os.Exit(0)
	}
}

func (k *Kernel) broadcastMessage(kernelID ActorID, message any) {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	for actorID := range k.Actors {
		if actorID != kernelID && k.isPermitted(kernelID, actorID, message) == nil {
			go func(id ActorID) {
				if err := k.SendInternal(kernelID, id, message, nil); err != nil {
					log.Warnf("Failed to send %T to actor %d: %v", message, id, err)
				}
			}(actorID)
		}
	}
}

func printStatus(k *Kernel) {
	k.Mu.RLock()
	defer k.Mu.RUnlock()
	log.Infof("Actors=%d\n", len(k.Actors))
	var ids []ActorID
	for Id := range k.Actors {
		ids = append(ids, Id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, Id := range ids {
		a := k.Actors[Id]
		log.Infof("  - Id=%2d Name=%-6s cpu(μs) %8d ops ipc(in=%3d out=%3d) Caps=%d\n",
			Id, a.Name, a.CpuOps, a.IpcIn, a.IpcOut, len(a.Caps))
	}
}

func (k *Kernel) handler(ctx *ActCtx, msg Message) HandlerSignal {
	switch payload := msg.Payload.(type) {
	case Broadcast:
		k.broadcastMessage(msg.From, payload.Payload)
		k.reply(ctx, msg, nil)
	case RequestShutdown:
		log.Infof("RequestShutdown: %d", payload.ExitCode)
		k.reply(ctx, msg, nil)
		printStatus(k)
		os.Exit(payload.ExitCode)
	default:
		log.Warnf("Unhandled message: %v", msg)
		printStatus(k)
		k.reply(ctx, msg, nil)
	}
	return Continue{}
}

func (k *Kernel) reply(ctx *ActCtx, msg Message, payload any) {
	if msg.Resp != nil {
		msg.Resp <- Message{From: ctx.Self, To: msg.From, Payload: payload}
	}
}
