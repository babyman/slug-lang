package kernel

import (
	"reflect"
	"sync/atomic"
)

type ActorID int64

type Rights uint64

const (
	RightRead  Rights = 1 << iota // e.g., fs.read, time.now
	RightWrite                    // e.g., fs.write
	RightExec                     // e.g., time.sleep / eval.run
)

// OpRights declares what Operations are required to invoke a given op on a service.
type OpRights map[reflect.Type]Rights

// Capability binds a sender to a target service actor with specific Operations and optional scope.
type Capability struct {
	ID      int64                `json:"Id"`
	Target  ActorID              `json:"target"`
	Rights  Rights               `json:"Operations"`
	Scope   map[reflect.Type]any `json:"scope,omitempty"`
	Revoked atomic.Bool          `json:"-"`
}

type Message struct {
	From    ActorID      `json:"from"`
	To      ActorID      `json:"to"`
	Payload any          `json:"payload,omitempty"`
	Resp    chan Message `json:"-"` // optional synchronous reply channel
}

type Actor struct {
	Id      ActorID
	Name    string
	inbox   chan Message
	handler Handler
	Caps    map[int64]*Capability // by cap ID
	// simple accounting
	CpuOps uint64
	IpcIn  uint64
	IpcOut uint64
}

type ActCtx struct {
	K    IKernel
	Self ActorID
}

type IKernel interface {
	ActorByName(name string) (ActorID, bool)
	SendInternal(from ActorID, to ActorID, payload any, respCh chan Message) error
}

type PrivilegedService interface {
	Initialize(k *Kernel)
}

type CapabilityView struct {
	ID      int64   `json:"Id"`
	Target  ActorID `json:"target"`
	Rights  Rights  `json:"Operations"`
	Revoked bool    `json:"revoked"`
}

// Actor handler is a function invoked for each incoming message.
// ==============================================================

type Handler func(ctx *ActCtx, msg Message) HandlerSignal

type HandlerSignal interface {
	Signal()
}

type Continue struct{}

func (c Continue) Signal() {}

type Terminate struct {
	Reason string
}

func (t Terminate) Signal() {}

type Restart struct {
	Reason string
}

func (r Restart) Signal() {}

type Error struct {
	Err error
}

func (e Error) Signal() {}
