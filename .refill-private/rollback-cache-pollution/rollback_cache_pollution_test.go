package rollbackcachepollution_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"seed-vigor-gate/internal/domain"
	"seed-vigor-gate/internal/store"
)

func TestRollbackDoesNotLeakCachedAggregateMutation(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	item, err := domain.CreateCase(domain.NewCase{
		ID:                "case-cache-rollback",
		AccessionCode:     "CACHE-ROLLBACK-001",
		Source:            "原始资源圃",
		HarvestedAt:       "2026-08-20",
		DeclaredSeedCount: 500,
		ProtocolCode:      "ISTA-2025",
		Now:               time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	created, _, err := repository.Create(context.Background(), "cache-create-001", "case.created", "接收员", item)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(context.Background(), created.ID); err != nil {
		t.Fatal(err)
	}

	rollbackCause := errors.New("受控 mutation 失败")
	_, _, err = repository.Update(context.Background(), created.ID, created.Version, "cache-update-001", "case.test_update", "试验员", nil, func(c *domain.QualificationCase) error {
		c.Source = "未提交的污染来源"
		c.Status = domain.StatusSamplingConfirmed
		return rollbackCause
	})
	if !errors.Is(err, rollbackCause) {
		t.Fatalf("期望事务因受控错误回滚，实际错误为 %v", err)
	}

	afterRollback, err := repository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterRollback.Source != "原始资源圃" || afterRollback.Status != domain.StatusDraft {
		t.Fatalf("回滚后缓存泄漏了未提交聚合: source=%q status=%q", afterRollback.Source, afterRollback.Status)
	}
}
