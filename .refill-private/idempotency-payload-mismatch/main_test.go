package idempotencypayloadmismatch_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/httpapi"
	"seed-vigor-gate/internal/store"
)

func TestIdempotencyKeyRejectsDifferentCreatePayload(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	handler := httpapi.New(application.NewService(repository)).Handler()

	post := func(accession string) *httptest.ResponseRecorder {
		body, err := json.Marshal(map[string]any{
			"idempotencyKey":    "same-create-key-001",
			"actor":             "接收员",
			"accessionCode":     accession,
			"source":            "本地测试资源圃",
			"harvestedAt":       "2026-08-20",
			"declaredSeedCount": 500,
			"protocolCode":      "ISTA-2025",
		})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/cases", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	if first := post("IDEMP-001"); first.Code != http.StatusCreated {
		t.Fatalf("first create failed: %d %s", first.Code, first.Body.String())
	}
	second := post("IDEMP-002")
	if second.Code != http.StatusConflict || !bytes.Contains(second.Body.Bytes(), []byte("IDEMPOTENCY_CONFLICT")) {
		t.Fatalf("same key with a different payload must conflict, got %d %s", second.Code, second.Body.String())
	}
}
