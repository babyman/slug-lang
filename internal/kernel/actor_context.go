package kernel

import (
	"fmt"
	"slug/internal/util/future"
	"time"
)

type ActCtx struct {
	K    IKernel
	Self ActorID
}

type IKernel interface {
	ActorByName(name string) (ActorID, bool)
	SendInternal(from ActorID, to ActorID, payload any, respCh chan Message) error
	RegisterCleanup(id ActorID, msg Message)
	SpawnChild(parent ActorID, name string, handler Handler) (ActorID, error)
}

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

func (c *ActCtx) SpawnChild(name string, handler Handler) (ActorID, error) {

	return c.K.SpawnChild(c.Self, name, handler)
}
