package httpapi

import (
	"net/http"
	"strconv"

	"seed-vigor-gate/internal/application"
)

func (s *Server) CreateCaseHandler(w http.ResponseWriter, r *http.Request) {
	var command application.CreateCaseCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	if command.IdempotencyKey == "" {
		command.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	item, err := s.service.CreateCase(r.Context(), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/cases/"+item.ID)
	writeJSON(w, 201, item)
}

func (s *Server) ListCasesHandler(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := s.service.ListCases(r.Context(), r.URL.Query().Get("status"), limit)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, items)
}

func (s *Server) GetCaseHandler(w http.ResponseWriter, r *http.Request) {
	item, err := s.service.Workbench(r.Context(), r.PathValue("caseId"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) TimelineHandler(w http.ResponseWriter, r *http.Request) {
	events, err := s.service.Timeline(r.Context(), r.PathValue("caseId"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, events)
}

func (s *Server) ConfirmSamplingHandler(w http.ResponseWriter, r *http.Request) {
	var command application.ConfirmSamplingCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.ConfirmSampling(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func fillMeta(r *http.Request, meta *application.WriteMeta) {
	if meta.IdempotencyKey == "" {
		meta.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
}
