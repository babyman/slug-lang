package kernel

import (
	"encoding/json"
	"net/http"
	"time"
)

// ===== Control Plane (HTTP) =====

type ControlPlane struct{ k *Kernel }

func (h *ControlPlane) routes() {
	http.HandleFunc("/actors", h.handleActors)
	http.HandleFunc("/send", h.handleSend)
	http.HandleFunc("/repl/eval", h.handleReplEval)
}

func (h *ControlPlane) handleActors(w http.ResponseWriter, r *http.Request) {
	h.k.mu.RLock()
	defer h.k.mu.RUnlock()
	type A struct {
		ID     ActorID          `json:"id"`
		Name   string           `json:"name"`
		CPUOps uint64           `json:"cpu_ops"`
		IPCIn  uint64           `json:"ipc_in"`
		IPCOut uint64           `json:"ipc_out"`
		Ops    OpRights         `json:"ops,omitempty"`
		Caps   []CapabilityView `json:"caps,omitempty"`
	}
	var out []A
	for id, a := range h.k.actors {
		ops := h.k.opsBySvc[id]
		caps := make([]CapabilityView, 0, len(a.caps))
		for _, c := range a.caps {
			caps = append(caps, CapabilityView{ID: c.ID, Target: c.Target, Rights: c.Rights, Revoked: c.Revoked.Load()})
		}
		out = append(out, A{ID: id, Name: a.name, CPUOps: a.cpuOps, IPCIn: a.ipcIn, IPCOut: a.ipcOut, Ops: ops, Caps: caps})
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (h *ControlPlane) handleSend(w http.ResponseWriter, r *http.Request) {
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
	fromID, ok := h.k.ActorByName(req.From)
	if !ok {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "unknown from"})
		return
	}
	toID, ok := h.k.ActorByName(req.To)
	if !ok {
		w.WriteHeader(404)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "unknown to"})
		return
	}
	// do a sync call through the kernel as if "from" sent it
	respCh := make(chan Message, 1)
	if err := h.k.sendInternal(fromID, toID, req.Payload, respCh); err != nil {
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
func (h *ControlPlane) handleReplEval(w http.ResponseWriter, r *http.Request) {
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
	replID, ok := h.k.ActorByName("repl")
	if !ok {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(sendResp{OK: false, Error: "REPL service missing"})
		return
	}
	// Kernel sends as if "repl-http" actor invoked REPL; for simplicity, reuse REPL as sender
	respCh := make(chan Message, 1)
	if err := h.k.sendInternal(replID, replID, map[string]any{"source": body.Source}, respCh); err != nil {
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
	Op      string `json:"op"`
	Payload any    `json:"payload"`
}

type sendResp struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	Reply any    `json:"reply,omitempty"`
}
