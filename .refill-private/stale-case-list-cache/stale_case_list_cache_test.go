package stale_case_list_cache_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/domain"
	"seed-vigor-gate/internal/httpapi"
	"seed-vigor-gate/internal/store"
)

func TestCreatedCaseInvalidatesPreviouslyReadList(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	handler := httpapi.New(application.NewService(repository)).Handler()

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/cases", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("首次列表状态码 = %d", first.Code)
	}
	var before struct {
		Data []domain.QualificationCase `json:"data"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &before); err != nil {
		t.Fatal(err)
	}
	if len(before.Data) != 0 {
		t.Fatalf("初始列表应为空，实际有 %d 项", len(before.Data))
	}

	payload := application.CreateCaseCommand{
		IdempotencyKey:    "private-create-001",
		Actor:             "接收员",
		AccessionCode:     "CACHE-001",
		Source:            "资源圃一号库",
		HarvestedAt:       "2026-08-20",
		DeclaredSeedCount: 500,
		ProtocolCode:      "ISTA-2025",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	created := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/cases", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(created, request)
	if created.Code != http.StatusCreated {
		t.Fatalf("创建状态码 = %d，响应 = %s", created.Code, created.Body.String())
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/cases", nil))
	if second.Code != http.StatusOK {
		t.Fatalf("再次列表状态码 = %d", second.Code)
	}
	var after struct {
		Data []domain.QualificationCase `json:"data"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &after); err != nil {
		t.Fatal(err)
	}
	if len(after.Data) != 1 || after.Data[0].AccessionCode != payload.AccessionCode {
		t.Fatalf("创建后列表未刷新：%+v", after.Data)
	}
}
