package privileged

import (
	"html/template"
	"net/http"
	"slug/internal/kernel"
	"slug/internal/logger"
	"sort"
)

// ===== Control Plane (HTTP) =====
var log = logger.NewLogger("control plane", kernel.SystemLogLevel())

type ControlPlane struct {
	kernel      *kernel.Kernel
	replSession kernel.ActorID
}

func (c *ControlPlane) Initialize(k *kernel.Kernel) {
	c.kernel = k
	c.routes()
	addr := ":8081"
	log.Infof("listening on http://localhost%s/", addr)
	go func() { log.Error(http.ListenAndServe(addr, nil)) }()
}

func (c *ControlPlane) routes() {
	http.HandleFunc("/", c.handleIndex)
	http.HandleFunc("/favicon.png", c.handleFavicon)
	http.HandleFunc("/assets/htmx-2.0.7.min.js", c.handleHtmx)
	http.HandleFunc("/assets/milligram-1.4.1.min.css", c.handleMilligram)
	http.HandleFunc("/assets/normalize-11.0.0.css", c.handleNormalize)
	http.HandleFunc("/actors", c.handleActors)
}

func (c *ControlPlane) handleFavicon(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/slug.png")
}

func (c *ControlPlane) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/control/index.html")
}

func (c *ControlPlane) handleHtmx(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/htmx-2.0.7.min.js")
}

func (c *ControlPlane) handleNormalize(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/normalize-11.0.0.css")
}

func (c *ControlPlane) handleMilligram(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/milligram-1.4.1.min.css")
}

func (c *ControlPlane) handleActors(w http.ResponseWriter, r *http.Request) {
	c.kernel.Mu.RLock()
	defer c.kernel.Mu.RUnlock()
	type A struct {
		ID     kernel.ActorID
		Name   string
		CPUOps uint64
		IPCIn  uint64
		IPCOut uint64
		Caps   int
	}
	var out []A
	for id, a := range c.kernel.Actors {
		out = append(out, A{ID: id, Name: a.Name, CPUOps: a.CpuOps, IPCIn: a.IpcIn, IPCOut: a.IpcOut, Caps: len(a.Caps)})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	t, _ := template.ParseFiles("webroot/control/templates/actors.html")
	w.Header().Set("Content-Type", "text/html")
	err := t.Execute(w, out)
	if err != nil {
		log.Error(err.Error())
	}
}
