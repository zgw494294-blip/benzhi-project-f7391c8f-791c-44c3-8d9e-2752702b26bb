package readyafterstoreclose_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"seed-vigor-gate/internal/application"
	"seed-vigor-gate/internal/httpapi"
	"seed-vigor-gate/internal/store"
)

func TestReadyFailsAfterStoreIsClosed(t *testing.T) {
	repository, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	handler := httpapi.New(application.NewService(repository)).Handler()
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("closed persistence layer must make readiness fail, got %d %s", response.Code, response.Body.String())
	}
}
