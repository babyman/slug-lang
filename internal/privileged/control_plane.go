package privileged

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slug/internal/kernel"
	"sort"
)

// ===== Control Plane (HTTP) =====

type ControlPlane struct {
	kernel      *kernel.Kernel
	replSession kernel.ActorID
}

type ActorMetrics struct {
	ID       kernel.ActorID
	ParentID kernel.ActorID
	Name     string
	CPUOps   uint64
	IPCIn    uint64
	IPCOut   uint64
	Caps     int
}

func (c *ControlPlane) Initialize(k *kernel.Kernel) {
	c.kernel = k
	c.routes()
	addr := ":8081"
	slog.Info("control-plane listening for connections on",
		slog.Any("url", fmt.Sprintf("http://localhost%s/", addr)))
	go func() { slog.Error("Control plane error", slog.Any("error", http.ListenAndServe(addr, nil))) }()
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
	var out []ActorMetrics
	for id, a := range c.kernel.Actors {
		out = append(out, ActorMetrics{ID: id, ParentID: a.Parent, Name: a.Name, CPUOps: a.CpuOps, IPCIn: a.IpcIn, IPCOut: a.IpcOut, Caps: len(a.Caps)})
	}
	// Build hierarchy: map parent -> children
	out = sortActorMetrics(out)
	t, _ := template.ParseFiles("webroot/control/templates/actors.html")
	w.Header().Set("Content-Type", "text/html")
	err := t.Execute(w, out)
	if err != nil {
		slog.Error("actors template error",
			slog.Any("error", err.Error()))
	}
}

func sortActorMetrics(out []ActorMetrics) []ActorMetrics {
	hierarchy := make(map[kernel.ActorID][]int)
	roots := []int{}
	for i := range out {
		if out[i].ParentID == 0 {
			roots = append(roots, i)
		} else {
			hierarchy[out[i].ParentID] = append(hierarchy[out[i].ParentID], i)
		}
	}

	// Sort children by ID within each parent
	for _, children := range hierarchy {
		sort.Slice(children, func(i, j int) bool {
			return out[children[i]].ID < out[children[j]].ID
		})
	}
	sort.Slice(roots, func(i, j int) bool {
		return out[roots[i]].ID < out[roots[j]].ID
	})

	// Flatten hierarchy depth-first
	sorted := []ActorMetrics{}
	var visit func(idx int)
	visit = func(idx int) {
		sorted = append(sorted, out[idx])
		for _, childIdx := range hierarchy[out[idx].ID] {
			visit(childIdx)
		}
	}
	for _, rootIdx := range roots {
		visit(rootIdx)
	}
	out = sorted
	return out
}
