package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/store"
)

func testServer(t *testing.T) (*Server, func()) {
	t.Helper()
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return New(application.NewService(repository)), func() { repository.Close() }
}

func TestWorkbenchPageAndCreate(t *testing.T) {
	server, closeStore := testServer(t)
	defer closeStore()
	page := httptest.NewRecorder()
	server.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "种子活力鉴定工作台") {
		t.Fatalf("page response: %d %s", page.Code, page.Body.String())
	}
	body, _ := json.Marshal(map[string]any{"idempotencyKey": "http-create-001", "actor": "接收员", "accessionCode": "ACC-HTTP-01", "source": "测试资源圃", "harvestedAt": "2025-08-01", "declaredSeedCount": 500, "protocolCode": "ISTA-2025"})
	request := httptest.NewRequest(http.MethodPost, "/api/cases", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 201 || !strings.Contains(response.Body.String(), "ACC-HTTP-01") {
		t.Fatalf("create response: %d %s", response.Code, response.Body.String())
	}
}

func TestJSONIsStrict(t *testing.T) {
	server, closeStore := testServer(t)
	defer closeStore()
	request := httptest.NewRequest(http.MethodPost, "/api/cases", strings.NewReader(`{"unknown":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != 400 || !strings.Contains(response.Body.String(), "INVALID_JSON") {
		t.Fatalf("response: %d %s", response.Code, response.Body.String())
	}
}
