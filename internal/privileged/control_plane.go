package privileged

import (
	"encoding/json"
	"net/http"
	"slug/internal/kernel"
	"slug/internal/logger"
	"slug/internal/svc/repl"
	"time"
)

// ===== Control Plane (HTTP) =====
var log = logger.NewLogger("control plane", kernel.SystemLogLevel())

type ControlPlane struct{ kernel *kernel.Kernel }

func (c *ControlPlane) Initialize(k *kernel.Kernel) {
	c.kernel = k
	c.routes()
	addr := ":8080"
	log.Infof("listening on %s", addr)
	go func() { log.Fatal(http.ListenAndServe(addr, nil)) }()
}

func (c *ControlPlane) routes() {
	http.HandleFunc("/actors", c.handleActors)
	http.HandleFunc("/send", c.handleSend)
	http.HandleFunc("/repl/eval", c.handleReplEval)
}

func (c *ControlPlane) handleActors(w http.ResponseWriter, r *http.Request) {
	c.kernel.Mu.RLock()
	defer c.kernel.Mu.RUnlock()
	type A struct {
		ID     kernel.ActorID          `json:"id"`
		Name   string                  `json:"name"`
		CPUOps uint64                  `json:"cpu_ops"`
		IPCIn  uint64                  `json:"ipc_in"`
		IPCOut uint64                  `json:"ipc_out"`
		Ops    kernel.OpRights         `json:"ops,omitempty"`
		Caps   []kernel.CapabilityView `json:"caps,omitempty"`
	}
	var out []A
	for id, a := range c.kernel.Actors {
		ops := c.kernel.OpsBySvc[id]
		caps := make([]kernel.CapabilityView, 0, len(a.Caps))
		for _, c := range a.Caps {
			caps = append(caps, kernel.CapabilityView{ID: c.ID, Target: c.Target, Rights: c.Rights, Revoked: c.Revoked.Load()})
		}
		out = append(out, A{ID: id, Name: a.Name, CPUOps: a.CpuOps, IPCIn: a.IpcIn, IPCOut: a.IpcOut, Ops: ops, Caps: caps})
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (c *ControlPlane) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var req sendReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: err.Error()})
		return
	}
	fromID, ok := c.kernel.ActorByName(req.From)
	if !ok {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "unknown from"})
		return
	}
	toID, ok := c.kernel.ActorByName(req.To)
	if !ok {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "unknown to"})
		return
	}
	// do a sync call through the kernel as if "from" sent it
	respCh := make(chan kernel.Message, 1)
	if err := c.kernel.SendInternal(fromID, toID, req.Payload, respCh); err != nil {
		w.WriteHeader(403)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: err.Error()})
		return
	}
	select {
	case reply := <-respCh:
		_ = json.NewEncoder(w).Encode(sendResp{OK: true, Reply: reply.Payload})
	case <-time.After(3 * time.Second):
		w.WriteHeader(504)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "timeout"})
	}
}

// REPL HTTP endpoint — clients POST {source} and the REPL actor evaluates it.
func (c *ControlPlane) handleReplEval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	var body struct {
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: err.Error()})
		return
	}
	replID, ok := c.kernel.ActorByName("repl")
	if !ok {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "REPL service missing"})
		return
	}
	// Create concrete RsEval message type for proper capability checking
	evalMsg := repl.RsEval{Source: body.Source}
	// Kernel sends as if "repl-http" actor invoked REPL; for simplicity, reuse REPL as sender
	respCh := make(chan kernel.Message, 1)
	if err := c.kernel.SendInternal(replID, replID, evalMsg, respCh); err != nil {
		w.WriteHeader(403)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: err.Error()})
		return
	}
	select {
	case reply := <-respCh:
		_ = json.NewEncoder(w).Encode(sendResp{OK: true, Reply: reply.Payload})
	case <-time.After(5 * time.Second):
		w.WriteHeader(504)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "timeout"})
	}
}

type sendReq struct {
	From    string `json:"from"` // actor name
	To      string `json:"to"`   // target actor name
	Payload any    `json:"payload"`
}

type sendResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Reply any    `json:"reply,omitempty"`
}
