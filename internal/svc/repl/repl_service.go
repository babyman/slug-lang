package repl

import (
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"reflect"
	"slug/internal/kernel"
	"slug/internal/svc"
)

// ===== REPL Service =====

type RsStart struct {
}

type RsStartResp struct {
	SessionID kernel.ActorID
}

type RsEval struct {
	SessionID kernel.ActorID
	Src       string
}

type RsEvalResp struct {
	Result string
	Error  error
}

var Operations = kernel.OpRights{
	reflect.TypeOf(RsEval{}): kernel.RightExec,
}

type ReplService struct {
	ReplSession int
	mux         *http.ServeMux
}

func NewReplService() *ReplService {
	repl := &ReplService{
		ReplSession: 0,
		mux:         http.NewServeMux(),
	}
	repl.routes()
	addr := ":8080"
	slog.Info("repl listening for connections on",
		slog.Any("url", fmt.Sprintf("http://localhost%s/", addr)))
	go func() { slog.Error("Repl service error", slog.Any("error", http.ListenAndServe(addr, repl.mux))) }()
	return repl
}

func (rs *ReplService) Handler(ctx *kernel.ActCtx, msg kernel.Message) kernel.HandlerSignal {
	switch payload := msg.Payload.(type) {
	case RsStart:
		worker := Repl{}
		workedId, _ := ctx.SpawnChild("repl", Operations, worker.Handler)
		svc.Reply(ctx, msg, RsStartResp{SessionID: workedId})
	case RsEval:
		ctx.SendAsync(payload.SessionID, msg)
	default:
		svc.Reply(ctx, msg, RsEvalResp{Error: errors.New("unknown op")})
	}
	return kernel.Continue{}
}

func (rs *ReplService) routes() {
	rs.mux.HandleFunc("/", rs.handleIndex)
	rs.mux.HandleFunc("/assets/htmx-2.0.7.min.js", rs.handleHtmx)
	rs.mux.HandleFunc("/assets/milligram-1.4.1.min.css", rs.handleMilligram)
	rs.mux.HandleFunc("/assets/normalize-11.0.0.css", rs.handleNormalize)
	rs.mux.HandleFunc("/repl", rs.handleRepl)
	rs.mux.HandleFunc("/repl/eval", rs.handleReplEval)
}

func (rs *ReplService) handleIndex(w http.ResponseWriter, r *http.Request) {
	slog.Info("handling index")
	http.ServeFile(w, r, "webroot/repl/index.html")
}

func (rs *ReplService) handleHtmx(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/htmx-2.0.7.min.js")
}

func (rs *ReplService) handleNormalize(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/normalize-11.0.0.css")
}

func (rs *ReplService) handleMilligram(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "webroot/assets/milligram-1.4.1.min.css")
}

func (rs *ReplService) handleRepl(w http.ResponseWriter, r *http.Request) {
	if rs.ReplSession == 0 {
		// create a repl
		//replID, _ = c.kernel.ActorByName(svc.ReplService)
		//res, _ := c.
	}
	var out []string
	t, _ := template.ParseFiles("webroot/repl/templates/repl.html")
	err := t.Execute(w, out)
	if err != nil {
		slog.Error("error rendering repl template", slog.Any("error", err.Error()))
	}
}

func (rs *ReplService) handleReplEval(w http.ResponseWriter, r *http.Request) {
	type A struct {
		Code string
	}
	value := r.FormValue("code")
	repl := A{
		Code: value,
	}
	t, _ := template.ParseFiles("webroot/templates/repl.html")
	err := t.Execute(w, repl)
	if err != nil {
		slog.Error("error rendering repl html", slog.Any("error", err.Error()))
	}
}
