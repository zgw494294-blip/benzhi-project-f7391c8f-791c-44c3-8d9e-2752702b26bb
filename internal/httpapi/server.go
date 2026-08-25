package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
	"time"

	"seed-vigor-gate/internal/application"
)

//go:embed web/*
var webFiles embed.FS

type Server struct {
	service *application.Service
	handler http.Handler
}

func New(service *application.Service) *Server {
	s := &Server{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.WorkbenchPageHandler)
	mux.HandleFunc("GET /workbench", s.WorkbenchPageHandler)
	assets, _ := fs.Sub(webFiles, "web")
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	mux.HandleFunc("GET /healthz", s.HealthHandler)
	mux.HandleFunc("GET /readyz", s.ReadyHandler)
	mux.HandleFunc("POST /api/cases", s.CreateCaseHandler)
	mux.HandleFunc("GET /api/cases", s.ListCasesHandler)
	mux.HandleFunc("GET /api/cases/{caseId}", s.GetCaseHandler)
	mux.HandleFunc("GET /api/cases/{caseId}/timeline", s.TimelineHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/sampling-plan/confirm", s.ConfirmSamplingHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/observations", s.RecordObservationHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/analysis", s.AnalyzeHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/deviations/{deviationId}/resolve", s.ResolveDeviationHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/review", s.ReviewHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/freeze", s.FreezeHandler)
	mux.HandleFunc("POST /api/cases/{caseId}/credential", s.IssueCredentialHandler)
	mux.HandleFunc("GET /api/credentials/{credentialNo}", s.GetCredentialHandler)
	mux.HandleFunc("GET /api/credentials/{credentialNo}/verify", s.VerifyCredentialHandler)
	s.handler = requestMiddleware(mux)
	return s
}

func (s *Server) Handler() http.Handler { return s.handler }

func (s *Server) WorkbenchPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/workbench" {
		writeProblem(w, r, http.StatusNotFound, "NOT_FOUND", "页面不存在", "")
		return
	}
	data, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		writeProblem(w, r, 500, "ASSET_ERROR", "工作台资源不可用", "")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(200)
	_, _ = w.Write(data)
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "time": time.Now().UTC()})
}
func (s *Server) ReadyHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ready"})
}
