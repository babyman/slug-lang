package evaluator

import (
	"fmt"
	"math/rand"
	"slug/internal/log"
	"slug/internal/object"
	"sync"
	"time"
)

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

var (
	System = NewActorSystem()
)

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

type AMessage interface {
	isMessage()
	String() string
}

type UserMessage struct {
	Payload any
}

func (UserMessage) isMessage() {}
func (m UserMessage) String() string {
	o, k := m.Payload.(object.Object)
	if k {
		return fmt.Sprintf("UserMessage{payload: %v}", o.Inspect())
	} else {
		return fmt.Sprintf("UserMessage{payload: %v}", m.Payload)
	}
}

type Shutdown struct{}

func (Shutdown) isMessage() {}
func (Shutdown) String() string {
	return "Shutdown"
}

type BindActor struct {
	actor *Actor
}

func (BindActor) isMessage() {}
func (m BindActor) String() string {
	return fmt.Sprintf("BindActor{Actor: %v}", m.actor.PID)
}

type UnbindActor struct {
	ActorPID int64
	Failed   bool
}

func (UnbindActor) isMessage() {}
func (m UnbindActor) String() string {
	return fmt.Sprintf("UnbindActor{Actor: %v, Failed: %v}", m.ActorPID, m.Failed)
}

type UnboundActor struct {
}

func (UnboundActor) isMessage() {}
func (m UnboundActor) String() string {
	return fmt.Sprintf("UnboundActor")
}

type UnbindActors struct{}

func (UnbindActors) isMessage() {}
func (m UnbindActors) String() string {
	return "UnbindActors"
}

type ActorExited struct {
	ActorPID       int64
	MailboxPID     int64
	Reason         string
	Result         object.Object
	Function       *object.Function
	Args           []object.Object
	LastMessage    *AMessage
	QueuedMessages []UserMessage
}

func (ActorExited) isMessage() {}
func (m ActorExited) String() string {
	return fmt.Sprintf("ActorExited{Actor: %v, mailbox: %d, ExitCode: %v, LastMessage: %v, QueuedMessages: %d}",
		m.ActorPID, m.MailboxPID, m.Reason, m.LastMessage, len(m.QueuedMessages))
}

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

type Actor struct {
	PID         int64
	MailboxPID  int64
	InTray      chan AMessage
	Mailbox     chan AMessage
	Evaluator   *Evaluator
	Function    *object.Function
	Args        []object.Object
	LastMessage *AMessage
	ExitMessage *ActorExited
}

func NewActor(
	evaluator *Evaluator,
	function *object.Function,
	args []object.Object,
) *Actor {
	actor := &Actor{
		PID:       System.NextMailboxId(),
		InTray:    make(chan AMessage, 1),
		Mailbox:   make(chan AMessage),
		Evaluator: evaluator,
		Function:  function,
		Args:      args,
	}

	evaluator.Actor = actor

	return actor
}

func (a *Actor) run() {
	defer close(a.Mailbox)
	var queue []UserMessage

	for {
		select {
		case item := <-a.InTray:
			switch m := item.(type) {
			case Shutdown:
				System.NotifySupervisor(a.MailboxPID, ActorExited{ActorPID: a.PID, MailboxPID: a.MailboxPID, Reason: "shutdown"})
				return
			case UnboundActor:
				if a.ExitMessage != nil {
					exitMsg := *a.ExitMessage
					exitMsg.QueuedMessages = append(exitMsg.QueuedMessages, queue...)
					System.NotifySupervisor(a.MailboxPID, exitMsg)
				} else {
					System.NotifySupervisor(a.MailboxPID, ActorExited{ActorPID: a.PID, MailboxPID: a.MailboxPID, Reason: "unbound"})
				}
				return
			case ActorExited:
				// we are supervising, relay the message
				a.Mailbox <- m
			case UserMessage:
				if a.ExitMessage == nil {
					a.Mailbox <- m
				} else {
					queue = append(queue, m)
				}
			}
		}
	}
}

func (a *Actor) start() {
	log.Trace("ACT: %d (%d) started\n", a.PID, a.MailboxPID)
	out := a.Evaluator.ApplyFunction("", a.Function, a.Args)
	reason := "return"
	switch out.(type) {
	case *object.Error:
		reason = "error"
	case *object.RuntimeError:
		reason = "error"
	}

	messages := make([]UserMessage, 0)
	messages = a.drainMailbox(messages, a.Mailbox)
	messages = a.drainMailbox(messages, a.InTray)

	a.ExitMessage = &ActorExited{
		ActorPID:       a.PID,
		MailboxPID:     a.MailboxPID,
		Result:         out,
		Function:       a.Function,
		Args:           a.Args,
		Reason:         reason,
		LastMessage:    a.LastMessage,
		QueuedMessages: messages,
	}

	if reason == "error" {
		System.UnbindFailedActor(a.MailboxPID, a.PID)
	} else {
		System.UnbindActor(a.MailboxPID, a.PID)
	}
}

