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
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var log = logger.NewLogger("kernel", logger.INFO)

// ===== Core Types =====

// SendSync sends and waits for a single reply.
func (c *ActCtx) SendSync(to ActorID, payload any) (Message, error) {
	respCh := make(chan Message, 1)
	err := c.K.SendInternal(c.Self, to, payload, respCh)
	if err != nil {
		log.Warnf("Error sending message to %d: %v", to, err)
		return Message{}, err
	}
	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(5 * time.Second):
		log.Warnf("E_DEADLINE: reply timeout, from %d to %d", c.Self, to)
		return Message{}, errors.New("E_DEADLINE: reply timeout")
	}
}

// SendAsync fire-and-forgets.
func (c *ActCtx) SendAsync(to ActorID, payload any) error {
	return c.K.SendInternal(c.Self, to, payload, nil)
}

// ===== Kernel =====

type Kernel struct {
	inbox              chan Message
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
		NextActorID:        1,
		inbox:              make(chan Message, 64),
	}

	kernel.OpsBySvc[KernelID] = OpRights{
		reflect.TypeOf(Shutdown{}): RightExec,
	}

	go func() {
		for msg := range kernel.inbox {
			kernel.handler(msg)
		}
	}()
	return kernel
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
		atomic.AddUint64(&a.IpcIn, 1)
		start := time.Now()
		a.handler(ctx, msg)
		atomic.AddUint64(&a.CpuOps, uint64(time.Since(start).Microseconds()))
	}
}

// RegisterPrivilegedService registers a service that needs kernel access
func (k *Kernel) RegisterPrivilegedService(name string, svc PrivilegedService) {
	k.PrivilegedServices[name] = svc
	svc.Initialize(k)
}

// Declare a service (actor) with op→rights mapping, enabling cap checks.
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

// resolveRights returns required rights for an op against a target service.
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

// hasCap checks if sender owns a non-revoked cap to target with required rights.
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

// sendInternal enqueues a message, enforcing capability checks for service ops.
func (k *Kernel) SendInternal(from ActorID, to ActorID, payload any, resp chan Message) error {
	if payload != nil {
		msgType := reflect.TypeOf(payload)
		if rights, ok := k.resolveRights(to, msgType); ok {
			if !k.hasCap(from, to, rights) {
				log.Warnf("E_POLICY: missing rights=%v for op %T from %d to target %d", rights, payload, from, to)
				return fmt.Errorf("E_POLICY: missing rights=%v for op %T to target %d", rights, payload, to)
			}
		}
	}
	var inbox chan Message = nil
	if to == KernelID {
		inbox = k.inbox
	} else {
		k.Mu.RLock()
		target := k.Actors[to]
		k.Mu.RUnlock()
		if target == nil {
			log.Warnf("E_NO_SUCH: target actor, from %d to %d", from, to)
			return errors.New("E_NO_SUCH: target actor")
		} else {
			inbox = target.inbox
		}
	}
	msg := Message{From: from, To: to, Payload: payload, Resp: resp}
	select {
	case inbox <- msg:
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
	cliID, ok := k.ActorByName("cli")
	if ok {
		go func() { _ = k.SendInternal(cliID, cliID, Boot{}, nil) }()
	}

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
		log.Infof("  - Id=%2d Name=%-6s cpu=%8d ops ipc(in=%3d out=%3d) Caps=%d\n",
			Id, a.Name, a.CpuOps, a.IpcIn, a.IpcOut, len(a.Caps))
	}
}
func (k *Kernel) handler(msg Message) {
	switch payload := msg.Payload.(type) {
	case Shutdown:
		log.Infof("Shutdown: %d", payload.ExitCode)
		if msg.Resp != nil {
			msg.Resp <- Message{From: KernelID, To: msg.From, Payload: nil}
		}
		os.Exit(payload.ExitCode)
	default:
		log.Warnf("Unhandled message: %v", msg)
		if msg.Resp != nil {
			msg.Resp <- Message{From: KernelID, To: msg.From, Payload: nil}
		}
	}
}
