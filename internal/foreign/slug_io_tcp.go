package foreign

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"slug/internal/dec64"
	"slug/internal/object"
	"sync"
)

var (
	ioTcpListeners       = map[int64]net.Listener{}
	ioTcpConns           = map[int64]net.Conn{}
	ioTcpNextID    int64 = 1
	ioTcpMutex     sync.Mutex
)

func nextIoTcpId() int64 {
	ioTcpMutex.Lock()
	defer ioTcpMutex.Unlock()
	id := ioTcpNextID<<16 | int64(rand.Intn(0xFFFF))
	ioTcpNextID++
	return id
}

func unpackString(arg object.Object, argName string) (string, error) {

	if arg.Type() != object.STRING_OBJ {
		return "", fmt.Errorf("argument to `%s` must be a STRING, got=%s", argName, arg.Type())
	}
	value := arg.(*object.String)
	return value.Value, nil
}

func unpackNumber(arg object.Object, argName string) (int64, error) {

	if arg.Type() != object.NUMBER_OBJ {
		return -1, fmt.Errorf("argument to `%s` must be a NUMBER, got=%s", argName, arg.Type())
	}
	value := arg.(*object.Number)
	return value.Value.ToInt64(), nil
}

func fnIoTcpBind() *object.Foreign {
	return &object.Foreign{
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

			id := nextIoTcpId()
			ioTcpListeners[id] = listener
			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnIoTcpAccept() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			listener, ok := ioTcpListeners[id]
			if !ok {
				return ctx.NewError("invalid listener ID")
			}

			conn, err := listener.Accept()
			if err != nil {
				return ctx.NewError(err.Error())
			}

			connID := nextIoTcpId()
			ioTcpConns[connID] = conn
			return &object.Number{Value: dec64.FromInt64(connID)}
		},
	}
}

func fnIoTcpConnect() *object.Foreign {
	return &object.Foreign{
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

			id := nextIoTcpId()
			ioTcpConns[id] = conn
			return &object.Number{Value: dec64.FromInt64(id)}
		},
	}
}

func fnIoTcpRead() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			max, err := unpackNumber(args[1], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			conn, ok := ioTcpConns[id]
			if !ok {
				return ctx.NewError("invalid conn ID")
			}

			buf := make([]byte, max)
			n, err := conn.Read(buf)
			if err != nil {
				if err == io.EOF {
					return ctx.Nil()
				} else {
					return ctx.NewError(err.Error())
				}
			}

			return &object.String{Value: string(buf[:n])}
		},
	}
}

func fnIoTcpWrite() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			data, err := unpackString(args[1], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			conn, ok := ioTcpConns[id]
			if !ok {
				return ctx.NewError("invalid conn ID")
			}

			n, err := conn.Write([]byte(data))
			if err != nil {
				return ctx.NewError(err.Error())
			}

			return &object.Number{Value: dec64.FromInt64(int64(n))}
		},
	}
}

func fnIoTcpClose() *object.Foreign {
	return &object.Foreign{
		Fn: func(ctx object.EvaluatorContext, args ...object.Object) object.Object {

			id, err := unpackNumber(args[0], "")
			if err != nil {
				return ctx.NewError(err.Error())
			}

			if l, ok := ioTcpListeners[id]; ok {
				l.Close()
				delete(ioTcpListeners, id)
				return ctx.Nil()
			}

			if c, ok := ioTcpConns[id]; ok {
				c.Close()
				delete(ioTcpConns, id)
				return ctx.Nil()
			}

			return ctx.Nil()
		},
	}
}
