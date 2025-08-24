package service

//
//import (
//	"slug/internal/kernel"
//	"sync"
//)
//
//// InMemFS: very simple path→bytes store. ops { read: READ, write: WRITE }
//type InMemFS struct {
//	mu    sync.RWMutex
//	files map[string][]byte
//}
//
//func NewInMemFS() *InMemFS { return &InMemFS{files: make(map[string][]byte)} }
//
//func (fs *InMemFS) Behavior(ctx *kernel.ActCtx, msg kernel.Message) {
//	switch msg.Op {
//	case "write":
//		path, _ := msg.Payload["path"].(string)
//		data, _ := msg.Payload["data"].([]byte)
//		if data == nil {
//			if s, _ := msg.Payload["data"].(string); s != "" {
//				data = []byte(s)
//			}
//		}
//		fs.mu.Lock()
//		fs.files[path] = data
//		fs.mu.Unlock()
//		if msg.Resp != nil {
//			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "write.ok", Payload: map[string]any{"bytes": len(data)}}
//		}
//	case "read":
//		path, _ := msg.Payload["path"].(string)
//		fs.mu.RLock()
//		b, ok := fs.files[path]
//		fs.mu.RUnlock()
//		if !ok {
//			if msg.Resp != nil {
//				msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: map[string]any{"error": "ENOENT"}}
//			}
//			return
//		}
//		if msg.Resp != nil {
//			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "read.ok", Payload: map[string]any{"data": string(b)}}
//		}
//	default:
//		if msg.Resp != nil {
//			msg.Resp <- kernel.Message{From: ctx.Self.Id, To: msg.From, Op: "err", Payload: map[string]any{"error": "unknown op"}}
//		}
//	}
//}
