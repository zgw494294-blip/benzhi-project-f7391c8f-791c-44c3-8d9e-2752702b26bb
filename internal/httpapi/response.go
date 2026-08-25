package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

type problem struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Field     string `json:"field,omitempty"`
	RequestID string `json:"requestId"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": value})
}
func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, message, field string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Code: code, Message: message, Field: field, RequestID: requestID(r.Context())})
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	var domainError *domain.DomainError
	switch {
	case errors.Is(err, application.ErrNotFound):
		writeProblem(w, r, 404, "NOT_FOUND", "请求的记录不存在", "")
	case errors.Is(err, application.ErrVersionConflict):
		writeProblem(w, r, 409, "VERSION_CONFLICT", "数据已被其他操作更新，请刷新后重试", "expectedVersion")
	case errors.Is(err, application.ErrIdempotencyConflict):
		writeProblem(w, r, 409, "IDEMPOTENCY_CONFLICT", "幂等键已用于其他命令", "idempotencyKey")
	case errors.As(err, &domainError):
		status := 422
		if domainError.Code == domain.CodeState || domainError.Code == domain.CodeFrozen {
			status = 409
		}
		if domainError.Code == domain.CodeNotFound {
			status = 404
		}
		writeProblem(w, r, status, string(domainError.Code), domainError.Message, domainError.Field)
	default:
		writeProblem(w, r, 500, "INTERNAL_ERROR", "服务处理失败", "")
	}
}
