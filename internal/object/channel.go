package object

import (
	"fmt"
	"sync"
)

type Channel struct {
	Capacity int
	ch       chan Object
	closed   bool
	mu       sync.Mutex
}

func NewChannel(capacity int) *Channel {
	if capacity < 0 {
		capacity = 0
	}
	return &Channel{
		Capacity: capacity,
		ch:       make(chan Object, capacity),
	}
}

func (c *Channel) Type() ObjectType { return CHANNEL_OBJ }
func (c *Channel) Inspect() string  { return fmt.Sprintf("<chan %d>", c.Capacity) }

func (c *Channel) GoChan() chan Object {
	return c.ch
}

func (c *Channel) Close() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	c.closed = true
	close(c.ch)
	return true
}

func (c *Channel) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}
