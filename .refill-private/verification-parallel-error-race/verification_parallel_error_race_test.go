package verification_parallel_error_race_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

var errBundleRead = errors.New("bundle read failed")
var errTimelineRead = errors.New("timeline read failed")

type synchronizedFailureRepository struct {
	readers sync.WaitGroup
}

func newSynchronizedFailureRepository() *synchronizedFailureRepository {
	repository := &synchronizedFailureRepository{}
	repository.readers.Add(2)
	return repository
}

func (r *synchronizedFailureRepository) Create(context.Context, string, string, string, *domain.QualificationCase) (*domain.QualificationCase, bool, error) {
	panic("unexpected Create")
}

func (r *synchronizedFailureRepository) Update(context.Context, string, int64, string, string, string, map[string]any, application.Mutation) (*domain.QualificationCase, bool, error) {
	panic("unexpected Update")
}

func (r *synchronizedFailureRepository) Get(context.Context, string) (*domain.QualificationCase, error) {
	panic("unexpected Get")
}

func (r *synchronizedFailureRepository) List(context.Context, string, int) ([]domain.QualificationCase, error) {
	panic("unexpected List")
}

func (r *synchronizedFailureRepository) Timeline(context.Context, string) ([]domain.AuditEvent, error) {
	r.readers.Done()
	r.readers.Wait()
	return nil, errTimelineRead
}

func (r *synchronizedFailureRepository) GetCredential(context.Context, string) (*domain.EligibilityCredential, error) {
	return &domain.EligibilityCredential{
		CredentialNo:     "CRED-RACE-001",
		CaseID:           "case-race-001",
		EvidenceBundleID: "bundle-race-001",
	}, nil
}

func (r *synchronizedFailureRepository) GetEvidenceBundle(context.Context, string) (*domain.EvidenceBundle, error) {
	r.readers.Done()
	r.readers.Wait()
	return nil, errBundleRead
}

func (r *synchronizedFailureRepository) IntegrityCheck(context.Context) error { return nil }
func (r *synchronizedFailureRepository) Close() error                         { return nil }

func TestVerifyCredentialParallelReadIsolation(t *testing.T) {
	service := application.NewService(newSynchronizedFailureRepository())
	if _, err := service.VerifyCredential(context.Background(), "CRED-RACE-001"); err == nil {
		t.Fatal("并行读取均失败时应返回 Repository 错误")
	}
}
