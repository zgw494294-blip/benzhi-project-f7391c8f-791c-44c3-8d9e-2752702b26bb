package application

import "seed-vigor-gate/internal/domain"

func fieldError(field, message string) error {
	return &domain.DomainError{Code: domain.CodeValidation, Field: field, Message: message}
}

func validateCreateMeta(key, actor string) error {
	if len(key) < 8 || len(key) > 120 {
		return fieldError("idempotencyKey", "幂等键长度须在 8 至 120 之间")
	}
	if actor == "" {
		return fieldError("actor", "操作人不能为空")
	}
	return nil
}
