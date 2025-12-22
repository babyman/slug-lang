package tcp

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"reflect"
	"slug/internal/dec64"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/svcutil"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SlugActorMessage{}): kernel.RightWrite,
}

// Common keys for the TCP messages
var (
	addrKey      = (&object.String{Value: "addr"}).MapKey()
	portKey      = (&object.String{Value: "port"}).MapKey()
	maxKey       = (&object.String{Value: "max"}).MapKey()
	creditsKey   = (&object.String{Value: "credits"}).MapKey()
	chunkSizeKey = (&object.String{Value: "chunkSize"}).MapKey()
	statusKey    = (&object.String{Value: "status"}).MapKey()
	reasonKey    = (&object.String{Value: "reason"}).MapKey()
	dataKey      = (&object.String{Value: "data"}).MapKey()
	remainingKey = (&object.String{Value: "remaining"}).MapKey()
)

type Service struct{}

type Listener struct {
	netListener net.Listener
}

type streamSub struct {
	reply     kernel.ActorID
	chunkSize int
	credits   int
	gate      chan struct{}
	stopChan  chan struct{}
}

type Connection struct {
	netConn    net.Conn
	subscriber *streamSub
}

func (s *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, ok := p.Msg.(*object.Map)
	if !ok {
		ctx.SendAsync(to, svcutil.ErrorResult("map expected"))
		return kernel.Continue{}
	}

	msgType, ok := m.Pairs[svcutil.MsgTypeKey]
	if !ok {
		ctx.SendAsync(to, svcutil.ErrorResult("missing type"))
		return kernel.Continue{}
	}

	switch msgType.Value.Inspect() {
	case "bind":
		addr := m.Pairs[addrKey].Value.Inspect()
		port := m.Pairs[portKey].Value.Inspect()
		listener := &Listener{}
		id, err := ctx.SpawnChild(fmt.Sprintf("tcp-listener: %s:%s", addr, port), Operations, listener.Handler)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.GrantChildAccess(msg.From, id, kernel.RightWrite, nil)
		ctx.ForwardAsync(id, msg)

	case "connect":
		conn := &Connection{}
		id, err := ctx.SpawnChild("tcp-connection", Operations, conn.Handler)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.GrantChildAccess(msg.From, id, kernel.RightWrite, nil)
		ctx.ForwardAsync(id, msg)
	}

	return kernel.Continue{}
}

func (l *Listener) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	// Handle System Exit
	if _, ok := msg.Payload.(kernel.Exit); ok {
		if l.netListener != nil {
			l.netListener.Close()
		}
		return kernel.Terminate{Reason: "system exit"}
	}

	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, _ := p.Msg.(*object.Map)
	msgType, _ := m.Pairs[svcutil.MsgTypeKey]

	switch msgType.Value.Inspect() {
	case "bind":
		addr := m.Pairs[addrKey].Value.Inspect()
		port := m.Pairs[portKey].Value.Inspect()
		lst, err := net.Listen("tcp", net.JoinHostPort(addr, port))
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Terminate{Reason: err.Error()}
		}
		l.netListener = lst
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "accept":
		if l.netListener == nil {
			ctx.SendAsync(to, svcutil.ErrorResult("not listening"))
			return kernel.Continue{}
		}
		netConn, err := l.netListener.Accept()
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}

		connActor := &Connection{netConn: netConn}
		connId, err := ctx.SpawnChild("tcp-accepted", Operations, connActor.Handler)
		if err != nil {
			netConn.Close()
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		// message must be forwarded to the child actor to pass permissions
		ctx.GrantChildAccess(msg.From, connId, kernel.RightWrite, nil)
		ctx.ForwardAsync(connId, msg)

	case "close":
		if l.netListener != nil {
			l.netListener.Close()
		}
		ctx.SendAsync(to, svcutil.CloseResult(ctx.Self))
		return kernel.Terminate{Reason: "closed"}
	}
	return kernel.Continue{}
}

