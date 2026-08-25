package httpapi

import (
	"net/http"

	"seed-vigor-gate/internal/application"
)

func (s *Server) RecordObservationHandler(w http.ResponseWriter, r *http.Request) {
	var command application.RecordObservationCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.RecordObservation(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	var command application.AnalyzeCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.Analyze(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) ResolveDeviationHandler(w http.ResponseWriter, r *http.Request) {
	var command application.ResolveDeviationCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.ResolveDeviation(r.Context(), r.PathValue("caseId"), r.PathValue("deviationId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) ReviewHandler(w http.ResponseWriter, r *http.Request) {
	var command application.ReviewCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.Review(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) FreezeHandler(w http.ResponseWriter, r *http.Request) {
	var command application.FreezeCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.Freeze(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, item)
}

func (s *Server) IssueCredentialHandler(w http.ResponseWriter, r *http.Request) {
	var command application.IssueCredentialCommand
	if !decodeOrProblem(w, r, &command) {
		return
	}
	fillMeta(r, &command.WriteMeta)
	item, err := s.service.IssueCredential(r.Context(), r.PathValue("caseId"), command)
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 201, item.Credential)
}

func (s *Server) GetCredentialHandler(w http.ResponseWriter, r *http.Request) {
	verification, err := s.service.VerifyCredential(r.Context(), r.PathValue("credentialNo"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, verification)
}

func (s *Server) VerifyCredentialHandler(w http.ResponseWriter, r *http.Request) {
	verification, err := s.service.VerifyCredential(r.Context(), r.PathValue("credentialNo"))
	if err != nil {
		handleError(w, r, err)
		return
	}
	writeJSON(w, 200, verification)
}