func (a *Actor) drainMailbox(messages []UserMessage, mailbox chan AMessage) []UserMessage {
	timeout := time.After(10 * time.Millisecond)
	for {
		select {
		case msg := <-mailbox:
			switch m := msg.(type) {
			case UserMessage:
				messages = append(messages, m)
			}
		case <-timeout:
			return messages
		default:
			return messages
		}
	}
}

func (a *Actor) WaitForMessage(timeout int64) (AMessage, bool) {
	if timeout <= 0 {
		msg := <-a.Mailbox
		a.LastMessage = &msg
		return msg, true
	}

	select {
	case msg := <-a.Mailbox:
		a.LastMessage = &msg
		return msg, true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		a.LastMessage = nil
		return nil, false
	}
}

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

type RelayMode string

const (
	RoundRobin RelayMode = "roundrobin"
	Broadcast  RelayMode = "broadcast"
)

type Mailbox struct {
	PID    int64
	Actors []*Actor
	InTray chan AMessage
	Mode   RelayMode
	Next   int
}

func NewMailbox(pid int64, mode RelayMode) *Mailbox {
	mailbox := &Mailbox{
		PID:    pid,
		Actors: make([]*Actor, 0),
		InTray: make(chan AMessage, 1),
		Mode:   mode,
		Next:   0,
	}
	return mailbox
}

func (m *Mailbox) run() {
	log.Trace("BOX: %d running\n", m.PID)
	var queue []AMessage

	for {
		select {
		case item := <-m.InTray:
			log.Debug("BOX: %d queue size %d, received: %s\n", m.PID, len(queue), item.String())
			switch msg := item.(type) {
			case Shutdown:
				//fmt.Printf("Mailbox %d received stop message\n", m.PID)
				System.RemoveMailbox(m.PID)
				for _, actor := range m.Actors {
					m.UnbindActor(actor.PID, false)
				}
				return
			case BindActor:
				m.BindActor(msg.actor)
			case UnbindActor:
				m.UnbindActor(msg.ActorPID, msg.Failed)
			case UnbindActors:
				for _, actor := range m.Actors {
					m.UnbindActor(actor.PID, false)
				}
			default:
				queue = append(queue, item)
			}

		default:
			if len(queue) > 0 && len(m.Actors) > 0 {
				message := queue[0]

				switch m.Mode {
				case Broadcast:
					m.broadcast(message)
				case RoundRobin:
					m.roundrobin(message)
				}

				queue = queue[1:]
			}
		}
	}
}

func (m *Mailbox) broadcast(message AMessage) {
	for _, actor := range m.Actors {
		actor.InTray <- message
	}
}

func (m *Mailbox) roundrobin(message AMessage) {
	if len(m.Actors) > 0 {
		actor := m.Actors[m.Next%len(m.Actors)]
		m.Next = (m.Next + 1) % len(m.Actors)
		actor.InTray <- message
	}
}

func (m *Mailbox) BindActor(actor *Actor) {
	m.Actors = append(m.Actors, actor)
	actor.MailboxPID = m.PID
}

func (m *Mailbox) UnbindActor(actorPID int64, failed bool) {
	for i, actor := range m.Actors {
		if actor.PID == actorPID {
			log.Trace("ACT: %d (%d), unbinding Actor\n", actorPID, m.PID)
			m.Actors = append(m.Actors[:i], m.Actors[i+1:]...)
			actor.InTray <- UnboundActor{}
			close(actor.InTray)
			if !failed && len(m.Actors) == 0 {
				m.InTray <- Shutdown{}
			}
			break
		}
	}
}

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

type ActorSystem struct {
	mailboxes           map[int64]*Mailbox
	mailboxesLock       sync.RWMutex
	supervisors         map[int64]int64
	supervisorsLock     sync.RWMutex
	mailboxRegistry     map[string]int64
	mailboxRegistryLock sync.RWMutex
	nextMailboxId       int64
	nextMailboxIdLock   sync.RWMutex
}

func NewActorSystem() *ActorSystem {
	return &ActorSystem{
		mailboxes:       make(map[int64]*Mailbox),
		supervisors:     make(map[int64]int64),
		mailboxRegistry: make(map[string]int64),
		nextMailboxId:   1,
	}
}

func (a *ActorSystem) NextMailboxId() int64 {
	a.nextMailboxIdLock.Lock()
	defer a.nextMailboxIdLock.Unlock()
	id := a.nextMailboxId<<16 | int64(rand.Intn(0xFFFF))
	a.nextMailboxId++
	return id
}

func (a *ActorSystem) NewMailbox(
	mode RelayMode,
) int64 {
	pid := a.NextMailboxId()
	mailbox := NewMailbox(pid, mode)

	go mailbox.run()

	a.mailboxesLock.Lock()
	a.mailboxes[pid] = mailbox
	a.mailboxesLock.Unlock()

	return pid
}

func (a *ActorSystem) BindNewActor(
	pid int64,
	function *object.Function,
	args ...object.Object,
) (int64, bool) {
	a.mailboxesLock.RLock()
	mailbox, exists := a.mailboxes[pid]
	a.mailboxesLock.RUnlock()

	if exists {
		evaluator := &Evaluator{
			envStack: []*object.Environment{function.Env},
		}

		actor := NewActor(evaluator, function, args)

		go actor.start()
		go actor.run()

		mailbox.InTray <- BindActor{actor: actor}

		log.Debug("ACT: %d (%d) Actor created and bound", actor.PID, mailbox.PID)

		return pid, true
	}
	return 0, false
}

