package kernel

// Slug Microkernel v1.1 — Actors-in-Kernel with HTTP REPL Service
// ---------------------------------------------------------------
// What’s new vs v1.0:
//  - REPL is now a first-class *service* (actor) exposed over HTTP.
//  - Evaluator service (stub) executes source sent by REPL; easy drop-in for your real tree-walker.
//  - Clean capability checks preserved: REPL talks to EVAL; EVAL talks to FS/TIME via granted caps.
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
//   curl -s localhost:8080/actors | jq
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
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// ===== Core Types =====

type Kernel struct {
	mu          sync.RWMutex
	nextActorID int64
	nextCapID   int64
	actors      map[ActorID]*Actor
	nameIdx     map[string]ActorID // convenient lookup by name
	opsBySvc    map[ActorID]OpRights
}

// SendSync sends and waits for a single reply.
func (c *ActCtx) SendSync(to ActorID, op string, payload any) (Message, error) {
	respCh := make(chan Message, 1)
	err := c.K.sendInternal(c.Self.Id, to, op, payload, respCh)
	if err != nil {
		return Message{}, err
	}
	select {
	case resp := <-respCh:
		return resp, nil
	case <-time.After(5 * time.Second):
		return Message{}, errors.New("E_DEADLINE: reply timeout")
	}
}

// SendAsync fire-and-forgets.
func (c *ActCtx) SendAsync(to ActorID, op string, payload any) error {
	return c.K.sendInternal(c.Self.Id, to, op, payload, nil)
}

// ===== Kernel =====

func NewKernel() *Kernel {

	return &Kernel{
		actors:   make(map[ActorID]*Actor),
		nameIdx:  make(map[string]ActorID),
		opsBySvc: make(map[ActorID]OpRights),
	}
}

// RegisterActor wires an actor into the kernel.
func (k *Kernel) RegisterActor(name string, beh Behavior) ActorID {
	k.mu.Lock()
	defer k.mu.Unlock()
	id := ActorID(k.nextActorID)
	k.nextActorID++
	act := &Actor{
		Id:       id,
		name:     name,
		inbox:    make(chan Message, 64),
		behavior: beh,
		caps:     make(map[int64]*Capability),
	}
	k.actors[id] = act
	if name != "" {
		k.nameIdx[name] = id
	}
	go k.runActor(act)
	return id
}

func (k *Kernel) runActor(a *Actor) {
	ctx := &ActCtx{K: k, Self: a}
	for msg := range a.inbox {
		atomic.AddUint64(&a.ipcIn, 1)
		start := time.Now()
		a.behavior(ctx, msg)
		atomic.AddUint64(&a.cpuOps, uint64(time.Since(start).Microseconds()))
	}
}

// Declare a service (actor) with op→rights mapping, enabling cap checks.
func (k *Kernel) RegisterService(name string, ops OpRights, beh Behavior) ActorID {
	id := k.RegisterActor(name, beh)
	k.mu.Lock()
	k.opsBySvc[id] = ops
	k.mu.Unlock()
	return id
}

// GrantCap issues a capability from kernel to a specific actor.
func (k *Kernel) GrantCap(to ActorID, target ActorID, rights Rights, scope map[string]any) *Capability {
	k.mu.Lock()
	defer k.mu.Unlock()
	capID := k.nextCapID
	k.nextCapID++
	cap := &Capability{ID: capID, Target: target, Rights: rights, Scope: scope}
	if a, ok := k.actors[to]; ok {
		a.caps[capID] = cap
		return cap
	}
	return nil
}

// resolveRights returns required rights for an op against a target service.
func (k *Kernel) resolveRights(target ActorID, op string) (Rights, bool) {
	k.mu.RLock()
	ops, ok := k.opsBySvc[target]
	k.mu.RUnlock()
	if !ok {
		return 0, false
	}
	r, ok := ops[op]
	return r, ok
}

// hasCap checks if sender owns a non-revoked cap to target with required rights.
func (k *Kernel) hasCap(sender ActorID, target ActorID, want Rights) bool {
	k.mu.RLock()
	a := k.actors[sender]
	k.mu.RUnlock()
	if a == nil {
		return false
	}
	for _, c := range a.caps {
		if c.Target == target && !c.Revoked.Load() && (c.Rights&want) == want {
			return true
		}
	}
	return false
}

// sendInternal enqueues a message, enforcing capability checks for service ops.
func (k *Kernel) sendInternal(from ActorID, to ActorID, op string, payload any, resp chan Message) error {
	// Resolve required rights for op (if target is a service with declared ops)
	if rights, ok := k.resolveRights(to, op); ok {
		if !k.hasCap(from, to, rights) {
			return fmt.Errorf("E_POLICY: missing rights=%v for op %q to target %d", rights, op, to)
		}
	}
	k.mu.RLock()
	target := k.actors[to]
	k.mu.RUnlock()
	if target == nil {
		return errors.New("E_NO_SUCH: target actor")
	}
	msg := Message{From: from, To: to, Op: op, Payload: payload, Resp: resp}
	select {
	case target.inbox <- msg:
		if a := k.getActor(from); a != nil {
			atomic.AddUint64(&a.ipcOut, 1)
		}
		return nil
	case <-time.After(2 * time.Second):
		return errors.New("E_BUSY: target inbox full")
	}
}

func (k *Kernel) getActor(id ActorID) *Actor {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.actors[id]
}

// name→ActorID lookup helpers
func (k *Kernel) ActorByName(name string) (ActorID, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	id, ok := k.nameIdx[name]
	return id, ok
}

// name→ActorID lookup helpers
func (k *Kernel) Start() {

	// if the CLI service is running send it a boot message
	cliID, ok := k.ActorByName("cli")
	if ok {
		go func() { _ = k.sendInternal(cliID, cliID, "boot", nil, nil) }()
	}

	// Kick off demo once
	demoID, _ := k.ActorByName("demo")
	go func() { _ = k.sendInternal(demoID, demoID, "start", nil, nil) }()

	// Control plane HTTP
	h := &ControlPlane{k: k}
	h.routes()
	addr := ":8080"
	log.Println("[kernel] control plane listening on", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, nil)) }()

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
	k.mu.RLock()
	defer k.mu.RUnlock()
	fmt.Printf("\n[kernel] actors=%d\n", len(k.actors))
	var ids []ActorID
	for Id := range k.actors {
		ids = append(ids, Id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for _, Id := range ids {
		a := k.actors[Id]
		fmt.Printf("  - Id=%2d name=%-6s cpu=%8d ops ipc(in=%3d out=%3d) caps=%d\n",
			Id, a.name, a.cpuOps, a.ipcIn, a.ipcOut, len(a.caps))
	}
}