func (c *Connection) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	// Handle System Exit
	if _, ok := msg.Payload.(kernel.Exit); ok {
		if c.subscriber != nil {
			close(c.subscriber.stopChan)
			c.subscriber = nil
		}
		if c.netConn != nil {
			c.netConn.Close()
		}
		return kernel.Terminate{Reason: "system exit"}
	}

	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := svcutil.ReplyTarget(msg)
	m, _ := p.Msg.(*object.Map)
	msgType, _ := m.Pairs[svcutil.MsgTypeKey]

	slog.Debug("Message received",
		slog.Any("type", msgType.Value.Inspect()),
		slog.Any("actor-id", ctx.Self),
		slog.Any("from", msg.From),
		slog.Any("reply-to", to),
		slog.Any("msgType", msgType),
	)

	switch msgType.Value.Inspect() {
	case "accept":
		// we are connected already, message the connection id to the caller
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "connect":
		addr := m.Pairs[addrKey].Value.Inspect()
		port := m.Pairs[portKey].Value.Inspect()
		var dialer net.Dialer
		conn, err := dialer.DialContext(ctx.Context, "tcp", net.JoinHostPort(addr, port))
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Terminate{Reason: err.Error()}
		}
		c.netConn = conn
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "read":
		maxVal, ok := m.Pairs[maxKey]
		maxSize := 4096
		if ok {
			if n, ok := maxVal.Value.(*object.Number); ok {
				maxSize = int(n.Value.ToInt64())
			}
		}

		buf := make([]byte, maxSize)
		n, err := c.netConn.Read(buf)
		if err != nil {
			if err == io.EOF {
				ctx.SendAsync(to, eofResult())
			} else {
				ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			}
			return kernel.Continue{}
		}
		ctx.SendAsync(to, dataResult(buf[:n]))

	case "write":
		dataObj, ok := m.Pairs[dataKey]
		if !ok {
			ctx.SendAsync(to, svcutil.ErrorResult("missing data"))
			return kernel.Continue{}
		}

		var raw []byte
		switch d := dataObj.Value.(type) {
		case *object.String:
			raw = []byte(d.Value)
		case *object.Bytes:
			raw = d.Value
		default:
			ctx.SendAsync(to, svcutil.ErrorResult("string or bytes expected for write"))
			return kernel.Continue{}
		}

		n, err := c.netConn.Write(raw)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
		} else {
			ctx.SendAsync(to, writeResult(n))
		}

	case "subscribe":
		credits := int(m.Pairs[creditsKey].Value.(*object.Number).Value.ToInt64())
		chunk := int(m.Pairs[chunkSizeKey].Value.(*object.Number).Value.ToInt64())

		// clamp credits to mailbox capacity
		cap, err := ctx.MailboxCapacity(to)
		if err != nil {
			ctx.SendAsync(to, svcutil.ErrorResult(err.Error()))
			return kernel.Continue{}
		}
		if credits > cap {
			credits = cap
		}

		// Clean up existing subscriber if any
		if c.subscriber != nil {
			close(c.subscriber.stopChan)
		}

		sub := &streamSub{
			reply:     to,
			chunkSize: chunk,
			credits:   credits,
			gate:      make(chan struct{}, 1),
			stopChan:  make(chan struct{}),
		}
		c.subscriber = sub
		go c.networkPump(ctx, sub)

		// If we have initial credits, open the gate
		if sub.credits > 0 {
			sub.gate <- struct{}{}
		}

	case "credit":
		if c.subscriber != nil {
			wasEmpty := c.subscriber.credits <= 0
			c.subscriber.credits += int(m.Pairs[creditsKey].Value.(*object.Number).Value.ToInt64())

			// If we just moved from 0 to >0 credits, open the gate
			if wasEmpty && c.subscriber.credits > 0 {
				select {
				case c.subscriber.gate <- struct{}{}:
				default: // Gate already open
				}
			}
		}

	case "_data_ready":
		if c.subscriber == nil {
			return kernel.Continue{}
		}

		c.subscriber.credits--

		if c.subscriber.credits > 0 {
			select {
			case c.subscriber.gate <- struct{}{}:
			default:
			}
		}

		status := m.Pairs[statusKey].Value.Inspect()
		if status == "eof" {
			ctx.SendAsync(c.subscriber.reply, eofResult())
			c.subscriber = nil
		} else if status == "error" {
			reason := m.Pairs[reasonKey].Value.Inspect()
			ctx.SendAsync(c.subscriber.reply, svcutil.ErrorResult(reason))
			c.subscriber = nil
		} else {
			data := m.Pairs[dataKey].Value.(*object.Bytes).Value
			rem := int(m.Pairs[remainingKey].Value.(*object.Number).Value.ToInt64())
			ctx.SendAsync(c.subscriber.reply, dataStreamResult(data, rem))
		}

	case "unsubscribe":
		if c.subscriber != nil {
			close(c.subscriber.stopChan)
			c.subscriber = nil
		}

	case "close":
		if c.subscriber != nil {
			close(c.subscriber.stopChan)
			c.subscriber = nil
		}
		if c.netConn != nil {
			c.netConn.Close()
		}
		ctx.SendAsync(to, svcutil.CloseResult(ctx.Self))
		return kernel.Terminate{Reason: "closed"}
	}
	return kernel.Continue{}
}

func (c *Connection) networkPump(ctx *kernel.ActCtx, sub *streamSub) {
	for {
		select {
		case <-ctx.Context.Done(): // Stop if the actor is terminated
			return
		case <-sub.stopChan:
			return
		case <-sub.gate:
			// We only reach here if the Actor opened the gate
			buf := make([]byte, sub.chunkSize)
			n, err := c.netConn.Read(buf)

			// Send back to actor to handle credit decrementing and gate logic
			ctx.SendAsync(ctx.Self, pumpResult(err, buf, n, sub.credits))

			if err != nil {
				return
			}
		}
	}
}

func dataResult(bytes []byte) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "data")
	svcutil.PutObj(resultMap, "bytes", &object.Bytes{Value: bytes})
	return svc.SlugActorMessage{Msg: resultMap}
}

func dataStreamResult(bytes []byte, credits int) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "data")
	svcutil.PutInt(resultMap, "remainingCredits", credits)
	svcutil.PutObj(resultMap, "bytes", &object.Bytes{Value: bytes})
	return svc.SlugActorMessage{Msg: resultMap}
}

func eofResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "eof")
	return svc.SlugActorMessage{Msg: resultMap}
}

func pumpResult(err error, buf []byte, n int, credits int) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "_data_ready")
	if err != nil {
		if err == io.EOF {
			svcutil.PutString(resultMap, "status", "eof")
		} else {
			svcutil.PutString(resultMap, "status", "error")
			svcutil.PutString(resultMap, "reason", err.Error())
		}
	} else {
		svcutil.PutString(resultMap, "status", "ok")
		svcutil.PutObj(resultMap, "data", &object.Bytes{Value: buf[:n]})
		svcutil.PutInt(resultMap, "remaining", credits-1)
	}
	return svc.SlugActorMessage{Msg: resultMap}
}

func writeResult(bytesWritten int) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	svcutil.PutString(resultMap, "type", "write")
	svcutil.PutInt(resultMap, "written", bytesWritten)
	return svc.SlugActorMessage{Msg: resultMap}
}
