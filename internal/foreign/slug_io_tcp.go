package foreign

import (
	"fmt"
	"io"
	"net"
	"slug/internal/dec64"
	"slug/internal/object"
	"sync"
)

var (
	ioTcpMu        sync.RWMutex
	ioTcpListeners = map[int64]net.Listener{}
	ioTcpConns     = map[int64]net.Conn{}
)

func fnIoTcpBind() *object.Foreign {
	return &object.Foreign{
		Name: "bind",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			addr, err := unpackString(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			port, err := unpackNumber(args[1], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
			if err != nil {
				return ctx.NewError(err.Error())
			}

			id := ctx.NextHandleID()
			ioTcpMu.Lock()
			ioTcpListeners[id] = listener
			ioTcpMu.Unlock()
			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnIoTcpAccept() *object.Foreign {
	return &object.Foreign{
		Name: "accept",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			ioTcpMu.RLock()
			listener, ok := ioTcpListeners[id]
			ioTcpMu.RUnlock()
			if !ok {
				return ctx.NewError("invalid listener ID")
			}

			conn, err := listener.Accept()
			if err != nil {
				return ctx.NewError(err.Error())
			}

			connID := ctx.NextHandleID()
			ioTcpMu.Lock()
			ioTcpConns[connID] = conn
			ioTcpMu.Unlock()
			return &object.Number{Value: dec64.FromInt64(connID)}
		},
	}
}

func fnIoTcpConnect() *object.Foreign {
	return &object.Foreign{
		Name: "connect",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			addr, err := unpackString(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			port, err := unpackNumber(args[1], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", addr, port))
			if err != nil {
				return ctx.NewError(err.Error())
			}

			id := ctx.NextHandleID()
			ioTcpMu.Lock()
			ioTcpConns[id] = conn
			ioTcpMu.Unlock()
			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnIoTcpRead() *object.Foreign {
	return &object.Foreign{
		Name: "read",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			max, err := unpackNumber(args[1], "")
			if err != nil {
				return ctx.NewError(tcpErrorMessage(id, err))
			}

			ioTcpMu.RLock()
			conn, ok := ioTcpConns[id]
			ioTcpMu.RUnlock()
			if !ok {
				return ctx.NewError("invalid conn ID")
			}

			buf := make([]byte, max)
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					return ctx.Nil()
				} else {
					return ctx.NewError(tcpErrorMessage(id, err))
				}
			}

			return &object.String{Value: string(buf[:n])}
		},
	}
}

func fnIoTcpWrite() *object.Foreign {
	return &object.Foreign{
		Name: "write",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			data, err := unpackString(args[1], "")
			if err != nil {
				return ctx.NewError(tcpErrorMessage(id, err))
			}

			ioTcpMu.RLock()
			conn, ok := ioTcpConns[id]
			ioTcpMu.RUnlock()
			if !ok {
				return ctx.NewError("invalid conn ID")
			}

			n, err := conn.Write([]byte(data))
			if err != nil {
				return ctx.NewError(tcpErrorMessage(id, err))
			}

			return &object.Number{Value: dec64.FromInt64(int64(n))}
		},
	}
}

func fnIoTcpClose() *object.Foreign {
	return &object.Foreign{
		Name: "close",
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			ioTcpMu.Lock()
			defer ioTcpMu.Unlock()

			if c, ok := ioTcpConns[id]; ok {
				c.Close()
				delete(ioTcpConns, id)
				return ctx.Nil()
			}

			if l, ok := ioTcpListeners[id]; ok {
				l.Close()
				delete(ioTcpListeners, id)
				return ctx.Nil()
			}

			return ctx.Nil()
		},
	}
}

func tcpErrorMessage(id int64, err error) string {
	return fmt.Sprintf("TCP error [%d] %s", id, err.Error())
}
