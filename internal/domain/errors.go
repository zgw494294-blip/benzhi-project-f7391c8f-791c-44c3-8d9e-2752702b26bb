package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation ErrorCode = "VALIDATION_FAILED"
	CodeConflict   ErrorCode = "VERSION_CONFLICT"
	CodeState      ErrorCode = "INVALID_STATE"
	CodeFrozen     ErrorCode = "EVIDENCE_FROZEN"
	CodeNotFound   ErrorCode = "NOT_FOUND"
)

type DomainError struct {
	Code    ErrorCode
	Field   string
	Message string
}

func (e *DomainError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
func invalid(field, message string) error {
	return &DomainError{Code: CodeValidation, Field: field, Message: message}
}
func invalidState(status Status, action string) error {
	return &DomainError{Code: CodeState, Field: "status", Message: fmt.Sprintf("状态 %s 不允许执行 %s", status, action)}
}
func frozenError() error {
	return &DomainError{Code: CodeFrozen, Message: "鉴定证据已冻结，不能再修改"}
}
