package detachedwritecontext_test

import (
	"context"
	"errors"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

type cancelAwareRepository struct {
	entered   chan struct{}
	release   chan struct{}
	persisted bool
}

func (r *cancelAwareRepository) Create(ctx context.Context, _, _, _ string, item *domain.QualificationCase) (*domain.QualificationCase, bool, error) {
	close(r.entered)
	<-r.release
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	r.persisted = true
	return item, false, nil
}

func (*cancelAwareRepository) Update(context.Context, string, int64, string, string, string, map[string]any, application.Mutation) (*domain.QualificationCase, bool, error) {
	panic("unexpected Update call")
}

func (*cancelAwareRepository) Get(context.Context, string) (*domain.QualificationCase, error) {
	panic("unexpected Get call")
}

func (*cancelAwareRepository) List(context.Context, string, int) ([]domain.QualificationCase, error) {
	panic("unexpected List call")
}

func (*cancelAwareRepository) Timeline(context.Context, string) ([]domain.AuditEvent, error) {
	panic("unexpected Timeline call")
}

func (*cancelAwareRepository) GetCredential(context.Context, string) (*domain.EligibilityCredential, error) {
	panic("unexpected GetCredential call")
}

func (*cancelAwareRepository) GetEvidenceBundle(context.Context, string) (*domain.EvidenceBundle, error) {
	panic("unexpected GetEvidenceBundle call")
}

func (*cancelAwareRepository) IntegrityCheck(context.Context) error { return nil }
func (*cancelAwareRepository) Close() error                         { return nil }

type createResult struct {
	err error
}

func TestCanceledCreateStopsBeforeRepositoryCommit(t *testing.T) {
	repository := &cancelAwareRepository{entered: make(chan struct{}), release: make(chan struct{})}
	service := application.NewService(repository)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan createResult, 1)

	go func() {
		_, err := service.CreateCase(ctx, application.CreateCaseCommand{
			IdempotencyKey:    "cancel-create-001",
			Actor:             "接收员",
			AccessionCode:     "CANCEL-001",
			Source:            "取消传播测试资源圃",
			HarvestedAt:       "2026-08-20",
			DeclaredSeedCount: 500,
			ProtocolCode:      "ISTA-2025",
		})
		done <- createResult{err: err}
	}()

	<-repository.entered
	cancel()
	close(repository.release)
	result := <-done

	if !errors.Is(result.err, context.Canceled) {
		t.Fatalf("TestCanceledCreateStopsBeforeRepositoryCommit: want context cancellation, got %v", result.err)
	}
	if repository.persisted {
		t.Fatalf("TestCanceledCreateStopsBeforeRepositoryCommit: repository committed after caller cancellation")
	}
}
