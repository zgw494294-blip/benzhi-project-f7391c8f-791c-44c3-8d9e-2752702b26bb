package application

import (
	"context"
	"errors"

	"seed-vigor-gate/internal/domain"
)

var ErrNotFound = errors.New("not found")
var ErrVersionConflict = errors.New("version conflict")
var ErrIdempotencyConflict = errors.New("idempotency conflict")

type Mutation func(*domain.QualificationCase) error

type Repository interface {
	Create(context.Context, string, string, string, *domain.QualificationCase) (*domain.QualificationCase, bool, error)
	Update(context.Context, string, int64, string, string, string, map[string]any, Mutation) (*domain.QualificationCase, bool, error)
	Get(context.Context, string) (*domain.QualificationCase, error)
	List(context.Context, string, int) ([]domain.QualificationCase, error)
	Timeline(context.Context, string) ([]domain.AuditEvent, error)
	GetCredential(context.Context, string) (*domain.EligibilityCredential, error)
	GetEvidenceBundle(context.Context, string) (*domain.EvidenceBundle, error)
	IntegrityCheck(context.Context) error
	Close() error
}
