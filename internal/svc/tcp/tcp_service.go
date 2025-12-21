package tcp

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"slug/internal/dec64"
	"slug/internal/kernel"
	"slug/internal/object"
	"slug/internal/svc"
	"slug/internal/svc/sqlutil"
)

var Operations = kernel.OpRights{
	reflect.TypeOf(svc.SlugActorMessage{}): kernel.RightWrite,
}

// Common keys for the TCP messages
var (
	AddrKey = (&object.String{Value: "addr"}).MapKey()
	PortKey = (&object.String{Value: "port"}).MapKey()
	DataKey = (&object.String{Value: "data"}).MapKey()
	MaxKey  = (&object.String{Value: "max"}).MapKey()
)

type Service struct{}

type Listener struct {
	netListener net.Listener
}

type Connection struct {
	netConn net.Conn
}

func (s *Service) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := sqlutil.ReplyTarget(msg)
	m, ok := p.Msg.(*object.Map)
	if !ok {
		ctx.SendAsync(to, errorResult("map expected"))
		return kernel.Continue{}
	}

	msgType, ok := m.Pairs[sqlutil.MsgTypeKey]
	if !ok {
		ctx.SendAsync(to, errorResult("missing type"))
		return kernel.Continue{}
	}

	switch msgType.Value.Inspect() {
	case "bind":
		addr := m.Pairs[AddrKey].Value.Inspect()
		port := m.Pairs[PortKey].Value.Inspect()
		listener := &Listener{}
		id, err := ctx.SpawnChild(fmt.Sprintf("tcp-listener (%s:%s)", addr, port), Operations, listener.Handler)
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.GrantChildAccess(msg.From, id, kernel.RightWrite, nil)
		ctx.ForwardAsync(id, msg)

	case "connect":
		conn := &Connection{}
		id, err := ctx.SpawnChild("tcp-connection", Operations, conn.Handler)
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Continue{}
		}
		ctx.GrantChildAccess(msg.From, id, kernel.RightWrite, nil)
		ctx.ForwardAsync(id, msg)
	}

	return kernel.Continue{}
}

func (l *Listener) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := sqlutil.ReplyTarget(msg)
	m, _ := p.Msg.(*object.Map)
	msgType, _ := m.Pairs[sqlutil.MsgTypeKey]

	switch msgType.Value.Inspect() {
	case "bind":
		addr := m.Pairs[AddrKey].Value.Inspect()
		port := m.Pairs[PortKey].Value.Inspect()
		lst, err := net.Listen("tcp", net.JoinHostPort(addr, port))
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Terminate{Reason: err.Error()}
		}
		l.netListener = lst
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "accept":
		if l.netListener == nil {
			ctx.SendAsync(to, errorResult("not listening"))
			return kernel.Continue{}
		}
		netConn, err := l.netListener.Accept()
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Continue{}
		}

		connActor := &Connection{netConn: netConn}
		connId, err := ctx.SpawnChild("tcp-accepted", Operations, connActor.Handler)
		if err != nil {
			netConn.Close()
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Continue{}
		}
		// message must be forwarded to the child actor to pass permissions
		ctx.GrantChildAccess(msg.From, connId, kernel.RightWrite, nil)
		ctx.ForwardAsync(connId, msg)

	case "close":
		if l.netListener != nil {
			l.netListener.Close()
		}
		ctx.SendAsync(to, sqlutil.Success())
		return kernel.Terminate{Reason: "closed"}
	}
	return kernel.Continue{}
}

func (c *Connection) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	p, ok := msg.Payload.(svc.SlugActorMessage)
	if !ok {
		return kernel.Continue{}
	}

	to := sqlutil.ReplyTarget(msg)
	m, _ := p.Msg.(*object.Map)
	msgType, _ := m.Pairs[sqlutil.MsgTypeKey]

	switch msgType.Value.Inspect() {
	case "accept":
		// we are connected already, message the connection id to the caller
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "connect":
		addr := m.Pairs[AddrKey].Value.Inspect()
		port := m.Pairs[PortKey].Value.Inspect()
		conn, err := net.Dial("tcp", net.JoinHostPort(addr, port))
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
			return kernel.Terminate{Reason: err.Error()}
		}
		c.netConn = conn
		ctx.SendAsync(to, svc.SlugActorMessage{Msg: &object.Number{
			Value: dec64.FromInt64(int64(ctx.Self)),
		}})

	case "read":
		maxVal, ok := m.Pairs[MaxKey]
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
				ctx.SendAsync(to, errorResult(err.Error()))
			}
			return kernel.Continue{}
		}
		ctx.SendAsync(to, dataResult(buf[:n]))

	case "write":
		dataObj, ok := m.Pairs[DataKey]
		if !ok {
			ctx.SendAsync(to, errorResult("missing data"))
			return kernel.Continue{}
		}

		var raw []byte
		switch d := dataObj.Value.(type) {
		case *object.String:
			raw = []byte(d.Value)
		case *object.Bytes:
			raw = d.Value
		default:
			ctx.SendAsync(to, errorResult("string or bytes expected for write"))
			return kernel.Continue{}
		}

		n, err := c.netConn.Write(raw)
		if err != nil {
			ctx.SendAsync(to, errorResult(err.Error()))
		} else {
			ctx.SendAsync(to, writeResult(n))
		}

	case "close":
		if c.netConn != nil {
			c.netConn.Close()
		}
		ctx.SendAsync(to, closeResult())
		return kernel.Terminate{Reason: "closed"}
	}
	return kernel.Continue{}
}

func errorResult(msg string) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	sqlutil.PutString(resultMap, "type", "error")
	sqlutil.PutString(resultMap, "msg", msg)
	return svc.SlugActorMessage{Msg: resultMap}
}

func dataResult(bytes []byte) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	sqlutil.PutString(resultMap, "type", "data")
	sqlutil.PutObj(resultMap, "bytes", &object.Bytes{Value: bytes})
	return svc.SlugActorMessage{Msg: resultMap}
}

func eofResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	sqlutil.PutString(resultMap, "type", "eof")
	return svc.SlugActorMessage{Msg: resultMap}
}

func writeResult(bytesWritten int) svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	sqlutil.PutString(resultMap, "type", "write")
	sqlutil.PutInt(resultMap, "written", bytesWritten)
	return svc.SlugActorMessage{Msg: resultMap}
}

func closeResult() svc.SlugActorMessage {
	resultMap := &object.Map{Pairs: make(map[object.MapKey]object.MapPair)}
	sqlutil.PutString(resultMap, "type", "closed")
	return svc.SlugActorMessage{Msg: resultMap}
}
