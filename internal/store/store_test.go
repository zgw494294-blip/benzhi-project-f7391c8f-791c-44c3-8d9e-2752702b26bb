package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
)

func TestStoreVersionAndIdempotency(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	item, err := domain.CreateCase(domain.NewCase{ID: "case-1", AccessionCode: "ACC-001", Source: "测试来源", HarvestedAt: "2024-01-02", DeclaredSeedCount: 500, ProtocolCode: "ISTA-2025", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	created, replayed, err := store.Create(context.Background(), "create-key-001", "case.created", "接收员", item)
	if err != nil || replayed {
		t.Fatalf("create failed: %v", err)
	}
	replay, replayed, err := store.Create(context.Background(), "create-key-001", "case.created", "接收员", item)
	if err != nil || !replayed || replay.ID != created.ID {
		t.Fatalf("idempotency replay failed: %v", err)
	}
	_, _, err = store.Update(context.Background(), created.ID, 99, "update-key-001", "sampling.confirmed", "接收员", nil, func(c *domain.QualificationCase) error { return nil })
	if !errors.Is(err, application.ErrVersionConflict) {
		t.Fatalf("want version conflict, got %v", err)
	}
}

func TestAuditSequence(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	item, _ := domain.CreateCase(domain.NewCase{ID: "case-2", AccessionCode: "ACC-002", Source: "测试来源", HarvestedAt: "2024-01-02", DeclaredSeedCount: 500, ProtocolCode: "ISTA-2025", Now: time.Now()})
	_, _, err = store.Create(context.Background(), "create-key-002", "case.created", "接收员", item)
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.Timeline(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Sequence < 1 {
		t.Fatalf("unexpected events: %+v", events)
	}
}