func (a *ActorSystem) Send(toPid int64, message AMessage) {
	a.mailboxesLock.RLock()
	mailbox, exists := a.mailboxes[toPid]
	a.mailboxesLock.RUnlock()
	if exists {
		mailbox.InTray <- message
	}
}

func (a *ActorSystem) SendData(toPid int64, data any) {
	a.Send(toPid, UserMessage{
		Payload: data,
	})
}

func (a *ActorSystem) UnbindFailedActor(toPid int64, actorPID int64) {
	a.Send(toPid, UnbindActor{ActorPID: actorPID, Failed: true})
}

func (a *ActorSystem) UnbindActor(toPid int64, actorPID int64) {
	a.Send(toPid, UnbindActor{ActorPID: actorPID, Failed: false})
}

func (a *ActorSystem) Supervise(supervisorPid, supervisedPid int64) {
	a.mailboxesLock.RLock()
	_, exists := a.mailboxes[supervisedPid]
	a.mailboxesLock.RUnlock()
	if exists {
		a.supervisorsLock.Lock()
		a.supervisors[supervisedPid] = supervisorPid
		a.supervisorsLock.Unlock()
	}
}

func (a *ActorSystem) Supervisor(pid int64) (int64, bool) {
	a.supervisorsLock.RLock()
	defer a.supervisorsLock.RUnlock()
	supervisorPid, exists := a.supervisors[pid]
	return supervisorPid, exists
}

func (a *ActorSystem) Unsupervise(supervisedPid int64) {
	delete(a.supervisors, supervisedPid)
}

func (a *ActorSystem) NotifySupervisor(pid int64, message AMessage) {
	if supervisor, exists := a.supervisors[pid]; exists {
		a.Send(supervisor, message)
	}
}

func (a *ActorSystem) Register(toPid int64, name string) {
	a.mailboxesLock.RLock()
	_, exists := a.mailboxes[toPid]
	a.mailboxesLock.RUnlock()
	if exists {
		a.mailboxRegistryLock.Lock()
		a.mailboxRegistry[name] = toPid
		a.mailboxRegistryLock.Unlock()
	}
}

func (a *ActorSystem) Unregister(name string) {
	a.mailboxRegistryLock.Lock()
	defer a.mailboxRegistryLock.Unlock()
	delete(a.mailboxRegistry, name)
}

func (a *ActorSystem) WhereIs(name string) (int64, bool) {
	a.mailboxRegistryLock.RLock()
	defer a.mailboxRegistryLock.RUnlock()
	pid, ok := a.mailboxRegistry[name]
	return pid, ok
}

func (a *ActorSystem) WhoIs(pid int64) (string, bool) {
	a.mailboxRegistryLock.RLock()
	defer a.mailboxRegistryLock.RUnlock()

	for name, registeredPid := range a.mailboxRegistry {
		if registeredPid == pid {
			return name, true
		}
	}
	return "", false
}

func (a *ActorSystem) RemoveMailbox(mailboxPid int64) {

	a.mailboxRegistryLock.Lock()
	for name, registeredPid := range a.mailboxRegistry {
		if registeredPid == mailboxPid {
			delete(a.mailboxRegistry, name)
		}
	}
	a.mailboxRegistryLock.Unlock()

	a.supervisorsLock.Lock()
	for supervised, supervisorPid := range a.supervisors {
		if supervisorPid == mailboxPid {
			delete(a.supervisors, supervised)
		}
	}
	a.supervisorsLock.Unlock()

	a.mailboxesLock.Lock()
	if _, exists := a.mailboxes[mailboxPid]; exists {
		delete(a.mailboxes, mailboxPid)
	}
	a.mailboxesLock.Unlock()
}

func (a *ActorSystem) Shutdown() {
	a.mailboxesLock.RLock()
	mailboxes := a.mailboxes
	a.mailboxesLock.RUnlock()

	for _, mailbox := range mailboxes {
		System.Send(mailbox.PID, Shutdown{})
		log.Trace("BOX SHUTDOWN: %d, closing InTray\n", mailbox.PID)
		close(mailbox.InTray)
	}
}

// =====================================================================================================================
// =====================================================================================================================
// =====================================================================================================================

func CreateMainThreadMailbox() *Actor {
	pid := System.NextMailboxId()
	actor := &Actor{
		PID:     System.NextMailboxId(),
		InTray:  make(chan AMessage, 1),
		Mailbox: make(chan AMessage),
	}

	mailbox := NewMailbox(pid, RoundRobin)
	mailbox.InTray <- BindActor{actor: actor}

	System.mailboxesLock.Lock()
	System.mailboxes[pid] = mailbox
	System.mailboxesLock.Unlock()

	log.Trace("ACT: main Actor %d (%d) created\n", actor.PID, actor.MailboxPID)

	// do not run the function for main
	//go Actor.start()
	go actor.run()

	go mailbox.run()

	return actor
}
