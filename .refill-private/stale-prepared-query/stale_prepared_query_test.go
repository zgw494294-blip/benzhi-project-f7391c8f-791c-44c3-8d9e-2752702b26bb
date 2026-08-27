package stale_prepared_query_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"seed-vigor-gate/internal/domain"
	"seed-vigor-gate/internal/store"
)

func TestReopenedStoreDoesNotReuseClosedPreparedQuery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	first, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	item, err := domain.CreateCase(domain.NewCase{
		ID:                "case-restart-001",
		AccessionCode:     "RESTART-001",
		Source:            "重启恢复测试资源圃",
		HarvestedAt:       "2026-08-20",
		DeclaredSeedCount: 500,
		ProtocolCode:      "ISTA-2025",
		Now:               time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = first.Create(context.Background(), "restart-create-001", "case.created", "接收员", item); err != nil {
		t.Fatal(err)
	}
	if _, err = first.Get(context.Background(), item.ID); err != nil {
		t.Fatalf("首次查询未能建立 prepared statement 缓存: %v", err)
	}
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	loaded, err := second.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("重新打开后的 Store.Get 复用了旧连接资源: %v", err)
	}
	if loaded.ID != item.ID || loaded.Version != item.Version {
		t.Fatalf("重启恢复结果错误: got=%+v want=%+v", loaded, item)
	}
}
