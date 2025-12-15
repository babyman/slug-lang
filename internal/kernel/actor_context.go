package kernel

import (
	"fmt"
	"log/slog"
	"reflect"
	"slug/internal/util/future"
	"time"
)

type ActCtx struct {
	K    IKernel
	Self ActorID
}

type SlugReceiver interface {
	WaitForMessage(timeout int64) (any, bool)
}

type IKernel interface {
	ActorByName(name string) (ActorID, bool)
	SendInternal(from ActorID, to ActorID, payload any, respCh chan Message) error
	RegisterCleanup(id ActorID, msg Message)
	SpawnChild(parent ActorID, name string, ops OpRights, handler Handler) (ActorID, error)
	Register(name string, pid ActorID)
	Unregister(name string) ActorID
	Registered() []string
	Lookup(name string) ActorID
	GrantChildAccess(granter ActorID, grantee ActorID, target ActorID, rights Rights, scope map[reflect.Type]any) (*Capability, error)
}

func (c *ActCtx) RegisterCleanup(msg Message) {
	c.K.RegisterCleanup(c.Self, msg)
}

func (c *ActCtx) ForwardAsync(to ActorID, msg Message) error {
	return c.K.SendInternal(msg.From, to, msg.Payload, msg.Resp)
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
			slog.Warn("Error sending message",
				slog.Any("to", to),
				slog.Any("from", c.Self),
				slog.Any("error", err))
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
	response, err := c.SendSyncWithTimeout(to, payload, defaultSendTimeout)
	if err != nil {
		slog.Error("error sending message",
			slog.Any("error", err))
		return Message{}, err
	}
	return response, err
}

func (c *ActCtx) SendSyncWithTimeout(to ActorID, payload any, timeout time.Duration) (Message, error) {
	f, err := c.SendFuture(to, payload)
	if err != nil {
		return Message{}, err
	}

	resp, err, ok := f.AwaitTimeout(timeout)
	if !ok {
		slog.Warn("E_DEADLINE: reply timeout",
			slog.Any("timeout", timeout),
			slog.Any("from", c.Self),
			slog.Any("to", to),
			slog.Any("payload", payload))
		return Message{}, fmt.Errorf("E_DEADLINE: reply timeout %v", timeout)
	}
	return resp, err
}

func (c *ActCtx) SpawnChild(name string, ops OpRights, handler Handler) (ActorID, error) {

	return c.K.SpawnChild(c.Self, name, ops, handler)
}

func (c *ActCtx) GrantChildAccess(
	grantee ActorID,
	target ActorID,
	rights Rights,
	scope map[reflect.Type]any) (*Capability, error) {

	return c.K.GrantChildAccess(c.Self, grantee, target, rights, scope)
}
