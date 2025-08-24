package kernel

import (
	"sync/atomic"
)

type ActorID int64

type Rights uint64

const (
	RightRead  Rights = 1 << iota // e.g., fs.read, time.now
	RightWrite                    // e.g., fs.write
	RightExec                     // e.g., time.sleep / eval.run
)

// OpRights declares what rights are required to invoke a given op on a service.
type OpRights map[string]Rights

// Capability binds a sender to a target service actor with specific rights and optional scope.
type Capability struct {
	ID      int64          `json:"Id"`
	Target  ActorID        `json:"target"`
	Rights  Rights         `json:"rights"`
	Scope   map[string]any `json:"scope,omitempty"`
	Revoked atomic.Bool    `json:"-"`
}

type Message struct {
	From    ActorID      `json:"from"`
	To      ActorID      `json:"to"`
	Payload any          `json:"payload,omitempty"`
	Resp    chan Message `json:"-"` // optional synchronous reply channel
}

// Actor behavior is a function invoked for each incoming message.
type Behavior func(ctx *ActCtx, msg Message)

type Actor struct {
	Id       ActorID
	name     string
	inbox    chan Message
	behavior Behavior
	caps     map[int64]*Capability // by cap ID
	// simple accounting
	cpuOps uint64
	ipcIn  uint64
	ipcOut uint64
}

type ActCtx struct {
	K    *Kernel
	Self *Actor
}

type CapabilityView struct {
	ID      int64   `json:"Id"`
	Target  ActorID `json:"target"`
	Rights  Rights  `json:"rights"`
	Revoked bool    `json:"revoked"`
}

//Kernel message payload types

type DemoStart struct {
}

type UnknownOperation struct {
}
